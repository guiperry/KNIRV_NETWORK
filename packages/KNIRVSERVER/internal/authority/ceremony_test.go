package authority

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// hkdfSHA256 is RFC 5869 written out in the test, so the ceremony is checked
// against an implementation that shares no code with it.
func hkdfSHA256(ikm, salt, info []byte, n int) []byte {
	ext := hmac.New(sha256.New, salt)
	ext.Write(ikm)
	prk := ext.Sum(nil)
	var out, t []byte
	for i := byte(1); len(out) < n; i++ {
		h := hmac.New(sha256.New, prk)
		h.Write(t)
		h.Write(info)
		h.Write([]byte{i})
		t = h.Sum(nil)
		out = append(out, t...)
	}
	return out[:n]
}

// fixedEntropy returns known bytes and remembers the challenge it was given.
type fixedEntropy struct {
	bytes     []byte
	kind      string
	err       error
	challenge [32]byte
	calls     int
}

func (f *fixedEntropy) Draw(_ context.Context, c [32]byte) ([]byte, string, error) {
	f.calls++
	f.challenge = c
	if f.err != nil {
		return nil, "", f.err
	}
	kind := f.kind
	if kind == "" {
		kind = EntropyCSPRNG
	}
	return append([]byte(nil), f.bytes...), kind, nil
}

func ceremonyParams(t *testing.T, e EntropySource) CeremonyParams {
	t.Helper()
	return CeremonyParams{
		LongTerm: testSigner(t, 1), Entropy: e, Epoch: 3, Scope: "*",
		Now: time.Unix(1_750_000_000, 0), TTL: 24 * time.Hour, Monotime: 1234567890123456789,
	}
}

func TestDeriveScalarMatchesAnIndependentKnownAnswer(t *testing.T) {
	// Known answers computed with a from-scratch HKDF in Python (RFC 5869).
	ikm, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f112210f47de98115")
	salt, _ := hex.DecodeString("fc8b64001c5fdd0f2f40fb67dae4a865a2c5bd17836676d6d5b58b7917e33717")
	got, err := deriveScalar(ikm, salt, 0)
	if err != nil || hex.EncodeToString(got) != "5a4d29e105d67329a592233b21347bba0f7b817db8a90394bf2e1693a5ff07fa" {
		t.Fatalf("scalar = %x, %v", got, err)
	}
	// A rejected first attempt retries with a counter byte after the info string.
	got, err = deriveScalar(ikm, salt, 1)
	if err != nil || hex.EncodeToString(got) != "6488d9da1c0e4e8bbdc1ff0e481867d31e3b1b464c8bac5ba0990cc778fa8c33" {
		t.Fatalf("retry scalar = %x, %v", got, err)
	}
	if _, err := deriveScalar(ikm, salt, maxScalarAttempts); err == nil {
		t.Fatal("endless rejection must give up with an error")
	}
}

func TestCeremonyDerivesTheSpecifiedKeyAndCertifiesIt(t *testing.T) {
	ent := &fixedEntropy{bytes: bytes.Repeat([]byte{0x42}, 32)}
	p := ceremonyParams(t, ent)
	sess, err := RunCeremony(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	// Re-derive independently: ikm = entropy || monotonic_ns, salt = sha256(challenge).
	var mono [8]byte
	binary.BigEndian.PutUint64(mono[:], uint64(p.Monotime))
	salt := sha256.Sum256(ent.challenge[:])
	scalar := hkdfSHA256(append(bytes.Repeat([]byte{0x42}, 32), mono[:]...), salt[:], []byte("KNIRV-ROOT-SESSION-V1"), 32)
	want, err := knirvsigning.NewLocalSigner(scalar)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sess.Signer.PublicKey(), want.PublicKey()) || sess.Signer.Address() != want.Address() {
		t.Fatal("the session key is not the one the specification derives")
	}

	c := sess.Cert
	if c.Operator != p.LongTerm.Address() || c.Epoch != 3 || c.Scope != "*" || c.Entropy != EntropyCSPRNG {
		t.Fatalf("certificate fields: %+v", c)
	}
	if c.NotBefore != p.Now.Unix()-60 || c.NotAfter != p.Now.Add(24*time.Hour).Unix() {
		t.Fatalf("certificate window %d..%d", c.NotBefore, c.NotAfter)
	}
	if !bytes.Equal(c.SessionPub, sess.Signer.PublicKey()) || bytes.Equal(c.SessionPub, p.LongTerm.PublicKey()) {
		t.Fatal("the certificate must name the new key, not the long-term key")
	}
	if err := c.Verify(p.LongTerm.PublicKey(), p.Now); err != nil {
		t.Fatalf("the long-term key's signature on the certificate does not verify: %v", err)
	}
	if err := c.Verify(testSigner(t, 9).PublicKey(), p.Now); err == nil {
		t.Fatal("the certificate verified under a key that did not sign it")
	}
}

func TestSessionKeyIsFreshEveryCeremony(t *testing.T) {
	a, err := RunCeremony(context.Background(), ceremonyParams(t, CSPRNG{}))
	if err != nil {
		t.Fatal(err)
	}
	b, err := RunCeremony(context.Background(), ceremonyParams(t, CSPRNG{})) // same time, same inputs
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a.Signer.PublicKey(), b.Signer.PublicKey()) {
		t.Fatal("two ceremonies produced the same key")
	}
}

