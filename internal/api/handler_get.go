package api

import (
	"errors"
	"net/http"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) GetJar(w http.ResponseWriter, r *http.Request, id spec.JarID, params spec.GetJarParams) {
	jar := database.ScrollJar{}
	jar.ID = id
	if err := app.models.ScrollJar.Get(r.Context(), &jar); err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
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
	if err := app.writeJSON(w, http.StatusOK, jar.Jar, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetJarScrolls(w http.ResponseWriter, r *http.Request, id spec.JarID) {
	jar := database.ScrollJar{}
	jar.ID = id

	if err := app.models.ScrollJar.Get(r.Context(), &jar); err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	scrolls, err := app.models.ScrollJar.GetAllScrolls(r.Context(), &jar)
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
	if err := app.writeJSON(w, http.StatusOK, outputScrolls, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) GetScroll(w http.ResponseWriter, r *http.Request, id spec.ScrollID, params spec.GetScrollParams) {
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
	if !scroll.Uploaded {
		app.notFoundResponse(w, r)
		return
	}

	jar := database.ScrollJar{}
	jar.ID = scroll.JarID

	if err := app.models.ScrollJar.Get(r.Context(), &jar); err != nil {
		app.serverErrorResponse(w, r, err)
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

	if err := app.writeJSON(
		w,
		http.StatusOK,
		spec.ScrollFetch{Scroll: scroll.Scroll, FetchURL: fetchURL},
		nil,
	); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
