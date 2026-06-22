// Package notify provides in-app and email notification primitives.
package notify

import (
	"context"
	"database/sql"
	"fmt"
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
