package httpapi

import (
	_ "embed"
	"net/http"
	"strings"
)

// openapiSpec is the embedded OpenAPI 3 document describing the public API.
//
//go:embed docs/openapi.json
var openapiSpec []byte

// swaggerHTML renders the Swagger UI shell. It loads Swagger UI from a CDN,
// points it at the embedded spec, and injects the session credentials plus the
// CSRF header so authenticated users can exercise the API directly ("Try it
// out"). The session cookie is sent automatically by the browser; the CSRF
// token is read from the ch_csrf cookie for mutating requests.
const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>Capital Hub · API Docs</title>
	<link rel="icon" href="/logo.svg" />
	<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
	<style>
		body { margin: 0; background: #fafafa; }
		.topbar { display: none; }
	</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
	<script>
		window.onload = async () => {
			// Ensure the CSRF cookie exists so mutating "Try it out" calls succeed.
			try { await fetch('/api/v1/auth/csrf', { credentials: 'include' }); } catch (e) {}

			window.ui = SwaggerUIBundle({
				url: '/api/v1/openapi.json',
				dom_id: '#swagger-ui',
				deepLinking: true,
				docExpansion: 'list',
				defaultModelsExpandDepth: 0,
				presets: [SwaggerUIBundle.presets.apis],
				withCredentials: true,
				requestInterceptor: (req) => {
					req.credentials = 'include';
					const m = document.cookie.match(/(?:^|;\s*)ch_csrf=([^;]+)/);
					if (m) { req.headers['X-CSRF-Token'] = decodeURIComponent(m[1]); }
					return req;
				}
			});
		};
	</script>
</body>
</html>`

// handleOpenAPISpec serves the embedded OpenAPI document.
func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(openapiSpec)
}

// handleSwaggerUI serves the interactive API documentation page.
func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}

// requireAuthRedirect gates browser-facing pages: unauthenticated visitors are
// redirected to the login page instead of receiving a JSON 401.
func (s *Server) requireAuthRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie(s.cfg.SessionCookieName)
		if err != nil || strings.TrimSpace(sessionCookie.Value) == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		user, err := s.auth.CurrentUser(r.Context(), sessionCookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r.WithContext(withUser(r, user)))
	})
}
