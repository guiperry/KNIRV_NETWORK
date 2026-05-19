package inferencer

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"time"
)

// generateSessionID creates a unique session identifier using crypto/rand.
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// ChatMessage represents a single turn in a chat conversation.
type ChatMessage struct {
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatSession represents a persistent conversation with the platform AI.
type ChatSession struct {
	ID        string        `json:"id"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	MaxTurns  int           `json:"max_turns"` // Max conversation turns before oldest is dropped
}

const (
	// DefaultChatMaxTurns is the default number of user+assistant turns kept in context.
	DefaultChatMaxTurns = 20
	// PlatformSystemPrompt instructs the LLM about the KNIRV platform context.
	PlatformSystemPrompt = `You are Ana, the default cognitive engine identity for the KNIRV Distributed Trusted Execution Network (D-TEN).

PLATFORM CONTEXT:
- KNIRV Network transforms AI failures into collective knowledge through sovereign blockchain layers.
- The platform features: DVE (Decentralized Virtual Environment) orchestration, validation engine, cognitive engine with adaptive learning, GraphRAG memory fabric, eBPF-based security monitoring, TEE container runtime, and cross-chain IBC communication.
- Available services: KNIRVCHAIN (blockchain), KNIRVServer (backend), KNIRVGateway (web gateway), KNIRVGraph (temporal hypergraph), KNIRVOracle (root-node oracle), KNIRVArena (3D RTS), KNIRVHeart (AI core).
- The Cognitive Engine manages adaptation learning, pattern analysis, guardrail enforcement, and agent telemetry.
- Users can deploy DVEs, run validation tasks, query the knowledge base, manage plugins, and interact with the blockchain.

RESPONSE STYLE:
- Be conversational, knowledgeable, and precise as Ana, the platform's cognitive engine identity.
- Reference platform capabilities when relevant to the user's question.
- If the user asks about something outside your knowledge, say so honestly.
- Keep responses substantive but not overly verbose.`
)

// ChatRequest is the payload sent to the chat endpoint.
type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

// ChatResponse is the payload returned from the chat endpoint.
type ChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
}

// ChatSessions manages all active chat sessions for the inference service.
type ChatSessions struct {
	mu       sync.RWMutex
	sessions map[string]*ChatSession
}

// NewChatSessions creates a new session manager.
func NewChatSessions() *ChatSessions {
	return &ChatSessions{
		sessions: make(map[string]*ChatSession),
	}
}

// GetOrCreate retrieves an existing session by ID or creates a new one.
func (cs *ChatSessions) GetOrCreate(sessionID string) *ChatSession {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if sessionID != "" {
		if session, ok := cs.sessions[sessionID]; ok {
			return session
		}
	}

	// Create new session
	session := &ChatSession{
		ID:        generateSessionID(),
		Messages:  make([]ChatMessage, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		MaxTurns:  DefaultChatMaxTurns,
	}
	cs.sessions[session.ID] = session
	return session
}

// Get retrieves a session by ID, returns nil if not found.
func (cs *ChatSessions) Get(sessionID string) *ChatSession {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.sessions[sessionID]
}

// Delete removes a session by ID.
func (cs *ChatSessions) Delete(sessionID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.sessions, sessionID)
}

// AddMessage appends a message to the session, trimming old turns if over MaxTurns.
func (s *ChatSession) AddMessage(role, content string) {
	msg := ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()

	// Trim oldest turn pairs (user+assistant) when over MaxTurns
	// Count turns as pairs of user+assistant messages
	if len(s.Messages) > s.MaxTurns*2 {
		excess := len(s.Messages) - s.MaxTurns*2
		// Round up to remove complete pairs
		if excess%2 != 0 {
			excess++
		}
		s.Messages = s.Messages[excess:]
	}
}

// BuildPrompt assembles the full prompt from system prompt, platform context, and message history.
func (s *ChatSession) BuildPrompt(systemPrompt string, platformContext string, newMessage string) string {
	prompt := systemPrompt + "\n\n"

	if platformContext != "" {
		prompt += "=== CURRENT PLATFORM STATE ===\n"
		prompt += platformContext + "\n\n"
	}

	prompt += "=== CONVERSATION HISTORY ===\n"
	for _, msg := range s.Messages {
		switch msg.Role {
		case "user":
			prompt += fmt.Sprintf("User: %s\n", msg.Content)
		case "assistant":
			prompt += fmt.Sprintf("Ana: %s\n", msg.Content)
		}
	}

	prompt += "\n=== CURRENT USER MESSAGE ===\n"
	prompt += fmt.Sprintf("User: %s\n\nAna:", newMessage)

	return prompt
}

// BuildPlatformContext returns a string describing the current platform state.
// This can be extended as the platform grows.
func BuildPlatformContext() string {
	// Static platform description — dynamic metrics can be injected by the handler
	return `The KNIRV D-TEN is operational with all core services running.
Users can interact with the cognitive engine, validation pipeline, memory fabric, and blockchain layers.
Node capabilities include DVE containers with TEE isolation, eBPF security monitoring, and GraphRAG-powered knowledge retrieval.`
}

// Chat processes a conversational message through the inference engine with session history.
// It wraps GenerateTextWithContext with session management and prompt assembly.
func (s *InferenceService) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	s.mutex.Lock()
	if !s.isRunning {
		s.mutex.Unlock()
		return nil, fmt.Errorf("inference service is not running")
	}
	sessions := s.chatSessions
	s.mutex.Unlock()

	if sessions == nil {
		return nil, fmt.Errorf("chat sessions not initialized")
	}

	// Get or create session
	session := sessions.GetOrCreate(req.SessionID)

	// Add user message to history
	session.AddMessage("user", req.Message)

	// Build the full prompt with history
	systemPrompt := PlatformSystemPrompt
	platformContext := BuildPlatformContext()
	fullPrompt := session.BuildPrompt(systemPrompt, platformContext, req.Message)

	log.Printf("Chat: session=%s, message=%q, history=%d turns",
		session.ID, req.Message, len(session.Messages)/2)

	// Call the inference engine
	response, err := s.GenerateTextWithContext(ctx, "", fullPrompt, "")
	if err != nil {
		log.Printf("Chat: inference error for session %s: %v", session.ID, err)
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	// Add assistant response to history
	session.AddMessage("assistant", response)

	return &ChatResponse{
		Response:  response,
		SessionID: session.ID,
	}, nil
}

// InitializeChatSessions sets up the chat session manager on the inference service.
// Must be called before the service starts accepting chat requests.
func (s *InferenceService) InitializeChatSessions() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.chatSessions = NewChatSessions()
	log.Println("ChatSessions: initialized")
}

// ClearChatSession removes a specific session's history.
func (s *InferenceService) ClearChatSession(sessionID string) {
	s.mutex.Lock()
	sessions := s.chatSessions
	s.mutex.Unlock()

	if sessions != nil {
		sessions.Delete(sessionID)
		log.Printf("ChatSessions: cleared session %s", sessionID)
	}
}
