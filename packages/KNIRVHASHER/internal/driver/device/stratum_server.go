// internal/driver/device/stratum_server.go
//
// A minimal, real Stratum V1 mining pool server. When CGMiner is available,
// KNIRVHASHER registers this as CGMiner's active pool (via CGMinerClient's
// addpool/switchpool RPCs) so that ComputeBatch/MineWork can hand real work
// to the live ASIC and get back a genuine, hardware-found nonce - instead of
// the previous approach of inferring a "nonce" from CGMiner's accepted-share
// counter.
//
// Wire-format handling here (byte order of prevhash/version/ntime/nbits,
// coinbase assembly, merkle root derivation, nonce encoding) follows the
// canonical SHA256d Stratum V1 dialect implemented by every Bitcoin ASIC
// miner client (CGMiner included) - cross-checked against the widely used
// nightminer.py reference client implementation. See swapWord/swapWords32.
package device

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	stratumExtranonce2Size  = 4
	stratumMaxTrackedJobs   = 64
	cgMinerBatchItemTimeout = 15 * time.Second

	// stratumSubscribeTimeout bounds how long startCGMinerStratumPool waits
	// for CGMiner to actually connect and subscribe after being redirected
	// to this pool via addpool/switchpool, before declaring the pool ready.
	// Without this wait, OpenDevice/NewCGMinerHashMethod would return
	// "operational" the instant the addpool/switchpool RPCs succeed - but
	// CGMiner's own reconnect-and-subscribe cycle isn't instantaneous, so a
	// caller's first real ComputeHash/ComputeBatch call could race it and
	// lose against a short external health-check timeout (this is exactly
	// what produced "server responded but client is in fallback mode" -
	// GetDeviceInfo succeeded, but the ComputeHash probe that follows it
	// timed out waiting on a CGMiner connection that hadn't landed yet).
	stratumSubscribeTimeout = 20 * time.Second

	// stratumFixedPort/User/Pass are fixed rather than randomly chosen per
	// process start. This lets a pool entry be configured once, by hand,
	// through CGMiner's own control surface - its RPC addpool (when the
	// running instance's API privileges allow it) or, on hardware like the
	// Bitmain S3 where the stock /etc/init.d/cgminer instance denies
	// addpool write access, the device's own web GUI under Miner
	// Configuration - and keep working across every future hasher-server
	// restart, since the port/credentials it points at never change. The
	// password carries no real security weight (see dispatch's
	// mining.authorize handler, which doesn't verify it) - this pool is
	// only ever reachable on loopback/the local subnet anyway - so a fixed
	// value is fine.
	stratumFixedPort = 18332
	stratumFixedUser = "knirvhasher"
	stratumFixedPass = "knirvhasher"
)

// diff1Target is the canonical Stratum/Bitcoin "difficulty 1" target: a
// share's hash (byte-reversed, interpreted as a big-endian integer) must be
// <= this value. Every SHA256d Stratum miner, CGMiner included, understands
// mining.set_difficulty(1) against this same target, so it is the safest
// difficulty to request - shares arrive in low-single-digit milliseconds at
// hundreds of GH/s, and any well-behaved client accepts it.
var diff1Target = new(big.Int).Lsh(big.NewInt(0xffff0000), 192)

// diff1NBitsHeader is an inert placeholder for the header's nbits field.
// Stratum share validity is governed entirely by mining.set_difficulty, not
// by the header's own nbits bytes (that field only matters for validating a
// hash against the *network's* real difficulty, which is not applicable
// here), so any fixed, well-formed value works.
var diff1NBitsHeader = func() [4]byte {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], 0x1d00ffff)
	return b
}()

// sha256d computes SHA256(SHA256(data)), the double-hash used throughout
// Bitcoin/Stratum for block headers and coinbase transactions.
func sha256d(data []byte) [32]byte {
	first := sha256.Sum256(data)
	return sha256.Sum256(first[:])
}

// swapWord reverses the 4 bytes of a 32-bit word. Stratum V1 clients apply
// this to the wire-format version/ntime/nbits fields to obtain the actual
// little-endian header bytes; since the transform is its own inverse, the
// server applies it the same way to go from header bytes to wire bytes.
func swapWord(b [4]byte) [4]byte {
	return [4]byte{b[3], b[2], b[1], b[0]}
}

