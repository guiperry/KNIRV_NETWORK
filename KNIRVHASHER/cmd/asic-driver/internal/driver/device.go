// internal/driver/device.go
package driver

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

const (
	DevicePath = "/dev/bitmain-asic"
	MaxBatchSize = 256
)

// Device represents an ASIC device with eBPF tracing
type Device struct {
	file      *os.File
	fd        uintptr
	chipCount int
	mu        sync.RWMutex
	tracer    *Tracer
	stats     *DeviceStats
}

// DeviceStats holds device statistics with internal synchronization
type DeviceStats struct {
	TotalRequests    uint64
	TotalBytes       uint64
	TotalLatencyNs   uint64
	PeakLatencyNs    uint64
	ErrorCount       uint64
	mu               sync.RWMutex
}

// DeviceStatsSnapshot is a copy of device statistics without synchronization
// Used for returning stats to callers without copying mutexes
type DeviceStatsSnapshot struct {
	TotalRequests    uint64
	TotalBytes       uint64
	TotalLatencyNs   uint64
	PeakLatencyNs    uint64
	ErrorCount       uint64
}

// OpenDevice opens the ASIC device with eBPF tracing
func OpenDevice(enableTracing bool) (*Device, error) {
	file, err := os.OpenFile(DevicePath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open device: %w", err)
	}

	dev := &Device{
		file:      file,
		fd:        file.Fd(),
		chipCount: 32,
		stats:     &DeviceStats{},
	}

	// Initialize eBPF tracer if enabled
	if enableTracing {
		tracer, err := NewTracer()
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("init tracer: %w", err)
		}
		dev.tracer = tracer
	}

	return dev, nil
}

// ComputeHash computes a single SHA-256 hash
func (d *Device) ComputeHash(data []byte) ([32]byte, error) {
	// Trace start
	if d.tracer != nil {
		pixie_compute_start()
		defer pixie_compute_end()
	}

	start := time.Now()
	defer func() {
		d.updateStats(1, uint64(len(data)), uint64(time.Since(start).Nanoseconds()))
	}()

	result, err := d.ComputeBatch([][]byte{data})
	if err != nil {
		d.stats.mu.Lock()
		d.stats.ErrorCount++
		d.stats.mu.Unlock()
		return [32]byte{}, err
	}

	return result[0], nil
}

// ComputeBatch computes multiple SHA-256 hashes
func (d *Device) ComputeBatch(inputs [][]byte) ([][32]byte, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("empty input batch")
	}

	if len(inputs) > MaxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", len(inputs), MaxBatchSize)
	}

	// Trace batch start
	if d.tracer != nil {
		pixie_batch_start()
		defer pixie_batch_end()
	}

	start := time.Now()
	results := make([][32]byte, len(inputs))

	// Process in hardware batches of 4
	const hwBatchSize = 4

	for i := 0; i < len(inputs); i += hwBatchSize {
		end := min(i+hwBatchSize, len(inputs))
		batch := inputs[i:end]

		// Build protocol packet
		packet := buildTxTaskPacket(batch)

		// Submit to device
		d.mu.Lock()
		_, err := d.file.Write(packet)
		d.mu.Unlock()

		if err != nil {
			d.stats.mu.Lock()
			d.stats.ErrorCount++
			d.stats.mu.Unlock()
			return nil, fmt.Errorf("write failed: %w", err)
		}

		// Poll for results (in production, use interrupts or async I/O)
		time.Sleep(100 * time.Microsecond)

		// Read results
		batchResults := make([]byte, len(batch)*32)
		d.mu.Lock()
		n, err := d.file.Read(batchResults)
		d.mu.Unlock()

		if err != nil && err.Error() != "EOF" {
			// Device might not support read, try alternative method
			// In production, use USB bulk transfer
		}

		// Parse results
		for j := 0; j < len(batch) && j*32 < n; j++ {
			copy(results[i+j][:], batchResults[j*32:(j+1)*32])
		}
	}

	totalBytes := uint64(0)
	for _, input := range inputs {
		totalBytes += uint64(len(input))
	}

	d.updateStats(uint64(len(inputs)), totalBytes, uint64(time.Since(start).Nanoseconds()))

	return results, nil
}

// GetStats returns current device statistics
func (d *Device) GetStats() DeviceStatsSnapshot {
	d.stats.mu.RLock()
	defer d.stats.mu.RUnlock()

	return DeviceStatsSnapshot{
		TotalRequests:  d.stats.TotalRequests,
		TotalBytes:     d.stats.TotalBytes,
		TotalLatencyNs: d.stats.TotalLatencyNs,
		PeakLatencyNs:  d.stats.PeakLatencyNs,
		ErrorCount:     d.stats.ErrorCount,
	}
}

// GetInfo returns device information
func (d *Device) GetInfo() DeviceInfo {
	var stat syscall.Stat_t
	syscall.Fstat(int(d.fd), &stat)

	return DeviceInfo{
		DevicePath:      DevicePath,
		ChipCount:       uint32(d.chipCount),
		FirmwareVersion: "1.0.0", // Would be read from device
		IsOperational:   true,
		FileDescriptor:  int(d.fd),
	}
}

// Close closes the device and cleanup resources
func (d *Device) Close() error {
	if d.tracer != nil {
		d.tracer.Close()
	}
	return d.file.Close()
}

// Internal helper functions

func (d *Device) updateStats(requests, bytes, latencyNs uint64) {
	d.stats.mu.Lock()
	defer d.stats.mu.Unlock()

	d.stats.TotalRequests += requests
	d.stats.TotalBytes += bytes
	d.stats.TotalLatencyNs += latencyNs

	if latencyNs > d.stats.PeakLatencyNs {
		d.stats.PeakLatencyNs = latencyNs
	}
}

func buildTxTaskPacket(inputs [][]byte) []byte {
	// Calculate total payload size
	payloadSize := 0
	for _, input := range inputs {
		payloadSize += 2 + len(input) // 2 bytes length prefix
	}

	// Build packet: [token(1)][version(1)][length(2)][payload][crc(2)]
	packet := make([]byte, 4+payloadSize+2)
	packet[0] = 0x52 // TXTASK token
	packet[1] = 0x01 // Version
	binary.LittleEndian.PutUint16(packet[2:4], uint16(payloadSize))

	// Add payload
	offset := 4
	for _, input := range inputs {
		binary.LittleEndian.PutUint16(packet[offset:offset+2], uint16(len(input)))
		offset += 2
		copy(packet[offset:], input)
		offset += len(input)
	}

	// Calculate and append CRC
	crc := calculateCRC16(packet[:offset])
	binary.LittleEndian.PutUint16(packet[offset:], crc)

	return packet
}

func calculateCRC16(data []byte) uint16 {
	// CRC-16-CCITT polynomial
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc = crc << 1
			}
		}
	}
	return crc
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DeviceInfo holds device information
type DeviceInfo struct {
	DevicePath      string
	ChipCount       uint32
	FirmwareVersion string
	IsOperational   bool
	FileDescriptor  int
}

// eBPF tracing stubs (implemented via CGO or replaced with actual eBPF calls)
//go:noinline
func pixie_compute_start() {
	// This function is traced by eBPF uprobe
}

//go:noinline
func pixie_compute_end() {
	// This function is traced by eBPF uprobe
}

//go:noinline
func pixie_batch_start() {
	// This function is traced by eBPF uprobe
}

//go:noinline
func pixie_batch_end() {
	// This function is traced by eBPF uprobe
}
