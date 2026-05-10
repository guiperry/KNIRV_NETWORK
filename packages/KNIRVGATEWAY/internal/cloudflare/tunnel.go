package cloudflare

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

// TunnelRunner manages a cloudflared process that connects a named
// Cloudflare Tunnel to a local upstream service.  It follows the same
// lifecycle conventions demonstrated in proxy-penguin: provision the tunnel
// via the Cloudflare API, then spawn cloudflared to maintain the connection.
type TunnelRunner struct {
	api          *API
	tunnelName   string
	tunnelID     string
	hostname     string
	servicePort  int

	cmd    *exec.Cmd
	cancel context.CancelFunc
	mu     sync.Mutex
	done   chan struct{}
}

// TunnelRunnerConfig holds the parameters for a TunnelRunner.
type TunnelRunnerConfig struct {
	APIToken    string // Cloudflare API token
	ZoneID      string // Cloudflare zone ID for DNS
	TunnelName  string // tunnel name (e.g. "knirv-gateway")
	Hostname    string // public subdomain (e.g. "gateway.knirv.network")
	ServicePort int    // local port the gateway listens on
}

// NewTunnelRunner creates a TunnelRunner.  Call Run() to provision
// and start the tunnel.
func NewTunnelRunner(cfg TunnelRunnerConfig) *TunnelRunner {
	return &TunnelRunner{
		api:         NewAPI(cfg.APIToken, cfg.ZoneID),
		tunnelName:  cfg.TunnelName,
		tunnelID:    "", // populated by provision()
		hostname:    cfg.Hostname,
		servicePort: cfg.ServicePort,
		done:        make(chan struct{}),
	}
}

// Run provisions the tunnel via the Cloudflare API (creating it and its
// DNS record if they don't exist), then spawns cloudflared.  It blocks
// until the context is cancelled, at which point cloudflared is stopped.
// Returns nil if the tunnel was started successfully; non-nil if
// provisioning or cloudflared startup fails.
func (tr *TunnelRunner) Run(ctx context.Context) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	// Step 1: validate token
	if err := tr.api.ValidateToken(); err != nil {
		return fmt.Errorf("cloudflare token validation failed: %w", err)
	}
	log.Println("[cloudflare] API token validated")

	// Step 2: provision tunnel
	if err := tr.provision(); err != nil {
		return fmt.Errorf("tunnel provisioning failed: %w", err)
	}

	// Step 3: locate cloudflared binary
	cfPath, err := findCloudflared()
	if err != nil {
		return fmt.Errorf("cloudflared not found: %w", err)
	}

	// Step 4: get tunnel token
	token, err := tr.api.GetTunnelToken(tr.tunnelID)
	if err != nil {
		return fmt.Errorf("failed to get tunnel token: %w", err)
	}

	// Step 5: spawn cloudflared
	ctx, cancel := context.WithCancel(ctx)
	tr.cancel = cancel

	// cloudflared tunnel run --token <token> <tunnelID>
	cmd := exec.CommandContext(ctx, cfPath, "tunnel", "run",
		"--token", token, tr.tunnelID)
	cmd.Stdout = &logWriter{prefix: "cloudflared"}
	cmd.Stderr = &logWriter{prefix: "cloudflared-err"}

	log.Printf("[cloudflare] starting cloudflared for tunnel %s (%s)",
		tr.tunnelName, tr.tunnelID)
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start cloudflared: %w", err)
	}
	tr.cmd = cmd
	log.Printf("[cloudflare] cloudflared running (pid %d)", cmd.Process.Pid)

	// Step 6: wait in background and signal when process exits
	go func() {
		defer close(tr.done)
		if err := cmd.Wait(); err != nil {
			log.Printf("[cloudflare] cloudflared exited: %v", err)
		} else {
			log.Println("[cloudflare] cloudflared stopped")
		}
	}()

	return nil
}