// swapWords32 applies swapWord independently to each of the eight 4-byte
// words of a 32-byte value, preserving word order. This is the transform
// Stratum V1 clients apply to the wire "prevhash" field to obtain header
// bytes (and, being self-inverse, what the server applies in reverse).
func swapWords32(b [32]byte) [32]byte {
	var out [32]byte
	for w := 0; w < 8; w++ {
		s := swapWord([4]byte{b[w*4], b[w*4+1], b[w*4+2], b[w*4+3]})
		copy(out[w*4:w*4+4], s[:])
	}
	return out
}

// hashMeetsTarget reports whether a header hash satisfies target, per
// Stratum/Bitcoin convention: the hash bytes are reversed, then compared as
// a big-endian integer against the target.
func hashMeetsTarget(hash [32]byte, target *big.Int) bool {
	var rev [32]byte
	for i := range hash {
		rev[i] = hash[31-i]
	}
	return new(big.Int).SetBytes(rev[:]).Cmp(target) <= 0
}

// stratumJob is one unit of ASIC work handed to CGMiner via mining.notify.
// versionHdr/prevHashHdr/ntimeHdr/nbitsHdr hold the actual header-ready
// bytes (offsets 0:4, 4:36, 68:72, 72:76 of the 80-byte header); the wire
// hex sent in mining.notify is derived from them via swapWord/swapWords32.
// The merkle root (offset 36:68) is never stored here - by Stratum's
// design it can only be derived from the coinbase once a share is
// submitted, so it is computed fresh in handleSubmit.
type stratumJob struct {
	id          string
	versionHdr  [4]byte
	prevHashHdr [32]byte
	ntimeHdr    [4]byte
	nbitsHdr    [4]byte
	coinb1      []byte
	coinb2      []byte

	resultCh  chan stratumResult
	responded int32
}

type stratumResult struct {
	nonce  uint32
	header [80]byte
	hash   [32]byte
}

// Nonce returns the hardware-found nonce this result was verified against.
func (r stratumResult) Nonce() uint32 { return r.nonce }

// Hash returns the independently-reverified double-SHA256 hash this result
// solved.
func (r stratumResult) Hash() [32]byte { return r.hash }

func (j *stratumJob) notifyParams(clean bool) []interface{} {
	versionWire := swapWord(j.versionHdr)
	prevHashWire := swapWords32(j.prevHashHdr)
	nbitsWire := swapWord(j.nbitsHdr)
	ntimeWire := swapWord(j.ntimeHdr)

	return []interface{}{
		j.id,
		hex.EncodeToString(prevHashWire[:]),
		hex.EncodeToString(j.coinb1),
		hex.EncodeToString(j.coinb2),
		[]string{}, // merkle_branch: empty - coinbase is the only "transaction"
		hex.EncodeToString(versionWire[:]),
		hex.EncodeToString(nbitsWire[:]),
		hex.EncodeToString(ntimeWire[:]),
		clean,
	}
}

// stratumClient is one TCP connection from a Stratum client (CGMiner).
type stratumClient struct {
	conn       net.Conn
	writeMu    sync.Mutex
	subscribed bool
}

func (c *stratumClient) send(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	data = append(data, '\n')

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, _ = c.conn.Write(data)
}

type rpcResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

