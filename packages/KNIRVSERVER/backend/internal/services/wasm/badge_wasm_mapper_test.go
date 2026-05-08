// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package wasm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── BadgeWASMMapper Tests ──────────────────────────────────────────────

func TestBadgeWASMMapper_RegisterBadge(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	tags := []string{
		"guardrail:policy:sandbox",
		"resolution:error:recovery",
		"resolution:restart",
		"patch:update",
		"unknown:tag",
	}

	err := mapper.RegisterBadge("badge-001", tags)
	if err != nil {
		t.Fatalf("RegisterBadge failed: %v", err)
	}

	entry, ok := mapper.GetEntry("badge-001")
	if !ok {
		t.Fatal("badge entry not found after registration")
	}

	if len(entry.Mappings) != len(tags) {
		t.Fatalf("expected %d mappings, got %d", len(tags), len(entry.Mappings))
	}

	// Verify tag classifications.
	for _, mp := range entry.Mappings {
		switch mp.Tag {
		case "guardrail:policy:sandbox":
			if mp.WASMType != WASMTypeRule {
				t.Errorf("tag %s: expected WASMType rule, got %s", mp.Tag, mp.WASMType)
			}
		case "resolution:error:recovery", "resolution:restart":
			if mp.WASMType != WASMTypeResolution {
				t.Errorf("tag %s: expected WASMType resolution, got %s", mp.Tag, mp.WASMType)
			}
		case "patch:update":
			if mp.WASMType != WASMTypePatch {
				t.Errorf("tag %s: expected WASMType patch, got %s", mp.Tag, mp.WASMType)
			}
		case "unknown:tag":
			if mp.WASMType != WASMTypeRule {
				t.Errorf("tag %s: expected WASMType rule (default), got %s", mp.Tag, mp.WASMType)
			}
		}
	}
}

func TestBadgeWASMMapper_LookupWASM(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	// Create a dummy .wasm file so lookup can find it.
	dummyWasm := filepath.Join(tmpDir, "rule_test.wasm")
	if err := os.WriteFile(dummyWasm, []byte{0x00, 0x61, 0x73, 0x6d}, 0644); err != nil {
		t.Fatalf("create dummy wasm: %v", err)
	}

	err := mapper.RegisterBadge("badge-002", []string{"guardrail:sandbox"})
	if err != nil {
		t.Fatalf("RegisterBadge failed: %v", err)
	}

	// Should find the dummy .wasm.
	wasmPath, wasmType, err := mapper.LookupWASM("badge-002", "guardrail:sandbox")
	if err != nil {
		t.Fatalf("LookupWASM failed: %v", err)
	}
	if wasmPath != dummyWasm {
		t.Errorf("expected wasm path %s, got %s", dummyWasm, wasmPath)
	}
	if wasmType != WASMTypeRule {
		t.Errorf("expected WASMType rule, got %s", wasmType)
	}

	// Non-existent badge.
	_, _, err = mapper.LookupWASM("badge-999", "guardrail:sandbox")
	if err == nil {
		t.Error("expected error for non-existent badge")
	}

	// Non-existent tag.
	_, _, err = mapper.LookupWASM("badge-002", "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent tag")
	}
}

func TestBadgeWASMMapper_RemoveBadge(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	mapper.RegisterBadge("badge-003", []string{"guardrail:test"})
	mapper.RemoveBadge("badge-003")

	_, ok := mapper.GetEntry("badge-003")
	if ok {
		t.Error("badge should have been removed")
	}
}

func TestBadgeWASMMapper_GetGuardrailAndResolutionWASM(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	// Create dummy .wasm files.
	ruleWasm := filepath.Join(tmpDir, "rule_guard.wasm")
	resWasm := filepath.Join(tmpDir, "resolution_rec.wasm")
	os.WriteFile(ruleWasm, []byte{0x00, 0x61, 0x73, 0x6d}, 0644)
	os.WriteFile(resWasm, []byte{0x00, 0x61, 0x73, 0x6d}, 0644)

	err := mapper.RegisterBadge("badge-004", []string{
		"guardrail:policy",
		"resolution:error",
	})
	if err != nil {
		t.Fatalf("RegisterBadge failed: %v", err)
	}

	guardrails := mapper.GetGuardrailWASM("badge-004")
	if len(guardrails) != 1 {
		t.Errorf("expected 1 guardrail mapping, got %d", len(guardrails))
	} else if guardrails[0].WASMType != WASMTypeRule {
		t.Errorf("expected rule type, got %s", guardrails[0].WASMType)
	}

	resolutions := mapper.GetResolutionWASM("badge-004")
	if len(resolutions) != 1 {
		t.Errorf("expected 1 resolution mapping, got %d", len(resolutions))
	} else if resolutions[0].WASMType != WASMTypeResolution {
		t.Errorf("expected resolution type, got %s", resolutions[0].WASMType)
	}
}

