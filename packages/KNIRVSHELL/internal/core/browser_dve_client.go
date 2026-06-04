package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	defaultBrowserDVEHeartbeatInterval = 45 * time.Second
	defaultBrowserDVEHeartbeatTimeout  = 90 * time.Second
	defaultBrowserDVERetryBaseDelay    = 1 * time.Second
	defaultBrowserDVERetryMaxDelay     = 30 * time.Second
	defaultBrowserDVERetryAttempts     = 10
	defaultBrowserDVEExtensionID       = "knirvshell"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// BrowserDVEIdentity captures the stable identity used by browser DVE nodes.
type BrowserDVEIdentity struct {
	NodeID         string
	DVEURI         string
	WalletAddress  string
	ExtensionID    string
	BrowserVersion string
}

// DeriveBrowserDVEIdentity constructs the deterministic identity used by the
// browser DVE registration and websocket handshake.
func DeriveBrowserDVEIdentity(walletAddress, extensionID, browserVersion string) BrowserDVEIdentity {
	extensionID = strings.TrimSpace(extensionID)
	if extensionID == "" {
		extensionID = defaultBrowserDVEExtensionID
	}

	browserVersion = strings.TrimSpace(browserVersion)
	if browserVersion == "" {
		browserVersion = fmt.Sprintf("knirvshell/%s", runtime.Version())
	}

	return BrowserDVEIdentity{
		NodeID:         DeriveBrowserDVENodeID(walletAddress, extensionID),
		DVEURI:         FormatBrowserDVEURI(walletAddress),
		WalletAddress:  walletAddress,
		ExtensionID:    extensionID,
		BrowserVersion: browserVersion,
	}
}

// DeriveBrowserDVENodeID creates the stable browser DVE node ID.
func DeriveBrowserDVENodeID(walletAddress, extensionID string) string {
	sum := sha256.Sum256([]byte(walletAddress + extensionID))
	return "dve-" + hex.EncodeToString(sum[:])[:16]
}

// FormatBrowserDVEURI returns the canonical browser DVE URI.
func FormatBrowserDVEURI(walletAddress string) string {
	return fmt.Sprintf("knirv://dve/%s/browser", walletAddress)
}

// BrowserDVEMessage mirrors the server websocket envelope.
type BrowserDVEMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// BrowserDVEHandler handles incoming websocket messages.
type BrowserDVEHandler func(BrowserDVEMessage)

// BrowserDVEClientOption customizes client behavior.
type BrowserDVEClientOption func(*BrowserDVEClient)

