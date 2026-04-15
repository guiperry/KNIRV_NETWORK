package dvemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/proto/hasher"
	"backend_server/internal/services/guardrails"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type HasherIntegration struct {
	hasherClient hasher.HasherServiceClient
	conn         *grpc.ClientConn
	guardrailMgr *guardrails.DynamicGuardrailManager
	db           interface {
		GetObjectsByPrefix(prefix string, fn func(key string, value []byte) bool) error
	}
	available      bool
	mu             sync.RWMutex
	socketPath     string
	trainingJobs   map[string]*TrainingJob
	trainingJobsMu sync.RWMutex
}

type TrainingJob struct {
	ID        string
	OrgID     string
	UserID    string
	Trigger   hasher.TrainingTrigger
	Status    string
	StartedAt time.Time
}

func NewHasherIntegration(socketPath string, db interface {
	GetObjectsByPrefix(prefix string, fn func(key string, value []byte) bool) error
}, guardrailMgr *guardrails.DynamicGuardrailManager) *HasherIntegration {
	return &HasherIntegration{
		db:           db,
		guardrailMgr: guardrailMgr,
		socketPath:   socketPath,
		available:    false,
		trainingJobs: make(map[string]*TrainingJob),
	}
}

func (hi *HasherIntegration) Connect(ctx context.Context) error {
	hi.mu.Lock()
	defer hi.mu.Unlock()

	var err error
	hi.conn, err = grpc.NewClient(
		fmt.Sprintf("unix:%s", hi.socketPath),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("HasherIntegration: Failed to connect to hasher gRPC at %s: %v", hi.socketPath, err)
		hi.available = false
		return err
	}

	hi.hasherClient = hasher.NewHasherServiceClient(hi.conn)
	hi.available = true
	log.Printf("HasherIntegration: Connected to hasher gRPC at %s", hi.socketPath)
	return nil
}

func (hi *HasherIntegration) Close() error {
	hi.mu.Lock()
	defer hi.mu.Unlock()

	if hi.conn != nil {
		err := hi.conn.Close()
		hi.conn = nil
		hi.hasherClient = nil
		hi.available = false
		return err
	}
	return nil
}

func (hi *HasherIntegration) IsAvailable() bool {
	hi.mu.RLock()
	defer hi.mu.RUnlock()
	return hi.available && hi.hasherClient != nil
}

