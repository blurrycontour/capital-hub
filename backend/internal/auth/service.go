package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"regexp"
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
	AvatarPath  string `json:"avatarPath"`
	IsAdmin     bool   `json:"isAdmin"`
	IsActive    bool   `json:"isActive"`
	Role        string `json:"role"`
	// HasPassword is false for accounts that can only sign in via OIDC (no local
	// password set), so the UI can disable password changes for them.
	HasPassword bool `json:"hasPassword"`
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
		`SELECT id, username, email, display_name, avatar_path, is_admin, is_active, password_hash, role
		 FROM users WHERE (username = ? OR email = ?) LIMIT 1`,
		identifier,
		identifier,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.AvatarPath, &isAdmin, &isActive, &passwordHash, &user.Role)
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
	user.HasPassword = true
	valid, err := VerifyPassword(password, passwordHash.String)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return nil, "", time.Time{}, errors.New("invalid credentials")
	}

	sessionID, expiresAt, err := s.createSession(ctx, user.ID, userAgent, remoteAddr)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return &user, sessionID, expiresAt, nil
}

// LoginByUserID creates a fresh session for a known user (used by OIDC flow).
func (s *Service) LoginByUserID(ctx context.Context, userID int64, userAgent, remoteAddr string) (string, time.Time, error) {
	if userID == 0 {
		return "", time.Time{}, errors.New("user id is required")
	}
	return s.createSession(ctx, userID, userAgent, remoteAddr)
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
	var hasPassword int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT u.id, u.username, u.email, u.display_name, u.avatar_path, u.is_admin, u.is_active, u.role,
		        (u.password_hash IS NOT NULL AND u.password_hash != '') AS has_password
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.id = ?
		   AND datetime(s.expires_at) > datetime('now')
		 LIMIT 1`,
		sessionID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.AvatarPath, &isAdmin, &isActive, &user.Role, &hasPassword)
	if err != nil {
		return nil, err
	}
	user.IsAdmin = isAdmin == 1
	user.IsActive = isActive == 1
	user.HasPassword = hasPassword == 1
	if !user.IsActive {
		return nil, errors.New("user is inactive")
	}
	return &user, nil
}

// UpdateProfile updates the editable profile fields for a user and returns the
// refreshed record. Only display name and email are user-editable.
func (s *Service) UpdateProfile(ctx context.Context, userID int64, displayName, email string) (*User, error) {
	displayName = strings.TrimSpace(displayName)
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, errors.New("email is required")
	}

	_, err := s.db.ExecContext(
		ctx,
		`UPDATE users SET display_name = ?, email = ?, updated_at = datetime('now') WHERE id = ?`,
		displayName,
		email,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	var user User
	var isAdmin int
	var isActive int
	var hasPassword int
	err = s.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, display_name, avatar_path, is_admin, is_active, role,
		        (password_hash IS NOT NULL AND password_hash != '') AS has_password
		 FROM users WHERE id = ? LIMIT 1`,
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.AvatarPath, &isAdmin, &isActive, &user.Role, &hasPassword)
	if err != nil {
		return nil, fmt.Errorf("reload user: %w", err)
	}
	user.HasPassword = hasPassword == 1
	user.IsAdmin = isAdmin == 1
	user.IsActive = isActive == 1
	return &user, nil
}

