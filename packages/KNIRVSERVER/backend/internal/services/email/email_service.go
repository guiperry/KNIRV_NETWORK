package email

import (
	"bytes"
	"fmt"
	"net/smtp"

	"backend_server/internal/config"
)

// EmailService handles sending emails
type EmailService struct {
	enabled      bool
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	fromEmail    string
	fromName     string
}

// NewEmailService creates a new email service
func NewEmailService() *EmailService {
	cfg, err := config.Load()
	if err != nil {
		// Return disabled service if config fails to load
		return &EmailService{}
	}

	return &EmailService{
		enabled:      cfg.Email.Enabled,
		smtpHost:     cfg.Email.SMTPHost,
		smtpPort:     cfg.Email.SMTPPort,
		smtpUser:     cfg.Email.SMTPUser,
		smtpPassword: cfg.Email.SMTPPassword,
		fromEmail:    cfg.Email.FromEmail,
		fromName:     cfg.Email.FromName,
	}
}

// IsEnabled returns true if the email service is enabled
func (es *EmailService) IsEnabled() bool {
	return es.enabled && es.smtpHost != "" && es.smtpUser != ""
}

// SendEmail sends an email
func (es *EmailService) SendEmail(to, subject, body string, isHTML bool) error {
	if !es.IsEnabled() {
		return fmt.Errorf("email service is not enabled")
	}

	// Build email headers
	headers := make(map[string]string)
	headers["From"] = es.formatAddress(es.fromName, es.fromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"

	if isHTML {
		headers["Content-Type"] = "text/html; charset=UTF-8"
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
	}

	// Build email message
	var message bytes.Buffer
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	// Connect to SMTP server
	auth := smtp.PlainAuth("", es.smtpUser, es.smtpPassword, es.smtpHost)
	addr := fmt.Sprintf("%s:%d", es.smtpHost, es.smtpPort)

	// Send email
	err := smtp.SendMail(addr, auth, es.fromEmail, []string{to}, message.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// formatAddress formats an email address with optional name
func (es *EmailService) formatAddress(name, email string) string {
	if name != "" {
		return fmt.Sprintf("%s <%s>", name, email)
	}
	return email
}

// SendVerificationEmail sends an email verification email
func (es *EmailService) SendVerificationEmail(to string, token string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	subject := "Verify your email address"
	body := fmt.Sprintf(`
		Hello,

		Thank you for registering! Please click the link below to verify your email address:

		%s/auth/verify-email?token=%s

		This link will expire in 24 hours.

		If you did not register for an account, please ignore this email.

		Best regards,
		The KNIRV Team
	`, cfg.Server.BaseURL, token)

	return es.SendEmail(to, subject, body, false)
}

// SendPasswordResetEmail sends a password reset email
func (es *EmailService) SendPasswordResetEmail(to string, token string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	subject := "Reset your password"
	body := fmt.Sprintf(`
		Hello,

		You have requested to reset your password. Please click the link below to reset your password:

		%s/auth/reset-password?token=%s

		This link will expire in 1 hour.

		If you did not request a password reset, please ignore this email.

		Best regards,
		The KNIRV Team
	`, cfg.Server.BaseURL, token)

	return es.SendEmail(to, subject, body, false)
}