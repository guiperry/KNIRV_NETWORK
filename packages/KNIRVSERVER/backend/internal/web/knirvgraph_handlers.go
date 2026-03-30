package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend_server/internal/database"
)

type SyncManager interface {
	QueueChange(change interface{})
}

type KnirvGraphHandlers struct {
	db          *database.BuntDBManager
	syncManager SyncManager
}

func NewKnirvGraphHandlers(db *database.BuntDBManager, syncManager SyncManager) *KnirvGraphHandlers {
	return &KnirvGraphHandlers{
		db:          db,
		syncManager: syncManager,
	}
}

type ErrorNodeRequest struct {
	TaskID           string                 `json:"taskId"`
	TaskName         string                 `json:"taskName"`
	TaskType         string                 `json:"taskType"`
	ErrorDetails     string                 `json:"errorDetails"`
	Timestamp        string                 `json:"timestamp"`
	Duration         int64                  `json:"duration"`
	WorkerID         string                 `json:"workerId"`
	Trace            string                 `json:"trace"`
	DocumentsTouched []string               `json:"documentsTouched"`
	UserMetadata     map[string]interface{} `json:"userMetadata"`
	PoliciesEnforced []string               `json:"policiesEnforced"`
	SubmittedAt      string                 `json:"submittedAt"`
	ResolutionStatus string                 `json:"resolutionStatus"`
}

type ErrorNodeResponse struct {
	NodeID    string `json:"nodeId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func (h *KnirvGraphHandlers) CreateErrorNode(w http.ResponseWriter, r *http.Request) {
	var req ErrorNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Generate a unique node ID
	nodeID := fmt.Sprintf("error_node_%d", time.Now().UnixNano())

	// Create the error node with additional metadata
	errorNode := map[string]interface{}{
		"node_id":           nodeID,
		"task_id":           req.TaskID,
		"task_name":         req.TaskName,
		"task_type":         req.TaskType,
		"error_details":     req.ErrorDetails,
		"timestamp":         req.Timestamp,
		"duration":          req.Duration,
		"worker_id":         req.WorkerID,
		"trace":             req.Trace,
		"documents_touched": req.DocumentsTouched,
		"user_metadata":     req.UserMetadata,
		"policies_enforced": req.PoliciesEnforced,
		"submitted_at":      req.SubmittedAt,
		"resolution_status": req.ResolutionStatus,
		"created_at":        time.Now().Format(time.RFC3339),
		"updated_at":        time.Now().Format(time.RFC3339),
	}

	// Store in database
	key := fmt.Sprintf("knirvgraph:error_nodes:%s", nodeID)
	if err := h.db.StoreJSON(key, errorNode); err != nil {
		log.Printf("Failed to store error node: %v", err)
		http.Error(w, "failed to store error node", http.StatusInternalServerError)
		return
	}

	// Queue for sync to embedded KNIRVGRAPH
	if h.syncManager != nil {
		h.syncManager.QueueChange(map[string]interface{}{
			"type":    "error_node",
			"data":    errorNode,
			"message": fmt.Sprintf("Commit error node: %s", nodeID),
			"author":  "knirvserver",
		})
	}

	// Also add to error queue for processing
	queueKey := fmt.Sprintf("knirvgraph:error_queue:%d", time.Now().UnixNano())
	queueEntry := map[string]interface{}{
		"node_id":   nodeID,
		"task_id":   req.TaskID,
		"queued_at": time.Now().Format(time.RFC3339),
		"status":    "pending",
	}
	if err := h.db.StoreJSON(queueKey, queueEntry); err != nil {
		log.Printf("Failed to add to error queue: %v", err)
		// Don't fail the request, just log the error
	}

	log.Printf("Error node created: %s for task: %s", nodeID, req.TaskID)

	response := ErrorNodeResponse{
		NodeID:    nodeID,
		Status:    "created",
		Message:   "Error node created and queued for resolution",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *KnirvGraphHandlers) GetErrorNodes(w http.ResponseWriter, r *http.Request) {
	var nodes []map[string]interface{}

	err := h.db.GetObjectsByPrefix("knirvgraph:error_nodes:", func(key string, value []byte) bool {
		var node map[string]interface{}
		if err := json.Unmarshal(value, &node); err == nil {
			nodes = append(nodes, node)
		}
		return true
	})

	if err != nil {
		log.Printf("Failed to get error nodes: %v", err)
		http.Error(w, "failed to get error nodes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func (h *KnirvGraphHandlers) GetErrorQueue(w http.ResponseWriter, r *http.Request) {
	var queue []map[string]interface{}

	err := h.db.GetObjectsByPrefix("knirvgraph:error_queue:", func(key string, value []byte) bool {
		var entry map[string]interface{}
		if err := json.Unmarshal(value, &entry); err == nil {
			queue = append(queue, entry)
		}
		return true
	})

	if err != nil {
		log.Printf("Failed to get error queue: %v", err)
		http.Error(w, "failed to get error queue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queue)
}
