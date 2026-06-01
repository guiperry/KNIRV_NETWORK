package fintech_validator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestConfig creates a minimal Config for tests.
func newTestConfig() *Config {
	return &Config{
		Enabled:               true,
		EnableAMLChecks:       true,
		EnableKYCChecks:       true,
		EnableSECChecks:       false,
		EnableBaselChecks:     false,
		EnableScenarioTesting: false,
	}
}

// newTestService builds a FinTechValidatorService with just enough to test
// the lifecycle methods (Start/Stop) without requiring real dependencies.
func newTestService() *FinTechValidatorService {
	cfg := newTestConfig()
	return &FinTechValidatorService{
		Config:  cfg,
		running: cfg.Enabled,
	}
}

func TestFinTechStart(t *testing.T) {
	svc := newTestService()
	// Stop first so we can start again
	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if !svc.IsRunning() {
		t.Error("expected IsRunning() == true after Start()")
	}
}

func TestFinTechStop(t *testing.T) {
	svc := newTestService()
	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
	if svc.IsRunning() {
		t.Error("expected IsRunning() == false after Stop()")
	}
}

func TestRequireEnabled_Returns503WhenDisabled(t *testing.T) {
	svc := newTestService()
	// Stop the service so it is disabled
	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	handlers := NewHandlers(svc)
	// Build a minimal router with just the requireEnabled middleware wrapping a dummy endpoint
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	protected := handlers.requireEnabled(next)

	req := httptest.NewRequest("GET", "/api/fintech/validate", nil)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if body["running"] != false {
		t.Errorf("expected running=false in response body, got %v", body["running"])
	}
}

func TestUpdateConfig_PartialPatch(t *testing.T) {
	svc := newTestService()
	// Initial state: KYC=true, SEC=false
	if !svc.Config.EnableKYCChecks {
		t.Fatal("expected EnableKYCChecks=true initially")
	}
	if svc.Config.EnableSECChecks {
		t.Fatal("expected EnableSECChecks=false initially")
	}

	secTrue := true
	kycFalse := false
	patch := &ConfigPatch{
		EnableSECChecks: &secTrue,
		EnableKYCChecks: &kycFalse,
	}
	if err := svc.UpdateConfig(patch); err != nil {
		t.Fatalf("UpdateConfig() failed: %v", err)
	}

	// Only the patched fields should change
	if !svc.Config.EnableSECChecks {
		t.Error("expected EnableSECChecks=true after patch")
	}
	if svc.Config.EnableKYCChecks {
		t.Error("expected EnableKYCChecks=false after patch")
	}
	// AML should be unchanged
	if !svc.Config.EnableAMLChecks {
		t.Error("expected EnableAMLChecks to remain true (unchanged)")
	}
}
