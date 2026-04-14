package dve

import (
	"context"
	"fmt"
	"log"

	hasherpb "backend_server/internal/proto/hasher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// HasherExporter handles exporting security data to the KNIRVHASHER 0_DATA_CONNECTOR via gRPC
type HasherExporter struct {
	grpcClient hasherpb.HasherServiceClient
	conn       *grpc.ClientConn
}

// NewHasherExporter creates a new HasherExporter connected to the 0_DATA_CONNECTOR
func NewHasherExporter(connectorAddr string) (*HasherExporter, error) {
	if connectorAddr == "" {
		connectorAddr = "localhost:50051" // Default connector address
	}
	conn, err := grpc.Dial(connectorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial connector: %w", err)
	}

	client := hasherpb.NewHasherServiceClient(conn)
	return &HasherExporter{
		grpcClient: client,
		conn:       conn,
	}, nil
}

// Close closes the gRPC connection
func (he *HasherExporter) Close() error {
	return he.conn.Close()
}

// ExportSecurityData exports user security data to the hasher connector
func (he *HasherExporter) ExportSecurityData(ctx context.Context, orgID, userID string, dataChunks <-chan []byte) error {
	stream, err := he.grpcClient.ExportSecurityData(ctx)
	if err != nil {
		return fmt.Errorf("start export stream: %w", err)
	}

	// Send data chunks
	for chunk := range dataChunks {
		err := stream.Send(&hasherpb.EncryptedChunk{
			Data:    chunk,
			ChunkId: fmt.Sprintf("%s-%s-%d", orgID, userID, len(chunk)),
			IsLast:  false,
		})
		if err != nil {
			return fmt.Errorf("send chunk: %w", err)
		}
	}

	// Close send and receive response
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("close stream: %w", err)
	}

	if resp.Status != "success" {
		return fmt.Errorf("export failed: %s", resp.Message)
	}

	log.Printf("Successfully exported security data for user %s in org %s", userID, orgID)
	return nil
}

// TriggerTraining initiates training for a user
func (he *HasherExporter) TriggerTraining(ctx context.Context, orgID, userID string, triggerType string) error {
	resp, err := he.grpcClient.TriggerTraining(ctx, &hasherpb.TrainingRequest{
		OrgId:   orgID,
		UserId:  userID,
		Trigger: hasherpb.TrainingTrigger_ON_DEMAND, // Map triggerType to enum
		Options: map[string]string{"type": triggerType},
	})
	if err != nil {
		return fmt.Errorf("trigger training: %w", err)
	}

	log.Printf("Training triggered: ID=%s, Status=%s", resp.TrainingId, resp.Status)
	return nil
}

// ValidateAction checks if an action is allowed based on trained security rules
func (he *HasherExporter) ValidateAction(ctx context.Context, orgID, userID, action string, context map[string]string) (*hasherpb.ActionResponse, error) {
	resp, err := he.grpcClient.ValidateAction(ctx, &hasherpb.ActionRequest{
		UserId:  userID,
		Action:  action,
		Context: context,
	})
	if err != nil {
		return nil, fmt.Errorf("validate action: %w", err)
	}

	return resp, nil
}
