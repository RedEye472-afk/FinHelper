// Package email sends transactional emails via a multi-provider chain:
// Resend → SendGrid → Brevo. Each step falls through to the next on failure
// so a single provider outage doesn't block user-facing flows.
//
// Source: adapted from C:\Users\user\Documents\finhelper\api\index.js
// (Express.js reference implementation).
package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Mail holds a single transactional email to be sent.
type Mail struct {
	To      string
	Subject string
	HTML    string
}

// Sender tries each configured provider in order until one succeeds.
type Sender struct {
	resendAPIKey   string
	sendGridAPIKey string
	brevoAPIKey    string
	brevoSender    string // sender email for Brevo
	fromEmail      string
	fromName       string
	client         *http.Client
	log            *slog.Logger
}

// NewSender creates a Sender. All API keys are optional — missing keys are
// skipped. At least one key must be configured or all sends will fail.
func NewSender(log *slog.Logger, fromEmail, fromName, resendKey, sendGridKey, brevoKey, brevoSender string) *Sender {
	return &Sender{
		resendAPIKey:   resendKey,
		sendGridAPIKey: sendGridKey,
		brevoAPIKey:    brevoKey,
		brevoSender:    brevoSender,
		fromEmail:      fromEmail,
		fromName:       fromName,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		log: log,
	}
}

// Send attempts delivery via the first available provider. Returns nil if any
// provider succeeded, or a combined error if all failed.
func (s *Sender) Send(m Mail) error {
	lastErr := error(nil)

	if s.resendAPIKey != "" {
		if err := s.tryResend(m); err == nil {
			return nil
		} else {
			lastErr = err
			s.log.Warn("email: Resend failed", "to", hashEmail(m.To), "error", err.Error())
		}
	}
	if s.sendGridAPIKey != "" {
		if err := s.trySendGrid(m); err == nil {
			return nil
		} else {
			lastErr = err
			s.log.Warn("email: SendGrid failed", "to", hashEmail(m.To), "error", err.Error())
		}
	}
	if s.brevoAPIKey != "" {
		if err := s.tryBrevo(m); err == nil {
			return nil
		} else {
			lastErr = err
			s.log.Warn("email: Brevo failed", "to", hashEmail(m.To), "error", err.Error())
		}
	}

	return fmt.Errorf("email: all providers failed — %w", lastErr)
}

// SendVerificationCode is a convenience wrapper for the verify-email template.
func (s *Sender) SendVerificationCode(to, code string) error {
	html := fmt.Sprintf(`<div style="font-family:sans-serif;max-width:480px;margin:0 auto;padding:24px;background:#f9fafb;border-radius:12px">
  <h2 style="color:#10b981;margin-top:0">Подтверждение email</h2>
  <p>Ваш код подтверждения:</p>
  <div style="font-size:32px;font-weight:bold;letter-spacing:8px;text-align:center;padding:16px;background:#fff;border-radius:8px;margin:16px 0;color:#10b981">%s</div>
  <p style="color:#6b7280;font-size:14px">Код действителен 10 минут.</p>
</div>`, code)
	return s.Send(Mail{
		To:      to,
		Subject: "Подтверждение регистрации — FinHelper",
		HTML:    html,
	})
}

// SendPasswordReset is a convenience wrapper for the password-reset template.
func (s *Sender) SendPasswordReset(to, resetURL string) error {
	html := fmt.Sprintf(`<div style="font-family:sans-serif;max-width:480px;margin:0 auto;padding:24px;background:#f9fafb;border-radius:12px">
  <h2 style="color:#10b981;margin-top:0">Сброс пароля</h2>
  <p>Нажмите кнопку ниже, чтобы сбросить пароль. Ссылка действительна 1 час.</p>
  <a href="%s" style="display:inline-block;padding:12px 24px;background:#10b981;color:#fff;text-decoration:none;border-radius:8px;font-weight:bold;margin:16px 0">Сбросить пароль</a>
  <p style="color:#6b7280;font-size:14px">Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.</p>
</div>`, resetURL)
	return s.Send(Mail{
		To:      to,
		Subject: "Сброс пароля — FinHelper",
		HTML:    html,
	})
}

// ---------------------------------------------------------------------------
// Provider implementations
// ---------------------------------------------------------------------------

func (s *Sender) tryResend(m Mail) error {
	body := map[string]any{
		"from":    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		"to":      []string{m.To},
		"subject": m.Subject,
		"html":    m.HTML,
	}
	return s.postJSON("https://api.resend.com/emails", s.resendAPIKey, body, "Bearer")
}

func (s *Sender) trySendGrid(m Mail) error {
	body := map[string]any{
		"personalizations": []map[string]any{
			{"to": []map[string]string{{"email": m.To}}},
		},
		"from": map[string]string{
			"email": s.fromEmail,
			"name":  s.fromName,
		},
		"subject": m.Subject,
		"content": []map[string]string{
			{"type": "text/html", "value": m.HTML},
		},
	}
	return s.postJSON("https://api.sendgrid.com/v3/mail/send", s.sendGridAPIKey, body, "Bearer")
}

func (s *Sender) tryBrevo(m Mail) error {
	senderEmail := s.brevoSender
	if senderEmail == "" {
		senderEmail = s.fromEmail
	}
	body := map[string]any{
		"sender": map[string]string{
			"name":  s.fromName,
			"email": senderEmail,
		},
		"to": []map[string]string{
			{"email": m.To},
		},
		"subject":     m.Subject,
		"htmlContent": m.HTML,
	}
	return s.postJSON("https://api.brevo.com/v3/smtp/email", s.brevoAPIKey, body, "api-key")
}

func (s *Sender) postJSON(url, apiKey string, body any, authScheme string) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", authScheme, apiKey))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("api: %s", resp.Status)
	}
	return nil
}

// hashEmail returns a masked version for logging (preserve domain, mask local).
func hashEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}
