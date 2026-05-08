package api

import (
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/kapilpokhrel/scrolljar/internal/spec"
)

// httpError is a known client error with a fixed status and message.
// Plain errors (not *httpError) are treated as unexpected server errors (500).
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

type validationHTTPError struct {
	data spec.ValidationError
}

func (e *validationHTTPError) Error() string { return "validation error" }

// Sentinel client errors.
var (
	errNotFound         = &httpError{http.StatusNotFound, "resources not found"}
	errInvalidCreds     = &httpError{http.StatusUnauthorized, "invalid credentials"}
	errInvalidJarPass   = &httpError{http.StatusUnauthorized, "invalid jar password"}
	errInactiveAccount  = &httpError{http.StatusForbidden, "your user account must be activated to access this resource"}
	errEntityTooLarge   = &httpError{http.StatusRequestEntityTooLarge, "entity too large"}
	errAlreadyUploaded  = &httpError{http.StatusConflict, "already uploaded"}
	errAlreadyActivated = &httpError{http.StatusServiceUnavailable, "account already activated"}
	errEditConflict     = &httpError{http.StatusConflict, "edit conflict; please try again"}
)

func errBadRequest(err error) *httpError {
	return &httpError{http.StatusBadRequest, err.Error()}
}

func errValidation(v spec.ValidationError) *validationHTTPError {
	return &validationHTTPError{v}
}

// handleError is the central error dispatcher called by outer handler shells.
func (app *Application) handleError(w http.ResponseWriter, r *http.Request, err error) {
	var he *httpError
	var ve *validationHTTPError
	switch {
	case errors.As(err, &he):
		app.errorResponse(w, r, he.status, spec.Error{Error: he.msg})
	case errors.As(err, &ve):
		app.validationErrorResponse(w, r, ve.data)
	default:
		app.serverErrorResponse(w, r, err)
	}
}

func (app *Application) logError(r *http.Request, err error) {
	app.logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
}

func (app *Application) errorResponse(w http.ResponseWriter, r *http.Request, status int, msg spec.Error) {
	if err := app.writeJSON(w, status, msg, nil); err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

func (app *Application) validationErrorResponse(w http.ResponseWriter, r *http.Request, data spec.ValidationError) {
	if err := app.writeJSON(w, http.StatusUnprocessableEntity, data, nil); err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

func (app *Application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	app.logger.Error(string(debug.Stack()))
	app.errorResponse(w, r, http.StatusInternalServerError, spec.Error{Error: "internal server error"})
}

func (app *Application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusUnauthorized, spec.Error{Error: "you must be authenticated to access this resource"})
}

func (app *Application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	app.errorResponse(w, r, http.StatusUnauthorized, spec.Error{Error: "invalid or missing authentication token"})
}

func (app *Application) globalMaxRateResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusTooManyRequests, spec.Error{Error: "server is on high load, please try again later"})
}

func (app *Application) ipMaxRateResponse(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusTooManyRequests, spec.Error{Error: "too many requests from your IP"})
}
