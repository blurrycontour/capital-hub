// Package httpapi wires the HTTP router, middleware, API routes, and the
// embedded frontend together into a single handler.
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/aditya/capital-hub/internal/auth"
	"github.com/aditya/capital-hub/internal/config"
	"github.com/aditya/capital-hub/internal/inventory"
	"github.com/aditya/capital-hub/internal/notify"
	"github.com/aditya/capital-hub/internal/web"
)

// Server holds shared dependencies for HTTP handlers.
type Server struct {
	cfg       *config.Config
	db        *sql.DB
	logger    *slog.Logger
	auth      *auth.Service
	notify    *notify.Service
	inventory *inventory.Service
	router    chi.Router
}

// New constructs a Server and builds its route tree.
func New(cfg *config.Config, db *sql.DB, logger *slog.Logger) (*Server, error) {
	s := &Server{
		cfg:       cfg,
		db:        db,
		logger:    logger,
		auth:      auth.NewService(db, cfg),
		notify:    notify.NewService(db),
		inventory: inventory.NewService(db),
	}
	if err := s.routes(); err != nil {
		return nil, err
	}
	return s, nil
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler { return s.router }

func (s *Server) routes() error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	if len(s.cfg.TrustedProxies) > 0 {
		r.Use(middleware.RealIP)
	}
	r.Use(requestLogger(s.logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(securityHeaders)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", s.handleHealth)

		api.Route("/auth", func(authRouter chi.Router) {
			authRouter.Get("/csrf", s.handleCSRFToken)
			authRouter.Get("/providers", s.handleAuthProviders)
			authRouter.Post("/login", s.handleLogin)
			authRouter.Get("/oidc/login", s.handleOIDCLogin)
			authRouter.Get("/oidc/callback", s.handleOIDCCallback)

			authRouter.With(s.requireAuth, s.requireCSRF).Post("/logout", s.handleLogout)
			authRouter.With(s.requireAuth).Get("/me", s.handleMe)
			authRouter.With(s.requireAuth, s.requireCSRF).Patch("/me", s.handleUpdateMe)
		})

		api.Route("/notifications", func(n chi.Router) {
			n.Use(s.requireAuth)
			n.Get("/", s.handleListNotifications)
			n.With(s.requireCSRF).Post("/{id}/read", s.handleMarkNotificationRead)
		})

		api.Route("/collections", func(c chi.Router) {
			c.Use(s.requireAuth)
			c.Get("/", s.handleListCollections)
			c.With(s.requireCSRF).Post("/", s.handleCreateCollection)
			c.Get("/{id}", s.handleGetCollection)
			c.With(s.requireCSRF).Patch("/{id}", s.handleUpdateCollection)
			c.With(s.requireCSRF).Delete("/{id}", s.handleDeleteCollection)
			c.Get("/{id}/stats", s.handleCollectionStats)
			c.Get("/{id}/items", s.handleListItems)
			c.With(s.requireCSRF).Post("/{id}/items", s.handleCreateItem)
		})

		api.Route("/items", func(it chi.Router) {
			it.Use(s.requireAuth)
			it.Get("/{id}", s.handleGetItem)
			it.With(s.requireCSRF).Patch("/{id}", s.handleUpdateItem)
			it.With(s.requireCSRF).Delete("/{id}", s.handleDeleteItem)
			it.With(s.requireCSRF).Post("/{id}/image", s.handleUploadItemImage)
			it.Get("/{id}/stats", s.handleItemStats)
			it.Get("/{id}/entries", s.handleListEntries)
			it.With(s.requireCSRF).Post("/{id}/entries", s.handleCreateEntry)
		})

		api.Route("/entries", func(e chi.Router) {
			e.Use(s.requireAuth)
			e.With(s.requireCSRF).Patch("/{id}", s.handleUpdateEntry)
			e.With(s.requireCSRF).Delete("/{id}", s.handleDeleteEntry)
		})

		api.With(s.requireAuth).Get("/search", s.handleSearch)
		api.With(s.requireAuth).Get("/stats/portfolio", s.handlePortfolioStats)

		api.Route("/admin", func(admin chi.Router) {
			admin.Use(s.requireAuth, s.requireAdmin)
			admin.Get("/users", s.handleAdminListUsers)
			admin.Get("/settings/smtp", s.handleAdminGetSMTPSettings)
			admin.With(s.requireCSRF).Put("/settings/smtp", s.handleAdminUpdateSMTPSettings)
			admin.With(s.requireCSRF).Post("/settings/smtp/test", s.handleAdminTestSMTP)
		})
	})

	// Liveness probe for orchestrators / proxies.
	r.Get("/healthz", s.handleHealth)

	// Serve user-uploaded files from disk. Requires a valid session so uploads
	// are not publicly enumerable.
	uploadFS := http.StripPrefix("/uploads/", http.FileServer(http.Dir(s.cfg.UploadsDir())))
	r.With(s.requireAuth).Handle("/uploads/*", uploadFS)

	assets, err := web.Assets()
	if err != nil {
		return err
	}
	r.Handle("/*", web.SPAHandler(assets))

	s.router = r
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK
	if err := s.db.PingContext(r.Context()); err != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
		s.logger.ErrorContext(r.Context(), "health check db ping failed", "error", err)
	}
	writeJSON(w, code, map[string]string{"status": status})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type apiError struct {
	Error string `json:"error"`
}

func writeAPIError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, apiError{Error: msg})
}

type contextKey string

const (
	ctxUserKey     contextKey = "auth.user"
	csrfCookieName            = "ch_csrf"
)

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie(s.cfg.SessionCookieName)
		if err != nil || strings.TrimSpace(sessionCookie.Value) == "" {
			writeAPIError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		user, err := s.auth.CurrentUser(r.Context(), sessionCookie.Value)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				s.logger.WarnContext(r.Context(), "resolve current user failed", "error", err)
			}
			writeAPIError(w, http.StatusUnauthorized, "invalid or expired session")
			return
		}

		next.ServeHTTP(w, r.WithContext(withUser(r, user)))
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := userFromContext(r)
		if user == nil || !user.IsAdmin {
			writeAPIError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(csrfCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			writeAPIError(w, http.StatusForbidden, "missing csrf cookie")
			return
		}
		token := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
		if token == "" || token != cookie.Value {
			writeAPIError(w, http.StatusForbidden, "invalid csrf token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withUser(r *http.Request, user *auth.User) context.Context {
	return context.WithValue(r.Context(), ctxUserKey, user)
}

func userFromContext(r *http.Request) *auth.User {
	v := r.Context().Value(ctxUserKey)
	if user, ok := v.(*auth.User); ok {
		return user
	}
	return nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
