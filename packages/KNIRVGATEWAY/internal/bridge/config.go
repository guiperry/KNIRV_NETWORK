// Package bridge implements KNIRVGATEWAY's omnichannel messaging bridge
// (product_packaging_alignment.md Phase 4): a hand-rolled Matrix appservice
// (maunium.net/go/mautrix + go.mau.fi/whatsmeow, both MPL-2.0) fronting a
// self-hosted, non-federating Tuwunel homeserver subprocess (Apache-2.0).
// It is additive, cross-cutting network infrastructure owned by
// KNIRVGATEWAY — the same ownership pattern internal/turnserver already
// establishes for WebRTC ICE/STUN/TURN — not a modification to KNIRVGATEWAY's
// existing routing, oracle-proxy, or DHT logic.
package bridge

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Config is the bridge's own configuration, derived from the gateway's
// top-level config.Config by the caller (internal/server). Keeping this a
// plain struct instead of importing internal/config keeps the bridge package
// self-contained and independently testable.
type Config struct {
	// Enabled gates the entire subsystem. Off by default — this is new,
	// optional infrastructure, not a dependency of the rest of the gateway.
	Enabled bool

	// DataDir holds everything the bridge owns at runtime: the extracted
	// Tuwunel binary, its generated config and database, the appservice
	// registration file, the WhatsApp session store, and the
	// conversation-mapping store.
	DataDir string

	// HomeserverDomain is the Matrix server_name Tuwunel answers as. Per
	// Tuwunel's own docs this can never change post-bootstrap without a
	// database wipe, so callers should pass the gateway's stable resolved
	// public host, not a value that can shift between restarts.
	HomeserverDomain string

	// HomeserverBinaryPath overrides the embedded Tuwunel binary with one
	// already present on disk. Empty means "extract the embedded binary".
	HomeserverBinaryPath string

	// BackendSocketPath is KNIRVSERVER's existing backend Unix socket. The
	// bridge rides this same socket to deliver inbound bridged messages into
	// the Phase 3 Expert-Advisor -> Supervisor pipeline, per Phase 4's
	// "reuse that path rather than inventing a new one" routing decision.
	BackendSocketPath string

	// InternalAuthToken gates the bridge's own outbound-relay callback
	// (POST /api/bridge/relay/outbound), reusing the shared
	// service-to-service token convention already used elsewhere in
	// KNIRVGATEWAY (e.g. the event-bundle mint proxy).
	InternalAuthToken string
}

func (c Config) socketPath(name string) string {
	return filepath.Join(c.DataDir, name)
}

// appserviceSocketPath is the Unix socket the Matrix homeserver (Tuwunel)
// pushes appservice transactions to. Internal-only — never routed through
// the public gateway router, matching the rest of KNIRVGATEWAY's
// internal-service-over-Unix-socket convention.
func (c Config) appserviceSocketPath() string { return c.socketPath("appservice.sock") }

// homeserverClientSocketPath is the Unix socket Tuwunel's own Client-Server
// API listens on, for the appservice's bot/ghost-user clients to call into.
func (c Config) homeserverClientSocketPath() string { return c.socketPath("homeserver-cs.sock") }

func (c Config) appservicesDir() string { return c.socketPath("appservices") }
func (c Config) registrationPath() string {
	return filepath.Join(c.appservicesDir(), "knirv-bridge.yaml")
}
func (c Config) homeserverConfigPath() string        { return c.socketPath("tuwunel.toml") }
func (c Config) homeserverBinaryExtractPath() string { return c.socketPath("tuwunel-bin") }
func (c Config) homeserverDataDir() string           { return filepath.Join(c.DataDir, "homeserver-data") }

// databasePath is the bridge's single local SQLite database — shared by
// whatsmeow's own sqlstore tables (session/device state it manages itself)
// and this package's bridge_conversations table (see store.go). One
// database file for everything the bridge persists locally, rather than a
// second bespoke store next to it.
func (c Config) databasePath() string { return c.socketPath("bridge.db") }

func (c Config) validate() error {
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("bridge data dir is required")
	}
	if strings.TrimSpace(c.HomeserverDomain) == "" {
		return fmt.Errorf("bridge homeserver domain is required")
	}
	return nil
}
