// Package httpapi wires the HTTP router, middleware, API routes, and the
// embedded frontend together into a single handler.
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
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

// Version is the build version string, set from main via ldflags.
var Version = "dev"

// Server holds shared dependencies for HTTP handlers.
type Server struct {
	cfg       *config.Config
	db        *sql.DB
	logger    *slog.Logger
	auth      *auth.Service
	notify    *notify.Service
	inventory *inventory.Service
	router    chi.Router
	// trustedProxies holds the parsed CH_TRUSTED_PROXIES entries. Requests
	// whose immediate peer is in this set may supply the real client IP via
	// X-Real-IP / X-Forwarded-For.
	trustedProxies []*net.IPNet
}

// New constructs a Server and builds its route tree.
func New(cfg *config.Config, db *sql.DB, logger *slog.Logger) (*Server, error) {
	s := &Server{
		cfg:            cfg,
		db:             db,
		logger:         logger,
		auth:           auth.NewService(db, cfg),
		notify:         notify.NewService(db),
		inventory:      inventory.NewService(db),
		trustedProxies: parseTrustedProxies(cfg.TrustedProxies, logger),
	}
	if err := s.routes(); err != nil {
		return nil, err
	}
	return s, nil
}

// parseTrustedProxies turns CIDR/IP strings into networks. Bare IPs become
// /32 or /128 networks. Invalid entries are logged and skipped.
func parseTrustedProxies(entries []string, logger *slog.Logger) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(entries))
	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				if ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			}
		}
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			if logger != nil {
				logger.Warn("ignoring invalid CH_TRUSTED_PROXIES entry", "entry", raw, "error", err)
			}
			continue
		}
		nets = append(nets, network)
	}
	return nets
}

// clientIP resolves the originating client IP for a request. When the immediate
// peer (r.RemoteAddr) is a trusted proxy, it honors X-Real-IP, then the
// left-most entry of X-Forwarded-For; otherwise it returns the peer address.
func (s *Server) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !s.peerIsTrusted(host) {
		return host
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		if net.ParseIP(realIP) != nil {
			return realIP
		}
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for _, part := range strings.Split(xff, ",") {
			candidate := strings.TrimSpace(part)
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}
	return host
}

