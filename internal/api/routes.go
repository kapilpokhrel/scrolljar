package api

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
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

	commonChain := alice.New(app.recoverPanic, app.globalRateLimiter)
	generalIPLimitChain := alice.New(app.ipRateLimiter("general"), app.authenticateUser)
	mediumIPLimitChain := alice.New(app.ipRateLimiter("medium"), app.authenticateUser)
	strictIPLimitChain := alice.New(app.ipRateLimiter("strict"), app.authenticateUser)
	authenticatedChain := mediumIPLimitChain.Append(app.requireAuthenticatedUser)

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.Handler(http.MethodGet, "/v1/ping", generalIPLimitChain.Then(http.HandlerFunc(app.ping)))

	router.Handler(http.MethodPost, "/v1/jar", mediumIPLimitChain.Then(http.HandlerFunc(app.postCreateScrollJarHandler)))
	router.Handler(http.MethodGet, "/v1/jar/:id", generalIPLimitChain.Then(http.HandlerFunc(app.getScrollJarHandler)))
	router.Handler(http.MethodDelete, "/v1/jar/:id", authenticatedChain.Then(http.HandlerFunc(app.deleteScrollJarHandler)))

	router.Handler(http.MethodPost, "/v1/scroll/:id", mediumIPLimitChain.Then(http.HandlerFunc(app.postCreateScrollHandler)))
	router.Handler(http.MethodGet, "/v1/scroll/:id", generalIPLimitChain.Then(http.HandlerFunc(app.getScrollHandler)))
	router.Handler(http.MethodPatch, "/v1/scroll/:id", authenticatedChain.Then(http.HandlerFunc(app.patchScrollHandler)))
	router.Handler(http.MethodDelete, "/v1/scroll/:id", authenticatedChain.Then(http.HandlerFunc(app.deleteScrollHandler)))

	router.Handler(http.MethodPost, "/v1/user/register", strictIPLimitChain.Then(http.HandlerFunc(app.postUserRegisterHandler)))
	router.Handler(http.MethodPut, "/v1/user/activate", strictIPLimitChain.Then(http.HandlerFunc(app.putUserActivationHandler)))
	router.Handler(http.MethodPost, "/v1/user/auth", mediumIPLimitChain.Then(http.HandlerFunc(app.postUserAuthHandler)))

	router.Handler(http.MethodGet, "/v1/user/get", authenticatedChain.Then(http.HandlerFunc(app.getUsersJarHandler)))

	router.Handler(http.MethodPost, "/v1/token/activation", strictIPLimitChain.Then(http.HandlerFunc(app.postUserActivationTokenHandler)))

	return commonChain.Then(router)
}
