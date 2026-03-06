package payment

import (
	"testing"
)

func TestNewPayPalService(t *testing.T) {
	clientID := "test_client_id"
	secret := "test_secret"
	environment := "sandbox"
	currency := "USD"

	service := NewPayPalService(clientID, secret, environment, currency)

	if service.clientID != clientID {
		t.Errorf("Expected clientID %s, got %s", clientID, service.clientID)
	}

	if service.secret != secret {
		t.Errorf("Expected secret %s, got %s", secret, service.secret)
	}

	if service.environment != environment {
		t.Errorf("Expected environment %s, got %s", environment, service.environment)
	}

	if service.currency != currency {
		t.Errorf("Expected currency %s, got %s", currency, service.currency)
	}

	if service.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestPayPalService_CreateOrder(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	rentalID := "rental_123"
	amount := 29.99
	currency := "USD"
	returnURL := "https://example.com/return"
	cancelURL := "https://example.com/cancel"

	order, err := service.CreateOrder(rentalID, amount, currency, returnURL, cancelURL)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	if order == nil {
		t.Fatal("Order should not be nil")
	}

	if order.RentalID != rentalID {
		t.Errorf("Expected rental ID %s, got %s", rentalID, order.RentalID)
	}

	if order.Amount != amount {
		t.Errorf("Expected amount %.2f, got %.2f", amount, order.Amount)
	}

	if order.Currency != currency {
		t.Errorf("Expected currency %s, got %s", currency, order.Currency)
	}

	if order.Status != "CREATED" {
		t.Errorf("Expected status CREATED, got %s", order.Status)
	}

	if order.CheckoutURL == "" {
		t.Error("Checkout URL should not be empty")
	}

	if order.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}
}

