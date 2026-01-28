package api

import (
	"errors"
	"net/http"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) DeleteJar(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	jar := database.ScrollJar{}
	jar.ID = id

	if !app.verifyJarCreator(id, w, r) {
		app.invalidCredentialsResponse(w, r)
		return
	}

	if err := app.models.ScrollJar.Delete(r.Context(), &jar); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	payload := spec.Message{
		Message: "scrolljar deleted sucessfully",
	}
	if err := app.writeJSON(w, http.StatusOK, payload, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) DeleteScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID) {
	scroll := database.Scroll{}
	scroll.ID = id

	if err := app.models.ScrollJar.GetScroll(r.Context(), &scroll); err != nil {
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

	if err := app.models.ScrollJar.DeleteScroll(r.Context(), &scroll); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	payload := spec.Message{
		Message: "scroll deleted sucessfully",
	}
	if err := app.writeJSON(w, http.StatusOK, payload, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
