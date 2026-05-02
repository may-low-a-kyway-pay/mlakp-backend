package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"mlakp-backend/api"
	"mlakp-backend/internal/auth"
	"mlakp-backend/internal/httpapi/handlers"
	"mlakp-backend/internal/httpapi/middleware"
	"mlakp-backend/internal/httpapi/response"
)

type RouterDeps struct {
	AuthHandler      *handlers.AuthHandler
	UserHandler      *handlers.UserHandler
	TokenManager     *auth.TokenManager
	ReadinessChecker interface {
		Ping(context.Context) error
	}
}

func NewRouter(logger *slog.Logger, deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /readyz", readyzHandler(deps.ReadinessChecker))
	mux.HandleFunc("GET /docs", docsHandler)
	mux.HandleFunc("GET /docs/openapi.yaml", openAPIHandler)

	if deps.AuthHandler != nil {
		mux.HandleFunc("POST /v1/auth/register", deps.AuthHandler.Register)
		mux.HandleFunc("POST /v1/auth/login", deps.AuthHandler.Login)
		mux.HandleFunc("POST /v1/auth/logout", deps.AuthHandler.Logout)
	}
	if deps.UserHandler != nil && deps.TokenManager != nil {
		authenticated := middleware.Authenticate(deps.TokenManager)
		mux.Handle("GET /v1/users/me", authenticated(http.HandlerFunc(deps.UserHandler.Me)))
	}

	return recoverPanic(logger)(requestLogger(logger)(secureHeaders(mux)))
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
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func readyzHandler(checker interface {
	Ping(context.Context) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if checker != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			if err := checker.Ping(ctx); err != nil {
				response.Error(w, http.StatusServiceUnavailable, "not_ready", "Service is not ready")
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ready",
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Default().Error("failed to write json response", "error", err)
	}
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")

		next.ServeHTTP(w, r)
	})
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
					writeJSON(w, http.StatusInternalServerError, map[string]map[string]string{
						"error": {
							"code":    "internal_error",
							"message": "Internal server error",
						},
					})
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
