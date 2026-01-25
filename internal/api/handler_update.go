package api

import (
	"errors"
	"net/http"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) PatchScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) {
	input := spec.ScrollPatchInput{}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	scroll := database.Scroll{}
	scroll.ID = id

	err = app.models.ScrollJar.GetScroll(&scroll)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if !app.verifyJarCreator(scroll.JarID, w, r) {
		app.invalidCredentialsResponse(w, r)
		return
	}

	if input.Title != nil {
		scroll.Title = *input.Title
	}
	if input.Format != nil {
		scroll.Format = *input.Format
	}

	err = app.models.ScrollJar.UpdateScroll(&scroll)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrEditConflict):
			app.errorResponse(w, r, http.StatusConflict, spec.Error{Error: "edit confict; please try again"})
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	app.getScrollURI(&scroll)

	user := app.contextGetUser(r)
	uploadToken, err := createScrollUploadToken(&scroll, user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	err = app.writeJSON(w, http.StatusOK, spec.CreateScrollOutput{
		Scroll:      scroll.Scroll,
		UploadToken: uploadToken,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
