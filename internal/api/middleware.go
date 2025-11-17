package api

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"

	"github.com/kapilpokhrel/scrolljar/internal/database"
)

func (app *Application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error("Internal server error", "error", err)
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, errors.New("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *Application) authenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, app.contextSetUser(r, nil))
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
		}

		tokenHash := sha256.Sum256([]byte(headerParts[1]))

		token := &database.Token{
			TokenHash: tokenHash[:],
		}

		err := app.models.Token.GetTokenByHash(token)
		if err != nil {
			switch {
			case errors.Is(err, database.ErrNoRecord):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)

			}
			return
		}

		user := &database.User{
			ID: token.UserID,
		}
		err = app.models.Users.GetByID(user)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		next.ServeHTTP(w, app.contextSetUser(r, user))
	})
}

func (app *Application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if user == nil {
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
}
