package graph

import (
	"fmt"
	"strings"
	"time"

	"backend_server/internal/storage/mdstorage"
)

// ContextRecord represents a human-readable reasoning trace
type ContextRecord struct {
	ID        string    `json:"id"`
	ErrorID   string    `json:"error_id"`
	AgentID   string    `json:"agent_id"`
	Trace     []string  `json:"trace"` // List of reasoning steps
	Result    string    `json:"result"`
	Timestamp time.Time `json:"timestamp"`
}

// ReasoningEngine implements NRV-based reasoning and record generation
type ReasoningEngine struct {
	storage *mdstorage.MarkdownStorageDriver
}

// NewReasoningEngine creates a new ReasoningEngine
func NewReasoningEngine(storage *mdstorage.MarkdownStorageDriver) *ReasoningEngine {
	return &ReasoningEngine{
		storage: storage,
	}
}

// GenerateTrace creates a .md ContextRecord for an error resolution
func (e *ReasoningEngine) GenerateTrace(agentID, errorID string, steps []string, result string) error {
	record := &ContextRecord{
		ID:        fmt.Sprintf("trace_%d", time.Now().UnixNano()),
		ErrorID:   errorID,
		AgentID:   agentID,
		Trace:     steps,
		Result:    result,
		Timestamp: time.Now(),
	}

	// Format as Markdown content
	var md strings.Builder
	md.WriteString(fmt.Sprintf("# Reasoning Trace: %s\n\n", record.ID))
	md.WriteString(fmt.Sprintf("**Agent:** %s  \n", record.AgentID))
	md.WriteString(fmt.Sprintf("**Error ID:** %s  \n", record.ErrorID))
	md.WriteString(fmt.Sprintf("**Timestamp:** %s  \n\n", record.Timestamp.Format(time.RFC1123)))
	
	md.WriteString("## Reasoning Steps\n")
	for i, step := range record.Trace {
		md.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}
	
	md.WriteString("\n## Result\n")
	md.WriteString(record.Result + "\n")

	doc := &mdstorage.MarkdownDocument{
		ID:        record.ID,
		Type:      "TRACE",
		Timestamp: record.Timestamp,
		Metadata: map[string]interface{}{
			"agent_id": agentID,
			"error_id": errorID,
		},
		Content: []byte(md.String()),
	}

	return e.storage.SaveDocument(doc)
}