type rpcNotification struct {
	ID     interface{}   `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// StratumServer is a minimal single-purpose Stratum V1 pool server: it only
// implements what a real ASIC miner (CGMiner) needs to fetch work and
// submit shares - subscribe, authorize, configure (declined), notify, and
// submit with independent share verification.
type StratumServer struct {
	listener        net.Listener
	port            int
	user            string
	pass            string
	extranonce1     [4]byte
	extranonce2Size int

	mu        sync.Mutex
	jobs      map[string]*stratumJob
	jobOrder  []string
	// atomic.Uint64, not a raw uint64 field: sync/atomic's function-style
	// ops (atomic.AddUint64(&x, ...)) require the target to be 8-byte
	// aligned, which the Go spec only guarantees for a struct's first
	// word - on 32-bit platforms (this runs on a MIPS Antminer) a uint64
	// field this far into the struct reliably panics with "unaligned
	// 64-bit atomic operation". The typed atomic.Uint64 handles its own
	// alignment correctly regardless of field position.
	nextJobID atomic.Uint64

	clientsMu sync.Mutex
	clients   map[*stratumClient]struct{}

	closed int32

	// diffMu guards difficulty/target: SetDifficulty can be called any
	// time relative to dispatch/handleSubmit running on client goroutines.
	diffMu     sync.RWMutex
	difficulty float64
	target     *big.Int
}

// NewStratumServer starts listening on port at bindIP (pass 0 to let the OS
// assign an ephemeral port, e.g. in tests) and begins accepting Stratum
// client connections. bindIP must be an address the target CGMiner instance
// can actually reach - "127.0.0.1" when CGMiner runs on this same host, or a
// routable local interface address when it runs elsewhere (see
// localAddrFor). Production callers should pass stratumFixedPort so a pool
// entry configured once (by hand, or via addpool) keeps working across
// restarts - see the constant's doc comment.
func NewStratumServer(bindIP string, port int, user, pass string) (*StratumServer, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort(bindIP, strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	var extranonce1 [4]byte
	if _, err := rand.Read(extranonce1[:]); err != nil {
		ln.Close()
		return nil, fmt.Errorf("generate extranonce1: %w", err)
	}

	s := &StratumServer{
		listener:        ln,
		port:            ln.Addr().(*net.TCPAddr).Port,
		user:            user,
		pass:            pass,
		extranonce1:     extranonce1,
		extranonce2Size: stratumExtranonce2Size,
		jobs:            make(map[string]*stratumJob),
		clients:         make(map[*stratumClient]struct{}),
		difficulty:      1,
		target:          diff1Target,
	}

	go s.acceptLoop()

	return s, nil
}

func (s *StratumServer) Port() int    { return s.port }
func (s *StratumServer) User() string { return s.user }
func (s *StratumServer) Pass() string { return s.pass }

// SetDifficulty changes the share difficulty new clients are told about
// (via mining.set_difficulty on their next mining.subscribe) and that
// handleSubmit verifies submitted shares against. Existing already-
// subscribed clients are not retroactively notified - call this before any
// client connects if it needs to apply from the start. Difficulty 1 (the
// default) is the easiest a standards-compliant Stratum client will ever
// need to accept; some ASIC firmware may silently refuse to mine jobs below
// whatever difficulty floor it considers sane, which is exactly what this
// exists to let a caller test.
func (s *StratumServer) SetDifficulty(difficulty float64) {
	if difficulty <= 0 {
		difficulty = 1
	}
	target := new(big.Int).Div(diff1Target, big.NewInt(int64(difficulty)))

	s.diffMu.Lock()
	s.difficulty = difficulty
	s.target = target
	s.diffMu.Unlock()
}

func (s *StratumServer) currentDifficultyAndTarget() (float64, *big.Int) {
	s.diffMu.RLock()
	defer s.diffMu.RUnlock()
	return s.difficulty, s.target
}

// Close stops accepting connections and closes every client connection.
func (s *StratumServer) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	s.clientsMu.Lock()
	for c := range s.clients {
		c.conn.Close()
	}
	s.clientsMu.Unlock()

	return s.listener.Close()
}

func (s *StratumServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if atomic.LoadInt32(&s.closed) == 1 {
				return
			}
			log.Printf("stratum: accept error: %v", err)
			continue
		}

		c := &stratumClient{conn: conn}
		s.clientsMu.Lock()
		s.clients[c] = struct{}{}
		s.clientsMu.Unlock()

		go s.handleClient(c)
	}
}

func (s *StratumServer) handleClient(c *stratumClient) {
	log.Printf("stratum: cgminer connected from %s", c.conn.RemoteAddr())

	defer func() {
		// An unrecovered panic in any goroutine - this one included - kills
		// the entire Go process, not just this connection. This handler
		// processes raw, ASIC-firmware-controlled input directly (real
		// CGMiner traffic, not just what this file's own tests generate),
		// so a panic here must never be allowed to take the whole
		// hasher-server process down with it - that previously showed up
		// as the first real mining.submit getting a bare "EOF" on the
		// gRPC side and every connection after that failing outright,
		// because the process had already died.
		if r := recover(); r != nil {
			log.Printf("stratum: PANIC recovered handling client %s: %v\n%s", c.conn.RemoteAddr(), r, debug.Stack())
		}

		s.clientsMu.Lock()
		delete(s.clients, c)
		s.clientsMu.Unlock()
		c.conn.Close()
		log.Printf("stratum: cgminer disconnected from %s", c.conn.RemoteAddr())
	}()

	reader := bufio.NewReader(c.conn)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			s.dispatch(c, line)
		}
		if err != nil {
			return
		}
	}
}

func (s *StratumServer) dispatch(c *stratumClient, line []byte) {
	// Logged unconditionally, not just unrecognized methods: mining.notify
	// is a one-way push with no acknowledgment, so if CGMiner silently
	// rejects or ignores a job (a bad field, or some handshake step this
	// particular firmware's stratum client expects before it'll actually
	// mine, e.g. mining.suggest_difficulty or mining.extranonce.subscribe)
	// there is no other way to see that anything went wrong - the
	// connection just goes quiet. Traffic here is a handful of messages at
	// startup, not a hot path, so this is cheap.
	log.Printf("stratum: recv from %s: %s", c.conn.RemoteAddr(), bytes.TrimSpace(line))

	var req struct {
		ID     interface{}       `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(line, &req); err != nil {
		log.Printf("stratum: failed to parse message from %s: %v", c.conn.RemoteAddr(), err)
		return
	}

	switch req.Method {
	case "mining.subscribe":
		result := []interface{}{
			[][2]string{{"mining.set_difficulty", "knirv1"}, {"mining.notify", "knirv1"}},
			hex.EncodeToString(s.extranonce1[:]),
			s.extranonce2Size,
		}
		c.send(rpcResponse{ID: req.ID, Result: result})
		c.subscribed = true
		difficulty, _ := s.currentDifficultyAndTarget()
		c.send(rpcNotification{ID: nil, Method: "mining.set_difficulty", Params: []interface{}{difficulty}})

	case "mining.authorize":
		c.send(rpcResponse{ID: req.ID, Result: true})

	case "mining.configure":
		// No extensions (e.g. version-rolling) supported - decline cleanly.
		c.send(rpcResponse{ID: req.ID, Result: map[string]interface{}{}})

	case "mining.submit":
		ok, errMsg := s.handleSubmit(req.Params)
		var errVal interface{}
		if !ok {
			errVal = []interface{}{23, errMsg, nil}
		}
		c.send(rpcResponse{ID: req.ID, Result: ok, Error: errVal})

	default:
		log.Printf("stratum: UNRECOGNIZED method %q from %s (see raw message above) - if cgminer's own client requires this to succeed before it will mine, this is likely why", req.Method, c.conn.RemoteAddr())
		if req.ID != nil {
			c.send(rpcResponse{ID: req.ID, Result: nil, Error: []interface{}{20, "unrecognized method", nil}})
		}
	}
}

