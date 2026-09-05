package asic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log" // Added for debugging output
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "knirvhasher/internal/proto/hasher/v1"
)

const (
	// DefaultASICServerAddress is the default gRPC server address for hasher-server
	DefaultASICServerAddress = "localhost:8888"
	// ConnectionTimeout for initial gRPC connection
	ConnectionTimeout = 5 * time.Second
	// ComputeHealthCheckTimeout bounds the real ComputeHash probe Connect()
	// makes below, separately from ConnectionTimeout. On a CGMiner-backed
	// server this round-trips through actual hardware over a real Stratum
	// pool connection (see internal/driver/device/stratum_server.go) -
	// and, observed in practice, that can take several seconds when
	// CGMiner is balancing hash time across multiple configured pools
	// (e.g. an existing higher-priority pool plus KNIRVHASHER's own),
	// not the sub-second round trip a bare gRPC call would suggest.
	// ConnectionTimeout alone (5s, shared with GetDeviceInfo before this
	// probe even starts) was cutting it off mid-flight.
	ComputeHealthCheckTimeout = 25 * time.Second
	// OperationTimeout for individual hash operations
	OperationTimeout        = 30 * time.Second
	legacyHasherServiceName = "hasher.v1.hasherService"
)

// ASICClient provides hash computation using ASIC hardware with software fallback
type ASICClient struct {
	client            pb.HasherServiceClient
	conn              *grpc.ClientConn
	useFallback       bool // true = use software SHA-256 instead of ASIC
	wasConnected      bool // true if we ever successfully connected (prevents silent fallback after connection)
	allowSoftFallback bool // if true, fall back to software on operation errors (default: false after successful connection)
	// fallbackOnConnectionLoss is enabled by direct mode. A direct-mode RPC is
	// the ASIC itself, so a transport failure means the simulator must continue
	// in software rather than leaving the caller with a dead hash method.
	fallbackOnConnectionLoss bool
	address                  string // gRPC server address
	chipCount                int    // Number of ASIC chips (from device info)
	lastConnectErr           error  // retained so callers can report why hardware was rejected
	legacyProtocol           bool   // pre-2026 service name used a lowercase hasherService
	mu                       sync.RWMutex
}

