package fintech_validator

import (
	"context"
	"fmt"
)

// PluginID returns the unique identifier for this builtin plugin.
func (s *FinTechValidatorService) PluginID() string { return "fintech-validator" }

// PluginName returns the human-readable name.
func (s *FinTechValidatorService) PluginName() string { return "FinTech Compliance Validator" }

// Category returns the plugin category.
func (s *FinTechValidatorService) Category() string { return "compliance" }

// Description returns a short description of the plugin.
func (s *FinTechValidatorService) Description() string {
	return "KYC/AML/SEC/Basel AI agent compliance with NRV tracing and PQC-signed evidence packs"
}

// Version returns the plugin version string.
func (s *FinTechValidatorService) Version() string { return "1.0.0" }

// Start marks the plugin as enabled and running.
func (s *FinTechValidatorService) Start(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Config == nil {
		return fmt.Errorf("fintech-validator: config is nil")
	}
	s.Config.Enabled = true
	s.running = true
	return nil
}

// Stop marks the plugin as disabled and not running.
func (s *FinTechValidatorService) Stop(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Config != nil {
		s.Config.Enabled = false
	}
	s.running = false
	return nil
}

// UpdateRawConfig applies a map-based patch to the plugin configuration at runtime.
// This satisfies the pluginmanagement.ConfigurablePlugin interface without creating
// an import cycle (the handler uses an interface assertion).
func (s *FinTechValidatorService) UpdateRawConfig(patch map[string]interface{}) error {
	if patch == nil {
		return nil
	}
	p := &ConfigPatch{}
	if v, ok := patch["enabled"].(bool); ok {
		p.Enabled = &v
	}
	if v, ok := patch["enable_aml"].(bool); ok {
		p.EnableAMLChecks = &v
	}
	if v, ok := patch["enable_kyc"].(bool); ok {
		p.EnableKYCChecks = &v
	}
	if v, ok := patch["enable_sec"].(bool); ok {
		p.EnableSECChecks = &v
	}
	if v, ok := patch["enable_basel"].(bool); ok {
		p.EnableBaselChecks = &v
	}
	if v, ok := patch["enable_scenarios"].(bool); ok {
		p.EnableScenarioTesting = &v
	}
	return s.UpdateConfig(p)
}
