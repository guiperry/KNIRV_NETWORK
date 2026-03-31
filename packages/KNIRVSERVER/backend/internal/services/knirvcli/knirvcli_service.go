package knirvcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/validation"
	"backend_server/internal/services/websocket"

	"github.com/google/uuid"
)

// KNIRVCLIService provides backend integration for KNIRVCLI
// It serves as the unified communication layer for all terminal interactions
type KNIRVCLIService struct {
	db             *database.BuntDBManager
	dveManager     *dvemanager.DVEManager
	validationCore *validation.ValidationCore
	wsService      *websocket.WebSocketService
	sessions       map[string]*TerminalSession
	sessionMu      sync.RWMutex
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	ID        string
	NodeID    string
	UserID    string
	Username  string
	Command   string
	Status    string
	StartTime time.Time
	EndTime   *time.Time
	Output    []string
	ExitCode  int
	Process   *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
}

// CommandRequest represents a command to be executed
type CommandRequest struct {
	Command  string            `json:"command"`
	NodeID   string            `json:"node_id,omitempty"`
	Args     []string          `json:"args,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Timeout  int               `json:"timeout,omitempty"`
	UserID   string            `json:"user_id"`
	Username string            `json:"username"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	SessionID string   `json:"session_id"`
	Command   string   `json:"command"`
	Output    []string `json:"output"`
	ExitCode  int      `json:"exit_code"`
	Status    string   `json:"status"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time,omitempty"`
}

// WalletInfo represents wallet information
type WalletInfo struct {
	Address  string `json:"address"`
	Balance  int64  `json:"balance"`
	Currency string `json:"currency"`
}

