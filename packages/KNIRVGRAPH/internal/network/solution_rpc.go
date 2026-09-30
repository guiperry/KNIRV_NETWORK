package network

import (
	"KNIRVGRAPH/internal/drq"
	"KNIRVGRAPH/internal/economics"
	"KNIRVGRAPH/internal/nrv"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// solutionRecorder is implemented by the App when its DRQ loop is running.
type solutionRecorder interface {
	RecordGradedSolution(sol *drq.Solution) (string, error)
}

// solutionBaseReward is the resolution reward before the grade bonus (0.01 NRN
// in the ProofOfSolution ledger's base units, unchanged from before grading).
var solutionBaseReward = big.NewInt(10000000)

// submitSolutionProof serves POST /economics/proof/solution: a KNIRVARENA
// swarm agent submits its resolution of an error node, as outputs for the
// node's sealed 8-test suite — the same tests that gate the resolving
// skill's badge.
//
// KNIRVGRAPH grades the outputs itself. The previous contract paid NRN on
// caller-reported efficiency/quality scores with no check at all; now:
//   - the suite must be sealed (all 8 tests contributed),
//   - the solution is recorded in the error's DRQ cluster either way (so the
//     cluster's validation rate reflects failures too),
//   - only a solution passing all 8 is Validated, credits its agent, counts
//     toward MinValidatedSolutions, and earns the resolution reward.
//
// Internal-token gated: it moves NRN, and the per-test verdicts would
// otherwise make it an oracle for the suite's private expected values.
func (rpc *RPCServer) submitSolutionProof(w http.ResponseWriter, r *http.Request) {
	if !requireInternalToken(w, r) {
		return
	}
	store := rpc.errorTestStore(w)
	if store == nil {
		return
	}
	var req struct {
		ErrorNodeID string            `json:"error_node_id"`
		SkillNodeID string            `json:"skill_node_id,omitempty"`
		SolverID    string            `json:"solver_id"`
		CodePackage string            `json:"code_package,omitempty"`
		SuiteHash   string            `json:"suite_hash,omitempty"`
		Outputs     map[string]string `json:"outputs"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid solution submission")
		return
	}
	if strings.TrimSpace(req.ErrorNodeID) == "" || strings.TrimSpace(req.SolverID) == "" {
		writeJSONError(w, http.StatusBadRequest, "error_node_id and solver_id are required")
		return
	}

	suite, err := store.Get(req.ErrorNodeID)
	if errors.Is(err, nrv.ErrTestSuiteNotFound) {
		writeJSONError(w, http.StatusConflict, "this error node has no tests yet; solutions are graded once its 8 tests are contributed")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.SuiteHash != "" && req.SuiteHash != suite.SuiteHash {
		writeJSONError(w, http.StatusConflict, "the error node's test suite changed since this solution was produced")
		return
	}
	report, err := suite.GradeOutputs(req.Outputs)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}

	now := time.Now().UTC()
	idSum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%d", req.ErrorNodeID, req.SolverID, suite.SuiteHash, req.CodePackage, now.UnixNano())))
	solution := &drq.Solution{
		SolutionID:      "sol-" + hex.EncodeToString(idSum[:8]),
		ErrorID:         req.ErrorNodeID,
		AgentID:         req.SolverID,
		CodePackage:     req.CodePackage,
		ProposalTime:    now,
		Validated:       report.AllPassed,
		ValidationScore: float64(report.Passed) / float64(report.Total),
		// The attestation is the suite the grade was made against.
		DVEAttestation: []byte(suite.SuiteHash),
	}

	response := map[string]interface{}{
		"solution_id":   solution.SolutionID,
		"error_node_id": req.ErrorNodeID,
		"solver_id":     req.SolverID,
		"grade":         report,
		"validated":     report.AllPassed,
	}

	recorder, ok := rpc.app.(solutionRecorder)
	if !ok || rpc.app == nil {
		response["drq"] = "not recorded: DRQ is not running on this node"
	} else if clusterID, err := recorder.RecordGradedSolution(solution); err != nil {
		response["drq"] = "not recorded: " + err.Error()
	} else if clusterID == "" {
		response["drq"] = "queued: the error is not clustered yet"
	} else {
		response["cluster_id"] = clusterID
		response["drq"] = "recorded"
	}

	if report.AllPassed && rpc.proofOfSolution != nil {
		// Quality is the grade itself (1.0 here); efficiency has no measured
		// source, so it contributes nothing rather than a self-reported number.
		event := economics.ResolutionEvent{
			ErrorNodeID:  req.ErrorNodeID,
			SkillNodeID:  req.SkillNodeID,
			SolverID:     req.SolverID,
			QualityScore: solution.ValidationScore,
			RewardEarned: new(big.Int).Add(solutionBaseReward, big.NewInt(int64(solution.ValidationScore*100))),
		}
		if err := rpc.proofOfSolution.ProcessSuccessfulResolution(event); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "graded, but the reward could not be processed: "+err.Error())
			return
		}
		response["reward_earned"] = event.RewardEarned.String()
	}

	rpc.logger.Info("Swarm solution graded against error-node suite",
		zap.String("error_id", req.ErrorNodeID), zap.String("solver", req.SolverID),
		zap.Int("passed", report.Passed), zap.Int("total", report.Total), zap.Bool("validated", report.AllPassed))
	writeJSON(w, http.StatusOK, response)
}
