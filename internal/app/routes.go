package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"mlakp-backend/api"
	"mlakp-backend/internal/auth"
	"mlakp-backend/internal/httpapi/handlers"
	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
	"mlakp-backend/internal/sessions"
)

type RouterDeps struct {
	AuthHandler      *handlers.AuthHandler
	UserHandler      *handlers.UserHandler
	GroupHandler     *handlers.GroupHandler
	ExpenseHandler   *handlers.ExpenseHandler
	DebtHandler      *handlers.DebtHandler
	PaymentHandler   *handlers.PaymentHandler
	DashboardHandler *handlers.DashboardHandler
	TokenManager     *auth.TokenManager
	SessionService   *sessions.Service
	AppEnv           string
	CORSOrigins      []string
	AuthRateLimiter  *middleware.RateLimiter
	ReadinessChecker interface {
		Ping(context.Context) error
	}
}

func NewRouter(logger *slog.Logger, deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// Public routes must stay outside the authenticated middleware.
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /readyz", readyzHandler(deps.ReadinessChecker))
	if docsEnabled(deps.AppEnv) {
		mux.HandleFunc("GET /docs", docsHandler)
		mux.HandleFunc("GET /docs/openapi.yaml", openAPIHandler)
	}

	if deps.AuthHandler != nil {
		authRateLimiter := deps.AuthRateLimiter
		if authRateLimiter == nil {
			authRateLimiter = middleware.NewRateLimiter(20, time.Minute)
		}

		mux.Handle("POST /v1/auth/register", authRateLimiter.Limit(http.HandlerFunc(deps.AuthHandler.Register)))
		mux.Handle("POST /v1/auth/login", authRateLimiter.Limit(http.HandlerFunc(deps.AuthHandler.Login)))
		mux.Handle("POST /v1/auth/refresh", authRateLimiter.Limit(http.HandlerFunc(deps.AuthHandler.Refresh)))
	}
	if deps.AuthHandler != nil && deps.TokenManager != nil {
		authenticated := middleware.Authenticate(deps.TokenManager, deps.SessionService)
		mux.Handle("POST /v1/auth/logout", authenticated(http.HandlerFunc(deps.AuthHandler.Logout)))
	}
	if deps.UserHandler != nil && deps.TokenManager != nil {
		authenticated := middleware.Authenticate(deps.TokenManager, deps.SessionService)
		mux.Handle("GET /v1/users/me", authenticated(http.HandlerFunc(deps.UserHandler.Me)))
		mux.Handle("PATCH /v1/users/me", authenticated(http.HandlerFunc(deps.UserHandler.UpdateMe)))
		mux.Handle("GET /v1/users/search", authenticated(http.HandlerFunc(deps.UserHandler.Search)))
	}
	if deps.GroupHandler != nil && deps.TokenManager != nil {
		authenticated := middleware.Authenticate(deps.TokenManager, deps.SessionService)
		mux.Handle("POST /v1/groups", authenticated(http.HandlerFunc(deps.GroupHandler.Create)))
		mux.Handle("GET /v1/groups", authenticated(http.HandlerFunc(deps.GroupHandler.List)))
		mux.Handle("GET /v1/groups/{groupID}", authenticated(http.HandlerFunc(deps.GroupHandler.Get)))
		mux.Handle("POST /v1/groups/{groupID}/members", authenticated(http.HandlerFunc(deps.GroupHandler.AddMember)))
	}
	if deps.ExpenseHandler != nil && deps.TokenManager != nil {
		authenticated := middleware.Authenticate(deps.TokenManager, deps.SessionService)
		mux.Handle("POST /v1/expenses", authenticated(http.HandlerFunc(deps.ExpenseHandler.Create)))
		mux.Handle("GET /v1/expenses/{expenseID}", authenticated(http.HandlerFunc(deps.ExpenseHandler.Get)))
		mux.Handle("GET /v1/groups/{groupID}/expenses", authenticated(http.HandlerFunc(deps.ExpenseHandler.ListByGroup)))
	}
	if deps.DebtHandler != nil && deps.TokenManager != nil {
		authenticated := middleware.Authenticate(deps.TokenManager, deps.SessionService)
		mux.Handle("GET /v1/debts", authenticated(http.HandlerFunc(deps.DebtHandler.List)))
		mux.Handle("POST /v1/debts/{debtID}", authenticated(http.HandlerFunc(deps.DebtHandler.Update)))
		mux.Handle("POST /v1/debts/{debtID}/review", authenticated(http.HandlerFunc(deps.DebtHandler.ReviewRejected)))
	}
	if deps.PaymentHandler != nil && deps.TokenManager != nil {
		authenticated := middleware.Authenticate(deps.TokenManager, deps.SessionService)
		mux.Handle("POST /v1/debts/{debtID}/payments", authenticated(http.HandlerFunc(deps.PaymentHandler.Mark)))
		mux.Handle("POST /v1/payments/{paymentID}", authenticated(http.HandlerFunc(deps.PaymentHandler.Update)))
	}
	if deps.DashboardHandler != nil && deps.TokenManager != nil {
		authenticated := middleware.Authenticate(deps.TokenManager, deps.SessionService)
		mux.Handle("GET /v1/dashboard", authenticated(http.HandlerFunc(deps.DashboardHandler.Get)))
	}

	handler := cors(deps.CORSOrigins)(mux)
	return recoverPanic(logger)(requestLogger(logger)(secureHeaders(deps.AppEnv)(handler)))
}

func docsEnabled(appEnv string) bool {
	return appEnv == "" || appEnv == "local" || appEnv == "test"
}

func cors(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.Header().Add("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func docsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(docsHTML)); err != nil {
		slog.Default().Error("failed to write docs response", "error", err)
	}
}

func openAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(api.OpenAPIYAML); err != nil {
		slog.Default().Error("failed to write openapi response", "error", err)
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	response.Success(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func readyzHandler(checker interface {
	Ping(context.Context) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if checker != nil {
			// Readiness should fail quickly instead of tying up health probes.
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			if err := checker.Ping(ctx); err != nil {
				response.Error(w, http.StatusServiceUnavailable, "not_ready", "Service is not ready")
				return
			}
		}

		response.Success(w, http.StatusOK, map[string]string{
			"status": "ready",
		})
	}
}

func secureHeaders(appEnv string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")

			if appEnv == "production" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
				w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
			}

			next.ServeHTTP(w, r)
		})
	}
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			logger.InfoContext(
				r.Context(),
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func recoverPanic(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic recovered", "panic", recovered)
					response.Error(w, http.StatusInternalServerError, "internal_error", "Internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

const docsHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>MLAKP API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #fff; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.addEventListener("load", function () {
      SwaggerUIBundle({
        url: "/docs/openapi.yaml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
        layout: "BaseLayout"
      });
    });
  </script>
</body>
</html>
`
