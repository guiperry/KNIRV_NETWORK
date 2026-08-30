package simulator

import "testing"

func TestASICAddressFromEnv(t *testing.T) {
	t.Setenv("DEVICE_IP", "192.0.2.10")
	if got, want := asicAddressFromEnv(), "192.0.2.10:8888"; got != want {
		t.Fatalf("asicAddressFromEnv() = %q, want %q", got, want)
	}

	t.Setenv("DEVICE_IP", "192.0.2.10:9443")
	if got, want := asicAddressFromEnv(), "192.0.2.10:9443"; got != want {
		t.Fatalf("asicAddressFromEnv() = %q, want %q", got, want)
	}
}

func TestInitializeRecognizesASICDeviceType(t *testing.T) {
	if got, want := methodTypeForDevice("hasher_asic"), "asic"; got != want {
		t.Fatalf("methodTypeForDevice() = %q, want %q", got, want)
	}
}
