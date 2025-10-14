package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *Application) Routes() http.Handler {
	// httprouter is fast alternative for http.ServerMux
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)
	router.HandlerFunc(http.MethodPost, "/v1/scrolljar", app.createPostHandler)
	router.HandlerFunc(http.MethodGet, "/v1/scrolljar/:jarID", app.GetScrollJarHandler)
	router.HandlerFunc(http.MethodGet, "/v1/scrolljar/:jarID/:scrollID", app.GetScrollHandler)

	return app.recoverPanic(router)
}
