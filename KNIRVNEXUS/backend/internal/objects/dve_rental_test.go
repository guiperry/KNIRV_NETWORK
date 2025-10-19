package objects

import (
	"testing"
	"time"
)

func TestDVERental_StructFields(t *testing.T) {
	now := time.Now()
	startTime := now
	endTime := now.Add(time.Hour * 24)

	resourceLimits := ResourceLimits{
		MaxCPU:       4.0,
		MaxMemory:    8589934592,   // 8GB in bytes
		MaxDisk:      107374182400, // 100GB in bytes
		MaxBandwidth: 1073741824,   // 1GB/s in bytes
	}

	usageMetrics := UsageMetrics{
		CPUUsage:     65.5,
		MemoryUsage:  4294967296,  // 4GB in bytes
		DiskUsage:    53687091200, // 50GB in bytes
		NetworkUsage: 536870912,   // 512MB in bytes
		LastUpdated:  now,
	}

	rental := DVERental{
		ID:               "rental-123",
		UserID:           "user-456",
		DVENodeID:        "dve-node-789",
		NRNAmount:        1000,
		RentalDuration:   86400, // 24 hours in seconds
		StartTime:        startTime,
		EndTime:          endTime,
		Status:           "active",
		PaymentTxHash:    "0xabcd1234567890",
		CDEEnvironmentID: "cde-env-123",
		ResourceLimits:   resourceLimits,
		UsageMetrics:     usageMetrics,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if rental.ID != "rental-123" {
		t.Errorf("Expected ID 'rental-123', got '%s'", rental.ID)
	}
	if rental.UserID != "user-456" {
		t.Errorf("Expected UserID 'user-456', got '%s'", rental.UserID)
	}
	if rental.NRNAmount != 1000 {
		t.Errorf("Expected NRNAmount 1000, got %d", rental.NRNAmount)
	}
	if rental.RentalDuration != 86400 {
		t.Errorf("Expected RentalDuration 86400, got %d", rental.RentalDuration)
	}
	if rental.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", rental.Status)
	}
	if rental.PaymentTxHash != "0xabcd1234567890" {
		t.Errorf("Expected PaymentTxHash '0xabcd1234567890', got '%s'", rental.PaymentTxHash)
	}
	if rental.ResourceLimits.MaxCPU != 4.0 {
		t.Errorf("Expected MaxCPU 4.0, got %f", rental.ResourceLimits.MaxCPU)
	}
	if rental.UsageMetrics.CPUUsage != 65.5 {
		t.Errorf("Expected CPUUsage 65.5, got %f", rental.UsageMetrics.CPUUsage)
	}
}

func TestResourceLimits_StructFields(t *testing.T) {
	limits := ResourceLimits{
		MaxCPU:       8.0,
		MaxMemory:    17179869184,  // 16GB in bytes
		MaxDisk:      214748364800, // 200GB in bytes
		MaxBandwidth: 2147483648,   // 2GB/s in bytes
	}

	if limits.MaxCPU != 8.0 {
		t.Errorf("Expected MaxCPU 8.0, got %f", limits.MaxCPU)
	}
	if limits.MaxMemory != 17179869184 {
		t.Errorf("Expected MaxMemory 17179869184, got %d", limits.MaxMemory)
	}
	if limits.MaxDisk != 214748364800 {
		t.Errorf("Expected MaxDisk 214748364800, got %d", limits.MaxDisk)
	}
	if limits.MaxBandwidth != 2147483648 {
		t.Errorf("Expected MaxBandwidth 2147483648, got %d", limits.MaxBandwidth)
	}
}

func TestUsageMetrics_StructFields(t *testing.T) {
	now := time.Now()

	metrics := UsageMetrics{
		CPUUsage:     75.2,
		MemoryUsage:  8589934592,   // 8GB in bytes
		DiskUsage:    107374182400, // 100GB in bytes
		NetworkUsage: 1073741824,   // 1GB in bytes
		LastUpdated:  now,
	}

	if metrics.CPUUsage != 75.2 {
		t.Errorf("Expected CPUUsage 75.2, got %f", metrics.CPUUsage)
	}
	if metrics.MemoryUsage != 8589934592 {
		t.Errorf("Expected MemoryUsage 8589934592, got %d", metrics.MemoryUsage)
	}
	if metrics.DiskUsage != 107374182400 {
		t.Errorf("Expected DiskUsage 107374182400, got %d", metrics.DiskUsage)
	}
	if metrics.NetworkUsage != 1073741824 {
		t.Errorf("Expected NetworkUsage 1073741824, got %d", metrics.NetworkUsage)
	}
	if !metrics.LastUpdated.Equal(now) {
		t.Errorf("Expected LastUpdated %v, got %v", now, metrics.LastUpdated)
	}
}

