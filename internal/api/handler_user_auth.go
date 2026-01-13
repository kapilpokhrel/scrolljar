package api

import (
	"errors"
	"net/http"
	"time"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
)

func (app *Application) AuthUser(w http.ResponseWriter, r *http.Request) {
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

	user := &database.User{}
	user.Email = string(input.Email)

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

	if !user.Activated {
		app.inactiveAccountResponse(w, r)
		return
	}

	authTokenText, authToken := generateToken(user.ID, database.ScopeAuthorization, time.Hour*24)
	refreshTokenText, refreshToken := generateToken(user.ID, database.ScopeRefresh, time.Hour*24*30) // 30 days

	err = app.models.Token.Insert(authToken)
	if err != nil {
		// TODO
		return
	}
	err = app.models.Token.Insert(refreshToken)
	if err != nil {
		// TODO
		return
	}

	payload := spec.AuthTokens{
		Authorization: spec.Token{
			Token:  authTokenText,
			Expiry: authToken.ExpiresAt.Time.UTC(),
		},
		Refresh: spec.Token{
			Token:  refreshTokenText,
			Expiry: refreshToken.ExpiresAt.Time.UTC(),
		},
	}
	app.writeJSON(w, http.StatusOK, payload, nil)
}
