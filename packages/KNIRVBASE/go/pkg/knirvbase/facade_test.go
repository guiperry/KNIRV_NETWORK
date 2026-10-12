package knirvbase

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stor "github.com/knirvcorp/knirvbase/internal/storage"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// E10: the Go DB exposes the same key-value / object / index / DHT / markdown
// surface as the TypeScript DB.
func newDB(t *testing.T) *DB {
	t.Helper()
	opts := DefaultOptions()
	opts.DataDir = t.TempDir()
	opts.Consensus.Enabled = false
	d, err := New(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Shutdown() })
	return d
}

func TestFacadeKeyValueAndObjects(t *testing.T) {
	d, ctx := newDB(t), context.Background()
	if err := d.Put(ctx, "k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	if got, err := d.Get(ctx, "k"); err != nil || string(got) != "v" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if err := d.DeleteKey(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.Get(ctx, "k"); len(got) != 0 {
		t.Fatalf("key still present after DeleteKey: %q", got)
	}

	type obj struct{ A int }
	if err := d.StoreObject(ctx, "o", obj{A: 7}); err != nil {
		t.Fatal(err)
	}
	var back obj
	if err := d.GetObject(ctx, "o", &back); err != nil || back.A != 7 {
		t.Fatalf("GetObject = %+v, %v", back, err)
	}

	target := filepath.Join(t.TempDir(), "o.md")
	if err := d.ProjectToMarkdown(ctx, "o", target); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(target); err != nil || len(b) == 0 {
		t.Fatalf("markdown projection empty: %v", err)
	}
}

func TestFacadeIndexes(t *testing.T) {
	d, ctx := newDB(t), context.Background()
	if err := d.CreateIndex(ctx, "users", "by_name", stor.IndexTypeBTree, []string{"name"}, false, "", nil); err != nil {
		t.Fatal(err)
	}
	if d.GetIndex(ctx, "users", "by_name") == nil || len(d.GetIndexesForCollection(ctx, "users")) != 1 {
		t.Fatal("index not registered")
	}
	if err := d.DropIndex(ctx, "users", "by_name"); err != nil {
		t.Fatal(err)
	}
	if len(d.GetIndexesForCollection(ctx, "users")) != 0 {
		t.Fatal("index not dropped")
	}
	if _, err := d.QueryIndex(ctx, "users", "by_name", map[string]interface{}{"value": "x"}); err == nil {
		t.Fatal("querying a dropped index should fail")
	}
}

func TestFacadeDHT(t *testing.T) {
	d, ctx := newDB(t), context.Background()
	if err := d.PutDHT(ctx, "dk", "dv", time.Minute); err != nil {
		t.Fatal(err)
	}
	got, err := d.GetDHT(ctx, "dk")
	if err != nil || len(got) == 0 {
		t.Fatalf("GetDHT = %v, %v", got, err)
	}
}

func TestLoadOrCreateKeyPair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys", "dataset.key.json")
	kp, err := LoadOrCreateKeyPair(path, "datasets")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode = %v, %v", info, err)
	}
	if dir, _ := os.Stat(filepath.Dir(path)); dir.Mode().Perm() != 0o700 {
		t.Fatalf("key dir mode = %v", dir.Mode().Perm())
	}
	again, err := LoadOrCreateKeyPair(path, "datasets")
	if err != nil || again.ID != kp.ID {
		t.Fatalf("second load gave a different key: %v %v", again, err)
	}
	d := DescribeKey(kp)
	if d.ID != kp.ID || d.Algorithm != "ML-DSA-65" || d.PublicKey == "" {
		t.Fatalf("DescribeKey = %+v", d)
	}

	// An unsupported (legacy) key file is an error and is left untouched.
	legacy := filepath.Join(t.TempDir(), "legacy.json")
	if err := os.WriteFile(legacy, []byte(`{"id":"x","algorithm":"Kyber-768+Dilithium-3"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateKeyPair(legacy, "x"); err == nil {
		t.Fatal("legacy key file must be rejected")
	}
	if b, _ := os.ReadFile(legacy); !strings.Contains(string(b), "Kyber") {
		t.Fatal("legacy key file was overwritten")
	}
}

func TestSignedDatasetFramesVerify(t *testing.T) {
	dir, ctx := t.TempDir(), context.Background()
	kp, err := LoadOrCreateKeyPair(filepath.Join(t.TempDir(), "k.json"), "d")
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewNRV(ctx, Options{DataDir: dir}, kp)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Shutdown()

	ds := d.Dataset("signed")
	b := &nrv.Bracket{ID: "b1", GoldenSeed: 7}
	if err := ds.AppendBracket(ctx, b, nrv.ThermoAtmosphere{}); err != nil {
		t.Fatal(err)
	}
	if err := ds.Flush(); err != nil {
		t.Fatal(err)
	}
	frames, err := ds.Frames()
	if err != nil || len(frames) != 1 {
		t.Fatalf("frames = %v, %v", frames, err)
	}
	sig, err := ds.FrameSignature(frames[0].ID)
	if err != nil || sig == "" {
		t.Fatalf("signature = %q, %v", sig, err)
	}
	if ok, err := ds.VerifyFrame(frames[0].ID, kp); err != nil || !ok {
		t.Fatalf("VerifyFrame = %v, %v", ok, err)
	}
	other, _ := LoadOrCreateKeyPair(filepath.Join(t.TempDir(), "o.json"), "o")
	if ok, _ := ds.VerifyFrame(frames[0].ID, other); ok {
		t.Fatal("wrong key must not verify")
	}
	if _, err := ds.FrameSignature("nope"); err == nil {
		t.Fatal("unknown frame must error")
	}
}
