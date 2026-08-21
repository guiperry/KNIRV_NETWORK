package dveevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// NewSubmissionCommitment derives server-verifiable content commitments. Raw
// proof bytes never enter the commitment or chain payload.
func NewSubmissionCommitment(submissionID, researcherCommitment, poc, report, scope, riskClassID string) (SubmissionCommitment, error) {
	if submissionID == "" || researcherCommitment == "" || poc == "" || report == "" || scope == "" || riskClassID == "" {
		return SubmissionCommitment{}, fmt.Errorf("submission commitment requires all fields")
	}
	hash := func(v string) string { sum := sha256.Sum256([]byte(v)); return hex.EncodeToString(sum[:]) }
	pocHash, reportHash, scopeHash := hash(poc), hash(report), hash(scope)
	return SubmissionCommitment{SchemaVersion: "dve.submission.v1", SubmissionID: submissionID, ResearcherCommitment: researcherCommitment, PoCHash: pocHash, ReportHash: reportHash, ScopeHash: scopeHash, DedupeFingerprint: hash(pocHash+reportHash+scopeHash+riskClassID), RiskClassID: riskClassID, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}
