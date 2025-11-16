package api

import (
	"errors"
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) deleteScrollJarHandler(w http.ResponseWriter, r *http.Request) {
	id := app.readIDParam(r)
	jar := database.ScrollJar{
		ID: id,
	}

	if !app.verifyJarCreator(id, w, r) {
		app.invalidAuthenticationTokenResponse(w, r)
		return
	}

	err := app.models.ScrollJar.Delete(&jar)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	env := envelope{"message": "scrolljar deleted sucessfully"}
	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) deleteScrollHandler(w http.ResponseWriter, r *http.Request) {
	id := app.readIDParam(r)
	scroll := database.Scroll{
		ID: id,
	}

	err := app.models.ScrollJar.GetScroll(&scroll)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if !app.verifyJarCreator(id, w, r) {
		app.invalidAuthenticationTokenResponse(w, r)
		return
	}

	err = app.models.ScrollJar.DeleteScroll(&scroll)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	env := envelope{"message": "scroll deleted sucessfully"}
	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