// handleSubmit independently reconstructs the header a submitted share
// claims to solve and reverifies its hash against the difficulty-1 target
// itself - the submitted nonce is never trusted blindly.
func (s *StratumServer) handleSubmit(params []json.RawMessage) (bool, string) {
	if len(params) < 5 {
		return false, "malformed submit"
	}

	var jobID, extranonce2Hex, nonceHex string
	if err := json.Unmarshal(params[1], &jobID); err != nil {
		return false, "bad job_id"
	}
	if err := json.Unmarshal(params[2], &extranonce2Hex); err != nil {
		return false, "bad extranonce2"
	}
	if err := json.Unmarshal(params[4], &nonceHex); err != nil {
		return false, "bad nonce"
	}

	s.mu.Lock()
	job, ok := s.jobs[jobID]
	s.mu.Unlock()
	if !ok {
		return false, "job not found"
	}

	extranonce2, err := hex.DecodeString(extranonce2Hex)
	if err != nil || len(extranonce2) != s.extranonce2Size {
		return false, "bad extranonce2"
	}

	nonceWire, err := hex.DecodeString(nonceHex)
	if err != nil || len(nonceWire) != 4 {
		return false, "bad nonce"
	}

	coinbase := make([]byte, 0, len(job.coinb1)+len(s.extranonce1)+len(extranonce2)+len(job.coinb2))
	coinbase = append(coinbase, job.coinb1...)
	coinbase = append(coinbase, s.extranonce1[:]...)
	coinbase = append(coinbase, extranonce2...)
	coinbase = append(coinbase, job.coinb2...)

	merkleRoot := sha256d(coinbase) // no merkle branches: coinbase is the only "tx"

	var header [80]byte
	copy(header[0:4], job.versionHdr[:])
	copy(header[4:36], job.prevHashHdr[:])
	copy(header[36:68], merkleRoot[:])
	copy(header[68:72], job.ntimeHdr[:])
	copy(header[72:76], job.nbitsHdr[:])
	// Wire nonce is big-endian; header field is its byte-reverse (little-endian).
	header[76], header[77], header[78], header[79] = nonceWire[3], nonceWire[2], nonceWire[1], nonceWire[0]

	hash := sha256d(header[:])
	_, target := s.currentDifficultyAndTarget()
	if !hashMeetsTarget(hash, target) {
		return false, "high-hash"
	}

	if atomic.CompareAndSwapInt32(&job.responded, 0, 1) {
		job.resultCh <- stratumResult{
			nonce:  binary.BigEndian.Uint32(nonceWire),
			header: header,
			hash:   hash,
		}
	}

	return true, ""
}

