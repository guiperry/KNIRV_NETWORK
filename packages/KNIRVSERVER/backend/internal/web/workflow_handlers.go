package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/workflow"
)

type WorkflowStep struct {
	StepID         int                    `json:"step_id"`
	Name           string                 `json:"name"`
	Command        string                 `json:"command"`
	ExpectedOutput string                 `json:"expected_output,omitempty"`
	RetryPolicy    *WorkflowRetryPolicy   `json:"retry_policy,omitempty"`
	Dependency     []int                  `json:"dependency,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

type WorkflowRetryPolicy struct {
	Count    int           `json:"count"`
	Interval time.Duration `json:"interval"`
}

type Workflow struct {
	WorkflowID  string         `json:"workflow_id"`
	NodeID      string         `json:"node_id,omitempty"`
	Steps       []WorkflowStep `json:"steps"`
	Status      string         `json:"status"`
	Logs        []string       `json:"logs"`
	CreatedAt   time.Time      `json:"created_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

type StepResult struct {
	StepID    int    `json:"step_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	Attempts  int    `json:"attempts"`
	StartedAt string `json:"started_at,omitempty"`
	EndedAt   string `json:"ended_at,omitempty"`
}

type WorkflowExecution struct {
	ExecutionID string       `json:"execution_id"`
	WorkflowID  string       `json:"workflow_id"`
	NodeID      string       `json:"node_id,omitempty"`
	Status      string       `json:"status"`
	Steps       []StepResult `json:"steps"`
	Logs        []string     `json:"logs"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
}

type WorkflowHandlers struct {
	dveManager      *dvemanager.DVEManager
	workflowService *workflow.WorkflowService
	logger          *zap.Logger
}

func NewWorkflowHandlers(dveManager *dvemanager.DVEManager, workflowService *workflow.WorkflowService) *WorkflowHandlers {
	return &WorkflowHandlers{
		dveManager:      dveManager,
		workflowService: workflowService,
		logger:          zap.NewNop(),
	}
}

func (wh *WorkflowHandlers) RegisterRoutes(r *mux.Router) {
	workflowRouter := r.PathPrefix("/api/workflow").Subrouter()
	workflowRouter.HandleFunc("/execute", wh.ExecuteWorkflow).Methods("POST", "OPTIONS")
	workflowRouter.HandleFunc("/executions", wh.ListExecutions).Methods("GET", "OPTIONS")
	workflowRouter.HandleFunc("/executions/{id}", wh.GetExecution).Methods("GET", "OPTIONS")
}

func (wh *WorkflowHandlers) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, `{"error":"invalid workflow JSON"}`, http.StatusBadRequest)
		return
	}
	if len(wf.Steps) == 0 {
		http.Error(w, `{"error":"workflow has no steps"}`, http.StatusBadRequest)
		return
	}
	if wf.WorkflowID == "" {
		wf.WorkflowID = fmt.Sprintf("wf-%d", time.Now().UnixNano())
	}

	if wh.workflowService != nil {
		steps := make([]workflow.WorkflowStep, len(wf.Steps))
		for i, s := range wf.Steps {
			interval := 0
			if s.RetryPolicy != nil {
				interval = int(s.RetryPolicy.Interval.Seconds())
			}
			steps[i] = workflow.WorkflowStep{
				StepID:      s.StepID,
				Name:        s.Name,
				Command:     s.Command,
				Dependency:  s.Dependency,
				Parameters:  s.Parameters,
				RetryPolicy: &workflow.RetryPolicy{Count: s.RetryPolicy.Count, Interval: time.Duration(interval) * time.Second},
			}
		}

		exec, err := wh.workflowService.ExecuteWorkflow(r.Context(), wf.WorkflowID, steps, wf.NodeID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}

		result := wh.convertExecution(exec)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	http.Error(w, `{"error":"workflow service not available"}`, http.StatusServiceUnavailable)
}

func (wh *WorkflowHandlers) convertExecution(exec *workflow.WorkflowExecution) *WorkflowExecution {
	steps := make([]StepResult, len(exec.Steps))
	for i, s := range exec.Steps {
		startedAt := ""
		if !s.StartedAt.IsZero() {
			startedAt = s.StartedAt.Format(time.RFC3339)
		}
		endedAt := ""
		if !s.EndedAt.IsZero() {
			endedAt = s.EndedAt.Format(time.RFC3339)
		}
		steps[i] = StepResult{
			StepID:    s.StepID,
			Name:      s.Name,
			Status:    s.Status,
			Output:    s.Output,
			Error:     s.Error,
			Attempts:  s.Attempts,
			StartedAt: startedAt,
			EndedAt:   endedAt,
		}
	}
	return &WorkflowExecution{
		ExecutionID: exec.ExecutionID,
		WorkflowID:  exec.WorkflowID,
		NodeID:      exec.NodeID,
		Status:      exec.Status,
		Steps:       steps,
		Logs:        exec.Logs,
		StartedAt:   exec.StartedAt,
		CompletedAt: exec.CompletedAt,
	}
}

func (wh *WorkflowHandlers) ListExecutions(w http.ResponseWriter, r *http.Request) {
	if wh.workflowService != nil {
		execs := wh.workflowService.ListExecutions()
		results := make([]*WorkflowExecution, len(execs))
		for i, e := range execs {
			results[i] = wh.convertExecution(e)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"executions": results, "total": len(results)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"executions": []interface{}{}, "total": 0})
}

func (wh *WorkflowHandlers) GetExecution(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if wh.workflowService != nil {
		exec, ok := wh.workflowService.GetExecution(id)
		if !ok {
			http.Error(w, `{"error":"execution not found"}`, http.StatusNotFound)
			return
		}
		result := wh.convertExecution(exec)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	http.Error(w, `{"error":"workflow service not available"}`, http.StatusServiceUnavailable)
}
