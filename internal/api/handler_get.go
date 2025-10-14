package api

import (
	"errors"
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) GetScrollJarHandler(w http.ResponseWriter, r *http.Request) {
	slugs, err := app.readSlugParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
	}

	jar := database.ScrollJar{
		Slug: slugs.jarSlug,
	}
	err = app.models.ScrollJar.Get(&jar)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	app.getJarURI(&jar)

	scrolls, err := app.models.ScrollJar.GetAllScrolls(&jar)
	if err != nil {
		app.serverErrorResponse(w, r, err)
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