func TestBadgeWASMMapper_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "wasm-mappings.json")

	// Create mapper with persistence.
	mapper1 := NewBadgeWASMMapper(tmpDir, persistPath)
	mapper1.RegisterBadge("badge-persist", []string{"guardrail:compliance"})

	// Create a second mapper that should load from disk.
	mapper2 := NewBadgeWASMMapper(tmpDir, persistPath)
	entry, ok := mapper2.GetEntry("badge-persist")
	if !ok {
		t.Fatal("persisted badge not loaded")
	}
	if entry.BadgeID != "badge-persist" {
		t.Errorf("expected badge ID 'badge-persist', got '%s'", entry.BadgeID)
	}
}

func TestBadgeWASMMapper_RefreshWASMDir(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	// Register badge when no WASM exists yet.
	err := mapper.RegisterBadge("badge-refresh", []string{"resolution:heal"})
	if err != nil {
		t.Fatalf("RegisterBadge failed: %v", err)
	}

	entry, _ := mapper.GetEntry("badge-refresh")
	if entry.Mappings[0].WASMPath != "" {
		t.Log("WASM already present — skipping empty check")
	}

	// Now create a WASM file.
	newWasm := filepath.Join(tmpDir, "resolution_heal.wasm")
	os.WriteFile(newWasm, []byte{0x00, 0x61, 0x73, 0x6d}, 0644)

	mapper.RefreshWASMDir()

	entry, _ = mapper.GetEntry("badge-refresh")
	if entry.Mappings[0].WASMPath == "" {
		t.Error("expected WASM path to be populated after refresh")
	}
}

func TestBadgeWASMMapper_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	ruleWasm := filepath.Join(tmpDir, "rule_stat.wasm")
	resWasm := filepath.Join(tmpDir, "resolution_stat.wasm")
	os.WriteFile(ruleWasm, []byte{0x00, 0x61, 0x73, 0x6d}, 0644)
	os.WriteFile(resWasm, []byte{0x00, 0x61, 0x73, 0x6d}, 0644)

	mapper.RegisterBadge("badge-stat-1", []string{"guardrail:audit", "resolution:heal"})
	mapper.RegisterBadge("badge-stat-2", []string{"guardrail:lsm"})

	stats := mapper.Stats()
	badges := stats["total_badges"].(int)
	if badges != 2 {
		t.Errorf("expected 2 badges in stats, got %d", badges)
	}
}

func TestBadgeWASMMapper_ListBadges(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	mapper.RegisterBadge("badge-a", []string{"guardrail:x"})
	mapper.RegisterBadge("badge-b", []string{"resolution:y"})

	ids := mapper.ListBadges()
	if len(ids) != 2 {
		t.Errorf("expected 2 badges, got %d", len(ids))
	}

	found := make(map[string]bool)
	for _, id := range ids {
		found[id] = true
	}
	if !found["badge-a"] || !found["badge-b"] {
		t.Error("expected both badge IDs in list")
	}
}

func TestBadgeWASMMapper_DefaultWASMDir(t *testing.T) {
	dir := DefaultWASMDir()
	if dir == "" {
		t.Error("DefaultWASMDir should not be empty")
	}
}

// ── Tag Classification Tests ──────────────────────────────────────────

func TestClassifyTag_Guardrail(t *testing.T) {
	tags := []string{
		"guardrail:sandbox",
		"policy:network",
		"compliance:audit",
		"lsm:restrict",
		"seccomp:block",
		"ontology:guard:file",
	}
	for _, tag := range tags {
		class := classifyTag(tag)
		if class != TagClassGuardrail {
			t.Errorf("tag %q: expected guardrail, got %s", tag, class)
		}
	}
}

func TestClassifyTag_Resolution(t *testing.T) {
	tags := []string{
		"resolution:error:epipe",
		"resolve:oom",
		"recovery:panic",
		"error:enoent",
		"remediation:restart",
		"heal:service",
		"self-heal:container",
		"restart:node",
		"rollback:config",
		"ontology:resolve:crash",
	}
	for _, tag := range tags {
		class := classifyTag(tag)
		if class != TagClassResolution {
			t.Errorf("tag %q: expected resolution, got %s", tag, class)
		}
	}
}

func TestClassifyTag_Patch(t *testing.T) {
	tags := []string{
		"patch:binary",
		"update:library",
		"hotfix:critical",
		"migrate:schema",
	}
	for _, tag := range tags {
		class := classifyTag(tag)
		if class != TagClassPatch {
			t.Errorf("tag %q: expected patch, got %s", tag, class)
		}
	}
}

func TestClassifyTag_Unknown(t *testing.T) {
	class := classifyTag("random:nonsense")
	if class != TagClassUnknown {
		t.Errorf("expected unknown, got %s", class)
	}
}

func TestClassToWASMType(t *testing.T) {
	if classToWASMType(TagClassGuardrail) != WASMTypeRule {
		t.Error("guardrail → rule mapping wrong")
	}
	if classToWASMType(TagClassResolution) != WASMTypeResolution {
		t.Error("resolution → resolution mapping wrong")
	}
	if classToWASMType(TagClassPatch) != WASMTypePatch {
		t.Error("patch → patch mapping wrong")
	}
	if classToWASMType(TagClassUnknown) != WASMTypeRule {
		t.Error("unknown → rule fallback wrong")
	}
}