func (s *StratumServer) newJob(versionHdr [4]byte, prevHashHdr [32]byte, ntimeHdr, nbitsHdr [4]byte, coinb1, coinb2 []byte) *stratumJob {
	id := strconv.FormatUint(s.nextJobID.Add(1), 16)
	return &stratumJob{
		id:          id,
		versionHdr:  versionHdr,
		prevHashHdr: prevHashHdr,
		ntimeHdr:    ntimeHdr,
		nbitsHdr:    nbitsHdr,
		coinb1:      coinb1,
		coinb2:      coinb2,
		resultCh:    make(chan stratumResult, 1),
	}
}

// NewJob builds a job whose header fields are freely chosen (no caller
// constraint on the actual header bytes), used by ComputeBatch where inputs
// are opaque byte blobs the driver is already free to map into a header
// however it likes - so there is no merkleroot-preservation issue here.
func (s *StratumServer) NewJob(input []byte, workID int) *stratumJob {
	prevHashHdr := sha256.Sum256(input)

	var versionHdr [4]byte
	binary.BigEndian.PutUint32(versionHdr[:], 1)

	var ntimeHdr [4]byte
	binary.BigEndian.PutUint32(ntimeHdr[:], uint32(time.Now().Unix()))

	coinb1 := append([]byte("KNIRVHASHER-CB-"), byte(workID>>8), byte(workID))

	return s.newJob(versionHdr, prevHashHdr, ntimeHdr, diff1NBitsHeader, coinb1, []byte{0, 0, 0, 0})
}

// NewJobFromHeader builds a job that preserves version/prevhash/ntime/nbits
// from an externally supplied 80-byte header (as used by Device.MineWork).
// The merkleroot (header bytes 36:68) is intentionally NOT preserved: real
// Stratum always derives it from the pool's own coinbase transaction, so a
// caller-chosen merkleroot cannot survive real pool-based mining. Callers
// that need the exact supplied header hashed verbatim must use the direct
// USB/kernel-device path instead, not CGMiner/stratum mode.
func (s *StratumServer) NewJobFromHeader(header []byte) *stratumJob {
	var versionHdr [4]byte
	copy(versionHdr[:], header[0:4])
	var prevHashHdr [32]byte
	copy(prevHashHdr[:], header[4:36])
	var ntimeHdr [4]byte
	copy(ntimeHdr[:], header[68:72])
	var nbitsHdr [4]byte
	copy(nbitsHdr[:], header[72:76])

	return s.newJob(versionHdr, prevHashHdr, ntimeHdr, nbitsHdr, []byte("KNIRVHASHER-MW"), []byte{0, 0, 0, 0})
}

