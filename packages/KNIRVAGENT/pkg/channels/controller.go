package channels

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/knirvcorp/knirvagent/pkg/bus"
	"github.com/knirvcorp/knirvagent/pkg/logger"
)

type ControllerChannel struct {
	*BaseChannel
	dveID         string
	sessionID     string
	serverURL     string
	authToken     string
	internalToken string
	evidenceDir   string
	client        *http.Client

	mu         sync.RWMutex
	conn       *websocket.Conn
	cancel     context.CancelFunc
	writeMu    sync.Mutex
	evidenceMu sync.Mutex
}

type controllerChatFrame struct {
	ID                string            `json:"id"`
	Type              string            `json:"type"`
	SessionID         string            `json:"session_id"`
	Sender            string            `json:"sender"`
	Content           string            `json:"content"`
	Timestamp         time.Time         `json:"timestamp"`
	TrustLevel        string            `json:"trust_level"`
	SignatureVerified bool              `json:"signature_verified"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

func NewControllerChannel(messageBus *bus.MessageBus) (*ControllerChannel, error) {
	sessionID := firstControllerEnv("DVE_SESSION_ID", "KNIRV_DVE_SESSION_ID", "KNIRV_SESSION_ID")
	dveID := firstControllerEnv("DVE_ID", "KNIRV_DVE_ID", "KNIRVAGENT_DVE_ID")
	if dveID == "" {
		return nil, errors.New("DVE_ID environment variable not set")
	}
	serverURL := strings.TrimRight(firstControllerEnv("KNIRV_SERVER_URL", "KNIRVAGENT_GATEWAY_URL"), "/")
	if serverURL == "" {
		serverURL = "http://host.docker.internal:8080"
	}
	evidenceDir := firstControllerEnv("KNIRV_DVE_SESSION_DIR", "KNIRV_SESSION_DIR")
	if evidenceDir == "" && sessionID != "" {
		evidenceDir = controllerEvidenceDir(sessionID)
	}
	return &ControllerChannel{
		BaseChannel:   NewBaseChannel("controller", nil, messageBus, nil),
		dveID:         dveID,
		sessionID:     sessionID,
		serverURL:     serverURL,
		authToken:     firstControllerEnv("KNIRVAGENT_AUTH_TOKEN", "KNIRV_AUTH_TOKEN", "AUTH_TOKEN"),
		internalToken: firstControllerEnv("KNIRVAGENT_INTERNAL_TOKEN"),
		evidenceDir:   evidenceDir,
		client:        &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (c *ControllerChannel) Start(ctx context.Context) error {
	if strings.TrimSpace(c.authToken) == "" && strings.TrimSpace(c.internalToken) == "" {
		return errors.New("controller channel requires KNIRVAGENT_AUTH_TOKEN or KNIRVAGENT_INTERNAL_TOKEN")
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()
	c.setRunning(true)
	go c.reconnectLoop(runCtx)
	logger.InfoCF("channels", "Controller channel started", map[string]interface{}{
		"session_id": c.sessionID,
		"server_url": c.serverURL,
	})
	return nil
}

func (c *ControllerChannel) Stop(context.Context) error {
	c.setRunning(false)
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *ControllerChannel) Send(ctx context.Context, message bus.OutboundMessage) error {
	if !c.IsRunning() {
		return errors.New("controller channel not running")
	}
	content := strings.TrimSpace(message.Content)
	if content == "" {
		return nil
	}
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return errors.New("controller chat socket is not connected")
	}
	c.writeMu.Lock()
	err := conn.WriteJSON(map[string]interface{}{
		"type":    "message",
		"content": content,
		"metadata": map[string]string{
			"origin": "knirvagent",
		},
	})
	c.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("write controller chat message: %w", err)
	}
	c.recordEvidence("chat.message.sent", controllerChatFrame{
		SessionID:  c.sessionID,
		Sender:     "agent",
		Content:    content,
		Timestamp:  time.Now().UTC(),
		TrustLevel: "locally_supervised",
	})
	return nil
}

func (c *ControllerChannel) reconnectLoop(ctx context.Context) {
	delay := time.Second
	for ctx.Err() == nil {
		err := c.connectAndRead(ctx)
		if ctx.Err() != nil {
			return
		}
		logger.WarnCF("channels", "Controller chat disconnected; reconnecting", map[string]interface{}{
			"session_id": c.sessionID,
			"error":      err.Error(),
			"retry_in":   delay.String(),
		})
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if delay < 30*time.Second {
			delay *= 2
		}
	}
}

func (c *ControllerChannel) connectAndRead(ctx context.Context) error {
	if err := c.resolveSession(ctx); err != nil {
		return err
	}
	socketURL, err := c.requestSocket(ctx)
	if err != nil {
		return err
	}
	headers := http.Header{}
	conn, response, err := websocket.DefaultDialer.DialContext(ctx, socketURL, headers)
	if err != nil {
		if response != nil {
			return fmt.Errorf("dial controller chat socket: HTTP %d: %w", response.StatusCode, err)
		}
		return fmt.Errorf("dial controller chat socket: %w", err)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.mu.Unlock()
		conn.Close()
	}()

	for {
		var frame controllerChatFrame
		if err := conn.ReadJSON(&frame); err != nil {
			return err
		}
		if frame.Type != "message" || strings.TrimSpace(frame.Content) == "" ||
			frame.Sender == "agent" {
			continue
		}
		if frame.Sender != "user" && !strings.HasPrefix(frame.Sender, "mobile:") {
			continue
		}
		metadata := cloneControllerMetadata(frame.Metadata)
		metadata["frame_id"] = frame.ID
		metadata["session_id"] = frame.SessionID
		metadata["sender"] = frame.Sender
		metadata["trust_level"] = frame.TrustLevel
		metadata["signature_verified"] = fmt.Sprintf("%t", frame.SignatureVerified)
		c.recordEvidence("chat.message.received", frame)
		c.HandleMessage(frame.Sender, c.sessionID, frame.Content, nil, metadata)
	}
}

func (c *ControllerChannel) resolveSession(ctx context.Context) error {
	c.mu.RLock()
	sessionID := c.sessionID
	c.mu.RUnlock()
	if sessionID != "" {
		return nil
	}

	if c.internalToken != "" {
		return c.resolveInternalSession(ctx)
	}

	endpoint := c.serverURL + "/dve/sessions"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.authToken)
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("discover controller DVE session: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("discover controller DVE session: HTTP %d", response.StatusCode)
	}
	var sessions []struct {
		ID            string    `json:"id"`
		EnvironmentID string    `json:"environment_id"`
		Status        string    `json:"status"`
		LastActivity  time.Time `json:"last_activity"`
	}
	if err := json.NewDecoder(response.Body).Decode(&sessions); err != nil {
		return fmt.Errorf("decode controller DVE sessions: %w", err)
	}
	var selected struct {
		ID           string
		LastActivity time.Time
	}
	for _, session := range sessions {
		if session.EnvironmentID != c.dveID ||
			(session.Status != "active" && session.Status != "idle") {
			continue
		}
		if selected.ID == "" || session.LastActivity.After(selected.LastActivity) {
			selected.ID = session.ID
			selected.LastActivity = session.LastActivity
		}
	}
	if selected.ID == "" {
		return fmt.Errorf("no active controller session found for DVE environment %s", c.dveID)
	}
	c.mu.Lock()
	c.sessionID = selected.ID
	if c.evidenceDir == "" {
		c.evidenceDir = controllerEvidenceDir(selected.ID)
	}
	c.mu.Unlock()
	return nil
}

func (c *ControllerChannel) resolveInternalSession(ctx context.Context) error {
	endpoint := c.serverURL + "/dve/controller-session?environment_id=" + url.QueryEscape(c.dveID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("X-KNIRV-Agent-Token", c.internalToken)
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("discover internal controller DVE session: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("discover internal controller DVE session: HTTP %d", response.StatusCode)
	}
	var decoded struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return fmt.Errorf("decode internal controller DVE session: %w", err)
	}
	if strings.TrimSpace(decoded.SessionID) == "" {
		return errors.New("internal controller DVE session response has no session_id")
	}
	c.mu.Lock()
	c.sessionID = decoded.SessionID
	if c.evidenceDir == "" {
		c.evidenceDir = controllerEvidenceDir(decoded.SessionID)
	}
	c.mu.Unlock()
	return nil
}

func (c *ControllerChannel) requestSocket(ctx context.Context) (string, error) {
	endpoint := fmt.Sprintf("%s/dve/sessions/%s/chat-socket", c.serverURL, url.PathEscape(c.sessionID))
	body, err := json.Marshal(map[string]string{"role": "agent", "dve_id": c.dveID})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	if c.internalToken != "" {
		request.Header.Set("X-KNIRV-Agent-Token", c.internalToken)
	} else {
		request.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("request controller chat socket: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request controller chat socket: HTTP %d", response.StatusCode)
	}
	var decoded struct {
		WebSocketURL string `json:"ws_url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode controller chat socket response: %w", err)
	}
	if strings.TrimSpace(decoded.WebSocketURL) == "" {
		return "", errors.New("controller chat socket response has no ws_url")
	}
	return decoded.WebSocketURL, nil
}

