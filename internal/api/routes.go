package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/api/spec"
)

func (app *Application) Ping(w http.ResponseWriter, r *http.Request) {
	output := envelope{
		"status":      "running",
		"uptime":      time.Since(app.startTime).String(),
		"environment": app.config.Env,
		"version":     "1.0.0",
	}
	app.writeJSON(w, http.StatusOK, output, nil)
}

type IPTier string

const (
	IPGeneral IPTier = "general"
	IPMedium  IPTier = "medium"
	IPStrict  IPTier = "strict"
)

var ipPolicy = map[string]IPTier{
	"GET /ping": IPGeneral,

	"GET /jar":         IPGeneral,
	"POST /jar":        IPMedium,
	"DELETE /jar":      IPMedium,
	"GET /jar/scrolls": IPGeneral,

	"GET /scroll":    IPGeneral,
	"POST /scroll":   IPMedium,
	"PATCH /scroll":  IPMedium,
	"DELETE /scroll": IPMedium,

	"GET /user":           IPMedium,
	"POST /user/auth":     IPMedium,
	"POST /user/register": IPStrict,
	"PUT /user/activate":  IPStrict,
	"GET /user/jars":      IPGeneral,

	"POST /token/activation": IPStrict,
}

func (app *Application) ipLimitPolicyMiddleware(next http.Handler) http.Handler {
	baseURL := "/v1"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, _ := strings.CutPrefix(r.URL.Path, baseURL)
		key := r.Method + " " + path
		tier := ipPolicy[key]
		if tier == "" {
			tier = IPGeneral
		}

		app.ipRateLimiter(string(tier))(next).ServeHTTP(w, r)
	})
}

func (app *Application) requireAuthIfContextFlag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the context has BearerAuthScopes key
		if _, ok := r.Context().Value(spec.BearerAuthScopes).([]string); ok {
			app.requireAuthenticatedUser(next).ServeHTTP(w, r)
			return
		}

		// No flag → skip auth
		next.ServeHTTP(w, r)
	})
}

func debugMiddleware(name string) spec.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return next
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("ENTER", name)
			next.ServeHTTP(w, r)
			log.Println("EXIT ", name)
		})
	}
}

func (app *Application) GetRouter() http.Handler {
	router := http.NewServeMux()
	middlewares := []spec.MiddlewareFunc{
		debugMiddleware("require_auth"),
		app.requireAuthIfContextFlag,
		debugMiddleware("auth"),
		app.authenticateUser,
		debugMiddleware("ip"),
		app.ipLimitPolicyMiddleware,
		debugMiddleware("rate"),
		app.globalRateLimiter,
		debugMiddleware("recover"),
		app.recoverPanic,
	}

	handler := spec.HandlerWithOptions(app, spec.StdHTTPServerOptions{
		BaseURL:     "/v1",
		BaseRouter:  router,
		Middlewares: middlewares,
	})

	return handler
}
