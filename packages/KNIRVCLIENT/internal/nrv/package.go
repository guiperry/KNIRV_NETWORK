package nrv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"knirvclient/types"
)

// SchemaVersion must match KNIRVGRAPH's ErrorNodeCommitSchema
// (packages/KNIRVGRAPH/internal/network/rpc.go).
const SchemaVersion = "knirv.error-node-commit.v1"

const toolName = "knirvclient"

// ErrorNodeCommit mirrors KNIRVGRAPH's rpc.ErrorNodeCommit struct field for
// field so its JSON encoding — and therefore the ErrorRoot hash computed
// over it — matches what the server independently recomputes.
type ErrorNodeCommit struct {
	SchemaVersion string                 `json:"schema_version"`
	ProjectID     string                 `json:"project_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	ErrorType     string                 `json:"error_type"`
	Description   string                 `json:"description"`
	Context       map[string]interface{} `json:"context"`
	Severity      int                    `json:"severity"`
	ErrorRoot     string                 `json:"error_root"`
	SignerID      string                 `json:"signer_id"`
	SigningKeyID  string                 `json:"signing_key_id"`
	Signature     string                 `json:"signature"`
}

// errorNodeCommitClaim is the subset of fields the ErrorRoot hash is
// computed over — must match KNIRVGRAPH's errorNodeCommitClaim exactly.
type errorNodeCommitClaim struct {
	ErrorType   string                 `json:"error_type"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Severity    int                    `json:"severity"`
}

// ComputeErrorRoot reimplements KNIRVGRAPH's errorNodeCommitRoot so the
// client can present a root the server will independently recompute and
// accept.
func ComputeErrorRoot(errorType, description string, context map[string]interface{}, severity int) (string, error) {
	canonical, err := json.Marshal(errorNodeCommitClaim{
		ErrorType:   errorType,
		Description: description,
		Context:     context,
		Severity:    severity,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal error commit claim: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// severityScore maps KNIRVCLIENT's string severity vocabulary
// ("critical"|"high"|"medium"|"low", see internal/risk/diagnoser.go) onto
// KNIRVGRAPH's 1-10 integer scale.
func severityScore(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 9
	case "high":
		return 7
	case "medium":
		return 5
	case "low":
		return 2
	default:
		return 5
	}
}

// gitCommit best-effort resolves the current HEAD commit for projectPath.
// Returns "" on any failure — this is enrichment, never a hard requirement.
func gitCommit(projectPath string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// BuildParams carries the fields common to every .nrv package regardless of
// which kind of issue it was built from.
type BuildParams struct {
	ProjectID   string
	SessionID   string
	ProjectPath string
	CodeContext string
}

// BuildFromSecurityVuln packages a types.SecurityVulnerability into an
// ErrorNodeCommit ready to sign and submit.
func BuildFromSecurityVuln(v types.SecurityVulnerability, p BuildParams) (*ErrorNodeCommit, error) {
	context := map[string]interface{}{
		"file_path":    v.FilePath,
		"line_number":  v.LineNumber,
		"code_context": p.CodeContext,
		"tool":         toolName,
		"cve":          v.CVE,
		"package":      v.Package,
		"version":      v.Version,
		"cvss":         v.CVSS,
		"fix_version":  v.FixVersion,
	}
	if commit := gitCommit(p.ProjectPath); commit != "" {
		context["git_commit"] = commit
	}

	errorType := "security_vulnerability:" + v.Type
	severity := severityScore(v.Severity)

	root, err := ComputeErrorRoot(errorType, v.Description, context, severity)
	if err != nil {
		return nil, err
	}

	return &ErrorNodeCommit{
		SchemaVersion: SchemaVersion,
		ProjectID:     p.ProjectID,
		SessionID:     p.SessionID,
		ErrorType:     errorType,
		Description:   v.Description,
		Context:       context,
		Severity:      severity,
		ErrorRoot:     root,
	}, nil
}

// BuildFromTechnicalDebt packages a types.TechnicalDebtItem into an
// ErrorNodeCommit ready to sign and submit.
func BuildFromTechnicalDebt(d types.TechnicalDebtItem, p BuildParams) (*ErrorNodeCommit, error) {
	context := map[string]interface{}{
		"file_path":    d.FilePath,
		"line_number":  d.LineNumber,
		"code_context": p.CodeContext,
		"tool":         toolName,
		"effort":       d.Effort,
		"remediation":  d.Remediation,
	}
	if commit := gitCommit(p.ProjectPath); commit != "" {
		context["git_commit"] = commit
	}

	errorType := "technical_debt:" + d.Type
	severity := severityScore(d.Severity)

	root, err := ComputeErrorRoot(errorType, d.Description, context, severity)
	if err != nil {
		return nil, err
	}

	return &ErrorNodeCommit{
		SchemaVersion: SchemaVersion,
		ProjectID:     p.ProjectID,
		SessionID:     p.SessionID,
		ErrorType:     errorType,
		Description:   d.Description,
		Context:       context,
		Severity:      severity,
		ErrorRoot:     root,
	}, nil
}

// Sign fills in the commit's signer identity fields using id, producing a
// KNIRVSDK-canonical signed message over the commit's ErrorRoot for chainID.
func Sign(commit *ErrorNodeCommit, id *Identity, chainID string) error {
	commit.SignerID = id.SignerID()
	commit.SigningKeyID = id.SigningKeyID()
	signature, err := id.Sign(chainID, commit.ErrorRoot)
	if err != nil {
		return err
	}
	commit.Signature = signature
	return nil
}

var unsafeFileChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// WritePackage writes commit as an indented JSON .nrv file under dir and
// returns the path written. This file *is* the .nrv package — identical to
// what gets submitted to KNIRVGRAPH, kept locally for inspection/audit.
func WritePackage(dir string, commit *ErrorNodeCommit) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create nrv package directory: %w", err)
	}

	safeType := unsafeFileChars.ReplaceAllString(commit.ErrorType, "_")
	shortRoot := strings.TrimPrefix(commit.ErrorRoot, "sha256:")
	if len(shortRoot) > 12 {
		shortRoot = shortRoot[:12]
	}
	fileName := fmt.Sprintf("%d-%s-%s.nrv", time.Now().UnixNano(), safeType, shortRoot)
	path := filepath.Join(dir, fileName)

	data, err := json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal nrv package: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write nrv package: %w", err)
	}
	return path, nil
}
