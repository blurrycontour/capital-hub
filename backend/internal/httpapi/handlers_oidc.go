package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// oidcEffectiveConfig is the merged OIDC configuration (env vars override DB
// values). It is built fresh per request so admin UI changes take effect
// without a server restart.
type oidcEffectiveConfig struct {
	Enabled           bool
	IssuerURL         string
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	AdminGroup        string
	ProviderName      string
	AllowRegistration bool
	// EnvFields contains the names of fields whose values come from environment
	// variables and therefore cannot be changed via the admin UI.
	EnvFields []string
}

// loadEffectiveOIDCConfig merges the persistent DB settings with any
// environment-variable overrides. Environment variables take priority.
func (s *Server) loadEffectiveOIDCConfig(ctx context.Context) (*oidcEffectiveConfig, error) {
	get := func(key string) string {
		v, _ := s.getSetting(ctx, key)
		return v
	}

	cfg := &oidcEffectiveConfig{
		Enabled:           get("oidc_enabled") == "1" || strings.EqualFold(get("oidc_enabled"), "true"),
		IssuerURL:         get("oidc_issuer_url"),
		ClientID:          get("oidc_client_id"),
		ClientSecret:      get("oidc_client_secret"),
		RedirectURL:       get("oidc_redirect_url"),
		AdminGroup:        get("oidc_admin_group"),
		ProviderName:      get("oidc_provider_name"),
		AllowRegistration: get("oidc_allow_registration") != "0" && !strings.EqualFold(get("oidc_allow_registration"), "false"),
	}
	if cfg.ProviderName == "" {
		cfg.ProviderName = s.cfg.OIDCProviderName
	}

	// Environment variables override DB values.
	if v := os.Getenv("CH_OIDC_ENABLED"); v != "" {
		cfg.Enabled = v == "1" || strings.EqualFold(v, "true")
		cfg.EnvFields = append(cfg.EnvFields, "enabled")
	} else if s.cfg.OIDCEnabled {
		cfg.Enabled = true
		cfg.EnvFields = append(cfg.EnvFields, "enabled")
	}
	if v := s.cfg.OIDCIssuerURL; v != "" {
		cfg.IssuerURL = v
		cfg.EnvFields = append(cfg.EnvFields, "issuerUrl")
	}
	if v := s.cfg.OIDCClientID; v != "" {
		cfg.ClientID = v
		cfg.EnvFields = append(cfg.EnvFields, "clientId")
	}
	if v := s.cfg.OIDCClientSecret; v != "" {
		cfg.ClientSecret = v
		cfg.EnvFields = append(cfg.EnvFields, "clientSecret")
	}
	if v := s.cfg.OIDCRedirectURL; v != "" {
		cfg.RedirectURL = v
		cfg.EnvFields = append(cfg.EnvFields, "redirectUrl")
	}
	if v := s.cfg.OIDCAdminGroup; v != "" {
		cfg.AdminGroup = v
		cfg.EnvFields = append(cfg.EnvFields, "adminGroup")
	}
	// Only count provider name as env-controlled when it differs from the default.
	if v := os.Getenv("CH_OIDC_PROVIDER_NAME"); v != "" && v != "OIDC" {
		cfg.ProviderName = v
		cfg.EnvFields = append(cfg.EnvFields, "providerName")
	}
	if v := os.Getenv("CH_OIDC_ALLOW_REGISTRATION"); v != "" {
		cfg.AllowRegistration = v != "0" && !strings.EqualFold(v, "false")
		cfg.EnvFields = append(cfg.EnvFields, "allowRegistration")
	} else {
		cfg.AllowRegistration = s.cfg.OIDCAllowRegistration
	}

	if cfg.EnvFields == nil {
		cfg.EnvFields = []string{}
	}
	return cfg, nil
}

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
	cfg, err := s.loadEffectiveOIDCConfig(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to load provider config")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"oidcEnabled":      cfg.Enabled,
		"oidcProviderName": cfg.ProviderName,
	})
}

func (s *Server) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	oidcCfg, err := s.loadEffectiveOIDCConfig(r.Context())
	if err != nil || !oidcCfg.Enabled {
		writeAPIError(w, http.StatusNotFound, "oidc is disabled")
		return
	}

	provider, verifier, oauthCfg, err := s.oidcObjects(r.Context(), oidcCfg)
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
	oidcCfg, err := s.loadEffectiveOIDCConfig(r.Context())
	if err != nil || !oidcCfg.Enabled {
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

	provider, verifier, oauthCfg, err2 := s.oidcObjects(r.Context(), oidcCfg)
	if err2 != nil {
		s.logger.ErrorContext(r.Context(), "init oidc failed", "error", err2)
		writeAPIError(w, http.StatusBadGateway, "oidc provider unavailable")
		return
	}

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

	// Many providers (e.g. Authelia) only put the `sub` claim in the ID token
	// and expose email/name/groups via the UserInfo endpoint. Fetch it and use
	// it to fill in any details the ID token did not carry.
	if userInfo, uiErr := provider.UserInfo(r.Context(), oauth2.StaticTokenSource(token)); uiErr == nil {
		var infoClaims oidcClaims
		if err := userInfo.Claims(&infoClaims); err == nil {
			if claims.Email == "" {
				claims.Email = infoClaims.Email
			}
			if claims.Name == "" {
				claims.Name = infoClaims.Name
			}
			if claims.PreferredUsername == "" {
				claims.PreferredUsername = infoClaims.PreferredUsername
			}
			if len(claims.Groups) == 0 {
				claims.Groups = infoClaims.Groups
			}
		}
	} else {
		s.logger.WarnContext(r.Context(), "oidc userinfo fetch failed", "error", uiErr)
	}

	makeAdmin := false
	if oidcCfg.AdminGroup != "" {
		for _, g := range claims.Groups {
			if strings.EqualFold(strings.TrimSpace(g), strings.TrimSpace(oidcCfg.AdminGroup)) {
				makeAdmin = true
				break
			}
		}
	}

	user, err := s.auth.ResolveOrCreateOIDCUser(
		r.Context(),
		oidcCfg.IssuerURL,
		claims.Subject,
		claims.Email,
		claims.PreferredUsername,
		claims.Name,
		makeAdmin,
		oidcCfg.AllowRegistration,
	)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "resolve oidc user failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to resolve oidc user")
		return
	}

	sessionID, expiresAt, err := s.auth.LoginByUserID(r.Context(), user.ID, r.UserAgent(), s.clientIP(r))
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

func (s *Server) oidcObjects(ctx context.Context, cfg *oidcEffectiveConfig) (*oidc.Provider, *oidc.IDTokenVerifier, *oauth2.Config, error) {
	if cfg.IssuerURL == "" || cfg.ClientID == "" {
		return nil, nil, nil, errors.New("oidc issuer URL and client ID are required")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, nil, nil, err
	}
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	return provider, verifier, oauthCfg, nil
}

// This endpoint is useful for quick diagnostics and frontend capability checks.
func (s *Server) handleOIDCDebugClaims(w http.ResponseWriter, r *http.Request) {
	oidcCfg, err := s.loadEffectiveOIDCConfig(r.Context())
	if err != nil || !oidcCfg.Enabled {
		writeAPIError(w, http.StatusNotFound, "oidc is disabled")
		return
	}

	provider, _, _, err := s.oidcObjects(r.Context(), oidcCfg)
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
