package api

import (
	"errors"
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) updatePatchHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
		Format  *string `json:"format"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	slugs, err := app.readSlugParam(r)
	if err != nil || slugs.scrollID == 0 {
		app.notFoundResponse(w, r)
		return
	}

	jar := database.ScrollJar{
		ID: slugs.jarID,
	}
	scroll := database.Scroll{
		ID:  slugs.scrollID,
		Jar: &jar,
	}

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
			app.errorResponse(w, r, http.StatusConflict, "edit config; please try again")
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	app.getScrollURI(&scroll)

	env := envelope{"scroll": scroll}

	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