func TestPrivateKeyAndEntropyNeverReachDisk(t *testing.T) {
	dir := t.TempDir()
	ent := &fixedEntropy{bytes: bytes.Repeat([]byte{0x77}, 32)}
	fixed := time.Unix(1_750_000_000, 0) // a fixed clock makes monotonic_ns exactly 0, so the scalar is derivable here
	m, err := NewSessionManager(SessionConfig{LongTerm: testSigner(t, 1), Entropy: ent, Scope: "*", TTL: time.Hour, EpochFloor: 1, Dir: dir, Now: func() time.Time { return fixed }})
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := OpenStore(dir)
	defer st.Close()
	sess, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Search every file in the directory for the raw entropy and the HKDF salt.
	salt := sha256.Sum256(ent.challenge[:])
	scalar := hkdfSHA256(append(bytes.Clone(ent.bytes), make([]byte, 8)...), salt[:], []byte("KNIRV-ROOT-SESSION-V1"), 32)
	if want, _ := knirvsigning.NewLocalSigner(scalar); want == nil || !bytes.Equal(want.PublicKey(), sess.Cert.SessionPub) {
		t.Fatal("test setup: could not reproduce the session key, so the search below would prove nothing")
	}
	secrets := []string{hex.EncodeToString(ent.bytes), hex.EncodeToString(salt[:]), hex.EncodeToString(scalar)}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		for _, s := range secrets {
			if bytes.Contains(data, []byte(s)) {
				t.Errorf("%s contains secret material %s...", e.Name(), s[:8])
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, sessionsFile))
	if err != nil {
		t.Fatal(err)
	}
	// sessions.jsonl is public facts only: a field list, never a private key field.
	for _, banned := range []string{"private", "scalar", "secret", "priv"} {
		if strings.Contains(strings.ToLower(string(data)), banned) {
			t.Errorf("sessions.jsonl mentions %q", banned)
		}
	}
	if !strings.Contains(string(data), hex.EncodeToString(sess.Cert.SessionPub)) {
		t.Error("the public key should be in the history")
	}
}

func TestCeremonyFailuresAreExplicit(t *testing.T) {
	ctx := context.Background()
	t.Run("ASIC entropy is not available yet", func(t *testing.T) {
		if _, err := RunCeremony(ctx, ceremonyParams(t, ASIC{})); !errors.Is(err, ErrASICUnavailable) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("entropy source error", func(t *testing.T) {
		if _, err := RunCeremony(ctx, ceremonyParams(t, &fixedEntropy{err: errors.New("device offline")})); err == nil || !strings.Contains(err.Error(), "device offline") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("too little entropy", func(t *testing.T) {
		if _, err := RunCeremony(ctx, ceremonyParams(t, &fixedEntropy{bytes: make([]byte, 31)})); err == nil {
			t.Fatal("31 bytes were accepted")
		}
	})
	t.Run("unknown entropy kind", func(t *testing.T) {
		if _, err := RunCeremony(ctx, ceremonyParams(t, &fixedEntropy{bytes: make([]byte, 32), kind: "trust-me"})); err == nil {
			t.Fatal("an unknown kind was written into a certificate")
		}
	})
	t.Run("asic kind is carried into the certificate", func(t *testing.T) {
		sess, err := RunCeremony(ctx, ceremonyParams(t, &fixedEntropy{bytes: bytes.Repeat([]byte{1}, 40), kind: EntropyCSPRNGASIC}))
		if err != nil || sess.Cert.Entropy != EntropyCSPRNGASIC || sess.Kind != EntropyCSPRNGASIC {
			t.Fatalf("%+v %v", sess, err)
		}
		if err := sess.Cert.Verify(testSigner(t, 1).PublicKey(), time.Unix(1_750_000_000, 0)); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("cancelled context", func(t *testing.T) {
		c, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := RunCeremony(c, ceremonyParams(t, CSPRNG{})); err == nil {
			t.Fatal("a cancelled ceremony succeeded")
		}
	})
	t.Run("bad parameters", func(t *testing.T) {
		for name, mut := range map[string]func(*CeremonyParams){
			"no epoch":          func(p *CeremonyParams) { p.Epoch = 0 },
			"no ttl":            func(p *CeremonyParams) { p.TTL = 0 },
			"no scope":          func(p *CeremonyParams) { p.Scope = "" },
			"no entropy source": func(p *CeremonyParams) { p.Entropy = nil },
			"no long-term key":  func(p *CeremonyParams) { p.LongTerm = nil },
		} {
			p := ceremonyParams(t, CSPRNG{})
			mut(&p)
			if _, err := RunCeremony(ctx, p); err == nil {
				t.Errorf("%s was accepted", name)
			}
		}
	})
}

// A session key must be able to sign a record that the verifier accepts with the
// long-term key as the only registered key.
func TestSessionKeySignsVerifyingRecords(t *testing.T) {
	long := testSigner(t, 1)
	now := time.Unix(1_750_000_000, 0)
	sess, err := RunCeremony(context.Background(), CeremonyParams{LongTerm: long, Entropy: CSPRNG{}, Epoch: 1, Scope: "*", Now: now, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	rec := &sdk.Record{
		Tier: sdk.TierT1, Domain: "evidence.commit", Scope: long.Address(), Operator: long.Address(), Epoch: 1,
		CanonicalActionHash: sha256.Sum256([]byte("x")), TargetBits: sdk.TierTargetBits(sdk.TierT1),
		WitnessKind: sdk.WitnessSoftware, SessionCert: sess.Cert,
	}
	if err := rec.Seal(context.Background(), sess.Signer, uint32(now.Unix())); err != nil {
		t.Fatal(err)
	}
	if err := sdk.Verify(rec, registryFor(long, KindRoot), now); err != nil {
		t.Fatalf("a record signed by the session key does not verify: %v", err)
	}
	if bytes.Equal(rec.Signature, nil) {
		t.Fatal("unsigned")
	}
}
