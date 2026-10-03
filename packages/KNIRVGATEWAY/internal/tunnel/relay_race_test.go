package tunnel

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestRelay_DoesNotRaceControlSocket is a regression test for the bug where
// TunnelManager.Relay used to io.Copy relay bytes directly onto a dev's
// control socket while control.go's IDENTIFY/PING scanner was still reading
// that same net.Conn concurrently — two unsynchronized readers on one
// socket. It drives a simulated control-plane reader (reading
// newline-delimited JSON, exactly as control.go's handleConnection does)
// concurrently with an external client relaying a payload over a *separate*
// dedicated data connection obtained via RequestRelayDataConn/
// BindRelayDataConn, and asserts both that the control reader only ever
// observes well-formed control messages and that the relayed payload
// (including bytes pipelined immediately after the external client's first
// line — the bufferedConn fix) survives byte-for-byte.
func TestRelay_DoesNotRaceControlSocket(t *testing.T) {
	logger := zap.NewNop()
	tm := NewTunnelManager(logger)

	const devID = "dev-under-test"
	// AddControlSocket hard-asserts socket.RemoteAddr().(*net.TCPAddr), so the
	// control socket must be a real TCP connection pair, not net.Pipe().
	controlListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("control listen: %v", err)
	}
	defer controlListener.Close()
	gatewayControlCh := make(chan net.Conn, 1)
	go func() {
		conn, err := controlListener.Accept()
		if err == nil {
			gatewayControlCh <- conn
		}
	}()
	devSideControl, err := net.Dial("tcp", controlListener.Addr().String())
	if err != nil {
		t.Fatalf("control dial: %v", err)
	}
	defer devSideControl.Close()
	gatewaySideControl := <-gatewayControlCh
	defer gatewaySideControl.Close()
	tm.AddControlSocket(devID, gatewaySideControl)

	relayListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer relayListener.Close()

	publicRelay := &PublicRelayListener{
		port:          relayListener.Addr().(*net.TCPAddr).Port,
		server:        relayListener,
		tunnelManager: tm,
		logger:        logger,
		config:        &Config{HTTPAPIPort: 0},
	}
	go publicRelay.acceptConnections()

	controlErrCh := make(chan error, 8)
	controlDone := make(chan struct{})

	// Simulates control.go's handleConnection read loop: continuously scan
	// newline-delimited JSON off the dev's side of the control socket. Any
	// non-JSON line is exactly what relay bytes leaking onto this socket
	// would look like, and fails the test. On RELAY_REQUEST, dial the public
	// relay port and bind, exactly as knirv-client's tunnel bridge will.
	go func() {
		defer close(controlDone)
		scanner := bufio.NewScanner(devSideControl)
		for scanner.Scan() {
			var msg ControlMessage
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				controlErrCh <- fmt.Errorf("control socket received non-control data (relay bytes leaked onto control plane?): %w (line=%q)", err, scanner.Text())
				return
			}
			if msg.Action != "RELAY_REQUEST" {
				controlErrCh <- fmt.Errorf("unexpected control action %q", msg.Action)
				return
			}
			if msg.RelaySessionToken == "" {
				controlErrCh <- fmt.Errorf("RELAY_REQUEST missing session token")
				return
			}
			dataConn, err := net.Dial("tcp", relayListener.Addr().String())
			if err != nil {
				controlErrCh <- fmt.Errorf("dev data conn dial: %w", err)
				return
			}
			bind, _ := json.Marshal(RelayBindMessage{RelaySessionToken: msg.RelaySessionToken})
			if _, err := dataConn.Write(append(bind, '\n')); err != nil {
				controlErrCh <- fmt.Errorf("dev data conn write: %w", err)
				return
			}
			// Echo everything the gateway relays onto this connection back to
			// it, standing in for a real bridge forwarding to a local server.
			go func() {
				io.Copy(dataConn, dataConn)
				dataConn.Close()
			}()
			return
		}
	}()

	// External client: connect to the public relay port, send the target
	// devId as the first line, then immediately pipeline a large payload in
	// the same write (no waiting for any ack) — this is exactly the pattern
	// that would lose data without the bufferedConn fix.
	externalConn, err := net.Dial("tcp", relayListener.Addr().String())
	if err != nil {
		t.Fatalf("external client dial: %v", err)
	}
	defer externalConn.Close()

	payload := bytes.Repeat([]byte("the-quick-brown-fox-jumps-over-the-lazy-dog-"), 4096) // ~180KB
	firstLine := []byte(fmt.Sprintf(`{"targetPeerId":%q}`, devID) + "\n")
	if _, err := externalConn.Write(append(firstLine, payload...)); err != nil {
		t.Fatalf("external client write: %v", err)
	}

	received := make([]byte, 0, len(payload))
	buf := make([]byte, 4096)
	_ = externalConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for len(received) < len(payload) {
		n, err := externalConn.Read(buf)
		received = append(received, buf[:n]...)
		if err != nil {
			if len(received) < len(payload) {
				t.Fatalf("external client read: %v (got %d/%d bytes)", err, len(received), len(payload))
			}
			break
		}
	}

	if !bytes.Equal(received, payload) {
		t.Fatalf("relayed payload corrupted: got %d bytes, want %d bytes", len(received), len(payload))
	}

	select {
	case err := <-controlErrCh:
		t.Fatalf("control plane error: %v", err)
	case <-controlDone:
		// Control reader observed exactly the RELAY_REQUEST and nothing else
		// — the control socket was never used as a data pipe.
	case <-time.After(2 * time.Second):
		t.Fatal("control reader goroutine did not complete")
	}
}