// Wait blocks until the cloudflared process exits.
func (tr *TunnelRunner) Wait() {
	<-tr.done
}

// Stop gracefully stops the cloudflared process.
func (tr *TunnelRunner) Stop() {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if tr.cancel != nil {
		tr.cancel()
	}
	if tr.cmd != nil && tr.cmd.Process != nil {
		log.Printf("[cloudflare] stopping cloudflared (pid %d)", tr.cmd.Process.Pid)
		tr.cmd.Process.Signal(os.Interrupt)
		// Let Wait() collect the exit.
	}
}

// TunnelID returns the provisioned tunnel ID (empty until Run succeeds).
func (tr *TunnelRunner) TunnelID() string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.tunnelID
}

// provision creates (or reuses) the Cloudflare Tunnel and its DNS record.
func (tr *TunnelRunner) provision() error {
	// Look for an existing tunnel with our name.
	existing, err := tr.api.FindTunnelByName(tr.tunnelName)
	if err != nil {
		return fmt.Errorf("list tunnels: %w", err)
	}

	var tunnel *Tunnel
	if existing != nil {
		log.Printf("[cloudflare] reusing existing tunnel %s (%s)", tr.tunnelName, existing.ID)
		tunnel = existing
	} else {
		log.Printf("[cloudflare] creating tunnel: %s", tr.tunnelName)
		t, err := tr.api.CreateTunnel(tr.tunnelName)
		if err != nil {
			return fmt.Errorf("create tunnel: %w", err)
		}
		tunnel = t
		log.Printf("[cloudflare] created tunnel %s (%s)", tr.tunnelName, tunnel.ID)
	}

	tr.tunnelID = tunnel.ID

	// Configure ingress: hostname → local service
	serviceURL := fmt.Sprintf("http://localhost:%d", tr.servicePort)
	ingress := []IngressRule{
		{Hostname: tr.hostname, Service: serviceURL},
	}
	if err := tr.api.ConfigureTunnel(tr.tunnelID, ingress); err != nil {
		return fmt.Errorf("configure tunnel: %w", err)
	}
	log.Printf("[cloudflare] configured ingress: %s → %s", tr.hostname, serviceURL)

	// Create DNS CNAME record (idempotent — Cloudflare returns an error
	// if the record already exists, which we treat as a success).
	if err := tr.api.CreateDNSRecord(tr.hostname, tr.tunnelID); err != nil {
		// Cloudflare returns error 81053 for duplicate records — that's OK.
		log.Printf("[cloudflare] DNS record note: %v", err)
	} else {
		log.Printf("[cloudflare] DNS record created: %s → %s.cfargotunnel.com",
			tr.hostname, tr.tunnelID)
	}

	return nil
}

// findCloudflared locates the cloudflared binary on the system.
func findCloudflared() (string, error) {
	// Respect explicit path from environment.
	if env := os.Getenv("CLOUDFLARED_PATH"); env != "" {
		if _, err := os.Stat(env); err == nil {
			return env, nil
		}
	}

	// Common install locations.
	candidates := []string{
		"/usr/local/bin/cloudflared",
		"/usr/bin/cloudflared",
		"/opt/cloudflared/cloudflared",
	}

	// Also check PATH.
	if p, err := exec.LookPath("cloudflared"); err == nil {
		candidates = append([]string{p}, candidates...)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("cloudflared not found — install from https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/")
}

// logWriter adapts a log prefix to an io.Writer for subprocess stdout/stderr.
type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (int, error) {
	scanner := bufio.NewScanner(
		&bytesReader{buf: p},
	)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			log.Printf("[%s] %s", w.prefix, line)
		}
	}
	return len(p), nil
}

// bytesReader is a minimal io.Reader over a byte slice for bufio.Scanner.
type bytesReader struct {
	buf []byte
	pos int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.buf) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.buf[r.pos:])
	r.pos += n
	return n, nil
}