type controllerEvidenceEvent struct {
	Index     int             `json:"index"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	PrevHash  string          `json:"prev_hash,omitempty"`
	Hash      string          `json:"hash,omitempty"`
}

type controllerEvidenceCanonical struct {
	Index     int             `json:"index"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	PrevHash  string          `json:"prev_hash,omitempty"`
}

func (c *ControllerChannel) recordEvidence(eventType string, frame controllerChatFrame) {
	if c.evidenceDir == "" {
		return
	}
	if err := c.appendEvidence(eventType, frame); err != nil {
		logger.WarnCF("channels", "Failed to append controller message evidence", map[string]interface{}{
			"session_id": c.sessionID,
			"error":      err.Error(),
		})
	}
}

func (c *ControllerChannel) appendEvidence(eventType string, frame controllerChatFrame) error {
	c.evidenceMu.Lock()
	defer c.evidenceMu.Unlock()

	eventsDir := filepath.Join(c.evidenceDir, "events")
	eventLogPath := filepath.Join(eventsDir, "eventlog.jsonl")
	hashChainPath := filepath.Join(eventsDir, "eventlog.hashchain")
	if _, err := os.Stat(filepath.Join(c.evidenceDir, "session.json")); err != nil {
		return fmt.Errorf("supervised session evidence directory is unavailable: %w", err)
	}
	if err := os.MkdirAll(eventsDir, 0o700); err != nil {
		return err
	}
	index, previousHash, err := lastControllerEvidence(eventLogPath)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]interface{}{
		"frame_id":           frame.ID,
		"session_id":         frame.SessionID,
		"sender":             frame.Sender,
		"content":            frame.Content,
		"origin":             "controller",
		"trust_level":        frame.TrustLevel,
		"signature_verified": frame.SignatureVerified,
		"metadata":           frame.Metadata,
	})
	if err != nil {
		return err
	}
	timestamp := frame.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	event := controllerEvidenceEvent{
		Index:     index,
		Timestamp: timestamp.UTC().Format(time.RFC3339Nano),
		Type:      eventType,
		Payload:   payload,
		PrevHash:  previousHash,
	}
	canonical, err := json.Marshal(controllerEvidenceCanonical{
		Index: event.Index, Timestamp: event.Timestamp, Type: event.Type,
		Payload: event.Payload, PrevHash: event.PrevHash,
	})
	if err != nil {
		return err
	}
	sum := sha256.Sum256(canonical)
	event.Hash = "sha256:" + hex.EncodeToString(sum[:])
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := appendControllerFile(eventLogPath, append(line, '\n')); err != nil {
		return err
	}
	return appendControllerFile(hashChainPath, []byte(event.Hash+"\n"))
}

