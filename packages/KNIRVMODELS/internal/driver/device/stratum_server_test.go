package device

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"
)

// TestWaitForSubscribedClientTimesOutWithNoClient guards against the exact
// regression this method exists to prevent: a caller treating the pool as
// ready before CGMiner has actually connected. With nobody ever connecting,
// it must return an error, not nil.
func TestWaitForSubscribedClientTimesOutWithNoClient(t *testing.T) {
	srv, err := NewStratumServer("127.0.0.1", 0, "testuser", "testpass")
	if err != nil {
		t.Fatalf("NewStratumServer: %v", err)
	}
	defer srv.Close()

	err = srv.WaitForSubscribedClient(500 * time.Millisecond)
	if err == nil {
		t.Fatal("expected a timeout error with no client ever connecting, got nil")
	}
}

// TestWaitForSubscribedClientReturnsOnceSubscribed confirms it unblocks
// promptly once a real client subscribes, rather than always sleeping out
// the full timeout.
func TestWaitForSubscribedClientReturnsOnceSubscribed(t *testing.T) {
	srv, err := NewStratumServer("127.0.0.1", 0, "testuser", "testpass")
	if err != nil {
		t.Fatalf("NewStratumServer: %v", err)
	}
	defer srv.Close()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", srv.Port()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	subscribeMsg, _ := json.Marshal(map[string]interface{}{
		"id": 1, "method": "mining.subscribe", "params": []interface{}{"test-client/1.0"},
	})
	if _, err := conn.Write(append(subscribeMsg, '\n')); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}

	start := time.Now()
	if err := srv.WaitForSubscribedClient(5 * time.Second); err != nil {
		t.Fatalf("WaitForSubscribedClient: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("WaitForSubscribedClient took %s to notice an already-subscribed client - should return promptly", elapsed)
	}
}

