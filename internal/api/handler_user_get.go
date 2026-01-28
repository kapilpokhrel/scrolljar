package api

import (
	"errors"
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) GetUser(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r) // user == nil check happens by requried authenticated user middlware
	if err := app.writeJSON(w, http.StatusOK, user.User, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetUserJars(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r) // user == nil check happens by requried authenticated user middlware

	jars, err := app.models.ScrollJar.GetAllByUserID(r.Context(), user.ID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
		default:
			app.serverErrorResponse(w, r, err)
		}
	}

	for i := range jars {
		app.getJarURI(jars[i])
	}

	outputJars := make([]spec.Jar, len(jars))
	for i, jar := range jars {
		outputJars[i] = jar.Jar
	}

	if err := app.writeJSON(w, http.StatusOK, outputJars, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