func (hi *HasherIntegration) OnGuardrailViolation(violation *guardrails.GuardrailViolation) error {
	if !hi.IsAvailable() {
		log.Printf("HasherIntegration: Skipping training trigger - hasher unavailable")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &hasher.TrainingRequest{
		OrgId:      "default",
		UserId:     violation.NodeID,
		Trigger:    hasher.TrainingTrigger_GUARDRAIL_VIOLATION,
		TrainingId: uuid.New().String(),
		Options: map[string]string{
			"violation_id":   violation.ID,
			"violation_type": violation.ViolationType,
			"severity":       violation.Severity,
			"guardrail_type": string(violation.GuardrailType),
		},
	}

	resp, err := hi.hasherClient.TriggerTraining(ctx, req)
	if err != nil {
		log.Printf("HasherIntegration: Failed to trigger training for violation %s: %v", violation.ID, err)
		return err
	}

	log.Printf("HasherIntegration: Training triggered for violation %s, training_id=%s, status=%s",
		violation.ID, resp.TrainingId, resp.Status)

	hi.trainingJobsMu.Lock()
	hi.trainingJobs[resp.TrainingId] = &TrainingJob{
		ID:        resp.TrainingId,
		OrgID:     req.OrgId,
		UserID:    req.UserId,
		Trigger:   req.Trigger,
		Status:    resp.Status,
		StartedAt: time.Now(),
	}
	hi.trainingJobsMu.Unlock()

	return nil
}

func (hi *HasherIntegration) OnValidationComplete(result *ValidationResult) error {
	if !hi.IsAvailable() {
		return nil
	}

	if result.Success {
		return nil
	}

	violation := &guardrails.GuardrailViolation{
		ID:            uuid.New().String(),
		NodeID:        result.NodeID,
		GuardrailType: guardrails.GuardrailTypeOntology,
		ViolationType: "validation_failure",
		Message:       fmt.Sprintf("Validation failed for task %s: %s", result.TaskID, result.Error),
		Severity:      "medium",
		Context: map[string]interface{}{
			"task_id": result.TaskID,
			"error":   result.Error,
		},
		Timestamp: time.Now(),
	}

	return hi.OnGuardrailViolation(violation)
}

type ValidationResult struct {
	TaskID   string
	NodeID   string
	Success  bool
	Error    string
	Duration time.Duration
}

func (hi *HasherIntegration) ExportUserData(orgID, userID string, dataType hasher.DataType) (<-chan *hasher.EncryptedChunk, error) {
	if !hi.IsAvailable() {
		return nil, fmt.Errorf("hasher not available")
	}

	ctx, cancel := context.WithCancel(context.Background())

	req := &hasher.ExportRequest{
		OrgId:     orgID,
		UserId:    userID,
		DataType:  dataType,
		Encrypted: true,
	}

	stream, err := hi.hasherClient.ExportSecurityData(ctx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start export: %w", err)
	}

	ch := make(chan *hasher.EncryptedChunk, 100)

	go func() {
		defer close(ch)
		defer cancel()

		for {
			chunk, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					log.Printf("HasherIntegration: Export stream error: %v", err)
				}
				return
			}

			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (hi *HasherIntegration) TriggerTraining(orgID, userID string, trigger hasher.TrainingTrigger, options map[string]string) (*hasher.TrainingResponse, error) {
	if !hi.IsAvailable() {
		return &hasher.TrainingResponse{
			TrainingId: "",
			Status:     "unavailable",
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := &hasher.TrainingRequest{
		OrgId:      orgID,
		UserId:     userID,
		Trigger:    trigger,
		Options:    options,
		TrainingId: uuid.New().String(),
	}

	resp, err := hi.hasherClient.TriggerTraining(ctx, req)
	if err != nil {
		return &hasher.TrainingResponse{
			TrainingId: "",
			Status:     "error",
		}, err
	}

	hi.trainingJobsMu.Lock()
	hi.trainingJobs[resp.TrainingId] = &TrainingJob{
		ID:        resp.TrainingId,
		OrgID:     orgID,
		UserID:    userID,
		Trigger:   trigger,
		Status:    resp.Status,
		StartedAt: time.Now(),
	}
	hi.trainingJobsMu.Unlock()

	return resp, nil
}

func (hi *HasherIntegration) GetTrainingStatus(trainingID string) (*hasher.TrainingStatusResponse, error) {
	if !hi.IsAvailable() {
		return nil, fmt.Errorf("hasher not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return hi.hasherClient.GetTrainingStatus(ctx, &hasher.TrainingStatusRequest{
		TrainingId: trainingID,
	})
}

func (hi *HasherIntegration) GetUserRules(orgID, userID string, ruleType hasher.RuleType) ([]*hasher.SecurityRule, error) {
	if !hi.IsAvailable() {
		return nil, fmt.Errorf("hasher not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := hi.hasherClient.GetUserRules(ctx, &hasher.RulesRequest{
		OrgId:    orgID,
		UserId:   userID,
		RuleType: ruleType,
	})
	if err != nil {
		return nil, err
	}

	return resp.Rules, nil
}

func (hi *HasherIntegration) ValidateAction(userID, action string, ctx map[string]string) (*guardrails.HasherSecurityDecision, error) {
	if !hi.IsAvailable() {
		return &guardrails.HasherSecurityDecision{
			Allowed:    true,
			Confidence: 0,
			Note:       "hasher_unavailable",
		}, nil
	}

	grpcCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &hasher.ActionRequest{
		UserId:  userID,
		Action:  action,
		Context: ctx,
	}

	resp, err := hi.hasherClient.ValidateAction(grpcCtx, req)
	if err != nil {
		log.Printf("HasherIntegration: ValidateAction failed: %v", err)
		return &guardrails.HasherSecurityDecision{
			Allowed:    true,
			Confidence: 0,
			Note:       fmt.Sprintf("hasher_error: %v", err),
		}, nil
	}

	return &guardrails.HasherSecurityDecision{
		Allowed:      resp.Allowed,
		Confidence:   float64(resp.Confidence),
		Violations:   resp.Violations,
		AppliedRules: resp.AppliedRules,
		SeedID:       resp.SeedId,
		DecisionID:   resp.DecisionId,
		Note:         "hasher_validated",
	}, nil
}

func (hi *HasherIntegration) StreamActivity(orgID string, eventTypes []string) (<-chan *hasher.ActivityEvent, error) {
	if !hi.IsAvailable() {
		return nil, fmt.Errorf("hasher not available")
	}

	ctx, cancel := context.WithCancel(context.Background())

	req := &hasher.StreamActivityRequest{
		OrgId:      orgID,
		EventTypes: eventTypes,
	}

	stream, err := hi.hasherClient.StreamActivity(ctx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start activity stream: %w", err)
	}

	ch := make(chan *hasher.ActivityEvent, 100)

	go func() {
		defer close(ch)
		defer cancel()

		for {
			event, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					log.Printf("HasherIntegration: Activity stream error: %v", err)
				}
				return
			}

			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (hi *HasherIntegration) GetTrainingJobs() []*TrainingJob {
	hi.trainingJobsMu.RLock()
	defer hi.trainingJobsMu.RUnlock()

	jobs := make([]*TrainingJob, 0, len(hi.trainingJobs))
	for _, job := range hi.trainingJobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (hi *HasherIntegration) ExportGuardrailsForUser(orgID, userID string) ([]byte, error) {
	var violations []*guardrails.GuardrailViolation

	err := hi.db.GetObjectsByPrefix("guardrail:violation:", func(key string, value []byte) bool {
		var v guardrails.GuardrailViolation
		if err := json.Unmarshal(value, &v); err != nil {
			return true
		}
		if userID == "" || v.NodeID == userID {
			violations = append(violations, &v)
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(violations)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (hi *HasherIntegration) ExportOntologyForUser(orgID, userID string) ([]byte, error) {
	var ontologyData []map[string]interface{}

	err := hi.db.GetObjectsByPrefix(fmt.Sprintf("ontology:user:%s:", userID), func(key string, value []byte) bool {
		var data map[string]interface{}
		if err := json.Unmarshal(value, &data); err != nil {
			return true
		}
		ontologyData = append(ontologyData, data)
		return true
	})
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(ontologyData)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (hi *HasherIntegration) RegisterWithDVEManager(dm *DVEManager) {
	if hi.guardrailMgr != nil {
		log.Printf("HasherIntegration: Registered with guardrail manager")
	}
	log.Printf("HasherIntegration: Registered with DVE Manager")
}

type HasherServiceServer struct {
	hasher.UnimplementedHasherServiceServer
	integration *HasherIntegration
}

func NewHasherServiceServer(integration *HasherIntegration) *HasherServiceServer {
	return &HasherServiceServer{
		integration: integration,
	}
}
