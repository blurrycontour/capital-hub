package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookieName = "ch_oidc_state"
	oidcNonceCookieName = "ch_oidc_nonce"
)

type oidcClaims struct {
	Subject           string   `json:"sub"`
	Email             string   `json:"email"`
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Groups            []string `json:"groups"`
}

func (s *Server) handleAuthProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"oidcEnabled": s.cfg.OIDCEnabled,
	})
}

func (s *Server) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.OIDCEnabled {
		writeAPIError(w, http.StatusNotFound, "oidc is disabled")
		return
	}

	provider, verifier, oauthCfg, err := s.oidcObjects(r.Context())
	if err != nil {
		s.logger.ErrorContext(r.Context(), "init oidc failed", "error", err)
		writeAPIError(w, http.StatusBadGateway, "oidc provider unavailable")
		return
	}
	_ = provider
	_ = verifier

	state, err := randomToken(32)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to start oidc flow")
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to start oidc flow")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((10 * time.Minute).Seconds()),
		Expires:  time.Now().UTC().Add(10 * time.Minute),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     oidcNonceCookieName,
		Value:    nonce,
		Path:     "/",
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((10 * time.Minute).Seconds()),
		Expires:  time.Now().UTC().Add(10 * time.Minute),
	})

	authURL := oauthCfg.AuthCodeURL(state, oidc.Nonce(nonce))
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.OIDCEnabled {
		writeAPIError(w, http.StatusNotFound, "oidc is disabled")
		return
	}

	stateCookie, err := r.Cookie(oidcStateCookieName)
	if err != nil || strings.TrimSpace(stateCookie.Value) == "" {
		writeAPIError(w, http.StatusBadRequest, "missing oidc state")
		return
	}
	if r.URL.Query().Get("state") != stateCookie.Value {
		writeAPIError(w, http.StatusBadRequest, "invalid oidc state")
		return
	}
	nonceCookie, err := r.Cookie(oidcNonceCookieName)
	if err != nil || strings.TrimSpace(nonceCookie.Value) == "" {
		writeAPIError(w, http.StatusBadRequest, "missing oidc nonce")
		return
	}

	provider, verifier, oauthCfg, err := s.oidcObjects(r.Context())
	if err != nil {
		s.logger.ErrorContext(r.Context(), "init oidc failed", "error", err)
		writeAPIError(w, http.StatusBadGateway, "oidc provider unavailable")
		return
	}
	_ = provider

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		writeAPIError(w, http.StatusBadRequest, "missing oidc code")
		return
	}

	token, err := oauthCfg.Exchange(r.Context(), code)
	if err != nil {
		s.logger.WarnContext(r.Context(), "oidc exchange failed", "error", err)
		writeAPIError(w, http.StatusUnauthorized, "oidc exchange failed")
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		writeAPIError(w, http.StatusUnauthorized, "missing id_token")
		return
	}

	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "invalid id_token")
		return
	}
	if idToken.Nonce != nonceCookie.Value {
		writeAPIError(w, http.StatusUnauthorized, "invalid oidc nonce")
		return
	}

	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		writeAPIError(w, http.StatusUnauthorized, "invalid oidc claims")
		return
	}

	makeAdmin := false
	if s.cfg.OIDCAdminGroup != "" {
		for _, g := range claims.Groups {
			if strings.EqualFold(strings.TrimSpace(g), strings.TrimSpace(s.cfg.OIDCAdminGroup)) {
				makeAdmin = true
				break
			}
		}
	}

	user, err := s.auth.ResolveOrCreateOIDCUser(
		r.Context(),
		s.cfg.OIDCIssuerURL,
		claims.Subject,
		claims.Email,
		claims.PreferredUsername,
		claims.Name,
		makeAdmin,
	)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "resolve oidc user failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to resolve oidc user")
		return
	}

	sessionID, expiresAt, err := s.auth.LoginByUserID(r.Context(), user.ID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "create oidc session failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	secure := requestIsSecure(r)
	http.SetCookie(w, s.auth.SessionCookie(sessionID, expiresAt, secure))
	http.SetCookie(w, csrfCookie(secure))
	clearCookie(w, oidcStateCookieName, secure)
	clearCookie(w, oidcNonceCookieName, secure)

	http.Redirect(w, r, "/", http.StatusFound)
}

func clearCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (s *Server) oidcObjects(ctx context.Context) (*oidc.Provider, *oidc.IDTokenVerifier, *oauth2.Config, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, s.cfg.OIDCIssuerURL)
	if err != nil {
		return nil, nil, nil, err
	}
	oauthCfg := &oauth2.Config{
		ClientID:     s.cfg.OIDCClientID,
		ClientSecret: s.cfg.OIDCClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  s.cfg.OIDCRedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: s.cfg.OIDCClientID})
	return provider, verifier, oauthCfg, nil
}

// This endpoint is useful for quick diagnostics and frontend capability checks.
func (s *Server) handleOIDCDebugClaims(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.OIDCEnabled {
		writeAPIError(w, http.StatusNotFound, "oidc is disabled")
		return
	}

	provider, _, _, err := s.oidcObjects(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "oidc provider unavailable")
		return
	}

	claims := map[string]any{}
	if err := provider.Claims(&claims); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		writeAPIError(w, http.StatusInternalServerError, "failed to read provider metadata")
		return
	}
	writeJSON(w, http.StatusOK, claims)
}

func writeJSONRaw(w http.ResponseWriter, code int, payload []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(payload)
}

func decodeJSONBody(r *http.Request, out any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}