// NewASICClient creates a new ASICClient with the specified server address
// It attempts to connect to the ASIC server and automatically falls back
// to software SHA-256 if the connection fails
func NewASICClient(address string) (*ASICClient, error) {
	if address == "" {
		address = DefaultASICServerAddress
	}

	c := &ASICClient{
		address:           address,
		useFallback:       false,
		wasConnected:      false,
		allowSoftFallback: true, // Allow fallback during initial connection
	}

	// Try to connect to ASIC server
	if err := c.Connect(); err != nil {
		// Fall back to software mode (only acceptable during initial connection)
		c.useFallback = true
		c.wasConnected = false
		return c, nil // Return successfully in fallback mode
	}

	// Connection succeeded - disable automatic soft fallback for operations
	c.wasConnected = true
	c.allowSoftFallback = false // After successful connection, don't silently fall back

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
		c.lastConnectErr = fmt.Errorf("failed to connect to hasher-server at %s: %w", c.address, err)
		return c.lastConnectErr
	}

	c.conn = conn
	c.client = pb.NewHasherServiceClient(conn)

	// Verify connection by getting device info
	infoCtx, infoCancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer infoCancel()

	info, err := c.client.GetDeviceInfo(infoCtx, &pb.GetDeviceInfoRequest{})
	if status.Code(err) == codes.Unimplemented {
		// Older deployed hasher-server binaries used a lowercase service name.
		// The protobuf messages and RPC methods are compatible, so retry only the
		// service route rather than rejecting otherwise healthy ASIC hardware.
		c.legacyProtocol = true
		info = new(pb.GetDeviceInfoResponse)
		err = c.invoke(infoCtx, "GetDeviceInfo", &pb.GetDeviceInfoRequest{}, info)
		if err == nil {
			log.Printf("ASIC at %s uses legacy gRPC service name %s; enabling compatibility mode", c.address, legacyHasherServiceName)
		}
	}
	if err != nil {
		conn.Close()
		c.useFallback = true
		c.lastConnectErr = fmt.Errorf("GetDeviceInfo RPC at %s failed: %w", c.address, err)
		return c.lastConnectErr
	}

	// Check if the ASIC device is actually operational
	if !info.IsOperational {
		conn.Close()
		c.useFallback = true
		c.lastConnectErr = fmt.Errorf("hasher-server at %s reports ASIC non-operational (device=%q chips=%d firmware=%q)", c.address, info.DevicePath, info.ChipCount, info.FirmwareVersion)
		return c.lastConnectErr
	}

	// Device metadata alone is not sufficient: a CGMiner-backed server can
	// report healthy chips while its direct hash interface is unavailable. Verify
	// the operation the training pipeline will actually use before accepting the
	// device as hardware-capable.
	probeCtx, probeCancel := context.WithTimeout(context.Background(), ComputeHealthCheckTimeout)
	defer probeCancel()

	probeInput := []byte("knirv-asic-compute-healthcheck")
	probeResponse := new(pb.ComputeHashResponse)
	if err := c.invoke(probeCtx, "ComputeHash", &pb.ComputeHashRequest{Data: probeInput}, probeResponse); err != nil {
		conn.Close()
		c.useFallback = true
		c.lastConnectErr = fmt.Errorf("ComputeHash health check at %s failed: %w", c.address, err)
		return c.lastConnectErr
	}
	probeExpected := sha256.Sum256(probeInput)
	if !bytes.Equal(probeResponse.Hash, probeExpected[:]) {
		conn.Close()
		c.useFallback = true
		c.lastConnectErr = fmt.Errorf("ComputeHash health check at %s returned an invalid digest", c.address)
		return c.lastConnectErr
	}

	c.chipCount = int(info.ChipCount)
	c.useFallback = false
	c.wasConnected = true
	c.allowSoftFallback = false // Once connected, don't allow silent fallback
	c.lastConnectErr = nil

	return nil
}

func (c *ASICClient) invoke(ctx context.Context, method string, in, out interface{}, opts ...grpc.CallOption) error {
	if c.legacyProtocol {
		return c.conn.Invoke(ctx, "/"+legacyHasherServiceName+"/"+method, in, out, opts...)
	}
	return c.conn.Invoke(ctx, "/hasher.v1.HasherService/"+method, in, out, opts...)
}

func (c *ASICClient) streamCompute(ctx context.Context, opts ...grpc.CallOption) (pb.HasherService_StreamComputeClient, error) {
	if !c.legacyProtocol {
		return c.client.StreamCompute(ctx, opts...)
	}
	stream, err := c.conn.NewStream(ctx, &pb.HasherService_ServiceDesc.Streams[0], "/"+legacyHasherServiceName+"/StreamCompute", opts...)
	if err != nil {
		return nil, err
	}
	return &legacyStreamComputeClient{ClientStream: stream}, nil
}

type legacyStreamComputeClient struct {
	grpc.ClientStream
}

func (c *legacyStreamComputeClient) Send(request *pb.StreamComputeRequest) error {
	return c.ClientStream.SendMsg(request)
}

func (c *legacyStreamComputeClient) Recv() (*pb.StreamComputeResponse, error) {
	response := new(pb.StreamComputeResponse)
	if err := c.ClientStream.RecvMsg(response); err != nil {
		return nil, err
	}
	return response, nil
}

// LastConnectionError returns the health-check failure that put this client
// into software fallback. It is diagnostic-only and never exposes credentials.
func (c *ASICClient) LastConnectionError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastConnectErr
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

// WasEverConnected returns true if the client ever successfully connected to ASIC
func (c *ASICClient) WasEverConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.wasConnected
}

// SetAllowSoftFallback enables or disables software fallback for operations
// When disabled, operation errors will be reported instead of silently using software
func (c *ASICClient) SetAllowSoftFallback(allow bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allowSoftFallback = allow
}

