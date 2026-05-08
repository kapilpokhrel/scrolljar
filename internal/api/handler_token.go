package api

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/kapilpokhrel/scrolljar/internal/spec"
)

func (app *Application) CreateActivationToken(w http.ResponseWriter, r *http.Request) {
	input := spec.LoginInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := input.Validate()
	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	user, err := app.store.GetUserByEmail(r.Context(), string(input.Email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	if !verifyHashPassword(input.Password, user.PasswordHash) {
		app.invalidCredentialsResponse(w, r)
		return
	}
	if user.Activated {
		app.errorResponse(w, r, http.StatusServiceUnavailable, spec.Error{Error: "account already activated"})
		return
	}

	tokenText, expiry, err := app.store.CreateActivationToken(r.Context(), user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, spec.Token{Token: tokenText, Expiry: expiry}, nil)
}
