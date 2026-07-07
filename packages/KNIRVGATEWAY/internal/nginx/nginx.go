package nginx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"go.uber.org/zap"
)

const (
	siteConfigPath    = "/etc/nginx/sites-available/knirvgateway"
	siteEnabledPath   = "/etc/nginx/sites-enabled/knirvgateway"
	certReloadService = "/etc/systemd/system/knirv-nginx-reload.service"
	certReloadPath    = "/etc/systemd/system/knirv-nginx-reload.path"
)

// EnsureNginx installs nginx if absent, writes a TLS reverse-proxy site config
// for the given domain pointing to KNIRVGATEWAY at localPort, then tests and
// reloads the config. Also sets up a systemd path unit to reload nginx when
// the TLS certificate is renewed.
//
// All errors are non-fatal — KNIRVGATEWAY continues without nginx if provisioning
// fails (Cloudflare tunnels remain the primary public ingress path).
func EnsureNginx(domain string, localPort int, certFile, keyFile string, logger *zap.Logger) error {
	if domain == "" {
		return fmt.Errorf("nginx provisioning: domain is required")
	}
	if certFile == "" {
		certFile = "/etc/knirvserver/tls/origin.crt"
	}
	if keyFile == "" {
		keyFile = "/etc/knirvserver/tls/origin.key"
	}

	if err := ensureInstalled(logger); err != nil {
		return fmt.Errorf("nginx install: %w", err)
	}

	if err := writeSiteConfig(domain, localPort, certFile, keyFile); err != nil {
		return fmt.Errorf("nginx site config: %w", err)
	}

	if err := enableSite(); err != nil {
		return fmt.Errorf("nginx enable site: %w", err)
	}

	if err := testAndReload(logger); err != nil {
		return fmt.Errorf("nginx reload: %w", err)
	}

	if err := setupCertReloadUnit(certFile, logger); err != nil {
		// Non-critical — cert reload falls back to manual or cron.
		logger.Warn("[nginx] failed to set up cert-reload path unit", zap.Error(err))
	}

	logger.Info("[nginx] provisioned",
		zap.String("domain", domain),
		zap.Int("upstream_port", localPort),
		zap.String("cert", certFile),
	)
	return nil
}

// ensureInstalled verifies nginx is on PATH; installs it if absent.
func ensureInstalled(logger *zap.Logger) error {
	if _, err := exec.LookPath("nginx"); err == nil {
		return nil
	}
	logger.Info("[nginx] not found — installing")
	return install()
}

func install() error {
	switch runtime.GOOS {
	case "linux":
		return installLinux()
	case "darwin":
		return installDarwin()
	default:
		return fmt.Errorf("auto-install unsupported on %s — install nginx manually", runtime.GOOS)
	}
}

func installLinux() error {
	// Detect package manager in preference order.
	if p, err := exec.LookPath("apt-get"); err == nil {
		if out, err := exec.Command(p, "install", "-y", "nginx").CombinedOutput(); err != nil {
			return fmt.Errorf("apt-get install nginx: %w\n%s", err, out)
		}
		return nil
	}
	if p, err := exec.LookPath("dnf"); err == nil {
		if out, err := exec.Command(p, "install", "-y", "nginx").CombinedOutput(); err != nil {
			return fmt.Errorf("dnf install nginx: %w\n%s", err, out)
		}
		return nil
	}
	if p, err := exec.LookPath("yum"); err == nil {
		if out, err := exec.Command(p, "install", "-y", "nginx").CombinedOutput(); err != nil {
			return fmt.Errorf("yum install nginx: %w\n%s", err, out)
		}
		return nil
	}
	if p, err := exec.LookPath("pacman"); err == nil {
		if out, err := exec.Command(p, "-S", "--noconfirm", "nginx").CombinedOutput(); err != nil {
			return fmt.Errorf("pacman install nginx: %w\n%s", err, out)
		}
		return nil
	}
	return fmt.Errorf("no supported package manager found (apt-get/dnf/yum/pacman) — install nginx manually")
}

func installDarwin() error {
	brew, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("homebrew not found — install nginx manually: https://nginx.org")
	}
	if out, err := exec.Command(brew, "install", "nginx").CombinedOutput(); err != nil {
		return fmt.Errorf("brew install nginx: %w\n%s", err, out)
	}
	return nil
}

var siteConfigTmpl = template.Must(template.New("nginx-site").Parse(`# Managed by KNIRVGATEWAY — do not edit manually.
server {
    listen 80;
    server_name {{.Domain}} www.{{.Domain}};
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name {{.Domain}} www.{{.Domain}};

    ssl_certificate     {{.CertFile}};
    ssl_certificate_key {{.KeyFile}};
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305;
    ssl_prefer_server_ciphers on;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 1d;

    # WebSocket + long-poll support (TURN, DHT signalling).
    location / {
        proxy_pass          http://127.0.0.1:{{.LocalPort}};
        proxy_http_version  1.1;
        proxy_set_header    Upgrade $http_upgrade;
        proxy_set_header    Connection "upgrade";
        proxy_set_header    Host $host;
        proxy_set_header    X-Real-IP $remote_addr;
        proxy_set_header    X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header    X-Forwarded-Proto $scheme;
        proxy_read_timeout  300s;
        proxy_send_timeout  300s;
        proxy_buffering     off;
    }
}
`))

