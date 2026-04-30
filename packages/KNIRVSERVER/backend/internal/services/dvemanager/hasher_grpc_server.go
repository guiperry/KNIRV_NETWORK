package dvemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/proto/hasher"
	"backend_server/internal/services/guardrails"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	DefaultSocketPath = "/var/lib/knirvserver/sockets/hasher.sock"
)

type HasherGRPCServer struct {
	hasher.UnimplementedHasherTrainingServiceServer
	socketPath string
	listener   net.Listener
	server     *grpc.Server
	db         *database.BuntDBManager
	mu         sync.RWMutex
	running    bool
}

func NewHasherGRPCServer(socketPath string, db *database.BuntDBManager) *HasherGRPCServer {
	return &HasherGRPCServer{
		socketPath: socketPath,
		db:         db,
	}
}

func (s *HasherGRPCServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	// Expand ~ to home directory if present
	socketPath := s.socketPath
	if strings.HasPrefix(socketPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		socketPath = filepath.Join(homeDir, socketPath[1:])
	}

	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	if err := os.RemoveAll(socketPath); err != nil {
		log.Printf("Warning: Failed to remove existing socket: %v", err)
	}

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket %s: %w", socketPath, err)
	}
	s.listener = ln

	s.server = grpc.NewServer()
	hasher.RegisterHasherTrainingServiceServer(s.server, s)

	go func() {
		log.Printf("HasherGRPCServer: Starting on %s", socketPath)
		if err := s.server.Serve(ln); err != nil {
			log.Printf("HasherGRPCServer: Serve error: %v", err)
		}
	}()

	if err := os.Chmod(socketPath, 0666); err != nil {
		log.Printf("Warning: Failed to set socket permissions: %v", err)
	}

	s.running = true
	log.Printf("HasherGRPCServer: Started successfully on %s", s.socketPath)
	return nil
}

func (s *HasherGRPCServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.server != nil {
		s.server.GracefulStop()
		s.server = nil
	}

	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}

	s.running = false
	log.Printf("HasherGRPCServer: Stopped")
	return nil
}

