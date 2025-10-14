package api

import (
	"errors"
	"net/http"
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