// SubmitJob registers the job and broadcasts it to every subscribed client
// via mining.notify with clean_jobs=true, so CGMiner immediately abandons
// any in-flight search and starts hashing this job. Returns an error if no
// subscribed CGMiner connection exists to receive it.
func (s *StratumServer) SubmitJob(job *stratumJob) error {
	s.mu.Lock()
	s.jobs[job.id] = job
	s.jobOrder = append(s.jobOrder, job.id)
	if len(s.jobOrder) > stratumMaxTrackedJobs {
		oldest := s.jobOrder[0]
		s.jobOrder = s.jobOrder[1:]
		delete(s.jobs, oldest)
	}
	s.mu.Unlock()

	notif := rpcNotification{ID: nil, Method: "mining.notify", Params: job.notifyParams(true)}

	sent := 0
	s.clientsMu.Lock()
	for c := range s.clients {
		if !c.subscribed {
			continue
		}
		c.send(notif)
		sent++
	}
	s.clientsMu.Unlock()

	if sent == 0 {
		return fmt.Errorf("no subscribed cgminer stratum connection to publish work to")
	}
	return nil
}

// WaitForResult blocks until the job's share is submitted and verified, or
// timeout elapses. Either way the job is removed from tracking.
func (s *StratumServer) WaitForResult(job *stratumJob, timeout time.Duration) (stratumResult, error) {
	defer func() {
		s.mu.Lock()
		delete(s.jobs, job.id)
		s.mu.Unlock()
	}()

	select {
	case res := <-job.resultCh:
		return res, nil
	case <-time.After(timeout):
		return stratumResult{}, fmt.Errorf("timed out waiting for cgminer to submit a verified share for job %s", job.id)
	}
}

