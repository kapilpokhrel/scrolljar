package api

import (
	"fmt"
	"net/http"
	"runtime/debug"

	spec "github.com/kapilpokhrel/scrolljar/internal/api/spec"
)

// This file serves as wrapper to send json error

func (app *Application) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)
	app.logger.Error(err.Error(), "method", method, "uri", uri)
}

func (app *Application) errorResponse(w http.ResponseWriter, r *http.Request, status int, msg spec.Error) {
	err := app.writeJSON(w, status, msg, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

func (app *Application) validationErrorResponse(w http.ResponseWriter, r *http.Request, data spec.ValidationError) {
	err := app.writeJSON(w, http.StatusUnprocessableEntity, data, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

func (app *Application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	app.logger.Error(string(debug.Stack()))
	app.errorResponse(w, r, http.StatusInternalServerError, spec.Error{Error: err.Error()})
}

func (app *Application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error(string(debug.Stack()))
	app.errorResponse(w, r, http.StatusBadRequest, spec.Error{Error: err.Error()})
}

func (app *Application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	error := spec.Error{Error: fmt.Sprintf("the %s method is not supported", r.Method)}
	app.errorResponse(w, r, http.StatusMethodNotAllowed, error)
}

func (app *Application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	error := spec.Error{Error: "resources not found"}
	app.errorResponse(w, r, http.StatusNotFound, error)
}

func (app *Application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	error := spec.Error{Error: "invalid authentication credentials"}
	app.errorResponse(w, r, http.StatusUnauthorized, error)
}

func (app *Application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	error := spec.Error{Error: "you must be authenticated to access this resource"}
	app.errorResponse(w, r, http.StatusUnauthorized, error)
}

func (app *Application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	error := spec.Error{Error: "your user account must be activated to access this resource"}
	app.errorResponse(w, r, http.StatusForbidden, error)
}

func (app *Application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWWW-Authenticate", "Bearer")

	error := spec.Error{Error: "invalid or missing authentication token"}
	app.errorResponse(w, r, http.StatusUnauthorized, error)
}

func (app *Application) globalMaxRateResponse(w http.ResponseWriter, r *http.Request) {
	error := spec.Error{Error: "Server is on high load. Please try again later"}
	app.errorResponse(w, r, http.StatusTooManyRequests, error)
}

func (app *Application) ipMaxRateResponse(w http.ResponseWriter, r *http.Request) {
	error := spec.Error{Error: "Too many requests from your ip"}
	app.errorResponse(w, r, http.StatusTooManyRequests, error)
}
