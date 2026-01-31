// internal/hasher/asic_client.go
package hasher

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "hasher/internal/proto/hasher/v1"
)

const (
	// DefaultASICServerAddress is the default gRPC server address for hasher-server
	DefaultASICServerAddress = "localhost:50051"
	// ConnectionTimeout for initial gRPC connection
	ConnectionTimeout = 5 * time.Second
	// OperationTimeout for individual hash operations
	OperationTimeout = 30 * time.Second
)

// ASICClient provides hash computation using ASIC hardware with software fallback
type ASICClient struct {
	client      pb.HasherServiceClient
	conn        *grpc.ClientConn
	useFallback bool   // true = use software SHA-256 instead of ASIC
	address     string // gRPC server address
	chipCount   int    // Number of ASIC chips (from device info)
	mu          sync.RWMutex
}

// NewASICClient creates a new ASICClient with the specified server address
// It attempts to connect to the ASIC server and automatically falls back
// to software SHA-256 if the connection fails
func NewASICClient(address string) (*ASICClient, error) {
	if address == "" {
		address = DefaultASICServerAddress
	}

	c := &ASICClient{
		address:     address,
		useFallback: false,
	}

	// Try to connect to ASIC server
	if err := c.Connect(); err != nil {
		// Fall back to software mode
		c.useFallback = true
		return c, nil // Return successfully in fallback mode
	}

	return c, nil
}

// Connect attempts to establish a gRPC connection to the hasher-server
// Sets fallback mode if connection fails
func (c *ASICClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		c.useFallback = true
		return fmt.Errorf("failed to connect to hasher-server at %s: %w", c.address, err)
	}

	c.conn = conn
	c.client = pb.NewHasherServiceClient(conn)

	// Verify connection by getting device info
	infoCtx, infoCancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer infoCancel()

	info, err := c.client.GetDeviceInfo(infoCtx, &pb.GetDeviceInfoRequest{})
	if err != nil {
		conn.Close()
		c.useFallback = true
		return fmt.Errorf("failed to get device info: %w", err)
	}

	c.chipCount = int(info.ChipCount)
	c.useFallback = false

	return nil
}

// IsUsingFallback returns true if the client is using software SHA-256 fallback
func (c *ASICClient) IsUsingFallback() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.useFallback
}

// IsConnected returns true if the client is connected to ASIC hardware
func (c *ASICClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && !c.useFallback
}

// GetChipCount returns the number of ASIC chips (0 if in fallback mode)
func (c *ASICClient) GetChipCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.chipCount
}

// ComputeHash computes a single SHA-256 hash using ASIC or software fallback
func (c *ASICClient) ComputeHash(data []byte) ([32]byte, error) {
	c.mu.RLock()
	useFallback := c.useFallback
	client := c.client
	c.mu.RUnlock()

	if useFallback {
		return c.computeHashSoftware(data), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	resp, err := client.ComputeHash(ctx, &pb.ComputeHashRequest{
		Data: data,
	})
	if err != nil {
		// On error, try software fallback for this operation
		return c.computeHashSoftware(data), nil
	}

	var result [32]byte
	copy(result[:], resp.Hash)
	return result, nil
}

// ComputeBatch computes multiple SHA-256 hashes in a batch
func (c *ASICClient) ComputeBatch(data [][]byte) ([][32]byte, error) {
	c.mu.RLock()
	useFallback := c.useFallback
	client := c.client
	c.mu.RUnlock()

	if useFallback {
		return c.computeBatchSoftware(data), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	resp, err := client.ComputeBatch(ctx, &pb.ComputeBatchRequest{
		Data:         data,
		MaxBatchSize: 256,
	})
	if err != nil {
		// On error, try software fallback
		return c.computeBatchSoftware(data), nil
	}

	results := make([][32]byte, len(resp.Hashes))
	for i, hash := range resp.Hashes {
		copy(results[i][:], hash)
	}
	return results, nil
}

// StreamComputeFunc is a callback function for streaming results
type StreamComputeFunc func(requestID uint64, hash [32]byte, latencyUs uint64)

// StreamCompute performs streaming hash computation for high-throughput scenarios
func (c *ASICClient) StreamCompute(data [][]byte, callback StreamComputeFunc) error {
	c.mu.RLock()
	useFallback := c.useFallback
	client := c.client
	c.mu.RUnlock()

	if useFallback {
		// Software fallback for streaming
		for i, d := range data {
			start := time.Now()
			hash := c.computeHashSoftware(d)
			latency := time.Since(start).Microseconds()
			callback(uint64(i+1), hash, uint64(latency))
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout*time.Duration(len(data)))
	defer cancel()

	stream, err := client.StreamCompute(ctx)
	if err != nil {
		// Fall back to software streaming
		for i, d := range data {
			start := time.Now()
			hash := c.computeHashSoftware(d)
			latency := time.Since(start).Microseconds()
			callback(uint64(i+1), hash, uint64(latency))
		}
		return nil
	}

	// Start receiver goroutine
	done := make(chan error, 1)
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				done <- err
				return
			}
			var hash [32]byte
			copy(hash[:], resp.Hash)
			callback(resp.RequestId, hash, resp.LatencyUs)
		}
	}()

	// Send all requests
	for i, d := range data {
		err := stream.Send(&pb.StreamComputeRequest{
			Data:      d,
			RequestId: uint64(i + 1),
		})
		if err != nil {
			stream.CloseSend()
			return fmt.Errorf("stream send failed: %w", err)
		}
	}

	stream.CloseSend()
	<-done

	return nil
}

// GetMetrics returns metrics from the ASIC server (nil if in fallback mode)
func (c *ASICClient) GetMetrics() (*pb.GetMetricsResponse, error) {
	c.mu.RLock()
	useFallback := c.useFallback
	client := c.client
	c.mu.RUnlock()

	if useFallback {
		return nil, fmt.Errorf("metrics not available in software fallback mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	return client.GetMetrics(ctx, &pb.GetMetricsRequest{})
}

// GetDeviceInfo returns device info from the ASIC server (nil if in fallback mode)
func (c *ASICClient) GetDeviceInfo() (*pb.GetDeviceInfoResponse, error) {
	c.mu.RLock()
	useFallback := c.useFallback
	client := c.client
	c.mu.RUnlock()

	if useFallback {
		return &pb.GetDeviceInfoResponse{
			DevicePath:      "software",
			ChipCount:       0,
			FirmwareVersion: "software-fallback",
			IsOperational:   true,
			UptimeSeconds:   0,
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	return client.GetDeviceInfo(ctx, &pb.GetDeviceInfoRequest{})
}

// Close closes the gRPC connection
func (c *ASICClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.client = nil
		return err
	}
	return nil
}

// computeHashSoftware computes SHA-256 using software (fallback mode)
func (c *ASICClient) computeHashSoftware(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// computeBatchSoftware computes multiple SHA-256 hashes using software
func (c *ASICClient) computeBatchSoftware(data [][]byte) [][32]byte {
	results := make([][32]byte, len(data))
	for i, d := range data {
		results[i] = sha256.Sum256(d)
	}
	return results
}