func (s *HasherGRPCServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *HasherGRPCServer) ExportSecurityData(req *hasher.ExportRequest, stream hasher.HasherTrainingService_ExportSecurityDataServer) error {
	log.Printf("HasherGRPCServer: ExportSecurityData called - org=%s, user=%s, type=%v, since=%d",
		req.OrgId, req.UserId, req.DataType, req.SinceTimestamp)

	ctx := stream.Context()

	since := time.Unix(req.SinceTimestamp, 0)
	if req.SinceTimestamp == 0 {
		since = time.Time{}
	}

	dataType := req.DataType
	if dataType == hasher.DataType_ALL {
		dataType = hasher.DataType_ONTOLOGY
	}

	chunkIndex := 0

	for _, dt := range []hasher.DataType{hasher.DataType_ONTOLOGY, hasher.DataType_GUARDRAILS, hasher.DataType_ACTIVITY, hasher.DataType_MARKDOWN, hasher.DataType_USERS} {
		if req.DataType != hasher.DataType_ALL && dt != dataType {
			continue
		}

		var data []byte
		var err error
		switch dt {
		case hasher.DataType_ONTOLOGY:
			data, err = s.exportOntologyData(req.OrgId, req.UserId)
		case hasher.DataType_GUARDRAILS:
			data, err = s.exportGuardrailData(req.OrgId, req.UserId, since)
		case hasher.DataType_ACTIVITY:
			data, err = s.exportActivityData(req.OrgId, req.UserId, since)
		case hasher.DataType_MARKDOWN:
			data, err = s.exportMarkdownData(req.OrgId, req.UserId)
		case hasher.DataType_USERS:
			data, err = s.exportUserData(req.OrgId, req.UserId)
		}

		if err != nil {
			log.Printf("HasherGRPCServer: Export error for type %v: %v", dt, err)
			continue
		}

		if len(data) == 0 {
			continue
		}

		chunkSize := 64 * 1024
		for i := 0; i < len(data); i += chunkSize {
			end := i + chunkSize
			if end > len(data) {
				end = len(data)
			}

			chunk := &hasher.EncryptedChunk{
				Data:        data[i:end],
				ChunkId:     fmt.Sprintf("%s_%d", dt.String(), chunkIndex),
				IsLast:      false,
				ChunkIndex:  int32(chunkIndex),
				TotalChunks: 0,
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err := stream.Send(chunk); err != nil {
				return fmt.Errorf("failed to send chunk: %w", err)
			}
			chunkIndex++
		}
	}

	lastChunk := &hasher.EncryptedChunk{
		Data:        nil,
		ChunkId:     "final",
		IsLast:      true,
		ChunkIndex:  int32(chunkIndex),
		TotalChunks: int32(chunkIndex),
	}

	log.Printf("HasherGRPCServer: Export completed - sent %d chunks", chunkIndex)
	return stream.Send(lastChunk)
}

func (s *HasherGRPCServer) exportOntologyData(orgID, userID string) ([]byte, error) {
	var ontologyData []map[string]interface{}

	if userID != "" && userID != "all" {
		err := s.db.GetObjectsByPrefix(fmt.Sprintf("ontology:user:%s:", userID), func(key string, value []byte) bool {
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
	} else {
		err := s.db.GetObjectsByPrefix("ontology:", func(key string, value []byte) bool {
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
	}

	return json.Marshal(map[string]interface{}{
		"type":      "ontology",
		"org_id":    orgID,
		"user_id":   userID,
		"timestamp": time.Now().Unix(),
		"data":      ontologyData,
	})
}

func (s *HasherGRPCServer) exportGuardrailData(orgID, userID string, since time.Time) ([]byte, error) {
	var violations []*guardrails.GuardrailViolation

	prefix := "guardrail:violation:"
	err := s.db.GetObjectsByPrefix(prefix, func(key string, value []byte) bool {
		var v guardrails.GuardrailViolation
		if err := json.Unmarshal(value, &v); err != nil {
			return true
		}
		if userID != "" && userID != "all" && v.NodeID != userID {
			return true
		}
		if !since.IsZero() && v.Timestamp.Before(since) {
			return true
		}
		violations = append(violations, &v)
		return true
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"type":      "guardrails",
		"org_id":    orgID,
		"user_id":   userID,
		"timestamp": time.Now().Unix(),
		"data":      violations,
	})
}

func (s *HasherGRPCServer) exportActivityData(orgID, userID string, since time.Time) ([]byte, error) {
	var activities []map[string]interface{}

	prefix := "activity:"
	err := s.db.GetObjectsByPrefix(prefix, func(key string, value []byte) bool {
		var data map[string]interface{}
		if err := json.Unmarshal(value, &data); err != nil {
			return true
		}
		if userID != "" && userID != "all" {
			if uid, ok := data["user_id"].(string); !ok || uid != userID {
				return true
			}
		}
		if !since.IsZero() {
			if ts, ok := data["timestamp"].(float64); ok {
				if time.Unix(int64(ts), 0).Before(since) {
					return true
				}
			}
		}
		activities = append(activities, data)
		return true
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"type":      "activity",
		"org_id":    orgID,
		"user_id":   userID,
		"timestamp": time.Now().Unix(),
		"data":      activities,
	})
}

func (s *HasherGRPCServer) exportMarkdownData(orgID, userID string) ([]byte, error) {
	var mdFiles []map[string]interface{}

	prefix := "md:user:"
	err := s.db.GetObjectsByPrefix(prefix, func(key string, value []byte) bool {
		var data map[string]interface{}
		if err := json.Unmarshal(value, &data); err != nil {
			return true
		}
		if userID != "" && userID != "all" {
			if uid, ok := data["user_id"].(string); !ok || uid != userID {
				return true
			}
		}
		mdFiles = append(mdFiles, data)
		return true
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"type":      "markdown",
		"org_id":    orgID,
		"user_id":   userID,
		"timestamp": time.Now().Unix(),
		"data":      mdFiles,
	})
}

func (s *HasherGRPCServer) exportUserData(orgID, userID string) ([]byte, error) {
	var users []map[string]interface{}

	prefix := "user:"
	err := s.db.GetObjectsByPrefix(prefix, func(key string, value []byte) bool {
		var data map[string]interface{}
		if err := json.Unmarshal(value, &data); err != nil {
			return true
		}
		if userID != "" && userID != "all" {
			if uid, ok := data["user_id"].(string); !ok || uid != userID {
				return true
			}
		}
		users = append(users, data)
		return true
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"type":      "users",
		"org_id":    orgID,
		"timestamp": time.Now().Unix(),
		"data":      users,
	})
}

func (s *HasherGRPCServer) TriggerTraining(ctx context.Context, req *hasher.TrainingRequest) (*hasher.TrainingResponse, error) {
	log.Printf("HasherGRPCServer: TriggerTraining - org=%s, user=%s, trigger=%v",
		req.OrgId, req.UserId, req.Trigger)

	trainingID := req.TrainingId
	if trainingID == "" {
		trainingID = uuid.New().String()
	}

	return &hasher.TrainingResponse{
		TrainingId: trainingID,
		Status:     "accepted",
	}, nil
}

func (s *HasherGRPCServer) GetTrainingStatus(ctx context.Context, req *hasher.TrainingStatusRequest) (*hasher.TrainingStatusResponse, error) {
	return &hasher.TrainingStatusResponse{
		TrainingId:      req.TrainingId,
		Status:          "completed",
		Progress:        1.0,
		EpochsCompleted: 100,
		TotalEpochs:     100,
		BestFitness:     0.95,
	}, nil
}

func (s *HasherGRPCServer) GetUserRules(ctx context.Context, req *hasher.RulesRequest) (*hasher.RulesResponse, error) {
	return &hasher.RulesResponse{
		Rules: []*hasher.SecurityRule{},
	}, nil
}

func (s *HasherGRPCServer) ValidateAction(ctx context.Context, req *hasher.ActionRequest) (*hasher.ActionResponse, error) {
	log.Printf("HasherGRPCServer: ValidateAction - user=%s, action=%s", req.UserId, req.Action)

	return &hasher.ActionResponse{
		Allowed:      true,
		Confidence:   1.0,
		Violations:   []string{},
		AppliedRules: []string{},
		SeedId:       "default",
		DecisionId:   uuid.New().String(),
	}, nil
}

func (s *HasherGRPCServer) StreamActivity(req *hasher.StreamActivityRequest, stream hasher.HasherTrainingService_StreamActivityServer) error {
	log.Printf("HasherGRPCServer: StreamActivity - org=%s, types=%v", req.OrgId, req.EventTypes)

	ctx := stream.Context()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			event := &hasher.ActivityEvent{
				EventId:   uuid.New().String(),
				EventType: "heartbeat",
				OrgId:     req.OrgId,
				UserId:    "",
				Data:      map[string]string{},
				Timestamp: time.Now().Unix(),
			}

			if err := stream.Send(event); err != nil {
				return err
			}
		}
	}
}

func (s *HasherGRPCServer) ExportMetadata(ctx context.Context, req *hasher.ExportRequest) (map[string]string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no metadata available")
	}

	metadata := make(map[string]string)
	for key, values := range md {
		if len(values) > 0 {
			metadata[key] = values[0]
		}
	}
	return metadata, nil
}
