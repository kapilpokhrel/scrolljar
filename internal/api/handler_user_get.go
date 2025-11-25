package api

import (
	"errors"
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) getUsersJarHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r) // user == nil check happens by requried authenticated user middlware

	jars, err := app.models.ScrollJar.GetAllByUserID(user.ID)
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

	env := envelope{"user": user, "jars": jars}

	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
