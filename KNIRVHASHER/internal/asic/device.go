package asic

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "hasher/internal/proto/pixie/v1"
)

const (
	// Pixie server address on ASIC device
	PIXIE_SERVER_ADDRESS = "192.168.1.99:50051"
)

type ASICDevice struct {
	pixieClient pb.PixieServiceClient
	pixieConn   *grpc.ClientConn
	chipCount   int
	frequency   int
}

// NewASICDevice creates a new ASIC driver that connects to the pixie-server
func NewASICDevice() (*ASICDevice, error) {
	d := &ASICDevice{}

	// Connect to pixie-server
	if err := d.connectPixie(); err != nil {
		return nil, fmt.Errorf("failed to connect to pixie-server: %w", err)
	}

	return d, nil
}

// connectPixie establishes a gRPC connection to pixie-server
func (d *ASICDevice) connectPixie() error {
	conn, err := grpc.Dial(PIXIE_SERVER_ADDRESS, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	d.pixieConn = conn
	d.pixieClient = pb.NewPixieServiceClient(conn)

	// Verify connection is working
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deviceInfo, err := d.pixieClient.GetDeviceInfo(ctx, &pb.GetDeviceInfoRequest{})
	if err != nil {
		conn.Close()
		return err
	}

	d.chipCount = int(deviceInfo.ChipCount)

	return nil
}

// Close the device
func (d *ASICDevice) Close() error {
	if d.pixieConn != nil {
		return d.pixieConn.Close()
	}
	return nil
}

// ComputeLayer sends a batch of hash computations for a single network layer
// to the ASIC using the pixie-driver and returns the results.
func (d *ASICDevice) ComputeLayer(input []byte, seeds [][32]byte) []byte {
	numNeurons := len(seeds)
	results := make([]byte, numNeurons*32)

	// Prepare all computation jobs for the layer
	jobs := make([][]byte, numNeurons)
	for i := 0; i < numNeurons; i++ {
		jobs[i] = append(input, seeds[i][:]...)
	}

	// Use pixie-driver (gRPC) for computation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := d.pixieClient.ComputeBatch(ctx, &pb.ComputeBatchRequest{
		Data:         jobs,
		MaxBatchSize: 32,
	})
	if err != nil {
		fmt.Printf("ComputeBatch failed: %v", err)
		return nil
	}

	for i, hash := range resp.Hashes {
		copy(results[i*32:(i+1)*32], hash)
	}

	return results
}

// ComputeHash computes a single hash using the pixie-driver
func (d *ASICDevice) ComputeHash(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := d.pixieClient.ComputeHash(ctx, &pb.ComputeHashRequest{
		Data: data,
	})
	if err != nil {
		return nil, err
	}

	return resp.Hash, nil
}

// GetMetrics returns performance metrics from the ASIC driver
func (d *ASICDevice) GetMetrics() (*pb.GetMetricsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return d.pixieClient.GetMetrics(ctx, &pb.GetMetricsRequest{})
}

// GetInfo retrieves device information
func (d *ASICDevice) GetInfo() (*pb.GetDeviceInfoResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return d.pixieClient.GetDeviceInfo(ctx, &pb.GetDeviceInfoRequest{})
}
