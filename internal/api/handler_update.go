package api

import (
	"errors"
	"net/http"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) PatchScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) {
	user := app.contextGetUser(r)
	if user == nil {
		app.invalidCredentialsResponse(w, r)
		return
	}

	input := spec.ScrollPatch{}

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
	if input.Content != nil {
		scroll.Content = *input.Content
	}

	err = app.models.ScrollJar.UpdateScroll(&scroll)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrEditConflict):
			app.errorResponse(w, r, http.StatusConflict, spec.Error{Error: "edit config; please try again"})
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	app.getScrollURI(&scroll)
	err = app.writeJSON(w, http.StatusOK, scroll.Scroll, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