// WithBrowserDVELogger overrides the default logger.
func WithBrowserDVELogger(logger *logrus.Logger) BrowserDVEClientOption {
	return func(c *BrowserDVEClient) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithBrowserDVECapabilities overrides the capabilities sent in registration.
func WithBrowserDVECapabilities(capabilities []string) BrowserDVEClientOption {
	return func(c *BrowserDVEClient) {
		c.capabilities = append([]string(nil), capabilities...)
	}
}

// WithBrowserDVEBadgeNFTIDs overrides the badge NFT IDs sent in registration.
func WithBrowserDVEBadgeNFTIDs(badgeNFTIDs []string) BrowserDVEClientOption {
	return func(c *BrowserDVEClient) {
		c.badgeNFTIDs = append([]string(nil), badgeNFTIDs...)
	}
}

// WithBrowserDVEHeartbeatInterval overrides the heartbeat interval.
func WithBrowserDVEHeartbeatInterval(interval time.Duration) BrowserDVEClientOption {
	return func(c *BrowserDVEClient) {
		if interval > 0 {
			c.heartbeatInterval = interval
		}
	}
}

// WithBrowserDVERetryPolicy overrides reconnect behavior.
func WithBrowserDVERetryPolicy(maxAttempts int, baseDelay, maxDelay time.Duration) BrowserDVEClientOption {
	return func(c *BrowserDVEClient) {
		if maxAttempts > 0 {
			c.maxReconnectAttempts = maxAttempts
		}
		if baseDelay > 0 {
			c.reconnectBaseDelay = baseDelay
		}
		if maxDelay > 0 {
			c.reconnectMaxDelay = maxDelay
		}
	}
}

// WithBrowserDVEHandler registers a handler for a specific websocket message type.
func WithBrowserDVEHandler(messageType string, handler BrowserDVEHandler) BrowserDVEClientOption {
	return func(c *BrowserDVEClient) {
		if handler == nil {
			return
		}
		c.handlers[messageType] = append(c.handlers[messageType], handler)
	}
}

// BrowserDVEClient manages the browser DVE websocket lifecycle.
type BrowserDVEClient struct {
	serverURL string
	authToken string
	identity  BrowserDVEIdentity

	capabilities []string
	badgeNFTIDs  []string

	heartbeatInterval    time.Duration
	heartbeatTimeout     time.Duration
	reconnectBaseDelay   time.Duration
	reconnectMaxDelay    time.Duration
	maxReconnectAttempts int

	logger *logrus.Logger

	mu                sync.RWMutex
	writeMu           sync.Mutex
	conn              *websocket.Conn
	connected         bool
	shouldReconnect   bool
	reconnecting      bool
	reconnectAttempts int
	heartbeatCancel   context.CancelFunc
	handlers          map[string][]BrowserDVEHandler
}

// NewBrowserDVEClient creates a browser DVE websocket client.
func NewBrowserDVEClient(serverURL, authToken string, identity BrowserDVEIdentity, opts ...BrowserDVEClientOption) *BrowserDVEClient {
	client := &BrowserDVEClient{
		serverURL:            strings.TrimRight(serverURL, "/"),
		authToken:            authToken,
		identity:             identity,
		capabilities:         []string{"policy-check", "signature-verify"},
		badgeNFTIDs:          nil,
		heartbeatInterval:    defaultBrowserDVEHeartbeatInterval,
		heartbeatTimeout:     defaultBrowserDVEHeartbeatTimeout,
		reconnectBaseDelay:   defaultBrowserDVERetryBaseDelay,
		reconnectMaxDelay:    defaultBrowserDVERetryMaxDelay,
		maxReconnectAttempts: defaultBrowserDVERetryAttempts,
		logger:               logrus.New(),
		handlers:             make(map[string][]BrowserDVEHandler),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Identity returns the client identity.
func (c *BrowserDVEClient) Identity() BrowserDVEIdentity {
	return c.identity
}

// Connect establishes the websocket connection and sends the registration envelope.
func (c *BrowserDVEClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	c.shouldReconnect = true
	c.reconnectAttempts = 0
	c.mu.Unlock()

	return c.connectOnce(ctx)
}

// Disconnect stops reconnect attempts and closes the websocket.
func (c *BrowserDVEClient) Disconnect() error {
	c.mu.Lock()
	c.shouldReconnect = false
	if cancel := c.heartbeatCancel; cancel != nil {
		cancel()
		c.heartbeatCancel = nil
	}
	conn := c.conn
	c.conn = nil
	c.connected = false
	c.mu.Unlock()

	if conn == nil {
		return nil
	}

	return conn.Close()
}

// SendMessage sends a typed websocket message.
func (c *BrowserDVEClient) SendMessage(messageType string, payload any) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("browser DVE websocket is not connected")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal browser DVE payload: %w", err)
	}

	message := BrowserDVEMessage{
		Type:    messageType,
		Payload: json.RawMessage(payloadBytes),
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	if err := conn.WriteJSON(message); err != nil {
		return err
	}

	return nil
}

// SendHeartbeat sends a heartbeat message to the browser DVE server.
func (c *BrowserDVEClient) SendHeartbeat() error {
	return c.SendMessage("heartbeat", map[string]any{
		"timestamp": time.Now().Unix(),
	})
}

// On registers a handler for a websocket message type.
func (c *BrowserDVEClient) On(messageType string, handler BrowserDVEHandler) {
	if handler == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[messageType] = append(c.handlers[messageType], handler)
}

// Run connects and blocks until the provided context is canceled.
func (c *BrowserDVEClient) Run(ctx context.Context) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	return c.Disconnect()
}

func (c *BrowserDVEClient) connectOnce(ctx context.Context) error {
	c.mu.RLock()
	if c.connected {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	dialURL, headers, err := c.buildDialTarget()
	if err != nil {
		return err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, dialURL, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to browser DVE websocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	if err := c.register(); err != nil {
		_ = conn.Close()
		c.mu.Lock()
		c.conn = nil
		c.connected = false
		c.mu.Unlock()
		return err
	}

	c.startHeartbeatLoop(ctx)
	go c.readLoop(ctx, conn)

	c.logger.Infof("Connected to browser DVE websocket at %s", dialURL)
	return nil
}

func (c *BrowserDVEClient) buildDialTarget() (string, http.Header, error) {
	parsed, err := url.Parse(c.serverURL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid browser DVE server URL: %w", err)
	}

	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
		// keep as-is
	default:
		return "", nil, fmt.Errorf("unsupported browser DVE server scheme: %s", parsed.Scheme)
	}

	parsed.Path = path.Join(parsed.Path, "/api/dve/browser/ws")
	query := parsed.Query()
	query.Set("wallet", c.identity.WalletAddress)
	parsed.RawQuery = query.Encode()

	headers := http.Header{}
	if c.authToken != "" {
		headers.Set("Authorization", "Bearer "+c.authToken)
	}

	return parsed.String(), headers, nil
}

func (c *BrowserDVEClient) register() error {
	payload := map[string]any{
		"node_id":         c.identity.NodeID,
		"capabilities":    c.capabilities,
		"badge_nft_ids":   c.badgeNFTIDs,
		"extension_id":    c.identity.ExtensionID,
		"browser_version": c.identity.BrowserVersion,
	}

	return c.SendMessage("ws_register", payload)
}

func (c *BrowserDVEClient) readLoop(ctx context.Context, conn *websocket.Conn) {
	defer c.handleDisconnect(ctx, conn)

	conn.SetReadLimit(65536)
	_ = conn.SetReadDeadline(time.Now().Add(c.heartbeatTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(c.heartbeatTimeout))
		return nil
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Warnf("Browser DVE websocket read error: %v", err)
			}
			return
		}

		var message BrowserDVEMessage
		if err := json.Unmarshal(raw, &message); err != nil {
			c.logger.Warnf("Browser DVE websocket received invalid JSON: %v", err)
			continue
		}

		c.emit(message)
	}
}