// peerIsTrusted reports whether the given peer host is a configured trusted
// proxy. If no proxies are configured, any peer is trusted so that forwarded
// headers work out of the box for single-proxy deployments (mirroring how
// X-Forwarded-Proto is already honored for the cookie Secure flag).
func (s *Server) peerIsTrusted(host string) bool {
	if len(s.trustedProxies) == 0 {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, network := range s.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
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
		api.Get("/version", s.handleVersion)
		api.With(s.requireAuth).Get("/openapi.json", s.handleOpenAPISpec)

		api.Route("/auth", func(authRouter chi.Router) {
			authRouter.Get("/csrf", s.handleCSRFToken)
			authRouter.Get("/providers", s.handleAuthProviders)
			authRouter.Post("/login", s.handleLogin)
			authRouter.Get("/oidc/login", s.handleOIDCLogin)
			authRouter.Get("/oidc/callback", s.handleOIDCCallback)

			authRouter.With(s.requireAuth, s.requireCSRF).Post("/logout", s.handleLogout)
			authRouter.With(s.requireAuth).Get("/me", s.handleMe)
			authRouter.With(s.requireAuth, s.requireCSRF).Patch("/me", s.handleUpdateMe)
			authRouter.With(s.requireAuth, s.requireCSRF).Post("/me/avatar", s.handleUploadAvatar)
			authRouter.With(s.requireAuth, s.requireCSRF).Post("/me/password", s.handleChangePassword)
			authRouter.With(s.requireAuth).Get("/me/preferences", s.handleGetPreferences)
			authRouter.With(s.requireAuth, s.requireCSRF).Patch("/me/preferences", s.handleUpdatePreferences)
			authRouter.With(s.requireAuth).Get("/me/sessions", s.handleListSessions)
			authRouter.With(s.requireAuth, s.requireCSRF).Delete("/me/sessions", s.handleRevokeOtherSessions)
			authRouter.With(s.requireAuth, s.requireCSRF).Delete("/me/sessions/{id}", s.handleRevokeSession)
			authRouter.With(s.requireAuth, s.requireCSRF).Post("/me/deletion-code", s.handleRequestAccountDeletion)
			authRouter.With(s.requireAuth, s.requireCSRF).Delete("/me", s.handleConfirmAccountDeletion)
		})

		api.Route("/notifications", func(n chi.Router) {
			n.Use(s.requireAuth)
			n.Get("/", s.handleListNotifications)
			n.Get("/unread-count", s.handleUnreadCount)
			n.With(s.requireCSRF).Post("/read-all", s.handleMarkAllNotificationsRead)
			n.With(s.requireCSRF).Post("/unread-all", s.handleMarkAllNotificationsUnread)
			n.With(s.requireCSRF).Delete("/", s.handleDeleteAllNotifications)
			n.With(s.requireCSRF).Post("/{id}/read", s.handleMarkNotificationRead)
			n.With(s.requireCSRF).Post("/{id}/unread", s.handleMarkNotificationUnread)
			n.With(s.requireCSRF).Delete("/{id}", s.handleDeleteNotification)
		})

		api.Route("/collections", func(c chi.Router) {
			c.Use(s.requireAuth)
			c.Get("/", s.handleListCollections)
			c.With(s.requireCSRF, s.requireNotReader).Post("/", s.handleCreateCollection)
			c.Get("/{id}", s.handleGetCollection)
			c.With(s.requireCSRF, s.requireNotReader).Patch("/{id}", s.handleUpdateCollection)
			c.With(s.requireCSRF, s.requireNotReader).Delete("/{id}", s.handleDeleteCollection)
			c.Get("/{id}/stats", s.handleCollectionStats)
			c.Get("/{id}/items", s.handleListItems)
			c.With(s.requireCSRF, s.requireNotReader).Post("/{id}/items", s.handleCreateItem)
			c.Get("/{id}/shares", s.handleListCollectionShares)
			c.With(s.requireCSRF, s.requireNotReader).Post("/{id}/shares", s.handleShareCollection)
			c.With(s.requireCSRF, s.requireNotReader).Delete("/{id}/shares/{userId}", s.handleUnshareCollection)
		})

		api.Route("/items", func(it chi.Router) {
			it.Use(s.requireAuth)
			it.Get("/", s.handleListAllItems)
			it.Get("/{id}", s.handleGetItem)
			it.With(s.requireCSRF, s.requireNotReader).Patch("/{id}", s.handleUpdateItem)
			it.With(s.requireCSRF, s.requireNotReader).Post("/{id}/move", s.handleMoveItem)
			it.With(s.requireCSRF, s.requireNotReader).Delete("/{id}", s.handleDeleteItem)
			it.With(s.requireCSRF, s.requireNotReader).Post("/{id}/image", s.handleUploadItemImage)
			it.With(s.requireCSRF, s.requireNotReader).Delete("/{id}/image", s.handleDeleteItemImage)
			it.With(s.requireCSRF, s.requireNotReader).Post("/{id}/cover", s.handleSetItemCover)
			it.With(s.requireCSRF, s.requireNotReader).Post("/{id}/attachments", s.handleUploadItemAttachment)
			it.With(s.requireCSRF, s.requireNotReader).Delete("/{id}/attachments", s.handleDeleteItemAttachment)
			it.Get("/{id}/stats", s.handleItemStats)
			it.Get("/{id}/entries", s.handleListEntries)
			it.With(s.requireCSRF, s.requireNotReader).Post("/{id}/entries", s.handleCreateEntry)
		})

		api.Route("/entries", func(e chi.Router) {
			e.Use(s.requireAuth)
			e.With(s.requireCSRF, s.requireNotReader).Patch("/{id}", s.handleUpdateEntry)
			e.With(s.requireCSRF, s.requireNotReader).Delete("/{id}", s.handleDeleteEntry)
			e.With(s.requireCSRF, s.requireNotReader).Post("/{id}/attachments", s.handleUploadEntryAttachment)
			e.With(s.requireCSRF, s.requireNotReader).Delete("/{id}/attachments", s.handleDeleteEntryAttachment)
		})

		api.With(s.requireAuth).Get("/search", s.handleSearch)
		api.With(s.requireAuth).Get("/stats/portfolio", s.handlePortfolioStats)
		api.With(s.requireAuth).Get("/stats/recent-items", s.handleRecentItems)

		api.Route("/admin", func(admin chi.Router) {
			admin.Use(s.requireAuth, s.requireAdmin)
			admin.Get("/users", s.handleAdminListUsers)
			admin.With(s.requireCSRF).Post("/users", s.handleAdminCreateUser)
			admin.With(s.requireCSRF).Patch("/users/{id}", s.handleAdminUpdateUser)
			admin.With(s.requireCSRF).Delete("/users/{id}", s.handleAdminDeleteUser)
			admin.Get("/settings/smtp", s.handleAdminGetSMTPSettings)
			admin.With(s.requireCSRF).Put("/settings/smtp", s.handleAdminUpdateSMTPSettings)
			admin.With(s.requireCSRF).Post("/settings/smtp/test", s.handleAdminTestSMTP)
			admin.Get("/settings/oidc", s.handleAdminGetOIDCSettings)
			admin.With(s.requireCSRF).Put("/settings/oidc", s.handleAdminUpdateOIDCSettings)
		})
	})

	// Liveness probe for orchestrators / proxies.
	r.Get("/healthz", s.handleHealth)

	// Interactive API documentation (Swagger UI), authenticated users only.
	r.With(s.requireAuthRedirect).Get("/api/docs", s.handleSwaggerUI)

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

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": Version})
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
	csrfCookieName string     = "ch_csrf"
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

func (s *Server) requireNotReader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := userFromContext(r)
		if user == nil {
			writeAPIError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if user.Role == "reader" {
			writeAPIError(w, http.StatusForbidden, "read-only access: this action requires editor or administrator role")
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