type siteConfigVars struct {
	Domain    string
	LocalPort int
	CertFile  string
	KeyFile   string
}

// writeSiteConfig renders the nginx site config. Skips if already identical.
func writeSiteConfig(domain string, localPort int, certFile, keyFile string) error {
	var buf strings.Builder
	if err := siteConfigTmpl.Execute(&buf, siteConfigVars{
		Domain:    domain,
		LocalPort: localPort,
		CertFile:  certFile,
		KeyFile:   keyFile,
	}); err != nil {
		return fmt.Errorf("render template: %w", err)
	}
	rendered := buf.String()

	existing, err := os.ReadFile(siteConfigPath)
	if err == nil && string(existing) == rendered {
		return nil // already up to date
	}

	if err := os.MkdirAll(filepath.Dir(siteConfigPath), 0755); err != nil {
		return fmt.Errorf("create sites-available dir: %w", err)
	}
	return os.WriteFile(siteConfigPath, []byte(rendered), 0644)
}

// enableSite creates the symlink in sites-enabled if not already there.
func enableSite() error {
	if err := os.MkdirAll(filepath.Dir(siteEnabledPath), 0755); err != nil {
		return fmt.Errorf("create sites-enabled dir: %w", err)
	}
	// Remove stale symlink if target changed.
	if target, err := os.Readlink(siteEnabledPath); err == nil && target == siteConfigPath {
		return nil // already linked
	}
	_ = os.Remove(siteEnabledPath)
	return os.Symlink(siteConfigPath, siteEnabledPath)
}

// testAndReload runs `nginx -t` then reloads the running nginx process.
func testAndReload(logger *zap.Logger) error {
	if out, err := exec.Command("nginx", "-t").CombinedOutput(); err != nil {
		return fmt.Errorf("nginx -t: %w\n%s", err, out)
	}

	// Prefer systemctl reload for clean signal delivery.
	if _, err := exec.LookPath("systemctl"); err == nil {
		if out, err := exec.Command("systemctl", "enable", "--now", "nginx").CombinedOutput(); err != nil {
			logger.Warn("[nginx] systemctl enable --now failed", zap.String("output", string(out)), zap.Error(err))
		}
		if out, err := exec.Command("systemctl", "reload", "nginx").CombinedOutput(); err != nil {
			// Fall through to nginx -s reload.
			logger.Warn("[nginx] systemctl reload failed, trying nginx -s reload",
				zap.String("output", string(out)), zap.Error(err))
			if out2, err2 := exec.Command("nginx", "-s", "reload").CombinedOutput(); err2 != nil {
				return fmt.Errorf("nginx -s reload: %w\n%s", err2, out2)
			}
		}
		return nil
	}

	if out, err := exec.Command("nginx", "-s", "reload").CombinedOutput(); err != nil {
		return fmt.Errorf("nginx -s reload: %w\n%s", err, out)
	}
	return nil
}

// setupCertReloadUnit writes a systemd path unit that triggers `nginx -s reload`
// whenever certFile is modified (i.e., when the auto-renewal goroutine rotates it).
func setupCertReloadUnit(certFile string, logger *zap.Logger) error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil // no systemd — skip silently
	}

	serviceContent := fmt.Sprintf(`[Unit]
Description=Reload nginx when KNIRV TLS certificate is renewed

[Service]
Type=oneshot
ExecStart=/usr/bin/env nginx -s reload
`)
	pathContent := fmt.Sprintf(`[Unit]
Description=Watch KNIRV TLS certificate for changes

[Path]
PathModified=%s

[Install]
WantedBy=multi-user.target
`, certFile)

	existingSvc, _ := os.ReadFile(certReloadService)
	if string(existingSvc) != serviceContent {
		if err := os.WriteFile(certReloadService, []byte(serviceContent), 0644); err != nil {
			return fmt.Errorf("write reload service: %w", err)
		}
	}

	existingPath, _ := os.ReadFile(certReloadPath)
	if string(existingPath) != pathContent {
		if err := os.WriteFile(certReloadPath, []byte(pathContent), 0644); err != nil {
			return fmt.Errorf("write reload path unit: %w", err)
		}
	}

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w\n%s", err, out)
	}
	if out, err := exec.Command("systemctl", "enable", "--now", "knirv-nginx-reload.path").CombinedOutput(); err != nil {
		return fmt.Errorf("enable cert-reload path unit: %w\n%s", err, out)
	}

	logger.Info("[nginx] cert-reload path unit active", zap.String("watching", certFile))
	return nil
}
