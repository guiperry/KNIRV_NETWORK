package device

import (
	"crypto/sha256"
	"sync"
	"testing"
	"time"
)

func TestComputeHashDirectASICNoUSBDevice(t *testing.T) {
	d := &Device{
		usbDevice:     nil,
		useDirectASIC: true,
		useUSB:        false,
		mu:            sync.RWMutex{},
	}
	_, err := d.computeHashDirectASIC(make([]byte, 80))
	if err == nil {
		t.Fatal("expected error when usbDevice is nil, got nil")
	}
}

func TestComputeHashDirectASICWithFileNoUSB(t *testing.T) {
	d := &Device{
		usbDevice:     nil,
		useDirectASIC: true,
		useUSB:        false,
		mu:            sync.RWMutex{},
	}
	_, err := d.computeHashDirectASIC(make([]byte, 80))
	if err == nil {
		t.Fatal("expected error when no USB device and no file, got nil")
	}
}

func TestComputeHashNonDirectModeFallsThroughToSoftware(t *testing.T) {
	d := &Device{
		useDirectASIC: false,
		stats:         &DeviceStats{},
		mu:            sync.RWMutex{},
	}
	data := []byte("test data for software path")
	hash, err := d.ComputeHash(data)
	if err != nil {
		t.Fatalf("ComputeHash: %v", err)
	}
	expected := sha256.Sum256(data)
	if hash != expected {
		t.Errorf("hash mismatch: got %x, want %x", hash, expected)
	}
}

func TestComputeHashDirectModeNon80BytesFallsThroughToSoftware(t *testing.T) {
	d := &Device{
		useDirectASIC: true,
		usbDevice:     nil,
		stats:         &DeviceStats{},
		mu:            sync.RWMutex{},
	}
	data := []byte("not exactly 80 bytes")
	hash, err := d.ComputeHash(data)
	if err != nil {
		t.Fatalf("ComputeHash with non-80 bytes in direct mode should fall through to software, got error: %v", err)
	}
	expected := sha256.Sum256(data)
	if hash != expected {
		t.Errorf("hash mismatch: got %x, want %x", hash, expected)
	}
}

func TestDirectASICWorkIDCounterSequential(t *testing.T) {
	d := &Device{
		mu: sync.RWMutex{},
	}

	id0 := d.directASICWorkID()
	id1 := d.directASICWorkID()
	id2 := d.directASICWorkID()

	if id0 != 0 {
		t.Errorf("first workID = %d, want 0", id0)
	}
	if id1 != 1 {
		t.Errorf("second workID = %d, want 1", id1)
	}
	if id2 != 2 {
		t.Errorf("third workID = %d, want 2", id2)
	}
}

func TestDirectASICWorkIDCounterWrapsUint8(t *testing.T) {
	d := &Device{
		workIDCounter: 254,
		mu:            sync.RWMutex{},
	}

	id0 := d.directASICWorkID()
	id1 := d.directASICWorkID()
	id2 := d.directASICWorkID()

	if id0 != 254 {
		t.Errorf("first workID = %d, want 254", id0)
	}
	if id1 != 255 {
		t.Errorf("second workID = %d, want 255", id1)
	}
	if id2 != 0 {
		t.Errorf("third workID = %d, want 0 (wrapped)", id2)
	}
}

func TestPollForDirectNonceNoDeviceInterface(t *testing.T) {
	d := &Device{
		usbDevice: nil,
		file:      nil,
		mu:        sync.RWMutex{},
	}

	_, err := d.pollForDirectNonce(50 * time.Millisecond)
	if err == nil {
		t.Fatal("expected error when no device interface is available, got nil")
	}
}

func TestUSBTxConfigFanPWMOnlyChangesFanFieldAndCRC(t *testing.T) {
	usb := &USBDevice{}
	off := usb.buildTxConfigPacket(0)
	on := usb.buildTxConfigPacket(directASICFanPWM)

	if len(off) != 28 || len(on) != 28 {
		t.Fatalf("TxConfig length = off:%d on:%d, want 28", len(off), len(on))
	}
	if off[10] != 0 {
		t.Errorf("off TxConfig fan PWM = 0x%02x, want 0", off[10])
	}
	if on[10] != directASICFanPWM {
		t.Errorf("on TxConfig fan PWM = 0x%02x, want 0x%02x", on[10], directASICFanPWM)
	}
	if on[4]&0x02 == 0 {
		t.Errorf("on TxConfig must retain the fan-control-effective flag: 0x%02x", on[4])
	}
	for i := range on {
		if i == 10 || i == 26 || i == 27 {
			continue
		}
		if off[i] != on[i] {
			t.Errorf("TxConfig byte %d changed from 0x%02x to 0x%02x", i, off[i], on[i])
		}
	}
	if got, want := CalculateCRC16(on[:26]), uint16(on[26])|uint16(on[27])<<8; got != want {
		t.Errorf("on TxConfig CRC = 0x%04x, want 0x%04x", want, got)
	}
	if got, want := CalculateCRC16(off[:26]), uint16(off[26])|uint16(off[27])<<8; got != want {
		t.Errorf("off TxConfig CRC = 0x%04x, want 0x%04x", want, got)
	}
}
