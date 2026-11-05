package authority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

func testSigner(t testing.TB, n byte) knirvsigning.RemoteSigner {
	t.Helper()
	key := make([]byte, 32)
	key[31] = n
	s, err := knirvsigning.NewLocalSigner(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// clock is a settable time source shared by a test and its service.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *clock { return &clock{t: time.Unix(1_750_000_000, 0)} }
func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}
func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

type logBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logBuf) Logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(&l.buf, format+"\n", args...)
}
func (l *logBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

type harness struct {
	svc    *Service
	store  *Store
	signer knirvsigning.RemoteSigner
	clock  *clock
	logs   *logBuf
	dir    string
	kind   string
}

type harnessOpt func(*Config)

func newHarness(t *testing.T, kind string, seed byte, opts ...harnessOpt) *harness {
	t.Helper()
	dir := t.TempDir()
	return buildHarness(t, dir, kind, seed, opts...)
}

func buildHarness(t *testing.T, dir, kind string, seed byte, opts ...harnessOpt) *harness {
	t.Helper()
	st, _, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	h := &harness{store: st, signer: testSigner(t, seed), clock: newClock(), logs: &logBuf{}, dir: dir, kind: kind}
	cfg := Config{Kind: kind, Signer: h.signer, Policy: DefaultPolicy(), Store: st, Now: h.clock.Now, Logf: h.logs.Logf}
	for _, o := range opts {
		o(&cfg)
	}
	svc, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h.svc = svc
	return h
}

func act(tier uint8, domain, scope, body string) Action {
	return Action{Tier: tier, Domain: domain, Scope: scope, Body: []byte(body)}
}

// registryFor builds the verifier's view of a single operator, independent of the
// service's own registry helper.
func registryFor(signer knirvsigning.RemoteSigner, kind string) sdk.Registry {
	op := sdk.Operator{Address: signer.Address(), LongTermPub: signer.PublicKey()}
	if kind == KindRoot {
		op.Kind, op.Scopes = sdk.OperatorRoot, []string{"*"}
	} else {
		op.Kind, op.Scopes = sdk.OperatorBoot, []string{signer.Address()}
	}
	return sdk.Registry{Operators: map[string]sdk.Operator{signer.Address(): op}}
}

// specDigest is an independent implementation of AUTHORITY_SPEC.md section 2.1,
// used to cross-check the digest the service commits to in the header.
func specDigest(tier uint8, domain, scope, operator string, epoch uint64, action, prev [32]byte) [32]byte {
	var b bytes.Buffer
	b.WriteString("KNIRV-AUTH-V2\x00")
	b.WriteByte(tier)
	for _, s := range []string{domain, scope, operator} {
		var l [4]byte
		binary.BigEndian.PutUint32(l[:], uint32(len(s)))
		b.Write(l[:])
		b.WriteString(s)
	}
	var e [8]byte
	binary.BigEndian.PutUint64(e[:], epoch)
	b.Write(e[:])
	b.Write(action[:])
	b.Write(prev[:])
	return sha256.Sum256(b.Bytes())
}

// scriptedWitnesser lets a test control mining.
type scriptedWitnesser struct {
	mine func(ctx context.Context, header [80]byte, bits uint32) (uint32, [32]byte, error)
	kind string
}

func (w scriptedWitnesser) Mine(ctx context.Context, header [80]byte, bits uint32) (uint32, [32]byte, error) {
	return w.mine(ctx, header, bits)
}
func (w scriptedWitnesser) Kind() string {
	if w.kind == "" {
		return sdk.WitnessSoftware
	}
	return w.kind
}
