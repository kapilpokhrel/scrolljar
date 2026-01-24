package api

import (
	"errors"
	"net/http"
	"time"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) GetJar(w http.ResponseWriter, r *http.Request, id spec.JarID, params spec.GetJarParams) {
	jar := database.ScrollJar{}
	jar.ID = id
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

	if jar.ExpiresAt.Time.Before(time.Now()) {
		app.notFoundResponse(w, r)
		return
	}

	if jar.Access == spec.AccessPrivate {
		passwordHeader := params.XPastePassword
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
	err = app.writeJSON(w, http.StatusOK, jar.Jar, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetJarScrolls(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	jar := database.ScrollJar{}
	jar.ID = id

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

	if jar.ExpiresAt.Time.Before(time.Now()) {
		app.notFoundResponse(w, r)
		return
	}

	scrolls, err := app.models.ScrollJar.GetAllScrolls(&jar)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	for i := range len(scrolls) {
		app.getScrollURI(scrolls[i])
	}

	outputScrolls := make([]spec.Scroll, len(scrolls))
	for i, scroll := range scrolls {
		outputScrolls[i] = scroll.Scroll
	}
	err = app.writeJSON(w, http.StatusOK, outputScrolls, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID, params spec.GetScrollParams) {
	scroll := database.Scroll{}
	scroll.ID = id

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

	if !scroll.Uploaded {
		app.notFoundResponse(w, r)
		return
	}

	jar := database.ScrollJar{}
	jar.ID = scroll.JarID

	err = app.models.ScrollJar.Get(&jar)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if jar.ExpiresAt.Time.Before(time.Now()) {
		app.notFoundResponse(w, r)
		return
	}

	if jar.Access == spec.AccessPrivate {
		passwordHeader := params.XPastePassword
		if passwordHeader == "" {
			app.invalidJarPassword(w, r)
			return
		}

		if !verifyHashPassword(passwordHeader, *jar.PasswordHash) {
			app.invalidJarPassword(w, r)
			return
		}
	}

	app.getScrollURI(&scroll)
	fetchURL, err := app.getScrollFetchURL(&scroll)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(
		w,
		http.StatusOK,
		spec.ScrollFetch{Scroll: scroll.Scroll, FetchURL: fetchURL},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
