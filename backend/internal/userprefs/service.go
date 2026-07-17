// Package userprefs stores per-user UI and notification preferences that are
// specific to Capital Hub. Core authentication lives in go-authkit; these
// application-owned columns extend the shared users table.
package userprefs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Preferences holds the per-user UI and notification preferences.
type Preferences struct {
	IncludeSharedInStats   bool   `json:"includeSharedInStats"`
	AmountDecimals         int    `json:"amountDecimals"`
	NumberFormat           string `json:"numberFormat"`
	NotifyCollectionShared bool   `json:"notifyCollectionShared"`
	NotifyItemAdded        bool   `json:"notifyItemAdded"`
	NotifyEntryAdded       bool   `json:"notifyEntryAdded"`
}

// Service reads and writes preference columns on the users table.
type Service struct {
	db *sql.DB
}

// NewService builds a preferences service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

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

// Get loads every preference for a user.
func (s *Service) Get(ctx context.Context, userID int64) (Preferences, error) {
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

// Set persists every preference for a user.
func (s *Service) Set(ctx context.Context, userID int64, p Preferences) error {
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