func TestPayPalService_CaptureOrder(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	orderID := "PAY-rental_123-1234567890"

	capture, err := service.CaptureOrder(orderID)
	if err != nil {
		t.Fatalf("Failed to capture order: %v", err)
	}

	if capture == nil {
		t.Fatal("Capture should not be nil")
	}

	if capture.Status != "COMPLETED" {
		t.Errorf("Expected status COMPLETED, got %s", capture.Status)
	}

	if capture.Amount <= 0 {
		t.Error("Amount should be positive")
	}

	if capture.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", capture.Currency)
	}

	if capture.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestPayPalService_GetOrderStatus(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	orderID := "PAY-rental_123-1234567890"

	status, err := service.GetOrderStatus(orderID)
	if err != nil {
		t.Fatalf("Failed to get order status: %v", err)
	}

	if status == nil {
		t.Fatal("Status should not be nil")
	}

	if status.ID != orderID {
		t.Errorf("Expected order ID %s, got %s", orderID, status.ID)
	}

	if status.Status != "COMPLETED" {
		t.Errorf("Expected status COMPLETED, got %s", status.Status)
	}

	if !status.Paid {
		t.Error("Order should be marked as paid")
	}

	if status.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestPayPalService_RefundCapture(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	captureID := "CAP-PAY-rental_123-1234567890"
	reason := "Customer request"

	err := service.RefundCapture(captureID, reason)
	if err != nil {
		t.Fatalf("Failed to refund capture: %v", err)
	}
}

func TestPayPalService_ValidateWebhookSignature(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	headers := map[string]string{
		"paypal-auth-algo":         "SHA256withRSA",
		"paypal-cert-url":          "https://api.paypal.com/v1/notifications/certs/CERT-360caa42-fca2a594-2c01f983",
		"paypal-transmission-id":   "69cd13f0-d67e-11e5-baa3-778b3bb2aa58",
		"paypal-transmission-sig":  "lm2RZ...==",
		"paypal-transmission-time": "2016-02-18T20:01:35Z",
	}
	body := []byte(`{"id":"WH-0HV1234567890ABCDE","event_version":"1.0","create_time":"2016-02-18T20:01:35Z","resource_type":"sale","event_type":"PAYMENT.SALE.COMPLETED","summary":"A successful sale payment was made","resource":{"id":"1VY989838W5252913","state":"completed","amount":{"total":"10.00","currency":"USD"},"payment_mode":"INSTANT_TRANSFER"}}`)

	err := service.ValidateWebhookSignature(headers, body)
	if err != nil {
		t.Fatalf("Failed to validate webhook signature: %v", err)
	}
}

func TestPayPalService_ProcessOrderCompleted(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	orderID := "PAY-rental_123-1234567890"

	err := service.ProcessOrderCompleted(orderID)
	if err != nil {
		t.Fatalf("Failed to process order completed: %v", err)
	}
}

func TestPayPalService_ProcessCaptureFailed(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	orderID := "PAY-rental_123-1234567890"
	reason := "Insufficient funds"

	err := service.ProcessCaptureFailed(orderID, reason)
	if err != nil {
		t.Fatalf("Failed to process capture failed: %v", err)
	}
}

func TestPayPalService_CalculateAmount(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	durationHours := int64(24)
	hourlyRate := 1.25
	expectedAmount := float64(durationHours) * hourlyRate

	amount := service.CalculateAmount(durationHours, hourlyRate)
	if amount != expectedAmount {
		t.Errorf("Expected amount %.2f, got %.2f", expectedAmount, amount)
	}
}

func TestPayPalService_FormatAmount(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	amount := 29.99
	expected := "$29.99"

	result := service.FormatAmount(amount)
	if result != expected {
		t.Errorf("Expected formatted amount %s, got %s", expected, result)
	}
}

func TestPayPalService_ValidateAmount(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	// Test valid amount
	err := service.ValidateAmount(29.99)
	if err != nil {
		t.Errorf("Valid amount should not return error: %v", err)
	}

	// Test zero amount
	err = service.ValidateAmount(0)
	if err == nil {
		t.Error("Zero amount should return error")
	}

	// Test negative amount
	err = service.ValidateAmount(-10.00)
	if err == nil {
		t.Error("Negative amount should return error")
	}

	// Test amount exceeding maximum
	err = service.ValidateAmount(100001.00)
	if err == nil {
		t.Error("Amount exceeding maximum should return error")
	}
}

func TestPayPalService_GetSupportedCurrencies(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	currencies := service.GetSupportedCurrencies()
	if len(currencies) == 0 {
		t.Error("Supported currencies should not be empty")
	}

	expectedCurrencies := []string{"USD", "EUR", "GBP", "CAD", "AUD", "JPY"}
	for _, expected := range expectedCurrencies {
		found := false
		for _, currency := range currencies {
			if currency == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected currency %s not found in supported currencies", expected)
		}
	}
}

func TestPayPalService_IsCurrencySupported(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	// Test supported currencies
	supportedCurrencies := []string{"USD", "EUR", "GBP", "CAD", "AUD", "JPY"}
	for _, currency := range supportedCurrencies {
		if !service.IsCurrencySupported(currency) {
			t.Errorf("Currency %s should be supported", currency)
		}
	}

	// Test unsupported currency
	if service.IsCurrencySupported("BTC") {
		t.Error("BTC should not be supported")
	}
}

func TestPayPalService_GetPaymentMethods(t *testing.T) {
	service := NewPayPalService("test_client", "test_secret", "sandbox", "USD")

	methods := service.GetPaymentMethods()
	if len(methods) == 0 {
		t.Error("Payment methods should not be empty")
	}

	expectedMethods := []string{"paypal", "venmo", "paylater"}
	for _, expected := range expectedMethods {
		found := false
		for _, method := range methods {
			if method == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected payment method %s not found", expected)
		}
	}
}

func TestPayPalService_GetBaseURL(t *testing.T) {
	// Test sandbox environment
	sandboxService := NewPayPalService("test_client", "test_secret", "sandbox", "USD")
	sandboxURL := sandboxService.GetBaseURL()
	expectedSandboxURL := "https://api.sandbox.paypal.com"
	if sandboxURL != expectedSandboxURL {
		t.Errorf("Expected sandbox URL %s, got %s", expectedSandboxURL, sandboxURL)
	}

	// Test production environment
	prodService := NewPayPalService("test_client", "test_secret", "production", "USD")
	prodURL := prodService.GetBaseURL()
	expectedProdURL := "https://api.paypal.com"
	if prodURL != expectedProdURL {
		t.Errorf("Expected production URL %s, got %s", expectedProdURL, prodURL)
	}
}

func TestPayPalService_GetCheckoutURL(t *testing.T) {
	orderID := "PAY-rental_123-1234567890"

	// Test sandbox environment
	sandboxService := NewPayPalService("test_client", "test_secret", "sandbox", "USD")
	sandboxURL := sandboxService.GetCheckoutURL(orderID)
	expectedSandboxURL := "https://www.sandbox.paypal.com/checkoutnow?token=" + orderID
	if sandboxURL != expectedSandboxURL {
		t.Errorf("Expected sandbox checkout URL %s, got %s", expectedSandboxURL, sandboxURL)
	}

	// Test production environment
	prodService := NewPayPalService("test_client", "test_secret", "production", "USD")
	prodURL := prodService.GetCheckoutURL(orderID)
	expectedProdURL := "https://www.paypal.com/checkoutnow?token=" + orderID
	if prodURL != expectedProdURL {
		t.Errorf("Expected production checkout URL %s, got %s", expectedProdURL, prodURL)
	}
}

func TestPayPalService_IsSandbox(t *testing.T) {
	// Test sandbox environment
	sandboxService := NewPayPalService("test_client", "test_secret", "sandbox", "USD")
	if !sandboxService.IsSandbox() {
		t.Error("Sandbox service should return true for IsSandbox()")
	}

	// Test production environment
	prodService := NewPayPalService("test_client", "test_secret", "production", "USD")
	if prodService.IsSandbox() {
		t.Error("Production service should return false for IsSandbox()")
	}
}

func TestPayPalService_GetClientID(t *testing.T) {
	clientID := "test_client_id"
	service := NewPayPalService(clientID, "test_secret", "sandbox", "USD")

	result := service.GetClientID()
	if result != clientID {
		t.Errorf("Expected client ID %s, got %s", clientID, result)
	}
}