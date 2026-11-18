package drq

// End-to-end coverage for MintSkillFromCluster (R13 §4 unit 3).
//
// Every external collaborator is a loopback fake: KNIRVORACLE and KNIRVCHAIN are
// httptest servers bound to Unix sockets (the production clients dial those
// sockets via KNIRV_ORACLE_SOCKET_PATH / KNIRV_CHAIN_SOCKET_PATH), the skill tower
// is an in-memory store, the uLoRA compiler and validator are stubs, and the DHT
// announcement goes through a RoundTripper that refuses any host other than the
// gateway address it expects. No network is used.
//
// Rollback coverage: MintSkillFromCluster reverts the tower record for failures
// at steps 4 (verify), 5 (uLoRA), 6 (chain mint) and, since R13 Q44, the
// post-chain steps 7 (burn), 8 (ownership/bounty) and 9 (DHT announce).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const e2eInternalToken = "e2e-internal-token"

// e2eCalls records every external call in arrival order.
type e2eCalls struct {
	mu    sync.Mutex
	calls []e2eCall
}

type e2eCall struct {
	step string
	body map[string]any
}

func (c *e2eCalls) add(step string, body map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, e2eCall{step: step, body: body})
}

func (c *e2eCalls) steps() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.calls))
	for _, call := range c.calls {
		out = append(out, call.step)
	}
	return out
}

// bodies returns the bodies of every call with the given step, in arrival order.
func (c *e2eCalls) bodies(step string) []map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []map[string]any
	for _, call := range c.calls {
		if call.step == step {
			out = append(out, call.body)
		}
	}
	return out
}

// e2eTower is the in-memory SkillTowerStore.
type e2eTower struct {
	mu      sync.Mutex
	records map[string]bool
	putErr  error
	delErr  error
	calls   *e2eCalls
}

func (s *e2eTower) PutSkill(_ context.Context, skillID, skillType string, capabilities []string, attributes map[string]interface{}) error {
	s.calls.add("tower.put", map[string]any{
		"id":           skillID,
		"type":         skillType,
		"capabilities": capabilities,
		"attributes":   attributes,
	})
	if s.putErr != nil {
		return s.putErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[skillID] = true
	return nil
}

func (s *e2eTower) DeleteSkill(_ context.Context, skillID string) error {
	s.calls.add("tower.delete", map[string]any{"id": skillID})
	if s.delErr != nil {
		return s.delErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, skillID)
	return nil
}

func (s *e2eTower) SkillResolvingError(context.Context, string) (*SkillRecord, error) {
	return nil, nil
}

func (s *e2eTower) has(skillID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.records[skillID]
}

func (s *e2eTower) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

// e2eOracleHandler answers the /oracle/v3/skills/* routes. fail maps a route
// suffix (verify, burn, ownership, bounty) to an HTTP status to return instead.
func e2eOracleHandler(calls *e2eCalls, fail map[string]int, verified bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-KNIRV-Internal-Token") != e2eInternalToken {
			http.Error(w, "bad internal token", http.StatusUnauthorized)
			return
		}
		step := strings.TrimPrefix(r.URL.Path, "/oracle/v3/skills/")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls.add("oracle."+step, body)
		if code, ok := fail[step]; ok {
			http.Error(w, "injected failure", code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if step == "verify" {
			_ = json.NewEncoder(w).Encode(map[string]any{"verified": verified, "reason": "e2e"})
			return
		}
		_, _ = w.Write([]byte("{}"))
	})
}

// e2eChainHandler answers /api/v1/event-bundles/mint with a receipt for the
// requested event, as KNIRVCHAIN does. failStatus != 0 returns that status.
func e2eChainHandler(calls *e2eCalls, failStatus int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-KNIRV-Internal-Token") != e2eInternalToken {
			http.Error(w, "bad internal token", http.StatusUnauthorized)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls.add("chain.mint", body)
		if failStatus != 0 {
			http.Error(w, "injected failure", failStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": knirvchainEventBundleMintSchema,
			"event_id":       body["event_id"],
			"bundle_hash":    "e2e-bundle-hash",
			"token_id":       "e2e-token",
			"transaction_id": "e2e-tx",
			"final":          true,
		})
	})
}

// e2eDHTTransport stands in for KNIRVGATEWAY's /dht/cache-resource route. The
// DHT client dials http://localhost:8080 through http.DefaultTransport, so any
// other host is a test failure rather than a real request.
type e2eDHTTransport struct {
	calls  *e2eCalls
	status int
	t      *testing.T
}

