package nrv

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"knirvclient/internal/config"
	"knirvclient/types"
)

// CodeContextProvider supplies surrounding source lines for a file/line —
// satisfied by guardian.ChromemManager (only this one method is needed
// here, so nrv doesn't need to import the guardian package).
type CodeContextProvider interface {
	GetCodeContext(filePath string, lineNumber int, contextLines int) (string, error)
}

// Reporter packages detected issues from a types.RiskAssessment into .nrv
// files and submits them to KNIRVGRAPH.
type Reporter struct {
	cfg         config.NRVConfig
	identity    *Identity
	client      *Client
	codeContext CodeContextProvider
}

// NewReporter constructs a Reporter. codeContext may be nil, in which case
// packages are built without surrounding source (code_context left empty).
func NewReporter(cfg config.NRVConfig, identity *Identity, client *Client, codeContext CodeContextProvider) *Reporter {
	return &Reporter{cfg: cfg, identity: identity, client: client, codeContext: codeContext}
}

func (r *Reporter) codeContextFor(filePath string, line int) string {
	if r.codeContext == nil {
		return ""
	}
	snippet, err := r.codeContext.GetCodeContext(filePath, line, r.cfg.ContextLines)
	if err != nil {
		return ""
	}
	return snippet
}

func isHighSeverity(severity string) bool {
	switch strings.ToLower(severity) {
	case "critical", "high":
		return true
	default:
		return false
	}
}

type reportJob struct {
	build func() (*ErrorNodeCommit, error)
	desc  string
}

// ReportAssessment packages and submits every security vulnerability, plus
// every critical/high-severity technical debt item, found in assessment.
// This is entirely best-effort: failures are logged, never returned —
// reporting to KNIRVGRAPH must never abort a scan cycle.
func (r *Reporter) ReportAssessment(ctx context.Context, projectID, sessionID, projectPath, outDir string, assessment *types.RiskAssessment) {
	if !r.cfg.Enable || assessment == nil {
		return
	}

	var jobs []reportJob

	for _, v := range assessment.SecurityVulns {
		v := v
		jobs = append(jobs, reportJob{
			build: func() (*ErrorNodeCommit, error) {
				return BuildFromSecurityVuln(v, BuildParams{
					ProjectID:   projectID,
					SessionID:   sessionID,
					ProjectPath: projectPath,
					CodeContext: r.codeContextFor(v.FilePath, v.LineNumber),
				})
			},
			desc: fmt.Sprintf("security_vulnerability %s (%s)", v.ID, v.FilePath),
		})
	}

	for _, d := range assessment.TechnicalDebt {
		if !isHighSeverity(d.Severity) {
			continue
		}
		d := d
		jobs = append(jobs, reportJob{
			build: func() (*ErrorNodeCommit, error) {
				return BuildFromTechnicalDebt(d, BuildParams{
					ProjectID:   projectID,
					SessionID:   sessionID,
					ProjectPath: projectPath,
					CodeContext: r.codeContextFor(d.FilePath, d.LineNumber),
				})
			},
			desc: fmt.Sprintf("technical_debt %s (%s)", d.ID, d.FilePath),
		})
	}

	if len(jobs) == 0 {
		return
	}

	log.Printf("📦 nrv: packaging %d issue(s) for KNIRVGRAPH submission", len(jobs))

	const maxConcurrent = 4
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, j := range jobs {
		j := j
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			r.reportOne(ctx, outDir, j)
		}()
	}
	wg.Wait()
}

func (r *Reporter) reportOne(ctx context.Context, outDir string, j reportJob) {
	commit, err := j.build()
	if err != nil {
		log.Printf("⚠️  nrv: failed to build package for %s: %v", j.desc, err)
		return
	}
	if err := Sign(commit, r.identity, r.cfg.ResolvedChainID()); err != nil {
		log.Printf("⚠️  nrv: failed to sign package for %s: %v", j.desc, err)
		return
	}

	if path, err := WritePackage(outDir, commit); err != nil {
		log.Printf("⚠️  nrv: failed to write package for %s: %v", j.desc, err)
	} else {
		log.Printf("📦 nrv: packaged %s -> %s", j.desc, path)
	}

	if err := r.client.SubmitErrorCommit(ctx, commit); err != nil {
		log.Printf("⚠️  nrv: failed to submit %s to KNIRVGRAPH: %v", j.desc, err)
		return
	}
	log.Printf("✅ nrv: submitted %s to KNIRVGRAPH", j.desc)
}