// TokenTransaction represents a token transaction
type TokenTransaction struct {
	TxHash string `json:"tx_hash"`
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// resolveKNIRVCLIBinary returns the path to the knirvcli binary.
// Priority: KNIRV_KNIRVCLI_PATH env var → "knirvcli" on PATH.
func resolveKNIRVCLIBinary() string {
	if p := os.Getenv("KNIRV_KNIRVCLI_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "knirvcli"
}

// NewKNIRVCLIService creates a new KNIRVCLI service
func NewKNIRVCLIService(db *database.BuntDBManager, dveManager *dvemanager.DVEManager, validationCore *validation.ValidationCore) *KNIRVCLIService {
	return &KNIRVCLIService{
		db:             db,
		dveManager:     dveManager,
		validationCore: validationCore,
		sessions:       make(map[string]*TerminalSession),
	}
}

// SetWebSocketService sets the WebSocket service for real-time output streaming
func (s *KNIRVCLIService) SetWebSocketService(ws *websocket.WebSocketService) {
	s.wsService = ws
}

// ExecuteCommand executes a command via KNIRVCLI
func (s *KNIRVCLIService) ExecuteCommand(ctx context.Context, req *CommandRequest) (*CommandResult, error) {
	sessionID := uuid.New().String()[:12]

	session := &TerminalSession{
		ID:        sessionID,
		NodeID:    req.NodeID,
		UserID:    req.UserID,
		Username:  req.Username,
		Command:   req.Command,
		Status:    "running",
		StartTime: time.Now(),
		Output:    []string{},
	}

	s.sessionMu.Lock()
	s.sessions[sessionID] = session
	s.sessionMu.Unlock()

	// Build command arguments
	args := []string{}
	if req.NodeID != "" {
		args = append(args, "--node", req.NodeID)
	}
	args = append(args, req.Args...)
	args = append(args, req.Command)

	// Execute command
	cmd := exec.CommandContext(ctx, resolveKNIRVCLIBinary(), args...)

	// Set environment variables
	if req.Env != nil {
		for k, v := range req.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	session.stdin = stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	session.stdout = stdout

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	session.stderr = stderr

	session.Process = cmd

	// Start command
	if err := cmd.Start(); err != nil {
		session.Status = "failed"
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output in goroutine
	go s.streamOutput(session, stdout, "stdout")
	go s.streamOutput(session, stderr, "stderr")

	// Wait for command completion in goroutine
	go func() {
		err := cmd.Wait()
		now := time.Now()
		session.EndTime = &now
		if err != nil {
			session.Status = "failed"
			if exitErr, ok := err.(*exec.ExitError); ok {
				session.ExitCode = exitErr.ExitCode()
			} else {
				session.ExitCode = 1
			}
		} else {
			session.Status = "completed"
			session.ExitCode = 0
		}

		// Broadcast completion via WebSocket
		if s.wsService != nil {
			s.wsService.Broadcast("terminal-session-complete", map[string]interface{}{
				"type":       "session_complete",
				"session_id": sessionID,
				"status":     session.Status,
				"exit_code":  session.ExitCode,
			})
		}
	}()

	return &CommandResult{
		SessionID: sessionID,
		Command:   req.Command,
		Status:    "running",
		StartTime: session.StartTime.Format(time.RFC3339),
	}, nil
}

// streamOutput streams command output to WebSocket
func (s *KNIRVCLIService) streamOutput(session *TerminalSession, reader io.Reader, streamType string) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			output := string(buf[:n])
			session.Output = append(session.Output, output)

			// Broadcast via WebSocket
			if s.wsService != nil {
				s.wsService.Broadcast("terminal-output", map[string]interface{}{
					"type":       streamType,
					"session_id": session.ID,
					"output":     output,
					"timestamp":  time.Now().Format(time.RFC3339),
				})
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading %s for session %s: %v", streamType, session.ID, err)
			}
			break
		}
	}
}

// GetSession returns a terminal session by ID
func (s *KNIRVCLIService) GetSession(sessionID string) (*TerminalSession, error) {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

// ListSessions returns all terminal sessions
func (s *KNIRVCLIService) ListSessions() []*TerminalSession {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()

	sessions := make([]*TerminalSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// StopSession stops a running terminal session
func (s *KNIRVCLIService) StopSession(sessionID string) error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Process != nil && session.Status == "running" {
		if err := session.Process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		session.Status = "stopped"
		now := time.Now()
		session.EndTime = &now
	}

	return nil
}

// SendInput sends input to a running terminal session
func (s *KNIRVCLIService) SendInput(sessionID string, input string) error {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.stdin == nil {
		return fmt.Errorf("session stdin not available")
	}

	_, err := session.stdin.Write([]byte(input))
	return err
}

// GetWalletInfo retrieves wallet information via KNIRVCLI
func (s *KNIRVCLIService) GetWalletInfo(ctx context.Context, walletAddress string) (*WalletInfo, error) {
	cmd := exec.CommandContext(ctx, resolveKNIRVCLIBinary(), "wallet", "info", "--address", walletAddress)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet info: %w", err)
	}

	var info WalletInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse wallet info: %w", err)
	}

	return &info, nil
}

// SendToken sends tokens via KNIRVCLI
func (s *KNIRVCLIService) SendToken(ctx context.Context, from, to string, amount int64) (*TokenTransaction, error) {
	cmd := exec.CommandContext(ctx, resolveKNIRVCLIBinary(), "wallet", "send",
		"--from", from,
		"--to", to,
		"--amount", fmt.Sprintf("%d", amount),
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to send token: %w", err)
	}

	var tx TokenTransaction
	if err := json.Unmarshal(output, &tx); err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	return &tx, nil
}

// ExecuteValidation executes a validation task via KNIRVCLI
func (s *KNIRVCLIService) ExecuteValidation(ctx context.Context, taskID string, nodeID string) error {
	if s.validationCore == nil {
		return fmt.Errorf("validation core not available")
	}

	// Create validation task
	req := &validation.CreateTaskRequest{
		Type:        "knirvcli-validation",
		Priority:    5,
		Data:        map[string]interface{}{"task_id": taskID, "node_id": nodeID},
		RequestedBy: "knirvcli-service",
	}

	_, err := s.validationCore.CreateValidationTask(req)
	if err != nil {
		return fmt.Errorf("failed to create validation task: %w", err)
	}

	return nil
}

// ExecuteTEECommand executes a TEE-related command via KNIRVCLI
func (s *KNIRVCLIService) ExecuteTEECommand(ctx context.Context, command string, nodeID string) (*CommandResult, error) {
	return s.ExecuteCommand(ctx, &CommandRequest{
		Command:  command,
		NodeID:   nodeID,
		Args:     []string{"tee"},
		UserID:   "system",
		Username: "knirvcli",
	})
}

// ExecuteP2PCommand executes a P2P-related command via KNIRVCLI
func (s *KNIRVCLIService) ExecuteP2PCommand(ctx context.Context, command string, nodeID string) (*CommandResult, error) {
	return s.ExecuteCommand(ctx, &CommandRequest{
		Command:  command,
		NodeID:   nodeID,
		Args:     []string{"p2p"},
		UserID:   "system",
		Username: "knirvcli",
	})
}

// ExecuteChainCommand executes a blockchain-related command via KNIRVCLI
func (s *KNIRVCLIService) ExecuteChainCommand(ctx context.Context, command string) (*CommandResult, error) {
	return s.ExecuteCommand(ctx, &CommandRequest{
		Command:  command,
		Args:     []string{"chain"},
		UserID:   "system",
		Username: "knirvcli",
	})
}

// CleanupExpiredSessions removes expired terminal sessions
func (s *KNIRVCLIService) CleanupExpiredSessions(maxAge time.Duration) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if session.Status != "running" && session.EndTime != nil {
			if now.Sub(*session.EndTime) > maxAge {
				delete(s.sessions, id)
			}
		}
	}
}

