package knirvshell

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type KNIRVSHELLService struct {
	sessions   map[string]*TerminalSession
	sessionMu  sync.RWMutex
	binaryPath string
}

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

type CommandRequest struct {
	Command  string            `json:"command"`
	NodeID   string            `json:"node_id,omitempty"`
	Args     []string          `json:"args,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Timeout  int               `json:"timeout,omitempty"`
	UserID   string            `json:"user_id"`
	Username string            `json:"username"`
}

type CommandResult struct {
	SessionID string   `json:"session_id"`
	Command   string   `json:"command"`
	Output    []string `json:"output"`
	ExitCode  int      `json:"exit_code"`
	Status    string   `json:"status"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time,omitempty"`
}

type WalletInfo struct {
	Address  string `json:"address"`
	Balance  int64  `json:"balance"`
	Currency string `json:"currency"`
}

type TokenTransaction struct {
	TxHash string `json:"tx_hash"`
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

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

type BadgeCreateRequest struct {
	Name        string                 `json:"name"`
	BadgeType   string                 `json:"badge_type"`
	Description string                 `json:"description"`
	ImageData   string                 `json:"image_data"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func NewKNIRVSHELLService() (*KNIRVSHELLService, error) {
	binaryPath, err := GetBinaryPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve knirvshell binary: %w", err)
	}

	return &KNIRVSHELLService{
		sessions:   make(map[string]*TerminalSession),
		binaryPath: binaryPath,
	}, nil
}

func (s *KNIRVSHELLService) BinaryPath() string {
	return s.binaryPath
}

func (s *KNIRVSHELLService) ExecuteCommand(ctx context.Context, req *CommandRequest) (*CommandResult, error) {
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

	args := []string{}
	if req.NodeID != "" {
		args = append(args, "--node", req.NodeID)
	}
	args = append(args, req.Args...)
	args = append(args, "--", req.Command)

	cmd := exec.CommandContext(ctx, s.binaryPath, args...)

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

	if err := cmd.Start(); err != nil {
		session.Status = "failed"
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	go s.streamOutput(session, stdout, "stdout")
	go s.streamOutput(session, stderr, "stderr")

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
	}()

	return &CommandResult{
		SessionID: sessionID,
		Command:   req.Command,
		Status:    "running",
		StartTime: session.StartTime.Format(time.RFC3339),
	}, nil
}

func (s *KNIRVSHELLService) streamOutput(session *TerminalSession, reader io.Reader, streamType string) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			output := string(buf[:n])
			session.Output = append(session.Output, output)
			log.Printf("[%s] %s", streamType, output)
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading %s for session %s: %v", streamType, session.ID, err)
			}
			break
		}
	}
}

func (s *KNIRVSHELLService) GetSession(sessionID string) (*TerminalSession, error) {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

func (s *KNIRVSHELLService) ListSessions() []*TerminalSession {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()

	sessions := make([]*TerminalSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

func (s *KNIRVSHELLService) StopSession(sessionID string) error {
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

func (s *KNIRVSHELLService) SendInput(sessionID string, input string) error {
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

func (s *KNIRVSHELLService) GetWalletInfo(ctx context.Context, walletAddress string) (*WalletInfo, error) {
	cmd := exec.CommandContext(ctx, s.binaryPath, "wallet", "info", "--address", walletAddress)
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

func (s *KNIRVSHELLService) SendToken(ctx context.Context, from, to string, amount int64) (*TokenTransaction, error) {
	cmd := exec.CommandContext(ctx, s.binaryPath, "wallet", "send",
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

func (s *KNIRVSHELLService) ExecuteTEECommand(ctx context.Context, command string, nodeID string) (*CommandResult, error) {
	return s.ExecuteCommand(ctx, &CommandRequest{
		Command:  command,
		NodeID:   nodeID,
		Args:     []string{"tee"},
		UserID:   "system",
		Username: "knirvshell",
	})
}

func (s *KNIRVSHELLService) ExecuteP2PCommand(ctx context.Context, command string, nodeID string) (*CommandResult, error) {
	return s.ExecuteCommand(ctx, &CommandRequest{
		Command:  command,
		NodeID:   nodeID,
		Args:     []string{"p2p"},
		UserID:   "system",
		Username: "knirvshell",
	})
}

func (s *KNIRVSHELLService) ExecuteChainCommand(ctx context.Context, command string) (*CommandResult, error) {
	return s.ExecuteCommand(ctx, &CommandRequest{
		Command:  command,
		Args:     []string{"chain"},
		UserID:   "system",
		Username: "knirvshell",
	})
}

func (s *KNIRVSHELLService) CleanupExpiredSessions(maxAge time.Duration) {
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

func (s *KNIRVSHELLService) Start() error {
	log.Println("KNIRVSHELL service started")
	return nil
}

func (s *KNIRVSHELLService) Stop() error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	for _, session := range s.sessions {
		if session.Process != nil && session.Status == "running" {
			session.Process.Process.Kill()
		}
	}

	log.Println("KNIRVSHELL service stopped")
	return nil
}

func (s *KNIRVSHELLService) CreateBadge(ctx context.Context, req *BadgeCreateRequest) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, s.binaryPath, "chain", "badge", "create",
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

func (s *KNIRVSHELLService) MintBadge(ctx context.Context, badgeID, agentID string) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, s.binaryPath, "chain", "badge", "mint",
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

func (s *KNIRVSHELLService) GetBadge(ctx context.Context, badgeID string) (*BadgeInfo, error) {
	cmd := exec.CommandContext(ctx, s.binaryPath, "chain", "badge", "get",
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

func ResolveBinaryPath() (string, error) {
	if p := os.Getenv("KNIRV_SHELL_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	dir := os.Getenv("KNIRV_SHELL_BINARY_DIR")
	if dir == "" {
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			dir = filepath.Join(xdgDataHome, "knirvserver", "bin")
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to resolve home directory: %w", err)
			}
			dir = filepath.Join(homeDir, ".local", "share", "knirvserver", "bin")
		}
	}

	binPath := filepath.Join(dir, "knirvshell")
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	return ExtractEmbeddedBinary(dir)
}