// SetFallbackOnConnectionLoss controls whether an RPC transport failure moves
// this client permanently into software simulation mode. It is intended for
// direct ASIC mode; non-direct callers preserve their existing error behavior.
func (c *ASICClient) SetFallbackOnConnectionLoss(enable bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallbackOnConnectionLoss = enable
}

// fallBackToSoftware atomically retires a failed hardware connection. Closing
// the connection also stops gRPC from retrying a dead transport in the
// background while the simulator is using its local SHA-256 implementation.
func (c *ASICClient) fallBackToSoftware(operation string, cause error) {
	c.mu.Lock()
	if c.useFallback {
		c.mu.Unlock()
		return
	}
	conn := c.conn
	c.conn = nil
	c.client = nil
	c.useFallback = true
	c.allowSoftFallback = true
	c.lastConnectErr = fmt.Errorf("ASIC %s failed; switched to software simulation: %w", operation, cause)
	c.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	log.Printf("ASIC %s failed: %v; switching to SOFTWARE SIMULATION mode", operation, cause)
}

// FallbackToSoftware explicitly retires the hardware connection and switches
// subsequent work to local software simulation.
func (c *ASICClient) FallbackToSoftware(operation string, cause error) {
	c.fallBackToSoftware(operation, cause)
}

// Reconnect attempts to reconnect to the ASIC server
// Returns error if reconnection fails
func (c *ASICClient) Reconnect() error {
	c.mu.Lock()
	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.client = nil
	}
	c.mu.Unlock()

	return c.Connect()
}

// GetChipCount returns the number of ASIC chips (0 if in fallback mode)
func (c *ASICClient) GetChipCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.chipCount
}

// ComputeHash returns the literal SHA-256 of data - always computed
// locally, never over the network, regardless of ASIC/fallback mode.
//
// This is deliberate, not a shortcut: hasher-server's own ComputeHash RPC
// handler (internal/driver/device/controller.go's Device.ComputeHash)
// always computes sha256.Sum256(data) in software too, for the same
// reason - a SHA-256 ASIC can't compute an arbitrary caller-specified hash
// on demand, only search for a nonce satisfying a target. Since the
// server-side answer is always identical to what this client can compute
// itself, round-tripping over the network for it never bought anything but
// latency - and callers that iterate this thousands of times (the seeder
// pipeline's per-candidate evolutionary search, which calls
// ComputeDoubleHash - and therefore this - many thousands of times per
// generation) were paying that latency thousands of times over for no
// benefit. The real, ASIC/CGMiner-backed path is ComputeBatch, untouched
// by this change; this method was never that path regardless of which
// hash-method is selected client-side.
func (c *ASICClient) ComputeHash(data []byte) ([32]byte, error) {
	return c.computeHashSoftware(data), nil
}

// ComputeDoubleHash computes double SHA-256 - see ComputeHash's doc
// comment for why this is always local.
func (c *ASICClient) ComputeDoubleHash(data []byte) ([32]byte, error) {
	first := c.computeHashSoftware(data)
	return c.computeHashSoftware(first[:]), nil
}

