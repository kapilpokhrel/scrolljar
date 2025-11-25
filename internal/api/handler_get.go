package api

import (
	"errors"
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) getScrollJarHandler(w http.ResponseWriter, r *http.Request) {
	id := app.readIDParam(r)
	jar := database.ScrollJar{
		ID: id,
	}
	err := app.models.ScrollJar.Get(&jar)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if jar.Access == database.AccessPrivate {
		passwordHeader := r.Header.Get("X-Paste-Password")
		if passwordHeader == "" {
			app.invalidCredentialsResponse(w, r)
			return
		}

		if !verifyHashPassword(passwordHeader, *jar.PasswordHash) {
			app.invalidCredentialsResponse(w, r)
			return
		}
	}

	app.getJarURI(&jar)

	scrolls, err := app.models.ScrollJar.GetAllScrolls(&jar)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	for i := range len(scrolls) {
		app.getScrollURI(scrolls[i])
	}

	env := envelope{"scrolljar": jar}
	env["scrolls"] = scrolls

	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) getScrollHandler(w http.ResponseWriter, r *http.Request) {
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

	jar := database.ScrollJar{
		ID: scroll.JarID,
	}

	err = app.models.ScrollJar.Get(&jar)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if jar.Access == database.AccessPrivate {
		passwordHeader := r.Header.Get("X-Paste-Password")
		if passwordHeader == "" {
			app.invalidCredentialsResponse(w, r)
			return
		}

		if !verifyHashPassword(passwordHeader, *jar.PasswordHash) {
			app.invalidCredentialsResponse(w, r)
			return
		}
	}

	app.getScrollURI(&scroll)

	env := envelope{"scroll": scroll}

	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