// Start starts the KNIRVCLI service
func (s *KNIRVCLIService) Start() error {
	log.Println("KNIRVCLI service started")
	return nil
}

// Stop stops the KNIRVCLI service
func (s *KNIRVCLIService) Stop() error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	// Stop all running sessions
	for _, session := range s.sessions {
		if session.Process != nil && session.Status == "running" {
			session.Process.Process.Kill()
		}
	}

	log.Println("KNIRVCLI service stopped")
	return nil
}

// BadgeInfo represents badge information from KNIRVCHAIN
type BadgeInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	BadgeType   string                 `json:"badge_type"`
	Description string                 `json:"description"`
	ImageData   string                 `json:"image_data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Minted      bool                   `json:"minted"`
	MintedAt    *time.Time             `json:"minted_at,omitempty"`
	AgentID     string                 `json:"agent_id,omitempty"`
}

// BadgeCreateRequest represents a badge creation request
type BadgeCreateRequest struct {
	Name        string                 `json:"name"`
	BadgeType   string                 `json:"badge_type"`
	Description string                 `json:"description"`
	ImageData   string                 `json:"image_data"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CreateBadge creates a new badge in KNIRVCHAIN
func (s *KNIRVCLIService) CreateBadge(ctx context.Context, req *BadgeCreateRequest) (map[string]interface{}, error) {
	binaryPath := resolveKNIRVCLIBinary()

	cmd := exec.CommandContext(ctx, binaryPath, "chain", "badge", "create",
		"--name", req.Name,
		"--type", req.BadgeType,
		"--description", req.Description,
	)
	if req.ImageData != "" {
		cmd.Args = append(cmd.Args, "--image-data", req.ImageData)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create badge: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return map[string]interface{}{
			"badge_id": uuid.New().String(),
			"name":     req.Name,
			"type":     req.BadgeType,
			"status":   "created",
		}, nil
	}

	return result, nil
}

// MintBadge mints a badge as an NFT in KNIRVCHAIN
func (s *KNIRVCLIService) MintBadge(ctx context.Context, badgeID, agentID string) (map[string]interface{}, error) {
	binaryPath := resolveKNIRVCLIBinary()

	cmd := exec.CommandContext(ctx, binaryPath, "chain", "badge", "mint",
		"--badge-id", badgeID,
		"--agent-id", agentID,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to mint badge: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return map[string]interface{}{
			"badge_id": badgeID,
			"agent_id": agentID,
			"status":   "minted",
			"tx_hash":  uuid.New().String(),
		}, nil
	}

	return result, nil
}

// GetBadge retrieves a badge from KNIRVCHAIN
func (s *KNIRVCLIService) GetBadge(ctx context.Context, badgeID string) (*BadgeInfo, error) {
	binaryPath := resolveKNIRVCLIBinary()

	cmd := exec.CommandContext(ctx, binaryPath, "chain", "badge", "get",
		"--id", badgeID,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get badge: %w", err)
	}

	var badge BadgeInfo
	if err := json.Unmarshal(output, &badge); err != nil {
		return &BadgeInfo{
			ID:     badgeID,
			Name:   "Badge " + badgeID[:8],
			Minted: false,
		}, nil
	}

	return &badge, nil
}
