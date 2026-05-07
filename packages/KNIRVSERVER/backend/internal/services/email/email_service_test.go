package email

import (
	"testing"
)

// TestNewEmailService tests the creation of email service
func TestNewEmailService(t *testing.T) {
	service := NewEmailService()
	if service == nil {
		t.Error("Expected non-nil EmailService")
	}
}

// TestIsEnabled tests the IsEnabled method
func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		service  *EmailService
		expected bool
	}{
		{
			name: "enabled with all fields",
			service: &EmailService{
				enabled:      true,
				smtpHost:     "smtp.gmail.com",
				smtpUser:     "test@gmail.com",
				smtpPassword: "password",
				fromEmail:    "test@gmail.com",
				fromName:     "Test",
			},
			expected: true,
		},
		{
			name: "disabled",
			service: &EmailService{
				enabled:      false,
				smtpHost:     "smtp.gmail.com",
				smtpUser:     "test@gmail.com",
				smtpPassword: "password",
				fromEmail:    "test@gmail.com",
				fromName:     "Test",
			},
			expected: false,
		},
		{
			name: "empty host",
			service: &EmailService{
				enabled:      true,
				smtpHost:     "",
				smtpUser:     "test@gmail.com",
				smtpPassword: "password",
				fromEmail:    "test@gmail.com",
				fromName:     "Test",
			},
			expected: false,
		},
		{
			name: "empty user",
			service: &EmailService{
				enabled:      true,
				smtpHost:     "smtp.gmail.com",
				smtpUser:     "",
				smtpPassword: "password",
				fromEmail:    "test@gmail.com",
				fromName:     "Test",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.service.IsEnabled()
			if result != tt.expected {
				t.Errorf("IsEnabled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestFormatAddress tests the formatAddress helper
func TestFormatAddress(t *testing.T) {
	service := &EmailService{}

	tests := []struct {
		name     string
		nameArg  string
		email    string
		expected string
	}{
		{
			name:     "with name",
			nameArg:  "KNIRV Network",
			email:    "noreply@knirv.com",
			expected: "KNIRV Network <noreply@knirv.com>",
		},
		{
			name:     "without name",
			nameArg:  "",
			email:    "noreply@knirv.com",
			expected: "noreply@knirv.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.formatAddress(tt.nameArg, tt.email)
			if result != tt.expected {
				t.Errorf("formatAddress() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestSendEmailDisabled tests sending email when service is disabled
func TestSendEmailDisabled(t *testing.T) {
	service := &EmailService{
		enabled: false,
	}

	err := service.SendEmail("test@example.com", "Test", "Body", false)
	if err == nil {
		t.Error("Expected error when sending with disabled service")
	}
}

// TestSendVerificationEmailDisabled tests sending verification email when disabled
func TestSendVerificationEmailDisabled(t *testing.T) {
	service := &EmailService{
		enabled: false,
	}

	err := service.SendVerificationEmail("test@example.com", "token123")
	if err == nil {
		t.Error("Expected error when sending verification email with disabled service")
	}
}

// TestSendPasswordResetEmailDisabled tests sending password reset email when disabled
func TestSendPasswordResetEmailDisabled(t *testing.T) {
	service := &EmailService{
		enabled: false,
	}

	err := service.SendPasswordResetEmail("test@example.com", "token123")
	if err == nil {
		t.Error("Expected error when sending password reset email with disabled service")
	}
}
