// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"fmt"
	"os"
	"time"
)

// RuntimeSelector selects the appropriate container runtime based on mode and host capabilities
type RuntimeSelector struct {
	NodePublicKey string
	NodeHostname  string
	NodePeerID    string
	GenesisTime   time.Time

	// Host capabilities
	hasDocker    bool
	hasKata      bool
	hasTEE       bool
	teeType      string // sgx, sev, tdx
	isPrivileged bool
}

// NewRuntimeSelector creates a new runtime selector
func NewRuntimeSelector() *RuntimeSelector {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}

	rs := &RuntimeSelector{
		NodePublicKey: "default-node-pubkey", // TODO: Load from config
		NodeHostname:  hostname,
		NodePeerID:    "default-peer-id", // TODO: Load from libp2p identity
		GenesisTime:   time.Now(),
	}

	// Detect host capabilities
	rs.detectCapabilities()

	return rs
}

// detectCapabilities detects available container runtimes and security features
func (rs *RuntimeSelector) detectCapabilities() {
	// Check for Docker
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		rs.hasDocker = true
	}

	// Check for Kata containers
	if _, err := os.Stat("/usr/bin/kata-runtime"); err == nil {
		rs.hasKata = true
	}

	// Check for TEE support
	rs.detectTEESupport()

	// Check if running as root/privileged
	rs.isPrivileged = os.Geteuid() == 0
}

// detectTEESupport detects hardware TEE support
func (rs *RuntimeSelector) detectTEESupport() {
	// Check for Intel SGX
	if _, err := os.Stat("/dev/sgx"); err == nil {
		rs.hasTEE = true
		rs.teeType = "sgx"
		return
	}

	// Check for AMD SEV
	if _, err := os.Stat("/dev/sev"); err == nil {
		rs.hasTEE = true
		rs.teeType = "sev"
		return
	}

	// Check for Intel TDX
	if _, err := os.Stat("/dev/tdx-guest"); err == nil {
		rs.hasTEE = true
		rs.teeType = "tdx"
		return
	}

	rs.hasTEE = false
}

// SelectRuntime selects the appropriate runtime based on mode and security level
func (rs *RuntimeSelector) SelectRuntime(mode RuntimeMode, securityLevel SecurityLevel) (ContainerRuntime, error) {
	switch mode {
	case RuntimeModeDocker:
		if !rs.hasDocker {
			return nil, fmt.Errorf("Docker runtime not available")
		}
		return NewDockerRuntime(), nil

	case RuntimeModeKata:
		if !rs.hasKata {
			return nil, fmt.Errorf("Kata runtime not available")
		}
		return NewKataRuntime(), nil

	case RuntimeModeTEE:
		if !rs.hasTEE {
			return nil, fmt.Errorf("TEE runtime not available (no hardware support)")
		}
		return NewTEERuntime(rs.teeType), nil

	case RuntimeModeNative:
		if !rs.isPrivileged {
			return nil, fmt.Errorf("Native runtime requires root privileges")
		}
		return NewNativeRuntime(), nil

	case RuntimeModeDVE, RuntimeModeObject:
		// For DVE and Object modes, select best available runtime
		// based on security level
		return rs.selectBestRuntime(securityLevel)

	default:
		return nil, fmt.Errorf("unknown runtime mode: %s", mode)
	}
}

// selectBestRuntime selects the best available runtime for the given security level
func (rs *RuntimeSelector) selectBestRuntime(securityLevel SecurityLevel) (ContainerRuntime, error) {
	switch securityLevel {
	case SecurityLevelExtreme:
		// Require hardware TEE
		if rs.hasTEE {
			return NewTEERuntime(rs.teeType), nil
		}
		// Fallback to Kata if TEE not available
		if rs.hasKata {
			return NewKataRuntime(), nil
		}
		return nil, fmt.Errorf("extreme security level requires TEE or Kata runtime")

	case SecurityLevelStrong:
		// Prefer Kata, fallback to Docker
		if rs.hasKata {
			return NewKataRuntime(), nil
		}
		if rs.hasDocker {
			return NewDockerRuntime(), nil
		}
		// Fallback to native if privileged
		if rs.isPrivileged {
			return NewNativeRuntime(), nil
		}
		return nil, fmt.Errorf("no suitable runtime available for strong security")

	case SecurityLevelBasic:
		// Use any available runtime
		if rs.hasDocker {
			return NewDockerRuntime(), nil
		}
		if rs.hasKata {
			return NewKataRuntime(), nil
		}
		if rs.isPrivileged {
			return NewNativeRuntime(), nil
		}
		return nil, fmt.Errorf("no container runtime available")

	default:
		return nil, fmt.Errorf("unknown security level: %s", securityLevel)
	}
}

// GetCapabilities returns the detected host capabilities
func (rs *RuntimeSelector) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"docker":     rs.hasDocker,
		"kata":       rs.hasKata,
		"tee":        rs.hasTEE,
		"tee_type":   rs.teeType,
		"privileged": rs.isPrivileged,
	}
}