// TestStratumServerRealMiningRoundTrip drives StratumServer over a real TCP
// loopback connection using an independent, from-scratch Stratum V1 client
// implementation - deliberately NOT reusing swapWord/swapWords32/sha256d
// from stratum_server.go - so the test exercises the wire protocol the way
// CGMiner would, rather than re-confirming this file's own helpers against
// themselves. It catches any asymmetry between what the server sends in
// mining.notify and what it expects back in mining.submit.
func TestStratumServerRealMiningRoundTrip(t *testing.T) {
	// Lower the share-difficulty target for this test only, so a passing
	// nonce is found in a handful of iterations instead of ~2^32 on
	// average (real difficulty-1 mining speed depends on real ASIC
	// hardware, not this test's CPU).
	origTarget := diff1Target
	diff1Target = new(big.Int).Lsh(big.NewInt(1), 250)
	defer func() { diff1Target = origTarget }()

	srv, err := NewStratumServer("127.0.0.1", 0, "testuser", "testpass")
	if err != nil {
		t.Fatalf("NewStratumServer: %v", err)
	}
	defer srv.Close()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", srv.Port()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	send := func(v interface{}) {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		b = append(b, '\n')
		if _, err := conn.Write(b); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	recvLine := func() map[string]interface{} {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		line, err := r.ReadBytes('\n')
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]interface{}
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("unmarshal %s: %v", line, err)
		}
		return m
	}

	// mining.subscribe
	send(map[string]interface{}{"id": 1, "method": "mining.subscribe", "params": []interface{}{"test-client/1.0"}})
	subResp := recvLine()
	subResult, _ := subResp["result"].([]interface{})
	if len(subResult) != 3 {
		t.Fatalf("unexpected subscribe result: %#v", subResp)
	}
	extranonce1Hex, _ := subResult[1].(string)
	extranonce2Size := int(subResult[2].(float64))
	extranonce1, err := hex.DecodeString(extranonce1Hex)
	if err != nil || len(extranonce1) != 4 {
		t.Fatalf("bad extranonce1: %q err=%v", extranonce1Hex, err)
	}

	// The server pushes mining.set_difficulty synchronously right after the
	// subscribe response, before we ever send mining.authorize - so it's
	// the next line on the wire, not the authorize response.
	diffNotif := recvLine()
	if diffNotif["method"] != "mining.set_difficulty" {
		t.Fatalf("expected mining.set_difficulty next, got %#v", diffNotif)
	}

	// mining.authorize
	send(map[string]interface{}{"id": 2, "method": "mining.authorize", "params": []interface{}{"testuser", "testpass"}})
	authResp := recvLine()
	if ok, _ := authResp["result"].(bool); !ok {
		t.Fatalf("authorize failed: %#v", authResp)
	}

	// Publish a job from the server side, concurrently with reading notify.
	job := srv.NewJob([]byte("hello world"), 7)
	notifyCh := make(chan map[string]interface{}, 1)
	go func() { notifyCh <- recvLine() }()

	if err := srv.SubmitJob(job); err != nil {
		t.Fatalf("SubmitJob: %v", err)
	}

	var notify map[string]interface{}
	select {
	case notify = <-notifyCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for mining.notify")
	}
	if notify["method"] != "mining.notify" {
		t.Fatalf("expected mining.notify, got %#v", notify)
	}

	params, _ := notify["params"].([]interface{})
	if len(params) != 9 {
		t.Fatalf("expected 9 mining.notify params, got %d: %#v", len(params), params)
	}
	jobID, _ := params[0].(string)
	prevHashWireHex, _ := params[1].(string)
	coinb1Hex, _ := params[2].(string)
	coinb2Hex, _ := params[3].(string)
	versionWireHex, _ := params[5].(string)
	nbitsWireHex, _ := params[6].(string)
	ntimeWireHex, _ := params[7].(string)

	if jobID != job.id {
		t.Fatalf("job id mismatch: got %q want %q", jobID, job.id)
	}

	// --- Independent client-side reconstruction of the header, matching
	// the canonical SHA256d Stratum V1 dialect (cross-checked against the
	// nightminer.py reference client), but written separately from
	// stratum_server.go's swapWord/swapWords32/sha256d so a bug shared by
	// both implementations wouldn't be masked. ---
	reverse4 := func(b []byte) []byte {
		return []byte{b[3], b[2], b[1], b[0]}
	}
	reverseWords := func(b []byte) []byte {
		out := make([]byte, len(b))
		for w := 0; w < len(b)/4; w++ {
			copy(out[w*4:w*4+4], reverse4(b[w*4:w*4+4]))
		}
		return out
	}
	mustHex := func(s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatalf("hex decode %q: %v", s, err)
		}
		return b
	}
	doubleSHA256 := func(b []byte) []byte {
		h1 := sha256.Sum256(b)
		h2 := sha256.Sum256(h1[:])
		return h2[:]
	}

	versionHdr := reverse4(mustHex(versionWireHex))
	prevHashHdr := reverseWords(mustHex(prevHashWireHex))
	nbitsHdr := reverse4(mustHex(nbitsWireHex))
	ntimeHdr := reverse4(mustHex(ntimeWireHex))
	coinb1 := mustHex(coinb1Hex)
	coinb2 := mustHex(coinb2Hex)

	extranonce2 := make([]byte, extranonce2Size) // all-zero is a valid choice

	coinbase := append(append(append(append([]byte{}, coinb1...), extranonce1...), extranonce2...), coinb2...)
	merkleRoot := doubleSHA256(coinbase)

	header := make([]byte, 80)
	copy(header[0:4], versionHdr)
	copy(header[4:36], prevHashHdr)
	copy(header[36:68], merkleRoot)
	copy(header[68:72], ntimeHdr)
	copy(header[72:76], nbitsHdr)

	// Brute-force a nonce satisfying the (test-lowered) target using our
	// own independently-implemented header + double-SHA256, then submit it
	// and confirm the server's own independent verification agrees.
	var foundNonce uint32
	var foundHash []byte
	found := false
	for n := uint32(0); n < 5_000_000; n++ {
		binary.LittleEndian.PutUint32(header[76:80], n)
		h := doubleSHA256(header)
		rev := make([]byte, 32)
		for i := 0; i < 32; i++ {
			rev[i] = h[31-i]
		}
		if new(big.Int).SetBytes(rev).Cmp(diff1Target) <= 0 {
			foundNonce = n
			foundHash = h
			found = true
			break
		}
	}
	if !found {
		t.Fatal("failed to find a passing nonce within search bound - target may be miscalibrated for this test")
	}

	nonceLE := []byte{byte(foundNonce), byte(foundNonce >> 8), byte(foundNonce >> 16), byte(foundNonce >> 24)}
	nonceWireHex := hex.EncodeToString(reverse4(nonceLE))

	resultCh := make(chan stratumResult, 1)
	errCh := make(chan error, 1)
	go func() {
		res, err := srv.WaitForResult(job, 5*time.Second)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- res
	}()

	send(map[string]interface{}{
		"id":     3,
		"method": "mining.submit",
		"params": []interface{}{
			"testuser", jobID, hex.EncodeToString(extranonce2), ntimeWireHex, nonceWireHex,
		},
	})

	submitResp := recvLine()
	if ok, _ := submitResp["result"].(bool); !ok {
		t.Fatalf("server rejected our share: %#v", submitResp)
	}

	select {
	case res := <-resultCh:
		if res.nonce != foundNonce {
			t.Errorf("nonce mismatch: server=%d test=%d", res.nonce, foundNonce)
		}
		if hex.EncodeToString(res.hash[:]) != hex.EncodeToString(foundHash) {
			t.Errorf("hash mismatch:\n server=%x\n test=  %x", res.hash, foundHash)
		}
	case err := <-errCh:
		t.Fatalf("WaitForResult: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for WaitForResult")
	}
}
