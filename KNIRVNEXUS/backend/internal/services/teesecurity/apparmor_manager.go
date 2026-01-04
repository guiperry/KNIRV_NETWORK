package teesecurity

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AppArmorManager handles AppArmor profile management
type AppArmorManager struct {
	config MACConfig
}

// NewAppArmorManager creates a new AppArmorManager instance
func NewAppArmorManager(config MACConfig) *AppArmorManager {
	return &AppArmorManager{config: config}
}

// LoadProfile loads the AppArmor profile into the kernel
func (am *AppArmorManager) LoadProfile() error {
	// Check if AppArmor is enabled
	if !am.isAppArmorEnabled() {
		return fmt.Errorf("AppArmor is not enabled on this system")
	}

	// Generate profile if needed
	profilePath := filepath.Join("/etc/apparmor.d", am.config.Profile)
	if am.config.AutoGenerate || !fileExists(profilePath) {
		if err := am.generateProfile(profilePath); err != nil {
			return err
		}
	}

	// Load profile using apparmor_parser
	cmd := exec.Command("apparmor_parser", "-r", profilePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load AppArmor profile: %s: %w", string(output), err)
	}

	log.Printf("AppArmor profile %s loaded successfully", am.config.Profile)
	return nil
}

// ApplyToProcess applies AppArmor profile to current process
func (am *AppArmorManager) ApplyToProcess() error {
	// Write profile name to /proc/self/attr/current
	profileLabel := am.config.Profile + " (enforce)"

	attrPath := "/proc/self/attr/current"
	if err := os.WriteFile(attrPath, []byte(profileLabel), 0); err != nil {
		return fmt.Errorf("failed to apply AppArmor profile: %w", err)
	}

	log.Printf("AppArmor profile %s applied to current process", am.config.Profile)
	return nil
}

// generateProfile generates an AppArmor profile from template
func (am *AppArmorManager) generateProfile(path string) error {
	profile := am.config.Template
	if profile == "" {
		profile = DefaultAppArmorProfile
	}

	// Replace placeholders
	profile = strings.ReplaceAll(profile, "{{PROFILE_NAME}}", am.config.Profile)

	if err := os.WriteFile(path, []byte(profile), 0644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	log.Printf("Generated AppArmor profile at %s", path)
	return nil
}

// isAppArmorEnabled checks if AppArmor is enabled on the system
func (am *AppArmorManager) isAppArmorEnabled() bool {
	_, err := os.Stat("/sys/kernel/security/apparmor")
	return err == nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetAppArmorStatus returns the current AppArmor status
func GetAppArmorStatus() (string, error) {
	// Check if AppArmor is enabled
	if !fileExists("/sys/kernel/security/apparmor") {
		return "disabled", nil
	}

	// Try to run aa-status command
	cmd := exec.Command("aa-status")
	output, err := cmd.Output()
	if err != nil {
		return "enabled (no aa-status)", nil
	}

	return fmt.Sprintf("enabled\n%s", string(output)), nil
}

// UnloadProfile unloads an AppArmor profile
func (am *AppArmorManager) UnloadProfile() error {
	if !am.isAppArmorEnabled() {
		return fmt.Errorf("AppArmor is not enabled")
	}

	cmd := exec.Command("apparmor_parser", "-R", filepath.Join("/etc/apparmor.d", am.config.Profile))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unload AppArmor profile: %s: %w", string(output), err)
	}

	log.Printf("AppArmor profile %s unloaded successfully", am.config.Profile)
	return nil
}

// SetProfileMode sets the profile to enforce or complain mode
func (am *AppArmorManager) SetProfileMode(mode string) error {
	if !am.isAppArmorEnabled() {
		return fmt.Errorf("AppArmor is not enabled")
	}

	var flag string
	if mode == "enforce" {
		flag = "-e"
	} else if mode == "complain" {
		flag = "-c"
	} else {
		return fmt.Errorf("invalid mode: %s (must be 'enforce' or 'complain')", mode)
	}

	cmd := exec.Command("aa-enforce", flag, am.config.Profile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set profile mode: %s: %w", string(output), err)
	}

	log.Printf("AppArmor profile %s set to %s mode", am.config.Profile, mode)
	return nil
}
