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
	grpcClient hasherpb.HasherTrainingServiceClient
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

	client := hasherpb.NewHasherTrainingServiceClient(conn)
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
// Returns a channel that yields encrypted chunks from the server
func (he *HasherExporter) ExportSecurityData(ctx context.Context, orgID, userID string, dataType hasherpb.DataType) (ch chan *hasherpb.EncryptedChunk, err error) {
	ch = make(chan *hasherpb.EncryptedChunk, 10)

	stream, err := he.grpcClient.ExportSecurityData(ctx, &hasherpb.ExportRequest{
		OrgId:     orgID,
		UserId:    userID,
		DataType:  dataType,
		Encrypted: true,
	})
	if err != nil {
		close(ch)
		return nil, fmt.Errorf("start export stream: %w", err)
	}

	go func() {
		defer close(ch)
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			ch <- chunk
		}
	}()

	return ch, nil
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
