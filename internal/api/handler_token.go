package api

import (
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/spec"
)

func (app *Application) CreateActivationToken(w http.ResponseWriter, r *http.Request) {
	if err := app.createActivationToken(w, r); err != nil {
		app.handleError(w, r, err)
	}
}

func (app *Application) createActivationToken(w http.ResponseWriter, r *http.Request) error {
	input := spec.LoginInput{}
	if err := app.readJSON(w, r, &input); err != nil {
		return errBadRequest(err)
	}
	v := input.Validate()
	if !v.Valid() {
		return errValidation(spec.ValidationError(*v))
	}

	user, err := app.store.GetUserByEmail(r.Context(), string(input.Email))
	if err != nil {
		return dbErr(err)
	}
	if !verifyHashPassword(input.Password, user.PasswordHash) {
		return errInvalidCreds
	}
	if user.Activated {
		return errAlreadyActivated
	}

	tokenText, expiry, err := app.store.CreateActivationToken(r.Context(), user.ID)
	if err != nil {
		return err
	}
	return app.writeJSON(w, http.StatusOK, spec.Token{Token: tokenText, Expiry: expiry}, nil)
}
