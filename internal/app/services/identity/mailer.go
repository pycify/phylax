package identity

import (
	"bytes"
	"context"
	"html/template"
	"log/slog"
	"time"

	"github.com/pycify/phylax/internal/config"
	gomail "gopkg.in/gomail.v2"
)

type Mailer interface {
	SendVerificationOTP(ctx context.Context, to string, code string, expiresAt time.Time) error
}

type LogMailer struct {
	logger *slog.Logger
}

func NewLogMailer(logger *slog.Logger) *LogMailer {
	return &LogMailer{logger: logger}
}

func (m *LogMailer) SendVerificationOTP(_ context.Context, to string, code string, expiresAt time.Time) error {
	m.logger.Info("email verification otp", "to", to, "code", code, "expires_at", expiresAt)
	return nil
}

type SMTPMailer struct {
	cfg config.MailConfig
}

func NewSMTPMailer(cfg config.MailConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) SendVerificationOTP(_ context.Context, to string, code string, expiresAt time.Time) error {
	var body bytes.Buffer
	if err := verificationTemplate.Execute(&body, map[string]any{
		"Code":      code,
		"ExpiresAt": expiresAt.Format(time.RFC1123),
	}); err != nil {
		return err
	}
	message := gomail.NewMessage()
	message.SetHeader("From", message.FormatAddress(m.cfg.FromEmail, m.cfg.FromName))
	message.SetHeader("To", to)
	message.SetHeader("Subject", "Verify your Phylax account")
	message.SetBody("text/html", body.String())
	dialer := gomail.NewDialer(m.cfg.SMTPHost, m.cfg.SMTPPort, m.cfg.SMTPUser, m.cfg.SMTPPass)
	return dialer.DialAndSend(message)
}

var verificationTemplate = template.Must(template.New("verification").Parse(`
<!doctype html>
<html>
  <body style="font-family: Arial, sans-serif; color: #111827;">
    <h2>Verify your Phylax account</h2>
    <p>Use this code to verify your email address:</p>
    <p style="font-size: 28px; font-weight: 700; letter-spacing: 4px;">{{ .Code }}</p>
    <p>This code expires at {{ .ExpiresAt }}.</p>
  </body>
</html>
`))
