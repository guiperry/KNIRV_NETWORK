package nrv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"knirvclient/types"
)

// wireClaim independently mirrors the exact wire shape KNIRVGRAPH's
// errorNodeCommitClaim expects, so this test fails if ComputeErrorRoot ever
// drifts from that contract (field names, order, or hash algorithm).
type wireClaim struct {
	ErrorType   string                 `json:"error_type"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Severity    int                    `json:"severity"`
}

func TestComputeErrorRoot_MatchesServerContract(t *testing.T) {
	context := map[string]interface{}{"file_path": "main.go", "line_number": float64(42)}

	canonical, err := json.Marshal(wireClaim{
		ErrorType:   "security_vulnerability:sql_injection",
		Description: "possible SQL injection",
		Context:     context,
		Severity:    9,
	})
	if err != nil {
		t.Fatalf("failed to marshal wire claim: %v", err)
	}
	sum := sha256.Sum256(canonical)
	want := "sha256:" + hex.EncodeToString(sum[:])

	got, err := ComputeErrorRoot("security_vulnerability:sql_injection", "possible SQL injection", context, 9)
	if err != nil {
		t.Fatalf("ComputeErrorRoot returned error: %v", err)
	}

	if got != want {
		t.Fatalf("ComputeErrorRoot() = %q, want %q", got, want)
	}
}

func TestComputeErrorRoot_Deterministic(t *testing.T) {
	context := map[string]interface{}{"b": 2, "a": 1}
	root1, err := ComputeErrorRoot("t", "d", context, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	root2, err := ComputeErrorRoot("t", "d", context, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root1 != root2 {
		t.Fatalf("ComputeErrorRoot is not deterministic: %q != %q", root1, root2)
	}
}

func TestBuildFromSecurityVuln(t *testing.T) {
	vuln := types.SecurityVulnerability{
		ID:          "vuln-1",
		Type:        "sql_injection",
		FilePath:    "main.go",
		LineNumber:  42,
		CVE:         "CVE-2024-0001",
		Package:     "database/sql",
		Version:     "1.0.0",
		Severity:    "critical",
		Description: "possible SQL injection",
		FixVersion:  "1.0.1",
		CVSS:        9.8,
	}

	commit, err := BuildFromSecurityVuln(vuln, BuildParams{
		ProjectID:   "proj-1",
		SessionID:   "sess-1",
		ProjectPath: t.TempDir(), // not a git repo — git_commit enrichment silently absent
		CodeContext: "func main() {}",
	})
	if err != nil {
		t.Fatalf("BuildFromSecurityVuln returned error: %v", err)
	}

	if commit.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", commit.SchemaVersion, SchemaVersion)
	}
	if commit.ErrorType != "security_vulnerability:sql_injection" {
		t.Errorf("ErrorType = %q", commit.ErrorType)
	}
	if commit.Severity != 9 {
		t.Errorf("Severity = %d, want 9 for critical", commit.Severity)
	}
	if commit.Context["cve"] != "CVE-2024-0001" {
		t.Errorf("Context[cve] = %v", commit.Context["cve"])
	}
	if commit.Context["code_context"] != "func main() {}" {
		t.Errorf("Context[code_context] = %v", commit.Context["code_context"])
	}
	if commit.ErrorRoot == "" {
		t.Error("ErrorRoot must not be empty")
	}
	if commit.SignerID != "" || commit.Signature != "" {
		t.Error("BuildFromSecurityVuln must not fill in signer identity — that's Sign()'s job")
	}
}

func TestBuildFromTechnicalDebt_SeverityMapping(t *testing.T) {
	cases := []struct {
		severity string
		want     int
	}{
		{"critical", 9},
		{"high", 7},
		{"medium", 5},
		{"low", 2},
		{"unknown", 5},
	}
	for _, tc := range cases {
		item := types.TechnicalDebtItem{Severity: tc.severity, Description: "d", Type: "t"}
		commit, err := BuildFromTechnicalDebt(item, BuildParams{ProjectPath: t.TempDir()})
		if err != nil {
			t.Fatalf("BuildFromTechnicalDebt(%q) returned error: %v", tc.severity, err)
		}
		if commit.Severity != tc.want {
			t.Errorf("severity %q -> %d, want %d", tc.severity, commit.Severity, tc.want)
		}
	}
}

func TestSign(t *testing.T) {
	dir := t.TempDir()
	id, err := LoadOrCreateIdentity(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity: %v", err)
	}

	commit := &ErrorNodeCommit{ErrorRoot: "sha256:deadbeef"}
	if err := Sign(commit, id, "knirv-testnet-1"); err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}

	if commit.SignerID != id.SignerID() {
		t.Errorf("SignerID = %q, want %q", commit.SignerID, id.SignerID())
	}
	if commit.SigningKeyID != id.SigningKeyID() {
		t.Errorf("SigningKeyID = %q, want %q", commit.SigningKeyID, id.SigningKeyID())
	}
	if commit.Signature == "" {
		t.Error("Signature must not be empty after Sign()")
	}
}

func TestWritePackage(t *testing.T) {
	dir := t.TempDir()
	commit := &ErrorNodeCommit{
		SchemaVersion: SchemaVersion,
		ErrorType:     "security_vulnerability:sql_injection",
		ErrorRoot:     "sha256:abc123",
	}

	path, err := WritePackage(dir, commit)
	if err != nil {
		t.Fatalf("WritePackage returned error: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("WritePackage wrote outside dir: %q", path)
	}
	if filepath.Ext(path) != ".nrv" {
		t.Errorf("WritePackage extension = %q, want .nrv", filepath.Ext(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written package: %v", err)
	}
	var roundTripped ErrorNodeCommit
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("written package is not valid ErrorNodeCommit JSON: %v", err)
	}
	if roundTripped.ErrorRoot != commit.ErrorRoot {
		t.Errorf("round-tripped ErrorRoot = %q, want %q", roundTripped.ErrorRoot, commit.ErrorRoot)
	}
}
