// internal/driver/device/cgminer_miner.go
// CGMiner integration for mining via its RPC API
// This provides deterministic nonces through CGMiner's proven implementation

package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const (
	cgminerHost    = "127.0.0.1"
	cgminerPort    = 4028
	cgminerTimeout = 30 * time.Second
)

// CGMinerMiner provides mining via CGMiner's RPC API
type CGMinerMiner struct {
	chipCount int
}

// NewCGMinerMiner creates a new CGMiner miner interface
func NewCGMinerMiner() *CGMinerMiner {
	return &CGMinerMiner{
		chipCount: 32, // Antminer S3
	}
}

// IsAvailable checks if CGMiner is running and responding
func (c *CGMinerMiner) IsAvailable() bool {
	addr := net.JoinHostPort(cgminerHost, fmt.Sprintf("%d", cgminerPort))
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// sendCommand sends a command to CGMiner and returns the response
func (c *CGMinerMiner) sendCommand(command string, params ...interface{}) (map[string]interface{}, error) {
	cmd := map[string]interface{}{
		"command": command,
	}
	if len(params) > 0 {
		cmd["parameter"] = params[0]
	}

	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	addr := net.JoinHostPort(cgminerHost, fmt.Sprintf("%d", cgminerPort))
	conn, err := net.DialTimeout("tcp", addr, cgminerTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cgminer: %w", err)
	}
	defer conn.Close()

	// Send command newline-terminated, not null-terminated - see the
	// matching comment in CGMinerClient.SendCommand (cgminer_client.go) for
	// why: this build's API rejects a null-terminated command as invalid
	// JSON, but the shell-based `echo ... | nc` health check (which
	// terminates with '\n') works against it.
	cmdData := append(cmdJSON, '\n')
	if _, err := conn.Write(cmdData); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	var buf bytes.Buffer
	tmp := make([]byte, 4096)
	for {
		n, err := conn.Read(tmp)
		if err != nil || n == 0 {
			break
		}
		buf.Write(tmp[:n])
	}

	// Parse response (remove null bytes)
	response := buf.Bytes()
	response = bytes.ReplaceAll(response, []byte{0x00}, []byte{})

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetDevices returns device information from CGMiner
func (c *CGMinerMiner) GetDevices() ([]map[string]interface{}, error) {
	resp, err := c.sendCommand("devs")
	if err != nil {
		return nil, err
	}

	devices, ok := resp["DEVS"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	result := make([]map[string]interface{}, len(devices))
	for i, d := range devices {
		if dev, ok := d.(map[string]interface{}); ok {
			result[i] = dev
		}
	}

	return result, nil
}

// GetStats returns mining statistics
func (c *CGMinerMiner) GetStats() (map[string]interface{}, error) {
	return c.sendCommand("summary")
}

// GetChipCount returns the number of chips
func (c *CGMinerMiner) GetChipCount() int {
	return c.chipCount
}

// StartCGMiner starts the CGMiner process (assumed already running by the
// caller today - see ensureCGMinerRunning in cmd/driver/hasher-server).
func StartCGMiner() error {
	// This would start cgminer process
	// For now, assume it's already running
	return nil
}