func (d e2eDHTTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
	if req.URL.Host != "localhost:8080" {
		d.t.Errorf("unexpected network request to %s", req.URL.Host)
		return nil, fmt.Errorf("e2e: refused request to %s", req.URL.Host)
	}
	d.calls.add("dht.announce", map[string]any{"path": req.URL.Path})
	code := d.status
	if code == 0 {
		code = http.StatusOK
	}
	return &http.Response{
		StatusCode: code,
		Status:     http.StatusText(code),
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("{}")),
		Request:    req,
	}, nil
}

// e2eUnixServer serves handler on a fresh Unix socket and returns its path.
// The directory is short on purpose: sun_path is limited to ~108 bytes.
func e2eUnixServer(t *testing.T, name string, handler http.Handler) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "drq-e2e-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, name)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: handler}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return sock
}

// e2eCompiler is a ULoRACompiler stub that records the compile step.
type e2eCompiler struct {
	calls *e2eCalls
	err   error
}

func (c e2eCompiler) CompileCluster(_ context.Context, _ string, _ []ULoRADatasetRecord, _ []ULoRATargetModel, _ ULoRAProvenance) (*ULoRACompileResult, error) {
	c.calls.add("ulora.compile", nil)
	if c.err != nil {
		return nil, c.err
	}
	return &ULoRACompileResult{BundleID: "e2e-bundle", ContentHash: "e2e-content-hash", TargetModels: []string{"llama-3-8b"}}, nil
}

// e2eValidator is a BundleValidator stub that records the gate and returns err.
type e2eValidator struct {
	calls *e2eCalls
	err   error
}

func (v e2eValidator) ValidateBundle(context.Context, *ErrorCluster, *ULoRACompileResult, []ULoRADatasetRecord) error {
	v.calls.add("ulora.validate", nil)
	return v.err
}

// e2eDocs resolves every member error to a fixed skill document.
type e2eDocs struct{}

func (e2eDocs) SkillDocForError(errorID string) (string, error) {
	return "fix for " + errorID, nil
}

type e2eOpts struct {
	towerPutErr error
	towerDelErr error
	oracleFail  map[string]int
	verified    bool
	chainFail   int
	dhtStatus   int
	// withULoRA wires a compiler and validator into the protocol.
	withULoRA    bool
	compileErr   error
	validatorErr error
}

type e2eHarness struct {
	t       *testing.T
	calls   *e2eCalls
	tower   *e2eTower
	smp     *SkillMintingProtocol
	cluster *ErrorCluster
	// skillID is the deterministic ID DiscoverSkill assigns to the cluster.
	skillID string
}

// e2eCluster is the converged cluster every case mints from. agent-a owns it
// (3 of 4 solutions), so bounties split 300/100 of a 400 NRN pot.
func e2eCluster() *ErrorCluster {
	return &ErrorCluster{
		ClusterID:   "cluster-e2e",
		OwnerAgent:  "agent-a",
		TotalBounty: 400,
		AgentCounts: map[string]int{"agent-a": 3, "agent-b": 1},
		Errors: []*ErrorNode{
			{Id: "e2", ErrorType: "runtime", Domain: "runtime", Description: "nil pointer", ModelOrigin: "llama-3-8b"},
			{Id: "e1", ErrorType: "runtime", Domain: "runtime", Description: "index out of range", ModelOrigin: "llama-3-8b"},
		},
		Status: CLUSTER_ACTIVE,
	}
}

