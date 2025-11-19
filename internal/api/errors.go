package api

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

// This file serves as wrapper to send json error

func (app *Application) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)
	app.logger.Error(err.Error(), "method", method, "uri", uri)
}

func (app *Application) errorResponse(w http.ResponseWriter, r *http.Request, status int, msg any) {
	env := envelope{"error": msg}

	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

func (app *Application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	app.logger.Error(string(debug.Stack()))
	app.errorResponse(w, r, http.StatusInternalServerError, err.Error())
}

func (app *Application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("the %s method is not supported", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, msg)
}

func (app *Application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	msg := "resources not found"
	app.errorResponse(w, r, http.StatusNotFound, msg)
}

func (app *Application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (app *Application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (app *Application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}

func (app *Application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWWW-Authenticate", "Bearer")

	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (app *Application) globalMaxRateResponse(w http.ResponseWriter, r *http.Request) {
	message := "Server is on high load. Please try again later"
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

func (app *Application) ipMaxRateResponse(w http.ResponseWriter, r *http.Request) {
	message := "Too many requests from your ip"
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}
