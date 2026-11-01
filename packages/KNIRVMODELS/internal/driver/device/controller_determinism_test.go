package device

import (
	"crypto/sha256"
	"sync"
	"testing"
)

// ComputeHash is documented to return the literal sha256(data). A SHA-256 ASIC
// cannot do that for arbitrary input: in direct mode it searches an 80-byte
// header for a nonce, so its output depends on hardware state and time. These
// tests pin that factuality determinism (HASHER validate/infer) never depends
// on the direct path unless the jitter engine explicitly opts in.
//
// A Device with no USB handle cannot reach hardware: computeHashDirectASIC
// returns an error. So "no error and exactly sha256" proves the hardware path
// was not entered.

var determinismLengths = []int{0, 1, 79, 80, 81, 4096}

func determinismInput(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i*31 + 7)
	}
	return data
}

func newDirectDevice(jitter bool) *Device {
	return &Device{
		useDirectASIC:   true,
		useDirectJitter: jitter,
		stats:           &DeviceStats{},
		mu:              sync.RWMutex{},
	}
}

func TestComputeHashIsLiteralSHA256InDirectModeWithoutJitter(t *testing.T) {
	d := newDirectDevice(false)
	for _, n := range determinismLengths {
		data := determinismInput(n)
		for run := 0; run < 3; run++ {
			got, err := d.ComputeHash(data)
			if err != nil {
				t.Fatalf("len %d run %d: reached hardware path with jitter off: %v", n, run, err)
			}
			if want := sha256.Sum256(data); got != want {
				t.Fatalf("len %d run %d: ComputeHash = %x, want sha256 %x", n, run, got, want)
			}
		}
	}
}

func TestComputeHashDirectJitterOnlyAffectsExactly80Bytes(t *testing.T) {
	d := newDirectDevice(true)
	for _, n := range determinismLengths {
		data := determinismInput(n)
		got, err := d.ComputeHash(data)
		if n == 80 {
			if err == nil {
				t.Fatal("len 80 with jitter on must take the hardware path (which errors without a USB handle)")
			}
			continue
		}
		if err != nil {
			t.Fatalf("len %d: non-header input reached hardware path: %v", n, err)
		}
		if want := sha256.Sum256(data); got != want {
			t.Fatalf("len %d: ComputeHash = %x, want sha256 %x", n, got, want)
		}
	}
}

func TestComputeHashMatchesWithDirectOnAndOff(t *testing.T) {
	off := &Device{stats: &DeviceStats{}, mu: sync.RWMutex{}}
	on := newDirectDevice(false)
	for _, n := range determinismLengths {
		data := determinismInput(n)
		a, errA := off.ComputeHash(data)
		b, errB := on.ComputeHash(data)
		if errA != nil || errB != nil {
			t.Fatalf("len %d: errors off=%v on=%v", n, errA, errB)
		}
		if a != b {
			t.Fatalf("len %d: direct on/off disagree: %x vs %x", n, a, b)
		}
	}
}
