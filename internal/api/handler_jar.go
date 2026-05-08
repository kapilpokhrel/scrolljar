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
	input := spec.CreateJarInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)
	v := input.Validate(user != nil)
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
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
		app.serverErrorResponse(w, r, err)
		return
	}

	createdScrolls := make([]spec.CreateScrollOutput, len(scrolls))
	for i, scroll := range scrolls {
		uploadToken, err := createScrollUploadToken(scroll.ID, scroll.JarID, user)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		createdScrolls[i] = spec.CreateScrollOutput{
			Scroll:      dbScrollToSpec(scroll),
			UploadToken: uploadToken,
		}
	}

	if err := app.writeJSON(w, http.StatusOK, spec.CreateJarOutput{
		Jar:     dbJarToSpec(jar),
		Scrolls: createdScrolls,
	}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetJar(w http.ResponseWriter, r *http.Request, id spec.JarID, params spec.GetJarParams) {
	jar, err := app.store.GetJar(r.Context(), id)
	if app.handleDBErr(w, r, err) {
		return
	}

	if err := checkJarPassword(jar, params.XPastePassword); err != nil {
		app.invalidCredentialsResponse(w, r)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, dbJarToSpec(jar), nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetJarScrolls(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	if _, err := app.store.GetJar(r.Context(), id); app.handleDBErr(w, r, err) {
		return
	}

	scrolls, err := app.store.GetScrollsByJar(r.Context(), id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	out := make([]spec.Scroll, len(scrolls))
	for i, s := range scrolls {
		out[i] = dbScrollToSpec(s)
	}
	if err := app.writeJSON(w, http.StatusOK, out, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) DeleteJar(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	if !app.requireJarCreator(w, r, id) {
		return
	}
	if err := app.store.DeleteJar(r.Context(), id); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if err := app.writeJSON(w, http.StatusOK, spec.Message{Message: "scrolljar deleted successfully"}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
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