// WaitForSubscribedClient blocks until at least one Stratum client (CGMiner)
// has connected and completed mining.subscribe, or timeout elapses. Callers
// should wait on this after registering the pool with CGMiner (addpool +
// switchpool) and before treating the pool as ready, so the first real
// ComputeHash/ComputeBatch call doesn't race CGMiner's own reconnect
// latency against a caller's much shorter health-check timeout.
func (s *StratumServer) WaitForSubscribedClient(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		s.clientsMu.Lock()
		for c := range s.clients {
			if c.subscribed {
				s.clientsMu.Unlock()
				return nil
			}
		}
		s.clientsMu.Unlock()

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for cgminer to connect and subscribe to the local stratum pool", timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// localAddrFor returns an address of this host that cgminerHost can reach:
// the loopback address when CGMiner runs on this same machine, or - since
// NewCGMinerHashMethod supports connecting to a CGMiner on a different host
// entirely - the local interface address the kernel would route through to
// reach cgminerHost, found via an unconnected UDP "dial" (no packets are
// actually sent; this only consults the routing table).
func localAddrFor(cgminerHost string) (string, error) {
	switch cgminerHost {
	case "", "127.0.0.1", "localhost", "::1":
		return "127.0.0.1", nil
	}

	conn, err := net.Dial("udp", net.JoinHostPort(cgminerHost, "1"))
	if err != nil {
		return "", fmt.Errorf("determine local address reachable from cgminer host %s: %w", cgminerHost, err)
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("unexpected local address type %T", conn.LocalAddr())
	}
	return localAddr.IP.String(), nil
}

// startCGMinerStratumPool starts a local Stratum V1 pool - bound to an
// address the CGMiner instance at cgminerHost can reach, whether that's
// this same machine or a remote miner - on a fixed port/credentials (see
// stratumFixedPort's doc comment), and best-effort tries to point CGMiner
// at it via addpool+switchpool over CGMiner's RPC API.
//
// That RPC registration is deliberately non-fatal: some CGMiner API
// privilege configurations reject addpool outright ("Access denied to
// 'addpool' command") even when read commands work fine - observed on a
// Bitmain S3's stock /etc/init.d/cgminer instance. Rather than fail
// outright, this falls through to WaitForSubscribedClient, which succeeds
// equally whether CGMiner learned about the pool from our RPC call or from
// a pool entry configured by hand through the device's own web GUI (Miner
// Configuration) pointing at this fixed port - which is exactly why the
// port/credentials are fixed rather than randomized per run.
func startCGMinerStratumPool(client *CGMinerClient, cgminerHost string) (*StratumServer, error) {
	bindIP, err := localAddrFor(cgminerHost)
	if err != nil {
		return nil, err
	}

	srv, err := NewStratumServer(bindIP, stratumFixedPort, stratumFixedUser, stratumFixedPass)
	if err != nil {
		return nil, err
	}

	// NOTE: an earlier version of this function called
	// /etc/init.d/cgminer restart here, on the theory that CGMiner only
	// decides which configured pool is "active" once, at its own startup,
	// and restarting it after our listener was already up would force a
	// fresh evaluation that lands on us. Reverted: on this device, that
	// restart command itself failed (exit status 1) and left CGMiner not
	// running at all afterward (the very next addpool attempt got
	// "connection refused" on its API port) - worse than the problem it
	// was meant to solve, and not safe to keep retrying blind given how
	// costly a device stuck in a bad state is to recover from here. If
	// this hypothesis is worth testing further, do it manually via the
	// device's own web GUI (Miner Configuration -> Save & Apply, which we
	// know reliably works) at a moment when hasher-server's listener is
	// already confirmed up, rather than from this code.

	url := fmt.Sprintf("stratum+tcp://%s:%d", bindIP, srv.Port())
	if _, err := client.SendCommand("addpool", fmt.Sprintf("%s,%s,%s", url, srv.User(), srv.Pass())); err != nil {
		log.Printf("Warning: addpool RPC failed (%v) - relying on a manually configured CGMiner pool entry pointing at %s instead, if one exists", err, url)
	} else if idx, err := findCGMinerPoolIndex(client, url); err != nil {
		log.Printf("Warning: could not locate newly added pool (%v) - relying on a manually configured CGMiner pool entry pointing at %s instead, if one exists", err, url)
	} else if _, err := client.SendCommand("switchpool", strconv.Itoa(idx)); err != nil {
		log.Printf("Warning: switchpool RPC failed (%v) - relying on a manually configured CGMiner pool entry pointing at %s instead, if one exists", err, url)
	}

	// Wait for CGMiner to connect, but don't fail startup if it hasn't yet
	// by this point.
	//
	// This used to be fatal on timeout, which assumed CGMiner would connect
	// promptly after addpool/switchpool. But on hardware where those RPCs
	// are denied (observed: this device's stock /etc/init.d/cgminer
	// instance) and the pool is only reachable via a pool entry configured
	// once through the device's own web GUI, CGMiner reconnects to a
	// "Dead" pool on its own internal schedule, which we don't control and
	// doesn't reliably land inside any short fixed window - observed
	// directly: 2 of 3 consecutive runs connected within 20s, one didn't.
	// Failing startup over that coin flip meant hasher-server never even
	// finished opening the device, so ComputeHash's own external retry
	// loop (which already runs over ~90s - see asic_client.go's
	// ComputeHealthCheckTimeout and deployment.go's ConnectTimeout) never
	// got a chance to catch a later reconnect either. Returning srv
	// regardless keeps the listener alive and accepting a connection
	// whenever CGMiner gets to it; ComputeBatch/MineWork correctly fail
	// fast (via SubmitJob's "no subscribed connection" check) until then,
	// rather than hanging, and succeed on whichever later external retry
	// lands after CGMiner actually reconnects.
	if err := srv.WaitForSubscribedClient(stratumSubscribeTimeout); err != nil {
		log.Printf("Warning: cgminer has not connected to %s yet (%v) - it may still connect on its own reconnect schedule; ComputeBatch/MineWork will fail fast until it does. If addpool RPC access is denied, configure it as a pool in CGMiner by hand (URL=%s:%d, worker=%s, password=%s), e.g. via the device's own web GUI under Miner Configuration",
			url, err, bindIP, srv.Port(), stratumFixedUser, stratumFixedPass)
	}

	return srv, nil
}

func findCGMinerPoolIndex(client *CGMinerClient, url string) (int, error) {
	resp, err := client.SendCommand("pools")
	if err != nil {
		return 0, err
	}
	pools, ok := resp["POOLS"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected pools response shape: %+v", resp)
	}
	for _, p := range pools {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if u, _ := pm["URL"].(string); u == url {
			if idx, ok := pm["POOL"].(float64); ok {
				return int(idx), nil
			}
		}
	}
	return 0, fmt.Errorf("pool %s not found in cgminer pool list after addpool", url)
}
