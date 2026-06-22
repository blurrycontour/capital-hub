package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aditya/capital-hub/internal/config"
)

// User is the authenticated principal used by the API layer.
type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	IsAdmin     bool   `json:"isAdmin"`
	IsActive    bool   `json:"isActive"`
}

// Service provides authentication/session operations backed by SQLite.
type Service struct {
	db  *sql.DB
	cfg *config.Config
}

// NewService builds an auth service.
func NewService(db *sql.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

// Login validates credentials and creates a fresh server-side session.
func (s *Service) Login(ctx context.Context, identifier, password, userAgent, remoteAddr string) (*User, string, time.Time, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || password == "" {
		return nil, "", time.Time{}, errors.New("identifier and password are required")
	}

	var user User
	var passwordHash sql.NullString
	var isAdmin int
	var isActive int

	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, display_name, is_admin, is_active, password_hash
		 FROM users WHERE (username = ? OR email = ?) LIMIT 1`,
		identifier,
		identifier,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &isAdmin, &isActive, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", time.Time{}, errors.New("invalid credentials")
		}
		return nil, "", time.Time{}, fmt.Errorf("query user: %w", err)
	}
	user.IsAdmin = isAdmin == 1
	user.IsActive = isActive == 1

	if !user.IsActive {
		return nil, "", time.Time{}, errors.New("user is inactive")
	}
	if !passwordHash.Valid || passwordHash.String == "" {
		return nil, "", time.Time{}, errors.New("local password login not enabled for this account")
	}
	valid, err := VerifyPassword(password, passwordHash.String)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return nil, "", time.Time{}, errors.New("invalid credentials")
	}

	sessionID, err := randomToken(32)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("generate session id: %w", err)
	}

	expiresAt := time.Now().UTC().Add(time.Duration(s.cfg.SessionTTLHours) * time.Hour)
	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, user_id, user_agent, ip, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		sessionID,
		user.ID,
		truncate(userAgent, 512),
		truncate(clientIP(remoteAddr), 128),
		sqliteTime(expiresAt),
	); err != nil {
		return nil, "", time.Time{}, fmt.Errorf("create session: %w", err)
	}

	return &user, sessionID, expiresAt, nil
}

// Logout removes a session from the server-side store.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// CurrentUser resolves a user from a valid non-expired session ID.
func (s *Service) CurrentUser(ctx context.Context, sessionID string) (*User, error) {
	if sessionID == "" {
		return nil, sql.ErrNoRows
	}

	var user User
	var isAdmin int
	var isActive int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT u.id, u.username, u.email, u.display_name, u.is_admin, u.is_active
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.id = ?
		   AND datetime(s.expires_at) > datetime('now')
		 LIMIT 1`,
		sessionID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &isAdmin, &isActive)
	if err != nil {
		return nil, err
	}
	user.IsAdmin = isAdmin == 1
	user.IsActive = isActive == 1
	if !user.IsActive {
		return nil, errors.New("user is inactive")
	}
	return &user, nil
}

// ListUsers returns all users ordered by creation date.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, username, email, display_name, is_admin, is_active
		 FROM users ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		var isAdmin int
		var isActive int
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &isAdmin, &isActive); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.IsAdmin = isAdmin == 1
		u.IsActive = isActive == 1
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

// EnsureBootstrapAdmin creates an initial admin user if configured and missing.
func (s *Service) EnsureBootstrapAdmin(ctx context.Context) error {
	username := strings.TrimSpace(s.cfg.BootstrapAdminUsername)
	email := strings.TrimSpace(s.cfg.BootstrapAdminEmail)
	password := s.cfg.BootstrapAdminPassword
	if username == "" && email == "" && password == "" {
		return nil
	}
	if username == "" || email == "" || password == "" {
		return errors.New("CH_BOOTSTRAP_ADMIN_USERNAME, CH_BOOTSTRAP_ADMIN_EMAIL and CH_BOOTSTRAP_ADMIN_PASSWORD must all be set together")
	}
	if err := ValidatePasswordStrength(password); err != nil {
		return fmt.Errorf("invalid bootstrap admin password: %w", err)
	}

	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE username = ? OR email = ? LIMIT 1`, username, email).Scan(&exists)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check bootstrap admin existence: %w", err)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO users (username, email, password_hash, display_name, is_admin, is_active)
		 VALUES (?, ?, ?, ?, 1, 1)`,
		username,
		email,
		hash,
		username,
	)
	if err != nil {
		return fmt.Errorf("insert bootstrap admin: %w", err)
	}
	return nil
}

// SessionCookie returns a secure cookie configured for the current environment.
func (s *Service) SessionCookie(sessionID string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   !s.cfg.IsDev(),
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	}
}

// ClearSessionCookie expires the session cookie.
func (s *Service) ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   !s.cfg.IsDev(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func randomToken(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func sqliteTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
