package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/spec"
)

const durYear = time.Hour * 25 * 365

func (app *Application) CreateJar(w http.ResponseWriter, r *http.Request) {
	if err := app.createJar(w, r); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) createJar(w http.ResponseWriter, r *http.Request) error {
	input := spec.CreateJarInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		return errBadRequest(err)
	}

	user := app.contextGetUser(r)
	v := input.Validate(user != nil)
	if !v.Valid() {
		return errValidation(spec.ValidationError(*v))
	}

	jarArg := buildInsertJarParams(input, user)
	scrollArgs := make([]database.InsertScrollParams, len(input.Scrolls))
	for i, s := range input.Scrolls {
		scrollArgs[i] = database.InsertScrollParams{
			Title:  pgtype.Text{String: s.Title, Valid: s.Title != ""},
			Format: pgtype.Text{String: s.Format, Valid: s.Format != ""},
		}
	}

	jar, scrolls, err := app.store.CreateJarWithScrolls(r.Context(), jarArg, scrollArgs)
	if err != nil {
		return err
	}

	createdScrolls := make([]spec.CreateScrollOutput, len(scrolls))
	for i, scroll := range scrolls {
		uploadToken, err := createScrollUploadToken(scroll.ID, scroll.JarID, user)
		if err != nil {
			return err
		}
		createdScrolls[i] = spec.CreateScrollOutput{
			Scroll:      dbScrollToSpec(scroll),
			UploadToken: uploadToken,
		}
	}

	return app.writeJSON(w, http.StatusOK, spec.CreateJarOutput{
		Jar:     dbJarToSpec(jar),
		Scrolls: createdScrolls,
	}, nil)
}

func (app *Application) GetJar(w http.ResponseWriter, r *http.Request, id spec.JarID, params spec.GetJarParams) {
	if err := app.getJar(w, r, id, params); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) getJar(w http.ResponseWriter, r *http.Request, id spec.JarID, params spec.GetJarParams) error {
	jar, err := app.store.GetJar(r.Context(), id)
	if err != nil {
		return dbErr(err)
	}
	if err := checkJarPassword(jar, params.XPastePassword); err != nil {
		return errInvalidCreds
	}
	return app.writeJSON(w, http.StatusOK, dbJarToSpec(jar), nil)
}

func (app *Application) GetJarScrolls(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	if err := app.getJarScrolls(w, r, id); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) getJarScrolls(w http.ResponseWriter, r *http.Request, id spec.JarID) error {
	if _, err := app.store.GetJar(r.Context(), id); err != nil {
		return dbErr(err)
	}
	scrolls, err := app.store.GetScrollsByJar(r.Context(), id)
	if err != nil {
		return err
	}
	out := make([]spec.Scroll, len(scrolls))
	for i, s := range scrolls {
		out[i] = dbScrollToSpec(s)
	}
	return app.writeJSON(w, http.StatusOK, out, nil)
}

func (app *Application) DeleteJar(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	if err := app.deleteJar(w, r, id); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) deleteJar(w http.ResponseWriter, r *http.Request, id spec.JarID) error {
	if err := app.requireJarCreator(r, id); err != nil {
		return err
	}
	if err := app.store.DeleteJar(r.Context(), id); err != nil {
		return err
	}
	return app.writeJSON(w, http.StatusOK, spec.Message{Message: "scrolljar deleted successfully"}, nil)
}

func buildInsertJarParams(input spec.CreateJarInput, user *database.UserAccount) database.InsertJarParams {
	arg := database.InsertJarParams{
		Name:      pgtype.Text{String: input.Name, Valid: input.Name != ""},
		Access:    int16(input.Access),
		Tags:      input.Tags,
		ExpiresAt: jarExpiryFromInput(input, user != nil),
	}
	if input.Password != "" {
		hash, _ := hashPassword(input.Password)
		arg.PasswordHash = pgtype.Text{String: hash, Valid: true}
	}
	if user != nil {
		arg.UserID = pgtype.Int8{Int64: user.ID, Valid: true}
	}
	return arg
}
