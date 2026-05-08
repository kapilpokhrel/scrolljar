package api

import (
	"errors"
	"net/http"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/spec"
)

func (app *Application) CreateScroll(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	if err := app.createScroll(w, r, id); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) createScroll(w http.ResponseWriter, r *http.Request, id spec.JarID) error {
	input := spec.CreateScrollInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		return errBadRequest(err)
	}
	if err := app.requireJarCreator(r, id); err != nil {
		return err
	}
	user := app.contextGetUser(r)
	scroll, err := app.store.InsertScroll(r.Context(), database.InsertScrollParams{
		JarID:  id,
		Title:  pgtype.Text{String: input.Title, Valid: input.Title != ""},
		Format: pgtype.Text{String: input.Format, Valid: input.Format != ""},
	})
	if err != nil {
		return err
	}
	uploadToken, err := createScrollUploadToken(scroll.ID, scroll.JarID, user)
	if err != nil {
		return err
	}
	return app.writeJSON(w, http.StatusOK, spec.CreateScrollOutput{
		Scroll:      dbScrollToSpec(scroll),
		UploadToken: uploadToken,
	}, nil)
}

func (app *Application) GetScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID, params spec.GetScrollParams) {
	if err := app.getScroll(w, r, id, params); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) getScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID, params spec.GetScrollParams) error {
	scroll, err := app.store.GetScroll(r.Context(), id)
	if err != nil {
		return dbErr(err)
	}
	if !scroll.Uploaded {
		return errNotFound
	}
	jar, err := app.store.GetJar(r.Context(), scroll.JarID)
	if err != nil {
		return dbErr(err)
	}
	if err := checkJarPassword(jar, params.XPastePassword); err != nil {
		return errInvalidJarPass
	}
	fetchURL, err := app.s3Bucket.GetScrollFetchURL(scroll.JarID, scroll.ID)
	if err != nil {
		return err
	}
	return app.writeJSON(w, http.StatusOK, spec.ScrollFetch{
		Scroll:   dbScrollToSpec(scroll),
		FetchURL: fetchURL,
	}, nil)
}

func (app *Application) PatchScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) {
	if err := app.patchScroll(w, r, id); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) patchScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) error {
	input := spec.ScrollPatchInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		return errBadRequest(err)
	}
	scroll, err := app.store.GetScroll(r.Context(), id)
	if err != nil {
		return dbErr(err)
	}
	if !scroll.Uploaded {
		return errNotFound
	}
	if err := app.requireJarCreator(r, scroll.JarID); err != nil {
		return err
	}
	if input.Title != nil {
		scroll.Title = pgtype.Text{String: *input.Title, Valid: true}
	}
	if input.Format != nil {
		scroll.Format = pgtype.Text{String: *input.Format, Valid: true}
	}
	updatedAt, err := app.store.UpdateScroll(r.Context(), database.UpdateScrollParams{
		Title:     scroll.Title,
		Format:    scroll.Format,
		ID:        scroll.ID,
		UpdatedAt: scroll.UpdatedAt,
	})
	if err != nil {
		return dbErrWithConflict(err)
	}
	scroll.UpdatedAt = updatedAt
	user := app.contextGetUser(r)
	uploadToken, err := createScrollUploadToken(scroll.ID, scroll.JarID, user)
	if err != nil {
		return err
	}
	return app.writeJSON(w, http.StatusOK, spec.CreateScrollOutput{
		Scroll:      dbScrollToSpec(scroll),
		UploadToken: uploadToken,
	}, nil)
}

func (app *Application) DeleteScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) {
	if err := app.deleteScroll(w, r, id); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) deleteScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) error {
	scroll, err := app.store.GetScroll(r.Context(), id)
	if err != nil {
		return dbErr(err)
	}
	if err := app.requireJarCreator(r, scroll.JarID); err != nil {
		return err
	}
	if err := app.store.DeleteScroll(r.Context(), id); err != nil {
		return err
	}
	return app.writeJSON(w, http.StatusOK, spec.Message{Message: "scroll deleted successfully"}, nil)
}

func (app *Application) UploadScroll(w http.ResponseWriter, r *http.Request, params spec.UploadScrollParams) {
	if err := app.uploadScroll(w, r, params); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) uploadScroll(w http.ResponseWriter, r *http.Request, params spec.UploadScrollParams) error {
	scrollID, jarID, userID, err := verifyScrollUploadToken(params.XUploadToken)
	if err != nil {
		return errNotFound
	}
	scroll, err := app.store.GetScroll(r.Context(), scrollID)
	if err != nil {
		return dbErr(err)
	}
	if scroll.Uploaded {
		return errAlreadyUploaded
	}

	var maxSize int64 = 1 * 1024 * 1024
	if userID >= 0 {
		maxSize = 5 * 1024 * 1024
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSize+1)

	key := filepath.Join(jarID, scrollID)
	_, err = app.s3Bucket.StreamingUpload(utf8ValidationReader{r: r.Body}, key)
	if err != nil {
		if errors.Is(err, utf8Err) {
			return errBadRequest(errors.New("invalid text content"))
		}
		if errors.Is(err, http.ErrBodyReadAfterClose) {
			return errEntityTooLarge
		}
		return err
	}

	updatedAt, err := app.store.SetScrollUploaded(r.Context(), database.SetScrollUploadedParams{
		ID:        scroll.ID,
		UpdatedAt: scroll.UpdatedAt,
	})
	if err != nil {
		return dbErrWithConflict(err)
	}
	scroll.UpdatedAt = updatedAt
	scroll.Uploaded = true

	fetchURL, err := app.s3Bucket.GetScrollFetchURL(jarID, scrollID)
	if err != nil {
		return err
	}
	return app.writeJSON(w, http.StatusOK, spec.ScrollFetch{
		Scroll:   dbScrollToSpec(scroll),
		FetchURL: fetchURL,
	}, nil)
}
