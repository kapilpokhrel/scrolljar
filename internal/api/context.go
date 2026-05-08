package api

import (
	"context"
	"net/http"

	"github.com/kapilpokhrel/scrolljar/internal/database"
)

type contextKey string // using builtin type for contextKey is discouraged as it can collide with other beyond our code

const userContextKey = contextKey("user")

func (app *Application) contextSetUser(r *http.Request, user *database.UserAccount) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func (app *Application) contextGetUser(r *http.Request) *database.UserAccount {
	user, ok := r.Context().Value(userContextKey).(*database.UserAccount)
	if !ok {
		panic("missing user in request context")
	}
	return user
}
