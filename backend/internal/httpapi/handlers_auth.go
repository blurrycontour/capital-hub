package httpapi

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	user, sessionID, expiresAt, err := s.auth.Login(
		r.Context(),
		req.Identifier,
		req.Password,
		r.UserAgent(),
		r.RemoteAddr,
	)
	if err != nil {
		s.logger.WarnContext(r.Context(), "login failed", "identifier", strings.TrimSpace(req.Identifier), "error", err)
		writeAPIError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	secure := requestIsSecure(r)
	http.SetCookie(w, s.auth.SessionCookie(sessionID, expiresAt, secure))
	csrf := csrfCookie(secure)
	http.SetCookie(w, csrf)
	writeJSON(w, http.StatusOK, map[string]any{
		"user":      user,
		"csrfToken": csrf.Value,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie(s.cfg.SessionCookieName)
	if err == nil && sessionCookie.Value != "" {
		if err := s.auth.Logout(r.Context(), sessionCookie.Value); err != nil {
			s.logger.WarnContext(r.Context(), "logout failed", "error", err)
		}
	}

	secure := requestIsSecure(r)
	http.SetCookie(w, s.auth.ClearSessionCookie(secure))
	http.SetCookie(w, clearCSRFCookie(secure))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

type updateProfileRequest struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	current := userFromContext(r)
	if current == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	updated, err := s.auth.UpdateProfile(r.Context(), current.ID, req.DisplayName, req.Email)
	if err != nil {
		s.logger.WarnContext(r.Context(), "update profile failed", "user_id", current.ID, "error", err)
		writeAPIError(w, http.StatusBadRequest, "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": updated})
}

// handleUploadAvatar stores an uploaded image and sets it as the user's avatar.
func (s *Server) handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	current := userFromContext(r)
	if current == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	stored, _, ok := s.saveUploadedFile(w, r, allowedImageExt, "upload avatar")
	if !ok {
		return
	}

	updated, prev, err := s.auth.SetAvatar(r.Context(), current.ID, stored)
	if err != nil {
		_ = os.Remove(filepath.Join(s.cfg.UploadsDir(), filepath.Base(stored)))
		s.logger.ErrorContext(r.Context(), "set avatar failed", "user_id", current.ID, "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to set avatar")
		return
	}
	if prev != "" {
		_ = os.Remove(filepath.Join(s.cfg.UploadsDir(), filepath.Base(prev)))
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": updated})
}

func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(c.Value) == "" {
		cookie := csrfCookie(requestIsSecure(r))
		http.SetCookie(w, cookie)
		writeJSON(w, http.StatusOK, map[string]string{"token": cookie.Value})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": c.Value})
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.auth.ListUsers(r.Context())
	if err != nil {
		s.logger.ErrorContext(r.Context(), "list users failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func csrfCookie(secure bool) *http.Cookie {
	token, err := randomToken(32)
	if err != nil {
		// Keep requests flowing even if entropy source is temporarily unavailable.
		token = base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("fallback-%d", time.Now().UnixNano())))
	}

	return &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(24 * time.Hour),
		MaxAge:   int((24 * time.Hour).Seconds()),
	}
}

func clearCSRFCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

// requestIsSecure reports whether the request reached the server over HTTPS,
// either directly (r.TLS) or via a TLS-terminating proxy that set
// X-Forwarded-Proto. The result controls the Secure attribute on cookies so
// they are stored both for plain-HTTP deployments (e.g. local Docker on
// http://localhost) and HTTPS deployments behind a reverse proxy.
func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		first := strings.TrimSpace(strings.Split(proto, ",")[0])
		return strings.EqualFold(first, "https")
	}
	return false
}

func randomToken(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func parseOptionalSessionCookie(r *http.Request, name string) (string, error) {
	c, err := r.Cookie(name)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", sql.ErrNoRows
		}
		return "", err
	}
	if strings.TrimSpace(c.Value) == "" {
		return "", sql.ErrNoRows
	}
	return c.Value, nil
}