// SetAvatar updates the user's avatar path and returns the previous one (so the
// caller can clean up the replaced file) along with the refreshed user.
func (s *Service) SetAvatar(ctx context.Context, userID int64, avatarPath string) (*User, string, error) {
	var prev string
	if err := s.db.QueryRowContext(ctx, `SELECT avatar_path FROM users WHERE id = ? LIMIT 1`, userID).Scan(&prev); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.New("user not found")
		}
		return nil, "", fmt.Errorf("lookup avatar: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET avatar_path = ?, updated_at = datetime('now') WHERE id = ?`,
		avatarPath, userID,
	); err != nil {
		return nil, "", fmt.Errorf("set avatar: %w", err)
	}

	var user User
	var isAdmin int
	var isActive int
	var hasPassword int
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, display_name, avatar_path, is_admin, is_active, role,
		        (password_hash IS NOT NULL AND password_hash != '') AS has_password
		 FROM users WHERE id = ? LIMIT 1`,
		userID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.AvatarPath, &isAdmin, &isActive, &user.Role, &hasPassword); err != nil {
		return nil, "", fmt.Errorf("reload user: %w", err)
	}
	user.IsAdmin = isAdmin == 1
	user.IsActive = isActive == 1
	user.HasPassword = hasPassword == 1
	return &user, prev, nil
}

// StatsIncludeShared reports whether the user wants collections shared with them
// to be counted in their dashboard portfolio totals.
func (s *Service) StatsIncludeShared(ctx context.Context, userID int64) (bool, error) {
	var v int
	if err := s.db.QueryRowContext(ctx,
		`SELECT include_shared_in_stats FROM users WHERE id = ? LIMIT 1`, userID,
	).Scan(&v); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("user not found")
		}
		return false, fmt.Errorf("load stats preference: %w", err)
	}
	return v == 1, nil
}

// SetStatsIncludeShared updates the dashboard "include shared collections"
// preference for the user.
func (s *Service) SetStatsIncludeShared(ctx context.Context, userID int64, include bool) error {
	v := 0
	if include {
		v = 1
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET include_shared_in_stats = ?, updated_at = datetime('now') WHERE id = ?`,
		v, userID,
	); err != nil {
		return fmt.Errorf("set stats preference: %w", err)
	}
	return nil
}

// Preferences holds the per-user UI and notification preferences.
type Preferences struct {
	IncludeSharedInStats   bool   `json:"includeSharedInStats"`
	AmountDecimals         int    `json:"amountDecimals"`
	NumberFormat           string `json:"numberFormat"`
	NotifyCollectionShared bool   `json:"notifyCollectionShared"`
	NotifyItemAdded        bool   `json:"notifyItemAdded"`
	NotifyEntryAdded       bool   `json:"notifyEntryAdded"`
}

func clampDecimals(n int) int {
	if n < 0 {
		return 0
	}
	if n > 2 {
		return 2
	}
	return n
}

// normalizeNumberFormat ensures the stored money formatting style is one of the
// supported values, defaulting to international grouping.
func normalizeNumberFormat(s string) string {
	switch s {
	case "indian", "european":
		return s
	default:
		return "international"
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GetPreferences loads every preference for a user.
func (s *Service) GetPreferences(ctx context.Context, userID int64) (Preferences, error) {
	var p Preferences
	var includeShared, notifShared, notifItem, notifEntry int
	if err := s.db.QueryRowContext(ctx,
		`SELECT include_shared_in_stats, amount_decimals, number_format, notify_collection_shared, notify_item_added, notify_entry_added
		 FROM users WHERE id = ? LIMIT 1`, userID,
	).Scan(&includeShared, &p.AmountDecimals, &p.NumberFormat, &notifShared, &notifItem, &notifEntry); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Preferences{}, errors.New("user not found")
		}
		return Preferences{}, fmt.Errorf("load preferences: %w", err)
	}
	p.IncludeSharedInStats = includeShared == 1
	p.NotifyCollectionShared = notifShared == 1
	p.NotifyItemAdded = notifItem == 1
	p.NotifyEntryAdded = notifEntry == 1
	p.AmountDecimals = clampDecimals(p.AmountDecimals)
	p.NumberFormat = normalizeNumberFormat(p.NumberFormat)
	return p, nil
}

