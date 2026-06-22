package httpapi

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type smtpSettingsResponse struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	From        string `json:"from"`
	UseTLS      bool   `json:"useTls"`
	PasswordSet bool   `json:"passwordSet"`
}

type smtpSettingsUpdate struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	UseTLS   bool   `json:"useTls"`
}

type smtpTestRequest struct {
	To string `json:"to"`
}

func (s *Server) handleAdminGetSMTPSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.loadSMTPSettings(r.Context())
	if err != nil {
		s.logger.ErrorContext(r.Context(), "load smtp settings failed", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to load smtp settings")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleAdminUpdateSMTPSettings(w http.ResponseWriter, r *http.Request) {
	var req smtpSettingsUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}

	req.Host = strings.TrimSpace(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	req.From = strings.TrimSpace(req.From)
	if req.Host == "" || req.Port <= 0 || req.Port > 65535 || req.From == "" {
		writeAPIError(w, http.StatusBadRequest, "host, valid port and from are required")
		return
	}
	if _, err := mail.ParseAddress(req.From); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid from email address")
		return
	}

	if err := s.upsertSetting(r.Context(), "smtp_host", req.Host); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to save smtp host")
		return
	}
	if err := s.upsertSetting(r.Context(), "smtp_port", strconv.Itoa(req.Port)); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to save smtp port")
		return
	}
	if err := s.upsertSetting(r.Context(), "smtp_username", req.Username); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to save smtp username")
		return
	}
	if req.Password != "" {
		if err := s.upsertSetting(r.Context(), "smtp_password", req.Password); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to save smtp password")
			return
		}
	}
	if err := s.upsertSetting(r.Context(), "smtp_from", req.From); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to save smtp from")
		return
	}
	if err := s.upsertSetting(r.Context(), "smtp_use_tls", boolToString(req.UseTLS)); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to save smtp tls setting")
		return
	}

	settings, err := s.loadSMTPSettings(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to load smtp settings")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleAdminTestSMTP(w http.ResponseWriter, r *http.Request) {
	var req smtpTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json payload")
		return
	}
	req.To = strings.TrimSpace(req.To)
	if req.To == "" {
		writeAPIError(w, http.StatusBadRequest, "to email is required")
		return
	}
	if _, err := mail.ParseAddress(req.To); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid recipient email address")
		return
	}

	settings, err := s.loadSMTPInternal(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to load smtp settings")
		return
	}
	if settings.Host == "" || settings.Port == 0 || settings.From == "" {
		writeAPIError(w, http.StatusBadRequest, "smtp settings are incomplete")
		return
	}

	if err := sendSMTPTestEmail(settings, req.To); err != nil {
		s.logger.ErrorContext(r.Context(), "smtp test email failed", "error", err)
		writeAPIError(w, http.StatusBadGateway, "smtp test failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type smtpInternal struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

func (s *Server) loadSMTPSettings(ctx context.Context) (*smtpSettingsResponse, error) {
	in, err := s.loadSMTPInternal(ctx)
	if err != nil {
		return nil, err
	}
	return &smtpSettingsResponse{
		Host:        in.Host,
		Port:        in.Port,
		Username:    in.Username,
		From:        in.From,
		UseTLS:      in.UseTLS,
		PasswordSet: in.Password != "",
	}, nil
}

func (s *Server) loadSMTPInternal(ctx context.Context) (*smtpInternal, error) {
	host, err := s.getSetting(ctx, "smtp_host")
	if err != nil {
		return nil, err
	}
	portRaw, err := s.getSetting(ctx, "smtp_port")
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(portRaw)
	username, err := s.getSetting(ctx, "smtp_username")
	if err != nil {
		return nil, err
	}
	password, err := s.getSetting(ctx, "smtp_password")
	if err != nil {
		return nil, err
	}
	from, err := s.getSetting(ctx, "smtp_from")
	if err != nil {
		return nil, err
	}
	useTLSRaw, err := s.getSetting(ctx, "smtp_use_tls")
	if err != nil {
		return nil, err
	}

	useTLS := useTLSRaw == "1" || strings.EqualFold(useTLSRaw, "true")
	return &smtpInternal{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		UseTLS:   useTLS,
	}, nil
}

func (s *Server) getSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ? LIMIT 1`, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (s *Server) upsertSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO settings (key, value, updated_at)
		 VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET
		 value = excluded.value,
		 updated_at = datetime('now')`,
		key,
		value,
	)
	return err
}

func sendSMTPTestEmail(cfg *smtpInternal, to string) error {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	fromAddress, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return fmt.Errorf("parse from address: %w", err)
	}

	headers := map[string]string{
		"From":         fromAddress.String(),
		"To":           to,
		"Subject":      "Capital-Hub SMTP Test",
		"Date":         time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}
	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(k)
		msg.WriteString(": ")
		msg.WriteString(v)
		msg.WriteString("\r\n")
	}
	msg.WriteString("\r\n")
	msg.WriteString("This is a Capital-Hub test email. SMTP configuration looks good.\r\n")

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	if cfg.UseTLS && cfg.Port == 465 {
		tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host})
		if err != nil {
			return fmt.Errorf("tls dial smtp: %w", err)
		}
		defer tlsConn.Close()

		client, err := smtp.NewClient(tlsConn, cfg.Host)
		if err != nil {
			return fmt.Errorf("new smtp client: %w", err)
		}
		defer client.Quit()

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
		if err := client.Mail(fromAddress.Address); err != nil {
			return fmt.Errorf("smtp mail from: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
		wc, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		if _, err := wc.Write([]byte(msg.String())); err != nil {
			_ = wc.Close()
			return fmt.Errorf("smtp write: %w", err)
		}
		if err := wc.Close(); err != nil {
			return fmt.Errorf("smtp close data: %w", err)
		}
		return nil
	}

	if err := smtp.SendMail(addr, auth, fromAddress.Address, []string{to}, []byte(msg.String())); err != nil {
		return fmt.Errorf("smtp send mail: %w", err)
	}
	return nil
}

func boolToString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
