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
	input := spec.CreateScrollInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if !app.requireJarCreator(w, r, id) {
		return
	}

	user := app.contextGetUser(r)
	scroll, err := app.store.InsertScroll(r.Context(), database.InsertScrollParams{
		JarID:  id,
		Title:  pgtype.Text{String: input.Title, Valid: input.Title != ""},
		Format: pgtype.Text{String: input.Format, Valid: input.Format != ""},
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	uploadToken, err := createScrollUploadToken(scroll.ID, scroll.JarID, user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, spec.CreateScrollOutput{
		Scroll:      dbScrollToSpec(scroll),
		UploadToken: uploadToken,
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID, params spec.GetScrollParams) {
	scroll, err := app.store.GetScroll(r.Context(), id)
	if app.handleDBErr(w, r, err) {
		return
	}
	if !scroll.Uploaded {
		app.notFoundResponse(w, r)
		return
	}

	jar, err := app.store.GetJar(r.Context(), scroll.JarID)
	if app.handleDBErr(w, r, err) {
		return
	}
	if err := checkJarPassword(jar, params.XPastePassword); err != nil {
		app.invalidJarPassword(w, r)
		return
	}

	fetchURL, err := app.s3Bucket.GetScrollFetchURL(scroll.JarID, scroll.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if err := app.writeJSON(w, http.StatusOK, spec.ScrollFetch{
		Scroll:   dbScrollToSpec(scroll),
		FetchURL: fetchURL,
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) PatchScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) {
	input := spec.ScrollPatchInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	scroll, err := app.store.GetScroll(r.Context(), id)
	if app.handleDBErr(w, r, err) {
		return
	}
	if !scroll.Uploaded {
		app.notFoundResponse(w, r)
		return
	}
	if !app.requireJarCreator(w, r, scroll.JarID) {
		return
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
	if app.handleDBErrWithConflict(w, r, err) {
		return
	}
	scroll.UpdatedAt = updatedAt

	user := app.contextGetUser(r)
	uploadToken, err := createScrollUploadToken(scroll.ID, scroll.JarID, user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, spec.CreateScrollOutput{
		Scroll:      dbScrollToSpec(scroll),
		UploadToken: uploadToken,
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) DeleteScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) {
	scroll, err := app.store.GetScroll(r.Context(), id)
	if app.handleDBErr(w, r, err) {
		return
	}
	if !app.requireJarCreator(w, r, scroll.JarID) {
		return
	}
	if err := app.store.DeleteScroll(r.Context(), id); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if err := app.writeJSON(w, http.StatusOK, spec.Message{Message: "scroll deleted successfully"}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) UploadScroll(w http.ResponseWriter, r *http.Request, params spec.UploadScrollParams) {
	scrollID, jarID, userID, err := verifyScrollUploadToken(params.XUploadToken)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	scroll, err := app.store.GetScroll(r.Context(), scrollID)
	if app.handleDBErr(w, r, err) {
		return
	}
	if scroll.Uploaded {
		app.errorResponse(w, r, http.StatusConflict, spec.Error{Error: "already uploaded"})
		return
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
			app.badRequestResponse(w, r, errors.New("invalid text content"))
			return
		}
		if errors.Is(err, http.ErrBodyReadAfterClose) {
			app.entityTooLarge(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	updatedAt, err := app.store.SetScrollUploaded(r.Context(), database.SetScrollUploadedParams{
		ID:        scroll.ID,
		UpdatedAt: scroll.UpdatedAt,
	})
	if app.handleDBErrWithConflict(w, r, err) {
		return
	}
	scroll.UpdatedAt = updatedAt
	scroll.Uploaded = true

	fetchURL, err := app.s3Bucket.GetScrollFetchURL(jarID, scrollID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if err := app.writeJSON(w, http.StatusOK, spec.ScrollFetch{
		Scroll:   dbScrollToSpec(scroll),
		FetchURL: fetchURL,
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