func newE2EHarness(t *testing.T, opts e2eOpts) *e2eHarness {
	t.Helper()
	calls := &e2eCalls{}

	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", e2eInternalToken)
	t.Setenv("KNIRV_SKILL_BURN_COST_NRN", "") // unset -> default 50 NRN
	oracleSock := e2eUnixServer(t, "oracle.sock", e2eOracleHandler(calls, opts.oracleFail, opts.verified))
	chainSock := e2eUnixServer(t, "chain.sock", e2eChainHandler(calls, opts.chainFail))
	t.Setenv("KNIRV_ORACLE_SOCKET_PATH", oracleSock)
	t.Setenv("KNIRV_CHAIN_SOCKET_PATH", chainSock)

	prev := http.DefaultTransport
	http.DefaultTransport = e2eDHTTransport{calls: calls, status: opts.dhtStatus, t: t}
	t.Cleanup(func() { http.DefaultTransport = prev })

	tower := &e2eTower{records: map[string]bool{}, putErr: opts.towerPutErr, delErr: opts.towerDelErr, calls: calls}
	deps := SkillMintingDeps{
		Graph:    NewKNIRVGRAPHClient(tower),
		Chain:    &KNIRVCHAINClient{},
		Oracle:   &KNIRVORACLEClient{},
		Discover: NewSkillDiscoveryEngine(nil),
	}
	if opts.withULoRA {
		deps.Compiler = e2eCompiler{calls: calls, err: opts.compileErr}
		deps.Validator = e2eValidator{calls: calls, err: opts.validatorErr}
		deps.SkillDocs = e2eDocs{}
		deps.TargetModels = map[string]ULoRATargetModel{
			"llama-3-8b": {Family: "llama", ParamCount: "8b", HiddenSize: 4096, IntermediateSize: 11008, NumLayers: 32, AttentionMechanism: "gqa", ActivationFunc: "silu"},
		}
		deps.ManifestVersion = "v1"
	}

	cluster := e2eCluster()
	id, err := NewSkillDiscoveryEngine(nil).DiscoverSkill(cluster)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	return &e2eHarness{
		t:       t,
		calls:   calls,
		tower:   tower,
		smp:     NewSkillMintingProtocol(deps),
		cluster: cluster,
		skillID: id.ID,
	}
}

func (h *e2eHarness) mint() (*SkillNode, error) {
	return h.smp.MintSkillFromCluster(context.Background(), h.cluster)
}

func expectSteps(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("call sequence:\n got  %v\n want %v", got, want)
	}
}

func strSlice(v any) []string {
	raw, _ := v.([]any)
	out := make([]string, 0, len(raw))
	for _, x := range raw {
		s, _ := x.(string)
		out = append(out, s)
	}
	return out
}

// TestMintSkillFromClusterEndToEnd: the happy path writes the tower record, verifies
// with KNIRVORACLE, mints the EventBundle on KNIRVCHAIN, burns, grants ownership,
// pays bounties and announces, in that order, with the right payloads.
func TestMintSkillFromClusterEndToEnd(t *testing.T) {
	h := newE2EHarness(t, e2eOpts{verified: true})
	skill, err := h.mint()
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if skill.ID != h.skillID {
		t.Fatalf("skill id = %q, want %q", skill.ID, h.skillID)
	}

	// Call order. The two bounty payments share one step name, so their order
	// (map iteration) does not matter.
	expectSteps(t, h.calls.steps(),
		"tower.put", "oracle.verify", "chain.mint", "oracle.burn",
		"oracle.ownership", "oracle.bounty", "oracle.bounty", "dht.announce")

	// Tower record.
	put := h.calls.bodies("tower.put")[0]
	if put["id"] != h.skillID || put["type"] != "mined" {
		t.Fatalf("tower put = %+v", put)
	}
	caps, _ := put["capabilities"].([]string)
	if !reflect.DeepEqual(caps, []string{"e1", "e2", "errors:e1,e2"}) {
		t.Fatalf("tower capabilities = %v", caps)
	}
	if !h.tower.has(h.skillID) {
		t.Fatal("tower record missing after successful mint")
	}

	// Verify.
	ver := h.calls.bodies("oracle.verify")[0]
	if ver["skill_id"] != h.skillID || ver["creator"] != "agent-a" {
		t.Fatalf("verify body = %+v", ver)
	}
	if !reflect.DeepEqual(strSlice(ver["resolves_errors"]), []string{"e1", "e2"}) {
		t.Fatalf("verify resolves_errors = %v", ver["resolves_errors"])
	}

	// EventBundle on KNIRVCHAIN.
	mint := h.calls.bodies("chain.mint")[0]
	if mint["schema_version"] != knirvchainEventBundleMintSchema ||
		mint["event_id"] != h.skillID || mint["minter_address"] != "agent-a" || mint["event_kind"] != "decision" {
		t.Fatalf("event bundle request = %+v", mint)
	}
	skills, _ := mint["skills"].([]any)
	if len(skills) != 1 || skills[0].(map[string]any)["id"] != h.skillID ||
		skills[0].(map[string]any)["ref"] != skill.CodePackageURI {
		t.Fatalf("event bundle skills = %+v, want ref %q", skills, skill.CodePackageURI)
	}
	var receipt knirvchainEventBundleReceipt
	if err := json.Unmarshal(skill.ValidationProof, &receipt); err != nil {
		t.Fatalf("validation proof is not a receipt: %v", err)
	}
	if receipt.EventID != h.skillID || receipt.BundleHash != "e2e-bundle-hash" || !receipt.Final {
		t.Fatalf("receipt = %+v", receipt)
	}

	// Burn.
	burn := h.calls.bodies("oracle.burn")[0]
	if burn["from"] != "agent-a" || burn["skill_id"] != h.skillID || burn["amount"] != "50" {
		t.Fatalf("burn body = %+v", burn)
	}

	// Ownership: fee = 10 * (1 + 0.1*2) = 12 NRN, perpetual.
	own := h.calls.bodies("oracle.ownership")[0]
	fee, _ := own["invocation_fee"].(float64)
	if own["agent_id"] != "agent-a" || own["skill_id"] != h.skillID || own["perpetual"] != true ||
		math.Abs(fee-12) > 1e-9 {
		t.Fatalf("ownership body = %+v", own)
	}

	// Bounty: 400 NRN split 3:1.
	got := map[string]string{}
	for _, b := range h.calls.bodies("oracle.bounty") {
		got[b["to"].(string)] = b["amount"].(string)
	}
	if !reflect.DeepEqual(got, map[string]string{"agent-a": "300", "agent-b": "100"}) {
		t.Fatalf("bounties = %v", got)
	}
}