func lastControllerEvidence(path string) (int, string, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 64<<10)
	scanner.Buffer(buffer, 1<<20)
	index := 0
	previousHash := ""
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var event controllerEvidenceEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return 0, "", fmt.Errorf("decode existing event log: %w", err)
		}
		index = event.Index + 1
		previousHash = event.Hash
	}
	return index, previousHash, scanner.Err()
}

func appendControllerFile(path string, value []byte) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(value); err != nil {
		return err
	}
	return file.Sync()
}

func controllerEvidenceDir(sessionID string) string {
	if explicit := firstControllerEnv("KNIRV_DVE_SESSION_DIR", "KNIRV_SESSION_DIR"); explicit != "" {
		return explicit
	}
	workspace := firstControllerEnv("KNIRV_WORKSPACE", "WORKSPACE_DIR")
	if workspace == "" {
		if cwd, err := os.Getwd(); err == nil {
			workspace = cwd
		}
	}
	evidenceSessionID := firstControllerEnv("KNIRV_EVIDENCE_SESSION_ID")
	if evidenceSessionID == "" && workspace != "" {
		if active, err := os.ReadFile(filepath.Join(workspace, ".knirv", "active-session")); err == nil {
			candidate := strings.TrimSpace(string(active))
			if candidate != "" && filepath.Base(candidate) == candidate {
				evidenceSessionID = candidate
			}
		}
	}
	if evidenceSessionID == "" {
		evidenceSessionID = sessionID
	}
	if workspace == "" {
		return ""
	}
	return filepath.Join(workspace, ".knirv", "sessions", evidenceSessionID)
}

func firstControllerEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func cloneControllerMetadata(source map[string]string) map[string]string {
	cloned := make(map[string]string, len(source)+5)
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