// ComputeHashRemote always calls the server's ComputeHash RPC, never
// computing locally. Used only by DirectASICHASHMethod - see its doc
// comment for why this is a distinct method from ComputeHash rather than a
// mode flag on it.
func (c *ASICClient) ComputeHashRemote(data []byte) ([32]byte, error) {
	c.mu.RLock()
	useFallback := c.useFallback
	fallbackOnConnectionLoss := c.fallbackOnConnectionLoss
	c.mu.RUnlock()

	if useFallback {
		return c.computeHashSoftware(data), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	resp := new(pb.ComputeHashResponse)
	if err := c.invoke(ctx, "ComputeHash", &pb.ComputeHashRequest{Data: data}, resp); err != nil {
		if fallbackOnConnectionLoss {
			c.fallBackToSoftware("ComputeHash RPC", err)
			return c.computeHashSoftware(data), nil
		}
		if c.allowSoftFallback || !c.wasConnected {
			return c.computeHashSoftware(data), nil
		}
		return [32]byte{}, fmt.Errorf("ComputeHashRemote RPC failed (no fallback): %w", err)
	}

	var hash [32]byte
	copy(hash[:], resp.Hash)
	return hash, nil
}

// ComputeBatch computes multiple SHA-256 hashes in a batch
func (c *ASICClient) ComputeBatch(data [][]byte) ([][32]byte, error) {
	c.mu.RLock()
	useFallback := c.useFallback
	allowFallback := c.allowSoftFallback
	wasConnected := c.wasConnected
	c.mu.RUnlock()

	if useFallback {
		return c.computeBatchSoftware(data), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	resp := new(pb.ComputeBatchResponse)
	err := c.invoke(ctx, "ComputeBatch", &pb.ComputeBatchRequest{
		Data:         data,
		MaxBatchSize: 256,
	}, resp)
	if err != nil {
		// Only fall back to software if allowed
		if allowFallback || !wasConnected {
			return c.computeBatchSoftware(data), nil
		}
		return nil, fmt.Errorf("ASIC batch operation failed (no fallback): %w", err)
	}

	results := make([][32]byte, len(resp.Hashes))
	for i, hash := range resp.Hashes {
		copy(results[i][:], hash)
	}
	return results, nil
}

// ComputeBatchWitness returns hashes plus hardware nonces when the server is
// direct ASIC capable. Empty nonces identify legacy/software responses.
func (c *ASICClient) ComputeBatchWitness(data [][]byte) ([][32]byte, []uint32, error) {
	if c.IsUsingFallback() {
		// DirectASICHASHMethod implements a *double* hash contract. Preserve it
		// when a direct device is unavailable; ComputeBatch's single-SHA fallback
		// would otherwise make identical headers produce different temporal
		// states solely because hardware was disconnected.
		hashes := make([][32]byte, len(data))
		for i, input := range data {
			first := c.computeHashSoftware(input)
			hashes[i] = c.computeHashSoftware(first[:])
		}
		return hashes, nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()
	resp := new(pb.ComputeBatchResponse)
	if err := c.invoke(ctx, "ComputeBatch", &pb.ComputeBatchRequest{Data: data, MaxBatchSize: uint32(len(data))}, resp); err != nil {
		return nil, nil, err
	}
	h := make([][32]byte, len(resp.Hashes))
	for i, v := range resp.Hashes {
		copy(h[i][:], v)
	}
	return h, resp.Nonces, nil
}

// StreamComputeFunc is a callback function for streaming results
type StreamComputeFunc func(requestID uint64, hash [32]byte, latencyUs uint64)

// StreamCompute performs streaming hash computation for high-throughput scenarios
func (c *ASICClient) StreamCompute(data [][]byte, callback StreamComputeFunc) error {
	c.mu.RLock()
	useFallback := c.useFallback
	allowFallback := c.allowSoftFallback
	wasConnected := c.wasConnected
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

	stream, err := c.streamCompute(ctx)
	if err != nil {
		// Only fall back to software if allowed
		if allowFallback || !wasConnected {
			for i, d := range data {
				start := time.Now()
				hash := c.computeHashSoftware(d)
				latency := time.Since(start).Microseconds()
				callback(uint64(i+1), hash, uint64(latency))
			}
			return nil
		}
		return fmt.Errorf("ASIC stream operation failed (no fallback): %w", err)
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
	c.mu.RUnlock()

	if useFallback {
		return nil, fmt.Errorf("metrics not available in software fallback mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTimeout)
	defer cancel()

	response := new(pb.GetMetricsResponse)
	if err := c.invoke(ctx, "GetMetrics", &pb.GetMetricsRequest{}, response); err != nil {
		return nil, err
	}
	return response, nil
}

// GetDeviceInfo returns device info from the ASIC server (nil if in fallback mode)
func (c *ASICClient) GetDeviceInfo() (*pb.GetDeviceInfoResponse, error) {
	c.mu.RLock()
	useFallback := c.useFallback
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

	response := new(pb.GetDeviceInfoResponse)
	if err := c.invoke(ctx, "GetDeviceInfo", &pb.GetDeviceInfoRequest{}, response); err != nil {
		return nil, err
	}
	return response, nil
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

// MineHeader performs mining on an 80-byte Bitcoin-style header to find the first valid nonce
// The mining uses Difficulty 1 target (nBits = 0x1d00ffff), meaning any hash with sufficient
// leading zeros is valid. At 500 GH/s, this typically finds a nonce in nanoseconds.
//
// This is the core operation for mining-based neural network activation:
// - Same header + same nonce range = same first valid nonce (deterministic)
// - The nonce becomes the activation value for the MiningNeuron
//
// Now offloads to ASIC hardware via gRPC MineWork method.
func (c *ASICClient) MineHeader(header []byte, nonceStart, nonceEnd uint32) (uint32, error) {
	if len(header) != 80 {
		return 0, fmt.Errorf("mining header must be exactly 80 bytes, got %d", len(header))
	}

	c.mu.RLock()
	useFallback := c.useFallback
	c.mu.RUnlock()

	if useFallback {
		return c.mineSoftware(header, nonceStart, nonceEnd)
	}

	// Use ASIC-accelerated mining via gRPC MineWork method
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	resp := new(pb.MineWorkResponse)
	err := c.invoke(ctx, "MineWork", &pb.MineWorkRequest{
		Header:     header,
		NonceStart: nonceStart,
		NonceEnd:   nonceEnd,
	}, resp)
	if err != nil {
		// If ASIC fails, fall back to software if allowed by current configuration (unlikely after successful connection)
		// Or if we were never successfully connected
		if c.allowSoftFallback || !c.wasConnected {
			log.Printf("Warning: ASIC MineWork failed, falling back to software mining: %v", err)
			return c.mineSoftware(header, nonceStart, nonceEnd)
		}
		return 0, fmt.Errorf("ASIC MineWork operation failed (no fallback): %w", err)
	}

	return resp.Nonce, nil
}

// mineSoftware performs software-based mining to find the first valid nonce
// Uses double SHA-256 (Bitcoin's hash function) with Difficulty 1 target
func (c *ASICClient) mineSoftware(header []byte, nonceStart, nonceEnd uint32) (uint32, error) {
	workHeader := make([]byte, 80)
	copy(workHeader, header)

	for nonce := nonceStart; nonce <= nonceEnd; nonce++ {
		// Set nonce in header (bytes 76-79, little-endian)
		workHeader[76] = byte(nonce)
		workHeader[77] = byte(nonce >> 8)
		workHeader[78] = byte(nonce >> 16)
		workHeader[79] = byte(nonce >> 24)

		// Double SHA-256 (Bitcoin mining)
		hash := c.doubleSHA256(workHeader)

		// Check if hash meets Difficulty 1 target
		// For Difficulty 1, the hash must be less than:
		// 0x00000000FFFF0000000000000000000000000000000000000000000000000000
		// Simplified check: first 4 bytes should have enough leading zeros
		if hash[0] == 0 && hash[1] == 0 && hash[2] == 0 && hash[3] < 0x10 {
			return nonce, nil
		}
	}

	// If no valid nonce found in range, return the last one
	// This shouldn't happen at Difficulty 1 with a reasonable range
	return nonceEnd, nil
}

// doubleSHA256 computes SHA256(SHA256(data)) - Bitcoin's hash function
func (c *ASICClient) doubleSHA256(data []byte) [32]byte {
	first := sha256.Sum256(data)
	return sha256.Sum256(first[:])
}

// MineHeaderBatch performs mining on multiple headers in parallel
// Useful for batch inference with MiningNeuron
func (c *ASICClient) MineHeaderBatch(headers [][]byte, nonceStart, nonceEnd uint32) ([]uint32, error) {
	results := make([]uint32, len(headers))
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for i, header := range headers {
		wg.Add(1)
		go func(idx int, h []byte) {
			defer wg.Done()
			nonce, err := c.MineHeader(h, nonceStart, nonceEnd)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}
			results[idx] = nonce
		}(i, header)
	}

	wg.Wait()
	return results, firstErr
}
