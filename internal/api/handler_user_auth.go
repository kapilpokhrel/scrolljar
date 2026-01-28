package api

import (
	"errors"
	"net/http"
	"time"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) AuthUser(w http.ResponseWriter, r *http.Request) {
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

	user := &database.User{}
	user.Email = string(input.Email)

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

	if !user.Activated {
		app.inactiveAccountResponse(w, r)
		return
	}

	authTokenText, authToken := generateToken(user.ID, database.ScopeAuthorization, time.Hour*24)
	refreshTokenText, refreshToken := generateToken(user.ID, database.ScopeRefresh, time.Hour*24*30) // 30 days

	tx, err := app.models.ScrollJar.GetTx(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	if err = app.models.Token.InsertTx(r.Context(), tx, authToken); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if err := app.models.Token.InsertTx(r.Context(), tx, refreshToken); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		app.serverErrorResponse(w, r, err)
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
