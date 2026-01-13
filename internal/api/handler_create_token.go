package api

import (
	"errors"
	"net/http"
	"time"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
)

func (app *Application) CreateActivationToken(w http.ResponseWriter, r *http.Request) {
	input := spec.Auth{}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(
		validator.Matches(string(input.Email), validator.EmailReg),
		"email",
		"must be a valid email address",
	)
	v.Check(
		len(input.Password) >= 8 && len(input.Password) <= 72,
		"password",
		"password must be atleast 8 characters long atmost 72 characters long",
	)

	if !v.Valid() {
		app.validationErrorResponse(w, r, spec.ValidationError(*v))
		return
	}

	user := &database.User{
		Email: string(input.Email),
	}

	err = app.models.Users.GetUserByEmail(user)
	if err != nil {
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

	err = app.models.Token.Insert(token)
	if err != nil {
		// TODO
		return
	}

	payload := spec.Token{
		Token:  tokenText,
		Expiry: token.ExpiresAt.Time.UTC(),
	}

	app.writeJSON(w, http.StatusOK, payload, nil)
}