// ── ResolutionService Tests ───────────────────────────────────────────

type mockRemediationSink struct {
	dispatched []*ResolutionResult
}

func (m *mockRemediationSink) DispatchRemediation(ctx context.Context, result *ResolutionResult) error {
	m.dispatched = append(m.dispatched, result)
	return nil
}

func TestResolutionService_SubmitAndStats(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	sink := &mockRemediationSink{}
	svc := NewResolutionService(mapper, sink, 2, 16)

	// Submit should succeed when buffer has room.
	signal := &ResolutionSignal{
		DVEID:     "dve-1",
		NodeID:    "node-1",
		BadgeID:   "badge-1",
		Tag:       "resolution:error",
		ErrorType: "ENOENT",
	}
	if !svc.Submit(signal) {
		t.Error("expected submit to succeed")
	}

	stats := svc.Stats()
	if stats.TotalSignals != 1 {
		t.Errorf("expected 1 total signal, got %d", stats.TotalSignals)
	}

	svc.Stop()
}

func TestResolutionService_ResolveWithoutWASM(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")

	// Register badge but no WASM file exists.
	mapper.RegisterBadge("badge-no-wasm", []string{"resolution:test"})

	sink := &mockRemediationSink{}
	svc := NewResolutionService(mapper, sink, 2, 16)

	signal := &ResolutionSignal{
		DVEID:   "dve-1",
		NodeID:  "node-1",
		BadgeID: "badge-no-wasm",
		Tag:     "resolution:test",
	}

	result, err := svc.Resolve(context.Background(), signal)
	if err == nil {
		t.Error("expected error when WASM is missing")
	}
	if result == nil {
		t.Fatal("expected non-nil result even on error")
	}
	if result.Error == "" {
		t.Error("expected error message in result")
	}

	stats := svc.Stats()
	if stats.WASMErrors == 0 {
		t.Error("expected WASM errors in stats")
	}

	svc.Stop()
}

func TestResolutionService_StartStop(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")
	sink := &mockRemediationSink{}
	svc := NewResolutionService(mapper, sink, 2, 8)

	ctx, cancel := context.WithCancel(context.Background())
	svc.Start(ctx, 2)

	time.Sleep(50 * time.Millisecond)
	cancel()
	svc.Stop()
}

// ── GuardrailMiddleware Tests ─────────────────────────────────────────

func TestGuardrailMiddleware_NoBadgeHeader(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")
	middleware := NewGuardrailMiddleware(mapper, 2)
	defer middleware.Close()

	// ValidateSession with empty badgeID should allow.
	allowed, class, err := middleware.ValidateSession(
		context.Background(), "dve-1", "", "")
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true when no badge")
	}
	if class != 0 {
		t.Errorf("expected class=0, got %d", class)
	}
}

func TestGuardrailMiddleware_MissingGuardrailWASM(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")
	mapper.RegisterBadge("badge-no-guard", []string{"resolution:test"})

	middleware := NewGuardrailMiddleware(mapper, 2)
	defer middleware.Close()

	// This badge has no guardrail tags → should allow.
	allowed, _, err := middleware.ValidateSession(
		context.Background(), "dve-1", "badge-no-guard", "")
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true when no guardrail WASM")
	}
}

func TestGuardrailMiddleware_Close(t *testing.T) {
	tmpDir := t.TempDir()
	mapper := NewBadgeWASMMapper(tmpDir, "")
	middleware := NewGuardrailMiddleware(mapper, 2)

	// Close should not panic even with no pools.
	middleware.Close()

	if len(middleware.pools) != 0 {
		t.Error("expected empty pools after close")
	}
}

// ── Wazero Pool Tests ─────────────────────────────────────────────────

func TestWazeroPool_NewAndClose(t *testing.T) {
	pool := NewWazeroPool("/nonexistent/path.wasm", 4)
	if pool == nil {
		t.Fatal("expected non-nil pool")
	}

	// CloseAll on empty pool should not panic.
	pool.CloseAll()
}

// ── Utility Tests ─────────────────────────────────────────────────────

func TestFileHash(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.bin")
	content := []byte("hello wasm world")
	os.WriteFile(tmpFile, content, 0644)

	hash, err := fileHash(tmpFile)
	if err != nil {
		t.Fatalf("fileHash failed: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d", len(hash))
	}
}

func TestFileHash_NonExistent(t *testing.T) {
	_, err := fileHash("/nonexistent/file.wasm")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestIsHex(t *testing.T) {
	if !isHex("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789") {
		t.Error("expected valid hex string")
	}
	if isHex("not-hex!") {
		t.Error("expected invalid hex string")
	}
	if isHex("") {
		t.Error("empty string should not be hex")
	}
}