func (c *BrowserDVEClient) emit(message BrowserDVEMessage) {
	c.mu.RLock()
	handlers := append([]BrowserDVEHandler(nil), c.handlers[message.Type]...)
	handlers = append(handlers, c.handlers["*"]...)
	c.mu.RUnlock()

	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		func(h BrowserDVEHandler) {
			defer func() {
				if r := recover(); r != nil {
					c.logger.Errorf("Browser DVE websocket handler panic for %s: %v", message.Type, r)
				}
			}()
			h(message)
		}(handler)
	}
}

func (c *BrowserDVEClient) startHeartbeatLoop(ctx context.Context) {
	heartbeatCtx, cancel := context.WithCancel(context.Background())

	c.mu.Lock()
	if previous := c.heartbeatCancel; previous != nil {
		previous()
	}
	c.heartbeatCancel = cancel
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(c.heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.SendHeartbeat(); err != nil {
					c.logger.Warnf("Browser DVE heartbeat failed: %v", err)
				}
			}
		}
	}()
}

func (c *BrowserDVEClient) handleDisconnect(ctx context.Context, conn *websocket.Conn) {
	c.mu.Lock()
	if c.conn == conn {
		c.conn = nil
	}
	wasConnected := c.connected
	c.connected = false
	if cancel := c.heartbeatCancel; cancel != nil {
		cancel()
		c.heartbeatCancel = nil
	}
	shouldReconnect := c.shouldReconnect
	reconnecting := c.reconnecting
	c.mu.Unlock()

	_ = conn.Close()

	if wasConnected {
		c.logger.Info("Browser DVE websocket disconnected")
	}

	if shouldReconnect && ctx.Err() == nil && !reconnecting {
		c.scheduleReconnect(ctx)
	}
}

func (c *BrowserDVEClient) scheduleReconnect(ctx context.Context) {
	c.mu.Lock()
	if c.reconnecting || !c.shouldReconnect {
		c.mu.Unlock()
		return
	}
	c.reconnecting = true
	c.mu.Unlock()

	go func() {
		defer func() {
			c.mu.Lock()
			c.reconnecting = false
			c.mu.Unlock()
		}()

		for {
			c.mu.RLock()
			shouldReconnect := c.shouldReconnect
			attempts := c.reconnectAttempts
			maxAttempts := c.maxReconnectAttempts
			baseDelay := c.reconnectBaseDelay
			maxDelay := c.reconnectMaxDelay
			c.mu.RUnlock()

			if !shouldReconnect || ctx.Err() != nil {
				return
			}

			if attempts >= maxAttempts {
				c.logger.Error("Browser DVE websocket max reconnect attempts reached")
				return
			}

			delay := baseDelay * time.Duration(1<<attempts)
			if delay > maxDelay {
				delay = maxDelay
			}
			delay += time.Duration(rand.Int63n(int64(1 * time.Second)))

			c.logger.Infof("Browser DVE websocket reconnecting in %s (attempt %d)", delay, attempts+1)

			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}

			if err := c.connectOnce(ctx); err != nil {
				c.logger.Warnf("Browser DVE websocket reconnect failed: %v", err)
				c.mu.Lock()
				c.reconnectAttempts++
				c.mu.Unlock()
				continue
			}

			c.mu.Lock()
			c.reconnectAttempts = 0
			c.mu.Unlock()
			return
		}
	}()
}