// SetPreferences persists every preference for a user.
func (s *Service) SetPreferences(ctx context.Context, userID int64, p Preferences) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET include_shared_in_stats = ?, amount_decimals = ?, number_format = ?, notify_collection_shared = ?,
		 notify_item_added = ?, notify_entry_added = ?, updated_at = datetime('now') WHERE id = ?`,
		boolToInt(p.IncludeSharedInStats), clampDecimals(p.AmountDecimals), normalizeNumberFormat(p.NumberFormat), boolToInt(p.NotifyCollectionShared),
		boolToInt(p.NotifyItemAdded), boolToInt(p.NotifyEntryAdded), userID,
	); err != nil {
		return fmt.Errorf("set preferences: %w", err)
	}
	return nil
}

// ListUsers returns all users ordered by creation date.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, username, email, display_name, avatar_path, is_admin, is_active, role
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
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.AvatarPath, &isAdmin, &isActive, &u.Role); err != nil {
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

// ChangePassword verifies the current password then replaces it with newPassword.
func (s *Service) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	var passwordHash sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = ? LIMIT 1`, userID).Scan(&passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return fmt.Errorf("lookup user: %w", err)
	}
	if !passwordHash.Valid || passwordHash.String == "" {
		return errors.New("password login is not enabled for this account")
	}
	valid, err := VerifyPassword(currentPassword, passwordHash.String)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return errors.New("current password is incorrect")
	}
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE id = ?`, hash, userID)
	return err
}

// AdminCreateUser creates a new local username/email + password account with
// the given role.
func (s *Service) AdminCreateUser(ctx context.Context, username, email, displayName, password, role string) (*User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(strings.ToLower(email))
	displayName = strings.TrimSpace(displayName)
	role = strings.TrimSpace(strings.ToLower(role))

	if username == "" || email == "" {
		return nil, errors.New("username and email are required")
	}
	switch role {
	case "administrator", "editor", "reader":
	default:
		return nil, errors.New("role must be one of: administrator, editor, reader")
	}
	if err := ValidatePasswordStrength(password); err != nil {
		return nil, err
	}

	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE username = ? OR email = ? LIMIT 1`, username, email).Scan(&exists)
	if err == nil {
		return nil, errors.New("a user with that username or email already exists")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check user existence: %w", err)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	if displayName == "" {
		displayName = username
	}
	isAdmin := 0
	if role == "administrator" {
		isAdmin = 1
	}

	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (username, email, password_hash, display_name, is_admin, is_active, role)
		 VALUES (?, ?, ?, ?, ?, 1, ?)`,
		username, email, hash, displayName, isAdmin, role,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("fetch new user id: %w", err)
	}

	return &User{
		ID:          id,
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		IsAdmin:     isAdmin == 1,
		IsActive:    true,
		Role:        role,
	}, nil
}

// AdminUpdateUser updates a user's role and active status. The caller must not
// be the same as the target (admins cannot demote themselves).
func (s *Service) AdminUpdateUser(ctx context.Context, callerID, targetID int64, role string, isActive bool) (*User, error) {
	role = strings.TrimSpace(strings.ToLower(role))
	switch role {
	case "administrator", "editor", "reader":
	default:
		return nil, errors.New("role must be one of: administrator, editor, reader")
	}
	if callerID == targetID && role != "administrator" {
		return nil, errors.New("cannot remove administrator role from your own account")
	}
	isAdmin := 0
	if role == "administrator" {
		isAdmin = 1
	}
	isActiveInt := 0
	if isActive {
		isActiveInt = 1
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET role = ?, is_admin = ?, is_active = ?, updated_at = datetime('now') WHERE id = ?`,
		role, isAdmin, isActiveInt, targetID,
	)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	var u User
	var isAdminR int
	var isActiveR int
	if err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, display_name, avatar_path, is_admin, is_active, role FROM users WHERE id = ? LIMIT 1`,
		targetID,
	).Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.AvatarPath, &isAdminR, &isActiveR, &u.Role); err != nil {
		return nil, fmt.Errorf("reload user: %w", err)
	}
	u.IsAdmin = isAdminR == 1
	u.IsActive = isActiveR == 1
	return &u, nil
}

// AdminDeleteUser permanently removes a user. Callers cannot delete themselves.
func (s *Service) AdminDeleteUser(ctx context.Context, callerID, targetID int64) error {
	if callerID == targetID {
		return errors.New("cannot delete your own account")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, targetID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// accountDeletionCodeTTL is how long a requested deletion code stays valid.
const accountDeletionCodeTTL = 15 * time.Minute

// hashDeletionCode returns a hex-encoded SHA-256 digest of the code so the
// plaintext is never stored at rest.
func hashDeletionCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// RequestAccountDeletion generates a one-time confirmation code for the given
// user, stores its hash, and returns the plaintext code together with the
// user's email so the caller can deliver it. An error is returned when the user
// has no email address on file.
func (s *Service) RequestAccountDeletion(ctx context.Context, userID int64) (code, email string, err error) {
	if userID == 0 {
		return "", "", errors.New("user id is required")
	}

	if err := s.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id = ?`, userID).Scan(&email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", errors.New("user not found")
		}
		return "", "", fmt.Errorf("load user email: %w", err)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return "", "", errors.New("no email address on file for this account")
	}

	// Six-digit numeric code.
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", "", fmt.Errorf("generate code: %w", err)
	}
	code = fmt.Sprintf("%06d", n.Int64())
	expires := time.Now().UTC().Add(accountDeletionCodeTTL).Format("2006-01-02 15:04:05")

	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO account_deletion_codes (user_id, code_hash, expires_at, created_at)
		 VALUES (?, ?, ?, datetime('now'))
		 ON CONFLICT(user_id) DO UPDATE SET
		   code_hash = excluded.code_hash,
		   expires_at = excluded.expires_at,
		   created_at = datetime('now')`,
		userID,
		hashDeletionCode(code),
		expires,
	); err != nil {
		return "", "", fmt.Errorf("store deletion code: %w", err)
	}

	return code, email, nil
}

// DeleteOwnAccount verifies the supplied confirmation code and, on success,
// permanently removes the user account (cascading to owned data).
func (s *Service) DeleteOwnAccount(ctx context.Context, userID int64, code string) error {
	if userID == 0 {
		return errors.New("user id is required")
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return errors.New("confirmation code is required")
	}

	var storedHash, expiresAt string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT code_hash, expires_at FROM account_deletion_codes WHERE user_id = ?`,
		userID,
	).Scan(&storedHash, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("no pending deletion request; request a code first")
		}
		return fmt.Errorf("load deletion code: %w", err)
	}

	expires, err := time.Parse("2006-01-02 15:04:05", expiresAt)
	if err != nil {
		return fmt.Errorf("parse expiry: %w", err)
	}
	if time.Now().UTC().After(expires) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM account_deletion_codes WHERE user_id = ?`, userID)
		return errors.New("confirmation code has expired; request a new one")
	}

	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(hashDeletionCode(code))) != 1 {
		return errors.New("incorrect confirmation code")
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// ResolveOrCreateOIDCUser finds or creates a user for an OIDC identity and
// links the identity to that user. When allowRegistration is false and no
// matching account exists (by existing OIDC identity or by email), an error is
// returned instead of creating a new user.
func (s *Service) ResolveOrCreateOIDCUser(
	ctx context.Context,
	provider,
	subject,
	email,
	preferredUsername,
	displayName string,
	makeAdmin bool,
	allowRegistration bool,
) (*User, error) {
	provider = strings.TrimSpace(provider)
	subject = strings.TrimSpace(subject)
	email = strings.TrimSpace(strings.ToLower(email))
	preferredUsername = strings.TrimSpace(preferredUsername)
	displayName = strings.TrimSpace(displayName)

	if provider == "" || subject == "" {
		return nil, errors.New("provider and subject are required")
	}

	var user User
	var isAdmin int
	var isActive int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT u.id, u.username, u.email, u.display_name, u.avatar_path, u.is_admin, u.is_active, u.role
		 FROM oidc_identities oi
		 JOIN users u ON u.id = oi.user_id
		 WHERE oi.provider = ? AND oi.subject = ?
		 LIMIT 1`,
		provider,
		subject,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.AvatarPath, &isAdmin, &isActive, &user.Role)
	if err == nil {
		user.IsAdmin = isAdmin == 1 || makeAdmin
		user.IsActive = isActive == 1
		if makeAdmin && !user.IsAdmin {
			if _, err := s.db.ExecContext(ctx, `UPDATE users SET is_admin = 1, role = 'administrator', updated_at = datetime('now') WHERE id = ?`, user.ID); err != nil {
				return nil, fmt.Errorf("promote oidc user to admin: %w", err)
			}
			user.IsAdmin = true
			user.Role = "administrator"
		}
		// Backfill display name / email for accounts created before the
		// provider returned these details (e.g. via UserInfo on later logins).
		if user.DisplayName == "" && displayName != "" {
			if _, err := s.db.ExecContext(ctx, `UPDATE users SET display_name = ?, updated_at = datetime('now') WHERE id = ?`, displayName, user.ID); err != nil {
				return nil, fmt.Errorf("backfill display name: %w", err)
			}
			user.DisplayName = displayName
		}
		if (user.Email == "" || strings.HasSuffix(user.Email, "@oidc.local")) && email != "" {
			// Ignore unique-constraint conflicts: another account may already
			// own this email, in which case we keep the existing placeholder.
			if _, err := s.db.ExecContext(ctx, `UPDATE users SET email = ?, updated_at = datetime('now') WHERE id = ?`, email, user.ID); err == nil {
				user.Email = email
			}
		}
		if !user.IsActive {
			return nil, errors.New("user is inactive")
		}
		return &user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup oidc identity: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin oidc transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userID, userEmail, userName, userDisplayName, userRole, err := s.findOrCreateOIDCUserTx(ctx, tx, email, preferredUsername, displayName, makeAdmin, allowRegistration)
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO oidc_identities (user_id, provider, subject)
		 VALUES (?, ?, ?)`,
		userID,
		provider,
		subject,
	); err != nil {
		return nil, fmt.Errorf("insert oidc identity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit oidc transaction: %w", err)
	}

	return &User{
		ID:          userID,
		Username:    userName,
		Email:       userEmail,
		DisplayName: userDisplayName,
		IsAdmin:     makeAdmin,
		IsActive:    true,
		Role:        userRole,
	}, nil
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
		`INSERT INTO users (username, email, password_hash, display_name, is_admin, is_active, role)
		 VALUES (?, ?, ?, ?, 1, 1, 'administrator')`,
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

func (s *Service) createSession(ctx context.Context, userID int64, userAgent, remoteAddr string) (string, time.Time, error) {
	sessionID, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate session id: %w", err)
	}

	expiresAt := time.Now().UTC().Add(time.Duration(s.cfg.SessionTTLHours) * time.Hour)
	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, user_id, user_agent, ip, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		sessionID,
		userID,
		truncate(userAgent, 512),
		truncate(clientIP(remoteAddr), 128),
		sqliteTime(expiresAt),
	); err != nil {
		return "", time.Time{}, fmt.Errorf("create session: %w", err)
	}
	return sessionID, expiresAt, nil
}

func (s *Service) findOrCreateOIDCUserTx(
	ctx context.Context,
	tx *sql.Tx,
	email,
	preferredUsername,
	displayName string,
	makeAdmin bool,
	allowRegistration bool,
) (int64, string, string, string, string, error) {
	if email != "" {
		var existingID int64
		var existingUsername string
		var existingDisplayName string
		var existingRole string
		var isAdmin int
		err := tx.QueryRowContext(
			ctx,
			`SELECT id, username, display_name, is_admin, role FROM users WHERE email = ? LIMIT 1`,
			email,
		).Scan(&existingID, &existingUsername, &existingDisplayName, &isAdmin, &existingRole)
		if err == nil {
			if makeAdmin && isAdmin != 1 {
				if _, err := tx.ExecContext(ctx, `UPDATE users SET is_admin = 1, role = 'administrator', updated_at = datetime('now') WHERE id = ?`, existingID); err != nil {
					return 0, "", "", "", "", fmt.Errorf("promote existing user to admin: %w", err)
				}
				existingRole = "administrator"
			}
			if existingDisplayName == "" && displayName != "" {
				if _, err := tx.ExecContext(ctx, `UPDATE users SET display_name = ?, updated_at = datetime('now') WHERE id = ?`, displayName, existingID); err != nil {
					return 0, "", "", "", "", fmt.Errorf("update display name: %w", err)
				}
				existingDisplayName = displayName
			}
			return existingID, email, existingUsername, existingDisplayName, existingRole, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, "", "", "", "", fmt.Errorf("lookup user by email: %w", err)
		}
	}
	if !allowRegistration {
		return 0, "", "", "", "", errors.New("user registration via OIDC is disabled")
	}

	usernameBase := normalizeUsername(preferredUsername)
	if usernameBase == "" {
		if email != "" {
			usernameBase = normalizeUsername(strings.SplitN(email, "@", 2)[0])
		}
	}
	if usernameBase == "" {
		usernameBase = "user"
	}
	username, err := uniqueUsernameTx(ctx, tx, usernameBase)
	if err != nil {
		return 0, "", "", "", "", err
	}

	userEmail := email
	if userEmail == "" {
		userEmail, err = uniqueGeneratedEmailTx(ctx, tx, username)
		if err != nil {
			return 0, "", "", "", "", err
		}
	}

	if displayName == "" {
		displayName = username
	}

	adminInt := 0
	role := "editor"
	if makeAdmin {
		adminInt = 1
		role = "administrator"
	}

	res, err := tx.ExecContext(
		ctx,
		`INSERT INTO users (username, email, display_name, is_admin, is_active, role)
		 VALUES (?, ?, ?, ?, 1, ?)`,
		username,
		userEmail,
		displayName,
		adminInt,
		role,
	)
	if err != nil {
		return 0, "", "", "", "", fmt.Errorf("insert oidc user: %w", err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		return 0, "", "", "", "", fmt.Errorf("fetch oidc user id: %w", err)
	}

	return userID, userEmail, username, displayName, role, nil
}

var usernameSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)

func normalizeUsername(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = usernameSanitizer.ReplaceAllString(v, "")
	v = strings.Trim(v, "._-")
	if len(v) > 32 {
		v = v[:32]
	}
	return v
}

func uniqueUsernameTx(ctx context.Context, tx *sql.Tx, base string) (string, error) {
	for i := 0; i < 1000; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s%d", base, i+1)
		}
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM users WHERE username = ? LIMIT 1`, candidate).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("check username uniqueness: %w", err)
		}
	}
	return "", errors.New("failed to allocate unique username")
}

func uniqueGeneratedEmailTx(ctx context.Context, tx *sql.Tx, username string) (string, error) {
	for i := 0; i < 1000; i++ {
		local := username
		if i > 0 {
			local = fmt.Sprintf("%s%d", username, i+1)
		}
		candidate := fmt.Sprintf("%s@oidc.local", local)
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM users WHERE email = ? LIMIT 1`, candidate).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("check email uniqueness: %w", err)
		}
	}
	return "", errors.New("failed to allocate generated email")
}

// SessionCookie returns a session cookie. The secure flag should reflect
// whether the request arrived over HTTPS so the cookie is stored both for
// plain-HTTP deployments and HTTPS deployments behind a TLS proxy.
func (s *Service) SessionCookie(sessionID string, expiresAt time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	}
}

// ClearSessionCookie expires the session cookie.
func (s *Service) ClearSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
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
