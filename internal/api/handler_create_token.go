package api

import (
	"errors"
	"net/http"
	"time"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
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

	user := &database.User{
		Email: string(input.Email),
	}

	if err := app.models.Users.GetUserByEmail(r.Context(), user); err != nil {
		switch {
		case errors.Is(err, database.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
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

	tokenText, token := generateToken(user.ID, database.ScopeActivation, time.Minute*5)

	if err := app.models.Token.Insert(r.Context(), token); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	payload := spec.Token{
		Token:  tokenText,
		Expiry: token.ExpiresAt.Time.UTC(),
	}

	app.writeJSON(w, http.StatusOK, payload, nil)
}
