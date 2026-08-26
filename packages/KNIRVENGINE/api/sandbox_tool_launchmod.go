package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// proxychainsConfig is the proxychains4 configuration written into the
// sandbox mount before launch.
type proxychainsConfig struct {
	ChainType         string             `json:"chainType"` // "random", "strict", "dynamic"
	DNSServers        []string           `json:"dnsServers,omitempty"`
	ProxyList         []proxychainsProxy `json:"proxyList"`
	QuietMode         bool               `json:"quietMode"`
	TCPReadTimeout    int                `json:"tcpReadTimeout,omitempty"`
	TCPConnectTimeout int                `json:"tcpConnectTimeout,omitempty"`
}

type proxychainsProxy struct {
	Type string `json:"type"` // "http", "socks4", "socks5"
	Host string `json:"host"`
	Port int    `json:"port"`
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
}

// handleToolProxychains handles POST /api/v1/sandboxes/{id}/tools/proxychains/configure.
// Writes a proxychains4 config into the session mount and flags the session
// to prepend proxychains4 to the target command at launch time.
func (m *SandboxManager) handleToolProxychains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := m.GetSession(sessionID)
	if err != nil {
		RespondWithNotFound(w, "Sandbox session")
		return
	}

	var cfg proxychainsConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		RespondWithValidationError(w, fmt.Sprintf("invalid proxychains config: %v", err))
		return
	}

	// Build the proxychains.conf content.
	conf := buildProxychainsConf(cfg)

	// Write it into the session's mount.
	confPath := filepath.Join(proxychainsConfDir(session), "proxychains.conf")
	if err := os.MkdirAll(filepath.Dir(confPath), 0o755); err != nil {
		RespondWithInternalError(w, fmt.Sprintf("failed to create config dir: %v", err))
		return
	}
	if err := os.WriteFile(confPath, []byte(conf), 0o644); err != nil {
		RespondWithInternalError(w, fmt.Sprintf("failed to write proxychains.conf: %v", err))
		return
	}

	session.mutex.Lock()
	session.proxychainsConf = confPath
	session.mutex.Unlock()

	RespondWithSuccess(w, map[string]string{
		"configPath": confPath,
		"status":     "configured",
	}, "proxychains4 configured — restart the sandbox to apply")
}

// proxychainsConfDir returns the directory where proxychains config lives.
func proxychainsConfDir(s *SandboxSession) string {
	return filepath.Join(sandboxToolsDir(), "proxychains", s.ID)
}

// buildProxychainsConf generates proxychains4 configuration from a config struct.
func buildProxychainsConf(cfg proxychainsConfig) string {
	var b strings.Builder

	// Chain type.
	switch cfg.ChainType {
	case "random":
		b.WriteString("random_chain\n")
	case "strict":
		b.WriteString("strict_chain\n")
	default:
		b.WriteString("dynamic_chain\n")
	}

	if cfg.QuietMode {
		b.WriteString("quiet_mode\n")
	}

	for _, dns := range cfg.DNSServers {
		b.WriteString(fmt.Sprintf("dns4 %s\n", dns))
	}

	if cfg.TCPReadTimeout > 0 {
		b.WriteString(fmt.Sprintf("tcp_read_time_out %d\n", cfg.TCPReadTimeout))
	}
	if cfg.TCPConnectTimeout > 0 {
		b.WriteString(fmt.Sprintf("tcp_connect_time_out %d\n", cfg.TCPConnectTimeout))
	}

	b.WriteString("\n[ProxyList]\n")
	for _, p := range cfg.ProxyList {
		line := fmt.Sprintf("%s %s %d", p.Type, p.Host, p.Port)
		if p.User != "" {
			line += fmt.Sprintf(" %s", p.User)
			if p.Pass != "" {
				line += fmt.Sprintf(" %s", p.Pass)
			}
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

// proxychainsPrefix returns the proxychains4 command prefix if a config
// is attached to the session, or empty string if not configured.
func proxychainsPrefix(s *SandboxSession) string {
	s.mutex.RLock()
	conf := s.proxychainsConf
	s.mutex.RUnlock()
	if conf == "" {
		return ""
	}
	return fmt.Sprintf("proxychains4 -f %s", conf)
}
