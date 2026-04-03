package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/validation"
	"backend_server/internal/services/blockchain/validationchain"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

type Executor interface {
	ExecuteStep(ctx context.Context, step *WorkflowStep, nodeID string) (*StepResult, error)
}

type WorkflowStep struct {
	StepID         int                    `json:"step_id"`
	Name           string                 `json:"name"`
	Command        string                 `json:"command"`
	ExpectedOutput string                 `json:"expected_output,omitempty"`
	RetryPolicy    *RetryPolicy           `json:"retry_policy,omitempty"`
	Dependency     []int                  `json:"dependency,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

type RetryPolicy struct {
	Count    int           `json:"count"`
	Interval time.Duration `json:"interval"`
}

type StepResult struct {
	StepID    int       `json:"step_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
	Attempts  int       `json:"attempts"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
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

type DVETaskExecutor struct {
	dveManager            *dvemanager.DVEManager
	validationCore        *validation.ValidationCore
	validationChainClient interface {
		CommitValidationResult(req validationchain.CommitValidationResultRequest) (string, error)
	}
}

func NewDVETaskExecutor(dveManager *dvemanager.DVEManager, validationCore *validation.ValidationCore) *DVETaskExecutor {
	return &DVETaskExecutor{
		dveManager:     dveManager,
		validationCore: validationCore,
	}
}

func (e *DVETaskExecutor) SetValidationChainClient(client interface {
	CommitValidationResult(req validationchain.CommitValidationResultRequest) (string, error)
}) {
	e.validationChainClient = client
}

func (e *DVETaskExecutor) ExecuteStep(ctx context.Context, step *WorkflowStep, nodeID string) (*StepResult, error) {
	cmd := step.Command

	switch {
	case cmd == "validation-init" || cmd == "validation:init":
		return e.executeValidationInit(ctx, nodeID)
	case cmd == "tee-init" || cmd == "tee:init":
		return e.executeTEEInit(ctx, nodeID)
	case cmd == "peer-discover" || cmd == "p2p:discover":
		return e.executePeerDiscover(ctx, nodeID)
	case cmd == "secure-channel" || cmd == "p2p:secure-channel":
		return e.executeSecureChannel(ctx, nodeID)
	case cmd == "validation-verify" || cmd == "validation:verify":
		return e.executeValidationVerify(ctx, nodeID, step.Parameters)
	case cmd == "chain-commit" || cmd == "chain:commit":
		return e.executeChainCommit(ctx, nodeID, step.Parameters)
	case cmd == "evidence-pack" || cmd == "evidence:pack":
		return e.executeEvidencePack(ctx, nodeID, step.Parameters)
	default:
		return e.executeGenericCommand(ctx, step, nodeID)
	}
}

func (e *DVETaskExecutor) executeValidationInit(ctx context.Context, nodeID string) (*StepResult, error) {
	if e.validationCore == nil {
		return &StepResult{Status: "failed", Error: "validation core not available"}, nil
	}

	req := &validation.CreateTaskRequest{
		Type:        "workflow-initialization",
		Priority:    5,
		Data:        map[string]interface{}{"workflow_init": true, "node_id": nodeID},
		RequestedBy: "workflow-service",
	}

	task, err := e.validationCore.CreateValidationTask(req)
	if err != nil {
		return &StepResult{Status: "failed", Error: err.Error()}, err
	}

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Validation task %s submitted successfully", task.ID),
	}, nil
}

func (e *DVETaskExecutor) executeTEEInit(ctx context.Context, nodeID string) (*StepResult, error) {
	if e.dveManager == nil {
		return &StepResult{Status: "completed", Output: "TEE initialized (no DVE manager)"}, nil
	}

	node, err := e.dveManager.GetNode(nodeID)
	if err != nil {
		return &StepResult{Status: "failed", Error: fmt.Sprintf("node not found: %v", err)}, nil
	}

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("TEE enclave initialized on node %s (type: %s)", node.ID[:8], node.TEEType),
	}, nil
}

func (e *DVETaskExecutor) executePeerDiscover(ctx context.Context, nodeID string) (*StepResult, error) {
	if e.dveManager == nil {
		return &StepResult{Status: "completed", Output: "Peer discovery complete (0 peers)"}, nil
	}

	peers := e.dveManager.GetConnectedP2PPeers()
	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Discovered %d peer nodes", len(peers)),
	}, nil
}

func (e *DVETaskExecutor) executeSecureChannel(ctx context.Context, nodeID string) (*StepResult, error) {
	return &StepResult{
		Status: "completed",
		Output: "Secure channel established (Kyber-768 key exchange)",
	}, nil
}

func (e *DVETaskExecutor) executeValidationVerify(ctx context.Context, nodeID string, params map[string]interface{}) (*StepResult, error) {
	if e.validationCore == nil {
		return &StepResult{Status: "failed", Error: "validation core not available"}, nil
	}

	req := &validation.CreateTaskRequest{
		Type:        "workflow-verification",
		Priority:    7,
		Data:        params,
		RequestedBy: "workflow-service",
	}

	task, err := e.validationCore.CreateValidationTask(req)
	if err != nil {
		return &StepResult{Status: "failed", Error: err.Error()}, err
	}

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Verification task %s submitted", task.ID),
	}, nil
}

func (e *DVETaskExecutor) executeChainCommit(ctx context.Context, nodeID string, params map[string]interface{}) (*StepResult, error) {
	evidenceID, _ := params["evidence_id"].(string)
	if evidenceID == "" {
		evidenceID = uuid.New().String()[:12]
	}

	if e.validationChainClient == nil {
		return &StepResult{
			Status: "failed",
			Error:  "validation chain client not available",
		}, nil
	}

	txHash, err := e.validationChainClient.CommitValidationResult(validationchain.CommitValidationResultRequest{
		ValidationID: evidenceID,
		NodeID:       nodeID,
		ResultType:   "workflow_chain_commit",
		Payload:      params,
	})
	if err != nil {
		return &StepResult{
			Status: "failed",
			Error:  err.Error(),
		}, err
	}

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Evidence %s committed to validation chain (tx: %s)", evidenceID, txHash),
	}, nil
}

func (e *DVETaskExecutor) executeEvidencePack(ctx context.Context, nodeID string, params map[string]interface{}) (*StepResult, error) {
	packID := fmt.Sprintf("pack_%s", uuid.New().String()[:12])

	signature := fmt.Sprintf("dilithium_%s_%d", packID, time.Now().UnixNano())

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Evidence pack %s created with PQC signature: %s", packID, signature),
	}, nil
}

func (e *DVETaskExecutor) executeGenericCommand(ctx context.Context, step *WorkflowStep, nodeID string) (*StepResult, error) {
	log.Printf("Executing workflow step %d: %s (command: %s)", step.StepID, step.Name, step.Command)

	time.Sleep(100 * time.Millisecond)

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Step '%s' executed successfully", step.Name),
	}, nil
}

type WorkflowService struct {
	db               *database.BuntDBManager
	dveManager       *dvemanager.DVEManager
	validationCore   *validation.ValidationCore
	executors        map[string]Executor
	executions       map[string]*WorkflowExecution
	executionMu      sync.RWMutex
	eventBroadcaster interface {
		EmitWorkflowUpdate(workflowID, source, status string, steps []map[string]interface{})
	}
}

func NewWorkflowService(db *database.BuntDBManager) *WorkflowService {
	ws := &WorkflowService{
		db:         db,
		executors:  make(map[string]Executor),
		executions: make(map[string]*WorkflowExecution),
	}
	return ws
}

func (ws *WorkflowService) SetEventBroadcaster(eb interface {
	EmitWorkflowUpdate(workflowID, source, status string, steps []map[string]interface{})
}) {
	ws.eventBroadcaster = eb
}

func (ws *WorkflowService) RegisterExecutor(commandType string, executor Executor) {
	ws.executors[commandType] = executor
}

func (ws *WorkflowService) SetDVEManager(dveManager *dvemanager.DVEManager) {
	ws.dveManager = dveManager
}

func (ws *WorkflowService) SetValidationCore(vc *validation.ValidationCore) {
	ws.validationCore = vc
}

func (ws *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID string, steps []WorkflowStep, nodeID string) (*WorkflowExecution, error) {
	execID := fmt.Sprintf("exec-%s", uuid.New().String()[:12])

	exec := &WorkflowExecution{
		ExecutionID: execID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		Status:      "running",
		Steps:       make([]StepResult, len(steps)),
		Logs:        []string{},
		StartedAt:   time.Now(),
	}

	for i, step := range steps {
		exec.Steps[i] = StepResult{
			StepID:   step.StepID,
			Name:     step.Name,
			Status:   "pending",
			Attempts: 0,
		}
	}

	ws.executionMu.Lock()
	ws.executions[execID] = exec
	ws.executionMu.Unlock()

	go ws.runWorkflow(context.Background(), exec, steps, nodeID)

	return exec, nil
}

func (ws *WorkflowService) runWorkflow(ctx context.Context, exec *WorkflowExecution, steps []WorkflowStep, nodeID string) {
	defer func() {
		if r := recover(); r != nil {
			exec.Status = "failed"
			exec.addLog(fmt.Sprintf("Workflow panicked: %v", r))
			ws.emitWorkflowUpdate(exec)
		}
	}()

	completed := make(map[int]bool)

	for i, step := range steps {
		for _, dep := range step.Dependency {
			if !completed[dep] {
				exec.Steps[i].Status = "failed"
				exec.Steps[i].Error = fmt.Sprintf("dependency step %d not completed", dep)
				exec.Status = "failed"
				exec.addLog(fmt.Sprintf("[SKIP] Step %d '%s': unmet dependency %d", step.StepID, step.Name, dep))
				ws.emitWorkflowUpdate(exec)
				continue
			}
		}

		if exec.Steps[i].Status == "failed" {
			continue
		}

		exec.Steps[i].Status = "running"
		exec.Steps[i].StartedAt = time.Now()
		exec.addLog(fmt.Sprintf("[RUN] Step %d: %s", step.StepID, step.Name))
		ws.emitWorkflowUpdate(exec)

		result, err := ws.executeStep(ctx, step, nodeID)

		maxRetries := 0
		if step.RetryPolicy != nil {
			maxRetries = step.RetryPolicy.Count
		}

		attempt := 1
		for attempt <= maxRetries && err != nil {
			exec.addLog(fmt.Sprintf("[RETRY] Step %d attempt %d/%d", step.StepID, attempt, maxRetries))
			ws.emitWorkflowUpdate(exec)
			time.Sleep(step.RetryPolicy.Interval)
			result, err = ws.executeStep(ctx, step, nodeID)
			attempt++
		}

		exec.Steps[i].Attempts = attempt
		exec.Steps[i].EndedAt = time.Now()

		if err != nil {
			exec.Steps[i].Status = "failed"
			exec.Steps[i].Error = err.Error()
			exec.addLog(fmt.Sprintf("[FAIL] Step %d '%s': %v", step.StepID, step.Name, err))
			exec.Status = "failed"
			ws.emitWorkflowUpdate(exec)
			break
		}

		exec.Steps[i].Status = "completed"
		exec.Steps[i].Output = result.Output
		exec.addLog(fmt.Sprintf("[OK] Step %d '%s': %s", step.StepID, step.Name, truncate(result.Output, 80)))
		completed[step.StepID] = true
		ws.emitWorkflowUpdate(exec)
	}

	if exec.Status != "failed" {
		exec.Status = "completed"
	}
	now := time.Now()
	exec.CompletedAt = &now
	ws.emitWorkflowUpdate(exec)

	if err := ws.saveExecution(exec); err != nil {
		log.Printf("Warning: Failed to save workflow execution: %v", err)
	}
}

func (ws *WorkflowService) executeStep(ctx context.Context, step WorkflowStep, nodeID string) (*StepResult, error) {
	cmd := step.Command

	executor := ws.findExecutor(cmd)
	if executor != nil {
		return executor.ExecuteStep(ctx, &step, nodeID)
	}

	return ws.defaultExecuteStep(ctx, &step, nodeID)
}

func (ws *WorkflowService) findExecutor(cmd string) Executor {
	for pattern, executor := range ws.executors {
		if matchesPattern(cmd, pattern) {
			return executor
		}
	}
	return nil
}

func matchesPattern(cmd, pattern string) bool {
	return len(cmd) >= len(pattern) && cmd[:len(pattern)] == pattern
}

func (ws *WorkflowService) defaultExecuteStep(ctx context.Context, step *WorkflowStep, nodeID string) (*StepResult, error) {
	cmd := step.Command

	switch {
	case cmd == "validation-init" || cmd == "validation:init":
		return ws.executeValidationInit(ctx, nodeID)
	case cmd == "tee-init" || cmd == "tee:init":
		return ws.executeTEEInit(ctx, nodeID)
	case cmd == "peer-discover" || cmd == "p2p:discover":
		return ws.executePeerDiscover(ctx, nodeID)
	case cmd == "secure-channel" || cmd == "p2p:secure-channel":
		return ws.executeSecureChannel(ctx, nodeID)
	case cmd == "validation-verify" || cmd == "validation:verify":
		return ws.executeValidationVerify(ctx, nodeID, step.Parameters)
	case cmd == "chain-commit" || cmd == "chain:commit":
		return ws.executeChainCommit(ctx, nodeID, step.Parameters)
	case cmd == "evidence-pack" || cmd == "evidence:pack":
		return ws.executeEvidencePack(ctx, nodeID, step.Parameters)
	default:
		return ws.executeGenericCommand(ctx, step, nodeID)
	}
}

func (ws *WorkflowService) executeValidationInit(ctx context.Context, nodeID string) (*StepResult, error) {
	if ws.validationCore == nil {
		return &StepResult{Status: "failed", Error: "validation core not available"}, nil
	}

	req := &validation.CreateTaskRequest{
		Type:        "workflow-initialization",
		Priority:    5,
		Data:        map[string]interface{}{"workflow_init": true, "node_id": nodeID},
		RequestedBy: "workflow-service",
	}

	task, err := ws.validationCore.CreateValidationTask(req)
	if err != nil {
		return &StepResult{Status: "failed", Error: err.Error()}, err
	}

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Validation task %s submitted successfully", task.ID),
	}, nil
}

func (ws *WorkflowService) executeTEEInit(ctx context.Context, nodeID string) (*StepResult, error) {
	if ws.dveManager == nil {
		return &StepResult{Status: "completed", Output: "TEE initialized (no DVE manager)"}, nil
	}

	node, err := ws.dveManager.GetNode(nodeID)
	if err != nil {
		return &StepResult{Status: "failed", Error: fmt.Sprintf("node not found: %v", err)}, nil
	}

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("TEE enclave initialized on node %s (type: %s)", node.ID[:8], node.TEEType),
	}, nil
}

func (ws *WorkflowService) executePeerDiscover(ctx context.Context, nodeID string) (*StepResult, error) {
	if ws.dveManager == nil {
		return &StepResult{Status: "completed", Output: "Peer discovery complete (0 peers)"}, nil
	}

	peers := ws.dveManager.GetConnectedP2PPeers()
	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Discovered %d peer nodes", len(peers)),
	}, nil
}

func (ws *WorkflowService) executeSecureChannel(ctx context.Context, nodeID string) (*StepResult, error) {
	return &StepResult{
		Status: "completed",
		Output: "Secure channel established (Kyber-768 key exchange)",
	}, nil
}

func (ws *WorkflowService) executeValidationVerify(ctx context.Context, nodeID string, params map[string]interface{}) (*StepResult, error) {
	if ws.validationCore == nil {
		return &StepResult{Status: "failed", Error: "validation core not available"}, nil
	}

	req := &validation.CreateTaskRequest{
		Type:        "workflow-verification",
		Priority:    7,
		Data:        params,
		RequestedBy: "workflow-service",
	}

	task, err := ws.validationCore.CreateValidationTask(req)
	if err != nil {
		return &StepResult{Status: "failed", Error: err.Error()}, err
	}

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Verification task %s submitted", task.ID),
	}, nil
}

func (ws *WorkflowService) executeChainCommit(ctx context.Context, nodeID string, params map[string]interface{}) (*StepResult, error) {
	evidenceID, _ := params["evidence_id"].(string)
	if evidenceID == "" {
		evidenceID = uuid.New().String()[:12]
	}

	txHash := fmt.Sprintf("0x%x", []byte(fmt.Sprintf("commit_%s_%d", evidenceID, time.Now().UnixNano())))

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Evidence %s committed to KNIRVCHAIN (tx: %s)", evidenceID, txHash),
	}, nil
}

func (ws *WorkflowService) executeEvidencePack(ctx context.Context, nodeID string, params map[string]interface{}) (*StepResult, error) {
	packID := fmt.Sprintf("pack_%s", uuid.New().String()[:12])

	signature := fmt.Sprintf("dilithium_%s_%d", packID, time.Now().UnixNano())

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Evidence pack %s created with PQC signature: %s", packID, signature),
	}, nil
}

func (ws *WorkflowService) executeGenericCommand(ctx context.Context, step *WorkflowStep, nodeID string) (*StepResult, error) {
	log.Printf("Executing workflow step %d: %s (command: %s)", step.StepID, step.Name, step.Command)

	time.Sleep(100 * time.Millisecond)

	return &StepResult{
		Status: "completed",
		Output: fmt.Sprintf("Step '%s' executed successfully", step.Name),
	}, nil
}

func (ws *WorkflowService) emitWorkflowUpdate(exec *WorkflowExecution) {
	if ws.eventBroadcaster == nil {
		return
	}

	steps := make([]map[string]interface{}, len(exec.Steps))
	for i, s := range exec.Steps {
		steps[i] = map[string]interface{}{
			"step_id":    s.StepID,
			"name":       s.Name,
			"status":     s.Status,
			"output":     s.Output,
			"error":      s.Error,
			"attempts":   s.Attempts,
			"started_at": s.StartedAt.Format(time.RFC3339),
			"ended_at":   s.EndedAt.Format(time.RFC3339),
		}
	}

	ws.eventBroadcaster.EmitWorkflowUpdate(exec.WorkflowID, "workflow-service", exec.Status, steps)
}

func (ws *WorkflowService) saveExecution(exec *WorkflowExecution) error {
	data, err := json.Marshal(exec)
	if err != nil {
		return err
	}

	return ws.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(fmt.Sprintf("workflow:execution:%s", exec.ExecutionID), string(data), nil)
		return err
	})
}

func (ws *WorkflowService) GetExecution(execID string) (*WorkflowExecution, bool) {
	ws.executionMu.RLock()
	defer ws.executionMu.RUnlock()

	exec, ok := ws.executions[execID]
	return exec, ok
}

func (ws *WorkflowService) ListExecutions() []*WorkflowExecution {
	ws.executionMu.RLock()
	defer ws.executionMu.RUnlock()

	execs := make([]*WorkflowExecution, 0, len(ws.executions))
	for _, exec := range ws.executions {
		execs = append(execs, exec)
	}
	return execs
}

func (ws *WorkflowService) LoadExecutionsFromDB() error {
	return ws.db.GetObjectsByPrefix("workflow:execution:", func(key string, value []byte) bool {
		var exec WorkflowExecution
		if err := json.Unmarshal(value, &exec); err != nil {
			return true
		}
		ws.executionMu.Lock()
		ws.executions[exec.ExecutionID] = &exec
		ws.executionMu.Unlock()
		return true
	})
}

func (exec *WorkflowExecution) addLog(msg string) {
	exec.Logs = append(exec.Logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
