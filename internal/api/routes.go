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
	router.HandlerFunc(http.MethodPost, "/v1/jar", app.postCreateScrollJarHandler)
	router.HandlerFunc(http.MethodPost, "/v1/scroll/:id", app.postCreateScrollHandler)
	router.HandlerFunc(http.MethodGet, "/v1/jar/:id", app.getScrollJarHandler)
	router.HandlerFunc(http.MethodGet, "/v1/scroll/:id", app.getScrollHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/scroll/:id", app.patchScrollHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/jar/:id", app.deleteScrollJarHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/scroll/:id", app.deleteScrollHandler)
	router.HandlerFunc(http.MethodPost, "/v1/user/register", app.postUserRegisterHandler)
	router.HandlerFunc(http.MethodPut, "/v1/user/activate", app.putUserActivationHandler)

	return app.recoverPanic(router)
}
