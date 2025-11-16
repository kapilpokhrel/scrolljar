package api

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

func (app *Application) ping(w http.ResponseWriter, r *http.Request) {
	output := envelope{
		"status":      "running",
		"uptime":      time.Since(app.startTime).String(),
		"environment": app.config.Env,
		"version":     "1.0.0",
	}
	app.writeJSON(w, http.StatusOK, output, nil)
}

func (app *Application) Routes() http.Handler {
	// httprouter is fast alternative for http.ServerMux
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)
	router.HandlerFunc(http.MethodGet, "/v1/ping", app.ping)
	router.HandlerFunc(http.MethodPost, "/v1/jar", app.postCreateScrollJarHandler)
	router.HandlerFunc(http.MethodPost, "/v1/scroll/:id", app.postCreateScrollHandler)
	router.HandlerFunc(http.MethodGet, "/v1/jar/:id", app.getScrollJarHandler)
	router.HandlerFunc(http.MethodGet, "/v1/scroll/:id", app.getScrollHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/scroll/:id", app.patchScrollHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/jar/:id", app.deleteScrollJarHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/scroll/:id", app.deleteScrollHandler)
	router.HandlerFunc(http.MethodPost, "/v1/user/register", app.postUserRegisterHandler)
	router.HandlerFunc(http.MethodPut, "/v1/user/activate", app.putUserActivationHandler)
	router.HandlerFunc(http.MethodPost, "/v1/user/auth", app.postUserAuthHandler)

	return app.recoverPanic(app.authenticateUser(router))
}
