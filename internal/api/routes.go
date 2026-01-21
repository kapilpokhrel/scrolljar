package api

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/flowchartsman/swaggerui"
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

type routeIPPolicy struct {
	method  string
	pattern *regexp.Regexp
	level   string
	mw      func(http.Handler) http.Handler
}

type RateLimiterFactory func(string) func(http.Handler) http.Handler

type routeIPLimiter struct {
	policies []routeIPPolicy
}

func NewRouteIPLimiter(factory RateLimiterFactory) routeIPLimiter {
	policies := []routeIPPolicy{
		{"GET", regexp.MustCompile(`^/ping$`), "General", nil},

		{"POST", regexp.MustCompile(`^/jar$`), "Medium", nil},
		{"GET", regexp.MustCompile(`^/jar/[^/]+$`), "General", nil},
		{"DELETE", regexp.MustCompile(`^/jar/[^/]+$`), "Medium", nil},
		{"GET", regexp.MustCompile(`^/jar/[^/]+/scrolls$`), "General", nil},

		{"GET", regexp.MustCompile(`^/scroll/[^/]+$`), "General", nil},
		{"POST", regexp.MustCompile(`^/scroll/[^/]+$`), "Medium", nil},
		{"PATCH", regexp.MustCompile(`^/scroll/[^/]+$`), "Medium", nil},
		{"DELETE", regexp.MustCompile(`^/scroll/[^/]+$`), "Medium", nil},

		{"GET", regexp.MustCompile(`^/user$`), "Medium", nil},
		{"POST", regexp.MustCompile(`^/user/auth$`), "Medium", nil},
		{"POST", regexp.MustCompile(`^/user/register$`), "Strict", nil},
		{"PUT", regexp.MustCompile(`^/user/activate$`), "Strict", nil},
		{"GET", regexp.MustCompile(`^/user/jars$`), "General", nil},

		{"POST", regexp.MustCompile(`^/token/activation$`), "Strict", nil},
	}
	for i, p := range policies {
		policies[i].mw = factory(p.level)
	}

	return routeIPLimiter{
		policies: policies,
	}
}

func (r *routeIPLimiter) Get(method, path string) func(http.Handler) http.Handler {
	for _, p := range r.policies {
		if p.method == method && p.pattern.MatchString(path) {
			return p.mw
		}
	}
	return nil
}

func (app *Application) ipLimitPolicyMiddleware(next http.Handler) http.Handler {
	baseURL := "/v1"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, _ := strings.CutPrefix(r.URL.Path, baseURL)
		limiter := app.ipLimiter.Get(r.Method, path)
		if limiter != nil {
			limiter(next).ServeHTTP(w, r)
		}
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
	router.Handle("GET /swagger/", http.StripPrefix("/swagger", swaggerui.Handler(spec.SpecFile)))

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
