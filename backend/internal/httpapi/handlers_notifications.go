package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsed
	}

	notifications, err := s.notify.ListByUser(r.Context(), user.ID, limit)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "list notifications failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": notifications})
}

func (s *Server) handleUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	count, err := s.notify.UnreadCount(r.Context(), user.ID)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "unread count failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to get unread count")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	if err := s.notify.MarkRead(r.Context(), user.ID, id); err != nil {
		s.logger.ErrorContext(r.Context(), "mark notification read failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to mark notification read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMarkNotificationUnread(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := s.notify.MarkUnread(r.Context(), user.ID, id); err != nil {
		s.logger.ErrorContext(r.Context(), "mark notification unread failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to mark notification unread")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteNotification(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := s.notify.DeleteNotification(r.Context(), user.ID, id); err != nil {
		s.logger.ErrorContext(r.Context(), "delete notification failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to delete notification")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err := s.notify.MarkAllRead(r.Context(), user.ID); err != nil {
		s.logger.ErrorContext(r.Context(), "mark all read failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to mark all as read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMarkAllNotificationsUnread(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err := s.notify.MarkAllUnread(r.Context(), user.ID); err != nil {
		s.logger.ErrorContext(r.Context(), "mark all unread failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to mark all as unread")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteAllNotifications(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r)
	if user == nil {
		writeAPIError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if err := s.notify.DeleteAll(r.Context(), user.ID); err != nil {
		s.logger.ErrorContext(r.Context(), "delete all notifications failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to delete all notifications")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
