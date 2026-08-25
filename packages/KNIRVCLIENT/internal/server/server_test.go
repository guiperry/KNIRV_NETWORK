package server

import (
	"errors"
	"net"
	"syscall"
	"testing"
)

func TestListenOnAvailablePortUsesOSSelectedPort(t *testing.T) {
	listener, err := listenOnAvailablePort(0)
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("network listeners are not permitted in this test environment: %v", err)
		}
		t.Fatalf("listenOnAvailablePort(0): %v", err)
	}
	defer listener.Close()

	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.IP.String() != "127.0.0.1" || address.Port == 0 {
		t.Fatalf("listener address = %v, want a loopback listener with a selected port", listener.Addr())
	}
}

func TestListenOnAvailablePortFallsBackWhenPreferredPortIsBusy(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("network listeners are not permitted in this test environment: %v", err)
		}
		t.Fatalf("reserve preferred port: %v", err)
	}
	defer busy.Close()

	preferredPort := busy.Addr().(*net.TCPAddr).Port
	listener, err := listenOnAvailablePort(preferredPort)
	if err != nil {
		t.Fatalf("listenOnAvailablePort(%d): %v", preferredPort, err)
	}
	defer listener.Close()

	if actualPort := listener.Addr().(*net.TCPAddr).Port; actualPort == preferredPort {
		t.Fatalf("selected busy port %d", actualPort)
	}
}
