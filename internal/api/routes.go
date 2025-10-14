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
	router.HandlerFunc(http.MethodPost, "/v1/jar", app.createPostHandler)
	router.HandlerFunc(http.MethodGet, "/v1/jar/:jarID", app.getScrollJarHandler)
	router.HandlerFunc(http.MethodGet, "/v1/jar/:jarID/scroll/:scrollID", app.getScrollHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/jar/:jarID/scroll/:scrollID", app.updatePatchHandler)

	return app.recoverPanic(router)
}
