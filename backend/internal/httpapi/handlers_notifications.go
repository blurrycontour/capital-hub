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
