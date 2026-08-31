package simulator

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"knirvhasher/pkg/hashing/core"
	"knirvhasher/pkg/hashing/methods/asic"
	"knirvhasher/pkg/hashing/methods/cuda"
	"knirvhasher/pkg/hashing/methods/software"
)

// HasherWrapper implements the HashSimulator interface using hasher's HashMethod
// This replaces the internal vhasher simulator with the production-grade hasher module
type HasherWrapper struct {
	hashMethod core.HashMethod
	config     *SimulatorConfig
	cache      map[string]uint32
	cacheMutex sync.RWMutex
	isRunning  bool
	mutex      sync.RWMutex
	methodType string // "auto", "asic", "software", "cuda"
}

// NewHasherWrapper creates a new HashSimulator wrapper using hasher's HashMethod
// It defaults to software implementation if no hardware is available
func NewHasherWrapper(config *SimulatorConfig) *HasherWrapper {
	if config == nil {
		config = &SimulatorConfig{
			DeviceType:     "software",
			MaxConcurrency: 100,
			TargetHashRate: 500000000,
			CacheSize:      10000,
			GPUDevice:      0,
			Timeout:        30,
		}
	}

	return &HasherWrapper{
		config:    config,
		cache:     make(map[string]uint32),
		isRunning: false,
	}
}

// Initialize sets up the HashMethod based on configuration
func (h *HasherWrapper) Initialize(config *SimulatorConfig) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if config != nil {
		h.config = config
		h.methodType = methodTypeForDevice(config.DeviceType)
	}

	// Initialize hash method based on type. DEVICE_IP is supplied to the
	// KNIRVHASHER subprocess from root.key by KNIRVSERVER; auto mode must prefer
	// that attached ASIC before considering CUDA or software.
	if h.hashMethod == nil {
		var err error
		h.hashMethod, err = h.createHashMethod()
		if err != nil {
			return fmt.Errorf("failed to create hash method: %w", err)
		}

		if err := h.hashMethod.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize hash method: %w", err)
		}
		fmt.Printf("[SIM] Using hash method: %s\n", h.hashMethod.Name())
	}

	h.isRunning = true
	return nil
}

func methodTypeForDevice(deviceType string) string {
	deviceType = strings.ToLower(deviceType)
	switch {
	case strings.Contains(deviceType, "direct"):
		return "direct"
	case strings.Contains(deviceType, "software"):
		return "software"
	case strings.Contains(deviceType, "asic"):
		return "asic"
	case strings.Contains(deviceType, "cuda"):
		return "cuda"
	default:
		return "auto"
	}
}

// createHashMethod creates the appropriate hash method based on configuration
func (h *HasherWrapper) createHashMethod() (core.HashMethod, error) {
	switch h.methodType {
	case "direct":
		return h.createDirectASICMethod()

	case "asic":
		return h.createASICMethod()

	case "cuda":
		// Directly create CUDA method
		cudaMethod := cuda.NewCudaMethod()
		if cudaMethod.IsAvailable() {
			return cudaMethod, nil
		}
		// Fall back to software if CUDA not available
		return software.NewSoftwareMethod(), nil

	case "software":
		return software.NewSoftwareMethod(), nil

	case "auto":
		if address := asicAddressFromEnv(); address != "" {
			method, err := h.createASICMethod()
			if err == nil && method.IsAvailable() {
				return method, nil
			}
			if err != nil {
				fmt.Printf("[SIM] ASIC at %s unavailable; falling back: %v\n", address, err)
			}
		}
		// Auto-detect: ASIC first, then CUDA, then software.
		if h.isCUDAAvailable() {
			cudaMethod := cuda.NewCudaMethod()
			if cudaMethod.IsAvailable() {
				return cudaMethod, nil
			}
		}
		return software.NewSoftwareMethod(), nil

	default:
		return software.NewSoftwareMethod(), nil
	}
}

func (h *HasherWrapper) createASICMethod() (core.HashMethod, error) {
	address := asicAddressFromEnv()
	if address == "" {
		return nil, fmt.Errorf("ASIC requested but DEVICE_IP is not set")
	}
	method := asic.NewASICMethod(address)
	if !method.IsAvailable() {
		if client := method.GetClient(); client != nil {
			if err := client.LastConnectionError(); err != nil {
				return nil, err
			}
		}
		return nil, fmt.Errorf("ASIC at %s is not operational", address)
	}
	return method, nil
}

func (h *HasherWrapper) createDirectASICMethod() (core.HashMethod, error) {
	address := asicAddressFromEnv()
	if address == "" {
		return nil, fmt.Errorf("direct ASIC requested but DEVICE_IP is not set")
	}
	method := asic.NewDirectASICMethod(address)
	if !method.IsAvailable() {
		if client := method.GetClient(); client != nil {
			if err := client.LastConnectionError(); err != nil {
				return nil, err
			}
		}
		return nil, fmt.Errorf("direct ASIC at %s is not operational", address)
	}
	return method, nil
}

func asicAddressFromEnv() string {
	address := strings.TrimSpace(os.Getenv("DEVICE_IP"))
	if address == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(address); err == nil {
		return address
	}
	return net.JoinHostPort(address, "8888")
}

// isCUDAAvailable checks if CUDA is available on the system with a 5-second timeout.
// nvidia-smi can hang indefinitely on systems with flaky GPU drivers, so a timeout
// prevents the data-seeder from blocking at startup.
func (h *HasherWrapper) isCUDAAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "nvidia-smi")
	err := cmd.Run()
	if err != nil {
		return false
	}
	return ctx.Err() == nil
}