func TestNRNPayment_StructFields(t *testing.T) {
	now := time.Now()
	confirmedAt := now.Add(time.Minute * 10)

	payment := NRNPayment{
		ID:            "payment-123",
		RentalID:      "rental-456",
		UserID:        "user-789",
		Amount:        2500,
		TxHash:        "0x1234567890abcdef",
		Status:        "confirmed",
		BlockHeight:   1000000,
		Confirmations: 12,
		CreatedAt:     now,
		ConfirmedAt:   &confirmedAt,
	}

	if payment.ID != "payment-123" {
		t.Errorf("Expected ID 'payment-123', got '%s'", payment.ID)
	}
	if payment.RentalID != "rental-456" {
		t.Errorf("Expected RentalID 'rental-456', got '%s'", payment.RentalID)
	}
	if payment.Amount != 2500 {
		t.Errorf("Expected Amount 2500, got %d", payment.Amount)
	}
	if payment.TxHash != "0x1234567890abcdef" {
		t.Errorf("Expected TxHash '0x1234567890abcdef', got '%s'", payment.TxHash)
	}
	if payment.Status != "confirmed" {
		t.Errorf("Expected Status 'confirmed', got '%s'", payment.Status)
	}
	if payment.BlockHeight != 1000000 {
		t.Errorf("Expected BlockHeight 1000000, got %d", payment.BlockHeight)
	}
	if payment.Confirmations != 12 {
		t.Errorf("Expected Confirmations 12, got %d", payment.Confirmations)
	}
	if payment.ConfirmedAt == nil || !payment.ConfirmedAt.Equal(confirmedAt) {
		t.Errorf("Expected ConfirmedAt %v, got %v", confirmedAt, payment.ConfirmedAt)
	}
}

func TestRentalPlan_StructFields(t *testing.T) {
	now := time.Now()
	updated := now.Add(time.Hour)

	resourceLimits := ResourceLimits{
		MaxCPU:       2.0,
		MaxMemory:    4294967296,  // 4GB in bytes
		MaxDisk:      53687091200, // 50GB in bytes
		MaxBandwidth: 536870912,   // 512MB/s in bytes
	}

	plan := RentalPlan{
		ID:             "plan-basic",
		Name:           "Basic Plan",
		Description:    "Basic DVE rental plan with 2 CPU cores and 4GB RAM",
		PricePerHour:   100,
		ResourceLimits: resourceLimits,
		MaxDuration:    604800, // 7 days in seconds
		MinDuration:    3600,   // 1 hour in seconds
		Features:       []string{"SSH Access", "Web Terminal", "File Transfer"},
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      updated,
	}

	if plan.ID != "plan-basic" {
		t.Errorf("Expected ID 'plan-basic', got '%s'", plan.ID)
	}
	if plan.Name != "Basic Plan" {
		t.Errorf("Expected Name 'Basic Plan', got '%s'", plan.Name)
	}
	if plan.PricePerHour != 100 {
		t.Errorf("Expected PricePerHour 100, got %d", plan.PricePerHour)
	}
	if plan.MaxDuration != 604800 {
		t.Errorf("Expected MaxDuration 604800, got %d", plan.MaxDuration)
	}
	if plan.MinDuration != 3600 {
		t.Errorf("Expected MinDuration 3600, got %d", plan.MinDuration)
	}
	if len(plan.Features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(plan.Features))
	}
	if !plan.IsActive {
		t.Error("Expected IsActive to be true")
	}
	if plan.ResourceLimits.MaxCPU != 2.0 {
		t.Errorf("Expected ResourceLimits MaxCPU 2.0, got %f", plan.ResourceLimits.MaxCPU)
	}
}

func TestDVERentalStats_StructFields(t *testing.T) {
	now := time.Now()

	planUsage1 := PlanUsage{
		PlanID:     "plan-basic",
		PlanName:   "Basic Plan",
		UsageCount: 150,
		Percentage: 60.0,
	}

	planUsage2 := PlanUsage{
		PlanID:     "plan-premium",
		PlanName:   "Premium Plan",
		UsageCount: 100,
		Percentage: 40.0,
	}

	stats := DVERentalStats{
		TotalRentals:      500,
		ActiveRentals:     25,
		TotalNRNCollected: 50000,
		AverageRentalTime: 43200, // 12 hours in seconds
		PopularPlans:      []PlanUsage{planUsage1, planUsage2},
		RevenueToday:      1000,
		Revenue7Days:      7500,
		Revenue30Days:     30000,
		Timestamp:         now,
	}

	if stats.TotalRentals != 500 {
		t.Errorf("Expected TotalRentals 500, got %d", stats.TotalRentals)
	}
	if stats.ActiveRentals != 25 {
		t.Errorf("Expected ActiveRentals 25, got %d", stats.ActiveRentals)
	}
	if stats.TotalNRNCollected != 50000 {
		t.Errorf("Expected TotalNRNCollected 50000, got %d", stats.TotalNRNCollected)
	}
	if stats.AverageRentalTime != 43200 {
		t.Errorf("Expected AverageRentalTime 43200, got %d", stats.AverageRentalTime)
	}
	if len(stats.PopularPlans) != 2 {
		t.Errorf("Expected 2 popular plans, got %d", len(stats.PopularPlans))
	}
	if stats.PopularPlans[0].Percentage != 60.0 {
		t.Errorf("Expected first plan percentage 60.0, got %f", stats.PopularPlans[0].Percentage)
	}
	if stats.Revenue30Days != 30000 {
		t.Errorf("Expected Revenue30Days 30000, got %d", stats.Revenue30Days)
	}
}

