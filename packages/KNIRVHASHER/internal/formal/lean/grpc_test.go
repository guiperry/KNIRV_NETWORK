package lean

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "knirvhasher/internal/proto/hasher/v1"
	"knirvhasher/pkg/hashing/proofasset"
	"google.golang.org/grpc"
)

func TestGRPCServerMetrics(t *testing.T) {
	store := &MemoryProofAssetStore{}
	srv := NewGRPCServer(&Worker{}, store)

	srv.metrics.RecordCheck(100, proofasset.StatusFormallyVerified)
	srv.metrics.RecordCheck(200, proofasset.StatusFormallyRejected)

	resp, err := srv.Metrics(context.Background(), &pb.MetricsRequest{})
	if err != nil {
		t.Fatalf("Metrics() error: %v", err)
	}

	if resp.TotalSubmissions != 2 {
		t.Errorf("TotalSubmissions = %d, want 2", resp.TotalSubmissions)
	}
	if resp.VerifiedCount != 1 {
		t.Errorf("VerifiedCount = %d, want 1", resp.VerifiedCount)
	}
	if resp.RejectedCount != 1 {
		t.Errorf("RejectedCount = %d, want 1", resp.RejectedCount)
	}
	if resp.QueueDepth != 0 {
		t.Errorf("QueueDepth = %d, want 0", resp.QueueDepth)
	}
}

func TestGRPCServerUnixSocket(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "knirvhasher-test-"+time.Now().Format("20060102150405")+".sock")
	_ = os.Remove(socketPath)

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
	}
	defer lis.Close()
	defer os.Remove(socketPath)

	grpcSrv := grpc.NewServer()
	defer grpcSrv.Stop()

	srv := NewGRPCServer(&Worker{}, &MemoryProofAssetStore{})
	pb.RegisterFormalVerificationServiceServer(grpcSrv, srv)

	go func() {
		_ = grpcSrv.Serve(lis)
	}()

	conn, err := grpc.Dial("unix://"+socketPath, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("failed to dial unix socket: %v", err)
	}
	defer conn.Close()

	client := pb.NewFormalVerificationServiceClient(conn)
	_, err = client.Metrics(context.Background(), &pb.MetricsRequest{})
	if err != nil {
		t.Fatalf("client Metrics() error: %v", err)
	}
}
