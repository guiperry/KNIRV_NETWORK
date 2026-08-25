package nrv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"knirvclient/internal/config"
	"knirvclient/types"
)

type stubCodeContext struct{}

func (stubCodeContext) GetCodeContext(filePath string, lineNumber int, contextLines int) (string, error) {
	return "// stub context", nil
}

func TestReportAssessment_OnlyHighSeverityDebtAndAllVulns(t *testing.T) {
	var mu sync.Mutex
	var submittedTypes []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body ErrorNodeCommit
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		submittedTypes = append(submittedTypes, body.ErrorType)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	identity, err := LoadOrCreateIdentity(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}

	reporter := NewReporter(
		config.NRVConfig{Enable: true, ContextLines: 5},
		identity,
		NewClient(server.URL),
		stubCodeContext{},
	)

	assessment := &types.RiskAssessment{
		SecurityVulns: []types.SecurityVulnerability{
			{ID: "v1", Type: "sql_injection", Severity: "critical", Description: "d", FilePath: "a.go"},
		},
		TechnicalDebt: []types.TechnicalDebtItem{
			{ID: "d1", Type: "complexity", Severity: "high", Description: "d", FilePath: "b.go"},
			{ID: "d2", Type: "todo", Severity: "low", Description: "d", FilePath: "c.go"},
		},
	}

	outDir := t.TempDir()
	reporter.ReportAssessment(context.Background(), "proj-1", "sess-1", t.TempDir(), outDir, assessment)

	mu.Lock()
	defer mu.Unlock()
	if len(submittedTypes) != 2 {
		t.Fatalf("submitted %d commits, want 2 (1 vuln + 1 high-severity debt item): %v", len(submittedTypes), submittedTypes)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("failed to read outDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("wrote %d .nrv files, want 2", len(entries))
	}
}

func TestReportAssessment_DisabledSkipsEverything(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	identity, err := LoadOrCreateIdentity(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}

	reporter := NewReporter(config.NRVConfig{Enable: false}, identity, NewClient(server.URL), stubCodeContext{})

	assessment := &types.RiskAssessment{
		SecurityVulns: []types.SecurityVulnerability{{ID: "v1", Type: "sql_injection", Severity: "critical"}},
	}
	reporter.ReportAssessment(context.Background(), "proj-1", "sess-1", t.TempDir(), t.TempDir(), assessment)

	if called {
		t.Fatal("expected no submission when NRVConfig.Enable is false")
	}
}
