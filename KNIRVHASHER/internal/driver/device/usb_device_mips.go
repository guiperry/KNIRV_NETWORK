//go:build mips || mipsle
// +build mips mipsle

// internal/driver/device/usb_device_mips.go
// Stub implementation for MIPS builds (gousb not available)
// On MIPS, we use the kernel device file directly instead of USB

package device

import (
	"fmt"
	"time"
)

// USBDevice stub - not available on MIPS
type USBDevice struct {
	chipCount int
}

// OpenUSBDevice returns an error on MIPS - USB access not supported
func OpenUSBDevice() (*USBDevice, error) {
	return nil, fmt.Errorf("USB device access not available on MIPS (use kernel device)")
}

// Close stub
func (d *USBDevice) Close() error {
	return nil
}

// SendPacket stub
func (d *USBDevice) SendPacket(data []byte) error {
	return fmt.Errorf("USB not available on MIPS")
}

// ReadPacket stub
func (d *USBDevice) ReadPacket(buffer []byte, timeout time.Duration) (int, error) {
	return 0, fmt.Errorf("USB not available on MIPS")
}

// GetChipCount stub
func (d *USBDevice) GetChipCount() int {
	return 0
}

// Initialize stub
func (d *USBDevice) Initialize() error {
	return fmt.Errorf("USB not available on MIPS")
}

// IsUSBDeviceAvailable returns false on MIPS
func IsUSBDeviceAvailable() bool {
	return false
}
