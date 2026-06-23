// Package notify provides in-app and email notification primitives.
package notify

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Service is the notifications subsystem entrypoint.
type Service struct {
	db *sql.DB
}

// NewService creates a notifications service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// InAppInput describes a notification that should be shown in-app to a user.
type InAppInput struct {
	UserID int64
	Type   string
	Title  string
	Body   string
	Link   string
}

// Notification represents a persisted in-app notification.
type Notification struct {
	ID        int64      `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Link      string     `json:"link"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// CreateInApp stores a notification in the database.
func (s *Service) CreateInApp(ctx context.Context, in InAppInput) error {
	if in.UserID == 0 || in.Type == "" || in.Title == "" {
		return fmt.Errorf("invalid notification input")
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO notifications (user_id, type, title, body, link)
		 VALUES (?, ?, ?, ?, ?)`,
		in.UserID,
		in.Type,
		in.Title,
		in.Body,
		in.Link,
	)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

// ListByUser returns recent notifications for a user.
func (s *Service) ListByUser(ctx context.Context, userID int64, limit int) ([]Notification, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user id is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, type, title, body, link, read_at, created_at
		 FROM notifications
		 WHERE user_id = ?
		 ORDER BY datetime(created_at) DESC
		 LIMIT ?`,
		userID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	out := make([]Notification, 0)
	for rows.Next() {
		var n Notification
		var readAt sql.NullString
		var createdAt string
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.Link, &readAt, &createdAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		parsedCreated, err := parseSQLiteTime(createdAt)
		if err != nil {
			return nil, err
		}
		n.CreatedAt = parsedCreated
		if readAt.Valid && strings.TrimSpace(readAt.String) != "" {
			parsedRead, err := parseSQLiteTime(readAt.String)
			if err != nil {
				return nil, err
			}
			n.ReadAt = &parsedRead
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}
	return out, nil
}

// MarkRead marks a notification as read if it belongs to the given user.
func (s *Service) MarkRead(ctx context.Context, userID, notificationID int64) error {
	if userID == 0 || notificationID == 0 {
		return fmt.Errorf("user id and notification id are required")
	}
	res, err := s.db.ExecContext(
		ctx,
		`UPDATE notifications
		 SET read_at = datetime('now')
		 WHERE id = ? AND user_id = ? AND read_at IS NULL`,
		notificationID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	_, _ = res.RowsAffected()
	return nil
}

func parseSQLiteTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	t, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.UTC)
	if err == nil {
		return t, nil
	}
	// fallback for providers returning RFC3339 strings
	t, err = time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", v, err)
	}
	return t.UTC(), nil
}
