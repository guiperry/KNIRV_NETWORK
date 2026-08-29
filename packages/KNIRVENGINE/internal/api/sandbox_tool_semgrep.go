package api

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

// SemgrepFinding matches the shape Semgrep.tsx already defines.
type SemgrepFinding struct {
	ID       string `json:"id"`
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	HasFix   bool   `json:"hasFix,omitempty"`
}

// semgrepJSONResult is the subset of Semgrep's --json output we parse.
type semgrepJSONResult struct {
	Results []struct {
		CheckID string `json:"check_id"`
		Path    string `json:"path"`
		Start   struct {
			Line int `json:"line"`
		} `json:"start"`
		Extra struct {
			Message  string `json:"message"`
			Severity string `json:"severity"`
			Metadata struct {
				Fixable bool `json:"fixable"`
			} `json:"metadata"`
		} `json:"extra"`
	} `json:"results"`
}

func init() {
	registerLane1Tool("semgrep", toolScanAdapter{
		binary: "semgrep",
		buildArgs: func(session *SandboxSession, args json.RawMessage) ([]string, error) {
			ruleset := "p/owasp-top-ten p/secrets p/golang"
			targetDir := mountedTargetDir(session)
			if args != nil {
				var req struct {
					Ruleset   string `json:"ruleset"`
					TargetDir string `json:"targetDir"`
				}
				if err := json.Unmarshal(args, &req); err == nil {
					if req.Ruleset != "" {
						ruleset = req.Ruleset
					}
					if req.TargetDir != "" {
						targetDir = req.TargetDir
					}
				}
			}
			return []string{"--config", ruleset, "--json", targetDir}, nil
		},
		parseOutput: func(stdout []byte) (json.RawMessage, error) {
			var sr semgrepJSONResult
			if err := json.Unmarshal(stdout, &sr); err != nil {
				return nil, fmt.Errorf("failed to parse semgrep JSON: %v", err)
			}
			findings := make([]SemgrepFinding, 0, len(sr.Results))
			for i, r := range sr.Results {
				f := SemgrepFinding{
					ID:       fmt.Sprintf("semgrep-%d", i),
					RuleID:   r.CheckID,
					Severity: r.Extra.Severity,
					File:     r.Path,
					Line:     r.Start.Line,
					Message:  r.Extra.Message,
					HasFix:   r.Extra.Metadata.Fixable,
				}
				// Make file path relative if it's absolute.
				if rel, err := filepath.Rel("", r.Path); err == nil && len(rel) < len(r.Path) {
					f.File = rel
				}
				findings = append(findings, f)
			}
			return json.Marshal(findings)
		},
	})
}
