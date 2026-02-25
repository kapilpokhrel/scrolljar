package api

import (
	"crypto/sha256"
	"errors"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/database"
	"golang.org/x/time/rate"
)

func (app *Application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error("Internal server error", "error", err)
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, errors.New("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

/*
http {
    # 1. TOTAL SERVER LIMIT (All users combined)
    # Using a static string "all" as the key makes this apply globally.
    limit_req_zone "all" zone=server_wide:10m rate=50r/s;

    # 2. PER-ENDPOINT LIMITS (Per IP)
    # These still use $binary_remote_addr to prevent one IP from hogging the endpoint.
    limit_req_zone $binary_remote_addr zone=read_limit:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=write_limit:10m rate=5r/s;

    # 3. MAX CONCURRENT CONNECTIONS (Global "User" Capacity)
    # Limits the total number of active connections the server will handle at once.
    limit_conn_zone "all_conns" zone=total_conns:10m;

    server {
        listen 80;

        # Apply the global connection limit to the whole server
        limit_conn total_conns 1000;

        # READ Endpoint
        location /api/read {
            # Checks both the global 50r/s bucket AND the per-IP 10r/s bucket
            limit_req zone=server_wide burst=20;
            limit_req zone=read_limit burst=5 nodelay;

            proxy_pass http://backend_read;
        }

        # WRITE Endpoint
        location /api/write {
            # Checks both the global 50r/s bucket AND the per-IP 5r/s bucket
            limit_req zone=server_wide burst=20;
            limit_req zone=write_limit burst=2 nodelay;

            proxy_pass http://backend_write;
        }
    }
}
*/

func (app *Application) globalRateLimiter(next http.Handler) http.Handler {
	// use nginx rate limiter
	limiter := rate.NewLimiter(rate.Limit(app.config.Rate.GlobalRps), app.config.Rate.GlobalBps)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			app.globalMaxRateResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *Application) ipRateLimiter(limitType string) func(http.Handler) http.Handler {
	// use nginx per location based rate limit

	var rps float64
	var bps int
	switch limitType {
	case "Medium":
		rps = app.config.Rate.IPRps / 5.0
		bps = int(math.Ceil(float64(app.config.Rate.IPBps) / 5.0))
	case "Strict":
		rps = app.config.Rate.IPRps / 10.0
		bps = int(math.Ceil(float64(app.config.Rate.IPBps) / 10.0))
	default:
		rps = app.config.Rate.IPRps
		bps = app.config.Rate.IPBps
	}

	type clientLimiter struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]clientLimiter)
	)

	go func() {
		ticker := time.NewTicker(time.Minute)
		for {
			<-ticker.C
			mu.Lock()

			for host, client := range clients {
				if time.Since(client.lastSeen) > time.Minute*5 {
					delete(clients, host)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
			}

			mu.Lock()
			limiter, ok := clients[host]
			if !ok {
				clients[host] = clientLimiter{limiter: rate.NewLimiter(
					rate.Limit(rps),
					bps,
				), lastSeen: time.Now()}
				limiter = clients[host]
			}
			mu.Unlock()

			if !limiter.limiter.Allow() {
				app.ipMaxRateResponse(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (app *Application) authenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, app.contextSetUser(r, nil))
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
		}

		tokenHash := sha256.Sum256([]byte(headerParts[1]))

		token := &database.Token{
			TokenHash: tokenHash[:],
		}

		if err := app.models.Token.GetTokenByHash(r.Context(), token); err != nil {
			switch {
			case errors.Is(err, database.ErrNoRecord):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)

			}
			return
		}

		user := &database.User{}
		user.ID = token.UserID

		if err := app.models.Users.GetByID(r.Context(), user); err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		next.ServeHTTP(w, app.contextSetUser(r, user))
	})
}

func (app *Application) requireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if user == nil {
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