func TestPlanUsage_StructFields(t *testing.T) {
	usage := PlanUsage{
		PlanID:     "plan-enterprise",
		PlanName:   "Enterprise Plan",
		UsageCount: 75,
		Percentage: 25.5,
	}

	if usage.PlanID != "plan-enterprise" {
		t.Errorf("Expected PlanID 'plan-enterprise', got '%s'", usage.PlanID)
	}
	if usage.PlanName != "Enterprise Plan" {
		t.Errorf("Expected PlanName 'Enterprise Plan', got '%s'", usage.PlanName)
	}
	if usage.UsageCount != 75 {
		t.Errorf("Expected UsageCount 75, got %d", usage.UsageCount)
	}
	if usage.Percentage != 25.5 {
		t.Errorf("Expected Percentage 25.5, got %f", usage.Percentage)
	}
}

func TestRentalRequest_StructFields(t *testing.T) {
	request := RentalRequest{
		UserID:        "user-123",
		PlanID:        "plan-premium",
		Duration:      172800, // 48 hours in seconds
		PaymentTxHash: "0xfedcba0987654321",
		PreferredDVE:  "dve-node-preferred",
	}

	if request.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", request.UserID)
	}
	if request.PlanID != "plan-premium" {
		t.Errorf("Expected PlanID 'plan-premium', got '%s'", request.PlanID)
	}
	if request.Duration != 172800 {
		t.Errorf("Expected Duration 172800, got %d", request.Duration)
	}
	if request.PaymentTxHash != "0xfedcba0987654321" {
		t.Errorf("Expected PaymentTxHash '0xfedcba0987654321', got '%s'", request.PaymentTxHash)
	}
	if request.PreferredDVE != "dve-node-preferred" {
		t.Errorf("Expected PreferredDVE 'dve-node-preferred', got '%s'", request.PreferredDVE)
	}
}

func TestRentalResponse_StructFields(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour * 24)

	credentials := CDECredentials{
		Username:    "user123",
		Password:    "securepass456",
		SSHKey:      "ssh-rsa AAAAB3NzaC1yc2E...",
		AccessToken: "token-abc123def456",
	}

	response := RentalResponse{
		Success:        true,
		RentalID:       "rental-789",
		DVENodeID:      "dve-node-123",
		CDEAccessURL:   "https://cde.knirv.com/env-123",
		CDECredentials: credentials,
		ExpiresAt:      expiresAt,
		Error:          "",
		Message:        "DVE rental successful",
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if response.RentalID != "rental-789" {
		t.Errorf("Expected RentalID 'rental-789', got '%s'", response.RentalID)
	}
	if response.DVENodeID != "dve-node-123" {
		t.Errorf("Expected DVENodeID 'dve-node-123', got '%s'", response.DVENodeID)
	}
	if response.CDEAccessURL != "https://cde.knirv.com/env-123" {
		t.Errorf("Expected CDEAccessURL 'https://cde.knirv.com/env-123', got '%s'", response.CDEAccessURL)
	}
	if response.CDECredentials.Username != "user123" {
		t.Errorf("Expected Username 'user123', got '%s'", response.CDECredentials.Username)
	}
	if response.Message != "DVE rental successful" {
		t.Errorf("Expected Message 'DVE rental successful', got '%s'", response.Message)
	}
	if !response.ExpiresAt.Equal(expiresAt) {
		t.Errorf("Expected ExpiresAt %v, got %v", expiresAt, response.ExpiresAt)
	}
}

func TestCDECredentials_StructFields(t *testing.T) {
	credentials := CDECredentials{
		Username:    "testuser",
		Password:    "testpass123",
		SSHKey:      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5...",
		AccessToken: "bearer-token-xyz789",
	}

	if credentials.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", credentials.Username)
	}
	if credentials.Password != "testpass123" {
		t.Errorf("Expected Password 'testpass123', got '%s'", credentials.Password)
	}
	if credentials.SSHKey != "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..." {
		t.Errorf("Expected SSHKey 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5...', got '%s'", credentials.SSHKey)
	}
	if credentials.AccessToken != "bearer-token-xyz789" {
		t.Errorf("Expected AccessToken 'bearer-token-xyz789', got '%s'", credentials.AccessToken)
	}
}