// TestMintSkillRollbackOnFailure covers each failure step that must roll back the
// tower record, and the case where the rollback itself fails.
func TestMintSkillRollbackOnFailure(t *testing.T) {
	t.Run("discover_fails_nothing_written", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{verified: true})
		h.cluster.OwnerAgent = ""
		if _, err := h.mint(); err == nil {
			t.Fatal("expected discovery to fail without an owner")
		}
		expectSteps(t, h.calls.steps())
		if h.tower.count() != 0 {
			t.Fatal("tower written before discovery succeeded")
		}
	})

	t.Run("tower_put_fails_no_downstream_calls", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{verified: true, towerPutErr: errors.New("tower down")})
		if _, err := h.mint(); err == nil || !strings.Contains(err.Error(), "tower down") {
			t.Fatalf("err = %v, want tower failure", err)
		}
		expectSteps(t, h.calls.steps(), "tower.put")
		if h.tower.count() != 0 {
			t.Fatal("tower record present after failed put")
		}
	})

	t.Run("oracle_verify_http_error_rolls_back_tower", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{verified: true, oracleFail: map[string]int{"verify": http.StatusInternalServerError}})
		_, err := h.mint()
		if err == nil || !strings.Contains(err.Error(), "(rollback: <nil>)") {
			t.Fatalf("err = %v, want verify failure with clean rollback", err)
		}
		expectSteps(t, h.calls.steps(), "tower.put", "oracle.verify", "tower.delete")
		if h.tower.count() != 0 {
			t.Fatal("tower record survived verify failure")
		}
	})

	t.Run("oracle_rejects_skill_rolls_back_tower", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{verified: false})
		if _, err := h.mint(); err == nil || !strings.Contains(err.Error(), "rejected") {
			t.Fatalf("err = %v, want rejection", err)
		}
		expectSteps(t, h.calls.steps(), "tower.put", "oracle.verify", "tower.delete")
		if h.tower.count() != 0 {
			t.Fatal("tower record survived rejection")
		}
	})

	t.Run("ulora_compile_fails_rolls_back_tower", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{verified: true, withULoRA: true, compileErr: errors.New("compiler down")})
		if _, err := h.mint(); err == nil || !strings.Contains(err.Error(), "(rollback: <nil>)") {
			t.Fatalf("err = %v, want compile failure with clean rollback", err)
		}
		expectSteps(t, h.calls.steps(), "tower.put", "oracle.verify", "ulora.compile", "tower.delete")
		if h.tower.count() != 0 {
			t.Fatal("tower record survived compile failure")
		}
	})

	t.Run("ulora_validation_refused_rolls_back_tower", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{verified: true, withULoRA: true, validatorErr: errors.New("gate refused")})
		if _, err := h.mint(); err == nil || !strings.Contains(err.Error(), "gate refused") {
			t.Fatalf("err = %v, want validation refusal", err)
		}
		// The gate refused, so nothing may reach KNIRVCHAIN.
		expectSteps(t, h.calls.steps(), "tower.put", "oracle.verify", "ulora.compile", "ulora.validate", "tower.delete")
		if h.tower.count() != 0 {
			t.Fatal("tower record survived validation refusal")
		}
	})

	t.Run("chain_mint_fails_rolls_back_tower_and_skips_burn", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{verified: true, chainFail: http.StatusBadGateway})
		if _, err := h.mint(); err == nil || !strings.Contains(err.Error(), "(rollback: <nil>)") {
			t.Fatalf("err = %v, want chain failure with clean rollback", err)
		}
		expectSteps(t, h.calls.steps(), "tower.put", "oracle.verify", "chain.mint", "tower.delete")
		if h.tower.count() != 0 {
			t.Fatal("tower record survived chain failure")
		}
	})

	t.Run("rollback_failure_is_reported_with_original_error", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{
			verified:    true,
			oracleFail:  map[string]int{"verify": http.StatusInternalServerError},
			towerDelErr: errors.New("tower delete refused"),
		})
		_, err := h.mint()
		if err == nil {
			t.Fatal("expected error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "verify skill") || !strings.Contains(msg, "tower delete refused") {
			t.Fatalf("err = %q, want both the verify error and the rollback error", msg)
		}
		// The orphan is still there, but the error says so.
		if !h.tower.has(h.skillID) {
			t.Fatal("expected orphaned tower record when delete is refused")
		}
	})
}