// SetHashMethod allows injection of a specific HashMethod (e.g., ASIC, CUDA)
func (h *HasherWrapper) SetHashMethod(method core.HashMethod) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.hashMethod = method
}

// SetMethodType sets the hash method type (auto, asic, software, cuda)
func (h *HasherWrapper) SetMethodType(methodType string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.methodType = methodType
}

// Shutdown cleans up the HashMethod
func (h *HasherWrapper) Shutdown() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.isRunning = false

	if h.hashMethod != nil {
		if err := h.hashMethod.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown hash method: %w", err)
		}
	}

	h.cacheMutex.Lock()
	h.cache = make(map[string]uint32)
	h.cacheMutex.Unlock()

	return nil
}

// SimulateHash performs deterministic hashing using the HashMethod
// Maps to: SHA256(seed || pass) with caching
func (h *HasherWrapper) SimulateHash(seed []byte, pass int) (uint32, error) {
	if !h.isRunning {
		return 0, fmt.Errorf("simulator is not running")
	}

	if h.hashMethod == nil {
		return 0, fmt.Errorf("hash method not initialized")
	}

	// Create cache key
	cacheKey := fmt.Sprintf("%x_%d", seed, pass)

	// Check cache
	h.cacheMutex.RLock()
	if cached, exists := h.cache[cacheKey]; exists {
		h.cacheMutex.RUnlock()
		return cached, nil
	}
	h.cacheMutex.RUnlock()

	// Combine seed with pass for deterministic hashing
	data := make([]byte, len(seed)+4)
	copy(data, seed)
	binary.LittleEndian.PutUint32(data[len(seed):], uint32(pass))

	// Use HashMethod for computation
	hash, err := h.hashMethod.ComputeHash(data)
	if err != nil {
		return 0, fmt.Errorf("hash computation failed: %w", err)
	}

	// Extract first 4 bytes as uint32 result (Big-Endian for simulator compatibility)
	result := binary.BigEndian.Uint32(hash[:4])

	// Cache result
	if len(h.cache) < h.config.CacheSize {
		h.cacheMutex.Lock()
		h.cache[cacheKey] = result
		h.cacheMutex.Unlock()
	}

	return result, nil
}

// SimulateBitcoinHeader returns first 4 bytes of Double-SHA256 as uint32
func (h *HasherWrapper) SimulateBitcoinHeader(header []byte) (uint32, error) {
	if !h.isRunning {
		return 0, fmt.Errorf("simulator is not running")
	}

	if h.hashMethod == nil {
		return 0, fmt.Errorf("hash method not initialized")
	}

	// Use MineHeader with a 0-0 range to just get the hash of the current header
	// or better, if the hashMethod supports a direct double-hash.
	// Most HashMethods implement double-hash internally for MineHeader.
	// But core.HashMethod doesn't have a direct "ComputeDoubleHash" exposed.
	// It has ComputeHash.

	hash1, err := h.hashMethod.ComputeHash(header)
	if err != nil {
		return 0, err
	}

	hash2, err := h.hashMethod.ComputeHash(hash1[:])
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(hash2[:4]), nil
}

// RecursiveMine performs the 21-pass temporal loop with associative jitter
// Returns the full 32-byte double-SHA256 hash of the final pass
func (h *HasherWrapper) RecursiveMine(header []byte, passes int) ([]byte, error) {
	if !h.isRunning {
		return nil, fmt.Errorf("simulator is not running")
	}

	if h.hashMethod == nil {
		return nil, fmt.Errorf("hash method not initialized")
	}

	// Delegate directly to the production-grade recursive mine implementation
	return h.hashMethod.ExecuteRecursiveMine(header, passes)
}

// ValidateSeed checks if seed produces target token in any pass
func (h *HasherWrapper) ValidateSeed(seed []byte, targetToken int32) (bool, error) {
	if !h.isRunning {
		return false, fmt.Errorf("simulator is not running")
	}

	// Check multiple passes
	for pass := 0; pass < 21; pass++ {
		nonce, err := h.SimulateHash(seed, pass)
		if err != nil {
			return false, err
		}

		if nonce == uint32(targetToken) {
			return true, nil
		}
	}

	return false, nil
}

// ProcessHeadersBatch processes multiple Bitcoin headers in parallel
func (h *HasherWrapper) ProcessHeadersBatch(headers [][]byte, targetTokenID uint32) ([]uint32, error) {
	if !h.isRunning {
		return nil, fmt.Errorf("simulator is not running")
	}

	results := make([]uint32, len(headers))
	for i, header := range headers {
		result, err := h.SimulateBitcoinHeader(header)
		if err != nil {
			continue
		}
		results[i] = result
	}

	return results, nil
}

// GetHashMethod returns the underlying HashMethod for advanced usage
func (h *HasherWrapper) GetHashMethod() core.HashMethod {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.hashMethod
}

// ClearCache clears the internal cache
func (h *HasherWrapper) ClearCache() {
	h.cacheMutex.Lock()
	defer h.cacheMutex.Unlock()
	h.cache = make(map[string]uint32)
}

// GetCacheSize returns current cache size
func (h *HasherWrapper) GetCacheSize() int {
	h.cacheMutex.RLock()
	defer h.cacheMutex.RUnlock()
	return len(h.cache)
}
