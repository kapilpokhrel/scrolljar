package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/validator"
)

func (app *Application) postUserActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	v.Check(
		validator.Matches(input.Email, validator.EmailReg),
		"email",
		"must be a valid email address",
	)
	v.Check(
		len(input.Password) >= 8 && len(input.Password) <= 72,
		"password",
		"password must be atleast 8 characters long atmost 72 characters long",
	)

	if !v.Valid() {
		app.errorResponse(w, r, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	user := &database.User{
		Email: input.Email,
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
		app.errorResponse(w, r, http.StatusServiceUnavailable, "account already activated")
		return
	}

	tokenText, token := generateToken(user.ID, database.ScopeActivation, time.Minute*5)

	err = app.models.Token.Insert(token)
	if err != nil {
		// TODO
		return
	}

	app.writeJSON(w, http.StatusOK, envelope{"token": tokenText, "expiry": token.ExpiresAt.Time.UTC()}, nil)
}