// TestMintSkillPostChainFailures covers the steps after the canonical chain mint
// (R13 Q44). A failure there rolls back the tower record, because the KNIRVCHAIN
// bundle cannot be reverted. The error must name the failed step, the chain
// bundle and transaction, and the rollback result. None of the earlier
// post-chain effects (burn, ownership, bounty) has an undo, so none is undone.
func TestMintSkillPostChainFailures(t *testing.T) {
	cases := []struct {
		name  string
		opts  e2eOpts
		steps []string
		// step is the label the error must carry for the failed post-chain step.
		step string
	}{
		{
			name:  "burn_fails",
			opts:  e2eOpts{verified: true, oracleFail: map[string]int{"burn": http.StatusInternalServerError}},
			steps: []string{"tower.put", "oracle.verify", "chain.mint", "oracle.burn", "tower.delete"},
			step:  "burn NRN",
		},
		{
			name:  "ownership_fails",
			opts:  e2eOpts{verified: true, oracleFail: map[string]int{"ownership": http.StatusInternalServerError}},
			steps: []string{"tower.put", "oracle.verify", "chain.mint", "oracle.burn", "oracle.ownership", "tower.delete"},
			step:  "reward/ownership/bounty distribution",
		},
		{
			name:  "bounty_fails",
			opts:  e2eOpts{verified: true, oracleFail: map[string]int{"bounty": http.StatusInternalServerError}},
			steps: []string{"tower.put", "oracle.verify", "chain.mint", "oracle.burn", "oracle.ownership", "oracle.bounty", "tower.delete"},
			step:  "reward/ownership/bounty distribution",
		},
		{
			name:  "dht_announce_fails",
			opts:  e2eOpts{verified: true, dhtStatus: http.StatusInternalServerError},
			steps: []string{"tower.put", "oracle.verify", "chain.mint", "oracle.burn", "oracle.ownership", "oracle.bounty", "oracle.bounty", "dht.announce", "tower.delete"},
			step:  "announce skill",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newE2EHarness(t, tc.opts)
			_, err := h.mint()
			if err == nil {
				t.Fatal("expected post-chain failure")
			}
			msg := err.Error()
			// The step, the chain bundle/tx, and a clean rollback result.
			for _, want := range []string{
				tc.step + " failed after KNIRVCHAIN mint",
				"chain event " + h.skillID,
				"tx e2e-tx",
				"bundle e2e-bundle-hash",
				"(rollback: <nil>)",
			} {
				if !strings.Contains(msg, want) {
					t.Fatalf("err = %q, missing %q", msg, want)
				}
			}
			expectSteps(t, h.calls.steps(), tc.steps...)
			if h.tower.has(h.skillID) {
				t.Fatalf("tower record %s survived post-chain failure", h.skillID)
			}
		})
	}

	t.Run("rollback_failure_is_reported_with_step_error", func(t *testing.T) {
		h := newE2EHarness(t, e2eOpts{
			verified:    true,
			oracleFail:  map[string]int{"burn": http.StatusInternalServerError},
			towerDelErr: errors.New("tower delete refused"),
		})
		_, err := h.mint()
		if err == nil {
			t.Fatal("expected error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "burn NRN failed after KNIRVCHAIN mint") ||
			!strings.Contains(msg, "tower delete refused") ||
			!strings.Contains(msg, "chain event "+h.skillID) {
			t.Fatalf("err = %q, want the step error, the rollback error and the chain id", msg)
		}
		// The orphan is still there, and the error says so.
		if !h.tower.has(h.skillID) {
			t.Fatal("expected orphaned tower record when delete is refused")
		}
	})
}
