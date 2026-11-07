package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"KNIRVGRAPH/internal/drq"
	"KNIRVGRAPH/internal/embeddings"
	"KNIRVGRAPH/internal/nrv"
	"KNIRVGRAPH/internal/types"

	"go.uber.org/zap"
)

// This file makes the DRQ knowledge-mining loop reachable: it converts ingested
// error nodes into DRQ clusters, drives ClusterManager's lifecycle on a ticker,
// and reports exactly where the loop currently stops.
//
// The loop is intentionally transparent about its limits rather than
// papering over them. Three stages are genuinely blocked on infrastructure that
// does not exist yet, and each reports itself instead of pretending:
//
//   - training:   no LoRA training backend exists, so a converged cluster stops
//                 at CLUSTER_ACTIVE with ErrLoRATrainingBackendNotImplemented.
//   - validation: KNIRVSERVER's validation routes accept only a Bearer
//                 JWT/session credential (there is no internal-service-token
//                 path on them), so a validator is only constructed when an
//                 operator supplies a service credential explicitly.
//   - royalty:    KNIRVCHAIN's royalty engine (CHAIN-3) exposes no route, so
//                 DistributeRoyalty reports ErrRoyaltyBackendUnavailable.
//
// Clustering itself is fully live: embeddings are real, so errors genuinely
// cluster by semantic similarity and the per-cluster convergence verdict is
// computable and visible.

const (
	// drqTickIntervalEnv overrides the lifecycle tick period.
	drqTickIntervalEnv = "KNIRV_DRQ_TICK_INTERVAL"
	// drqDefaultTickInterval is the lifecycle tick period.
	drqDefaultTickInterval = 30 * time.Second

	// validationServiceURLEnv points at KNIRVSERVER's validation API.
	validationServiceURLEnv = "KNIRV_VALIDATION_SERVICE_URL"
	// validationServiceTokenEnv is the Bearer credential presented to it.
	validationServiceTokenEnv = "KNIRV_VALIDATION_SERVICE_TOKEN"
	// validationServiceTokenFileEnv names a file holding that credential. It is
	// re-read on every resolve, so a secrets provider can supply it after startup.
	validationServiceTokenFileEnv = "KNIRV_VALIDATION_SERVICE_TOKEN_FILE"

	// drqSimilarityThresholdEnv overrides the clustering similarity threshold.
	drqSimilarityThresholdEnv = "KNIRV_DRQ_SIMILARITY_THRESHOLD"
)

// drqRuntime owns the DRQ loop's state.
type drqRuntime struct {
	clustering     *drq.DRQClusterManager
	clusterManager *drq.ClusterManager
	trainer        *drq.LoRATrainer
	logger         *zap.Logger

	// errorSource is where ingested error nodes are read from. It is an
	// interface so tests can supply a fixed set of errors without standing up
	// the whole NRV system.
	errorSource errorSource

	// ingest bookkeeping
	mu        sync.Mutex
	ingested  map[string]string // error id -> cluster id
	lastError map[string]string // cluster id -> last reported lifecycle error
	ticks     uint64
	skipped   uint64

	// validator resolves the validation service on first use; status reports
	// whether it exists so a blocked validation stage can be explained precisely.
	validator *lateValidationService
}

// errorSource supplies ingested error nodes.
type errorSource interface {
	GetAllErrorNodes() []*nrv.ErrorNode
}

// drqStatus is the loop's observable state, re-exported from the drq package so
// the RPC layer can serve the same type.
type drqStatus = drq.Status

// clusterView is the per-cluster status detail (alias for readability below).
type clusterView = drq.ClusterView

// newDRQRuntime builds the loop's components against KNIRVGRAPH's real systems.
//
// embeddingService may be nil — the app's processing pipeline builds one only
// when Processing.Enabled, and mining must not be silently disabled by an
// unrelated feature toggle. When it is nil a dedicated service is built from
// the app's own EmbeddingConfig instead.
func newDRQRuntime(nrvSystem *nrv.NRVSystem, embeddingService *embeddings.EmbeddingService, config *Config, logger *zap.Logger) (*drqRuntime, error) {
	if nrvSystem == nil {
		return nil, fmt.Errorf("DRQ requires the NRV system to read ingested errors from")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	if embeddingService == nil {
		service, err := drqEmbeddingService(config, logger)
		if err != nil {
			return nil, err
		}
		embeddingService = service
	}

	// The clustering manager needs DRQ's float64 adapter over the shared
	// embedding service; without it clustering would be defined by nothing.
	embeddingModel := drq.NewEmbeddingModelWithService(drq.BERT_BASE, embeddingService)

	clustering, err := drq.NewDRQClusterManager(embeddingModel, similarityThreshold(), drq.DefaultMaxClusterSize)
	if err != nil {
		return nil, fmt.Errorf("build DRQ clustering manager: %w", err)
	}

	trainer := drq.NewLoRATrainer(&drq.LLMModel{Name: "knirv-base"}, nil, drq.NewDVEClient(nil), nil)

	// The skill minting protocol is fully wired to real backends: KNIRVGRAPH's
	// NRV skill registry, KNIRVCHAIN's event-bundle mint and KNIRVORACLE's
	// skill-economics routes.
	//
	// The skill tower store is shared between minting and the uLoRA corpus: a
	// member error's validated fix is read back from the same record minting
	// wrote, so the corpus cannot drift from what was actually minted.
	towerStore := drq.NewNRVSkillTowerStore(nrvSystem)
	// The compiler and gate are wired from the environment. With neither
	// configured the protocol still mints skill.md skills and simply does not
	// mint adapters; with a compiler but no validator the mint stops at the gate
	// rather than publishing an unvalidated bundle.
	uloraCompiler, uloraValidator, uloraModelSpecs, uloraManifestVersion := newULoRAMintingConfig()

	minter := drq.NewSkillMintingProtocol(drq.SkillMintingDeps{
		Graph:           drq.NewKNIRVGRAPHClient(towerStore),
		Chain:           &drq.KNIRVCHAINClient{},
		Oracle:          &drq.KNIRVORACLEClient{},
		Discover:        drq.NewSkillDiscoveryEngine(nil),
		SkillDocs:       drq.NewTowerSkillDocSource(towerStore),
		Compiler:        uloraCompiler,
		Validator:       uloraValidator,
		TargetModels:    uloraModelSpecs,
		ManifestVersion: uloraManifestVersion,
	})

	// KNIRVGRAPH starts inside backend_server, before the Transaction and Validation
	// Chains and the KNIRVSERVER HTTP API it validates against. The service is
	// therefore resolved when a cluster first needs it, not here.
	validator := newLateValidationService(logger)

	clusterManager := drq.NewClusterManager(drq.ClusterDeps{
		Trainer:   trainer,
		Minter:    minter,
		Validator: validator,
	})

	rt := &drqRuntime{
		clustering:     clustering,
		clusterManager: clusterManager,
		trainer:        trainer,
		logger:         logger,
		errorSource:    nrvSystem,
		ingested:       make(map[string]string),
		lastError:      make(map[string]string),
		validator:      validator,
	}

	logger.Info("DRQ loop initialized",
		zap.Float64("similarity_threshold", clustering.SimilarityThreshold()),
		zap.Bool("validator_configured", validator.configured()),
		zap.String("validator_resolution", "on first use, after the platform services are up"))
	return rt, nil
}

// similarityThreshold resolves the clustering threshold from the environment.
func similarityThreshold() float64 {
	raw := strings.TrimSpace(os.Getenv(drqSimilarityThresholdEnv))
	if raw == "" {
		return drq.DefaultSimilarityThreshold
	}
	var value float64
	if _, err := fmt.Sscanf(raw, "%f", &value); err != nil || value <= 0 || value > 1 {
		return drq.DefaultSimilarityThreshold
	}
	return value
}

// tickInterval resolves the lifecycle tick period.
func tickInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv(drqTickIntervalEnv))
	if raw == "" {
		return drqDefaultTickInterval
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return drqDefaultTickInterval
	}
	return parsed
}

// run drives the loop until ctx is cancelled.
func (rt *drqRuntime) run(ctx context.Context) {
	interval := tickInterval()
	rt.logger.Info("DRQ loop started", zap.Duration("interval", interval))
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Ingest once immediately so a restart does not wait a full interval.
	rt.step(ctx)

	for {
		select {
		case <-ctx.Done():
			rt.logger.Info("DRQ loop stopped", zap.Uint64("ticks", rt.ticks))
			return
		case <-ticker.C:
			rt.step(ctx)
		}
	}
}

// step performs one ingest + lifecycle pass.
func (rt *drqRuntime) step(ctx context.Context) {
	rt.ingest()
	rt.advance(ctx)

	rt.mu.Lock()
	rt.ticks++
	rt.mu.Unlock()
}

// ingest clusters every error node that has not yet been clustered, and hands
// each resulting cluster to the lifecycle.
func (rt *drqRuntime) ingest() {
	errors := rt.nrvErrors()
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for _, node := range errors {
		if node == nil {
			continue
		}
		id := strings.TrimSpace(node.Id)
		if id == "" {
			continue
		}
		if _, done := rt.ingested[id]; done {
			continue
		}

		converted := toDRQErrorNode(node)
		if converted == nil {
			continue
		}
		clusterID, err := rt.clustering.ClusterError(converted)
		if err != nil {
			// Record the failure against the error id so a persistent problem
			// is not retried silently forever without being visible.
			rt.ingested[id] = ""
			rt.logger.Warn("DRQ: failed to cluster error",
				zap.String("error_id", id), zap.Error(err))
			rt.skipped++
			continue
		}
		rt.ingested[id] = clusterID
	}
}

// nrvErrors reads the ingested error nodes.
func (rt *drqRuntime) nrvErrors() []*nrv.ErrorNode {
	if rt.errorSource == nil {
		return nil
	}
	return rt.errorSource.GetAllErrorNodes()
}

// advance runs one lifecycle step for every tracked cluster, reporting each
// distinct failure once so a permanently blocked stage is visible without
// flooding the log every tick.
func (rt *drqRuntime) advance(ctx context.Context) {
	// Track any cluster the clustering manager knows about that the lifecycle
	// is not yet driving.
	for clusterID, cluster := range rt.clustering.Clusters() {
		if _, tracked := rt.clusterManager.Cluster(clusterID); !tracked {
			if err := rt.clusterManager.Track(cluster); err != nil {
				rt.logger.Warn("DRQ: failed to track cluster",
					zap.String("cluster_id", clusterID), zap.Error(err))
			}
		}
	}

	for _, clusterID := range rt.clusterManager.TrackedClusterIDs() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := rt.clusterManager.ProcessCluster(ctx, clusterID)
		rt.report(clusterID, err)
	}
}

// report logs a lifecycle outcome, de-duplicating repeated identical errors.
func (rt *drqRuntime) report(clusterID string, err error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if err == nil {
		// Clear the memo so a *future* recurrence of the same failure is
		// reported again rather than being suppressed by a stale entry.
		delete(rt.lastError, clusterID)
		return
	}
	message := err.Error()
	if previous, seen := rt.lastError[clusterID]; seen && previous == message {
		return
	}
	rt.lastError[clusterID] = message
	rt.logger.Warn("DRQ lifecycle stage blocked",
		zap.String("cluster_id", clusterID), zap.String("reason", message))
}

// Status returns an observable snapshot of the loop.
func (rt *drqRuntime) Status() drqStatus {
	rt.mu.Lock()
	ticks := rt.ticks
	ingested := len(rt.ingested)
	lastErrors := make(map[string]string, len(rt.lastError))
	for id, msg := range rt.lastError {
		lastErrors[id] = msg
	}
	rt.mu.Unlock()

	status := drqStatus{
		Enabled:             true,
		Ticks:               ticks,
		IngestedErrors:      ingested,
		ClustersByStatus:    make(map[string]int),
		ValidatorConfigured: rt.validator.configured(),
		TrainerBackend:      "none (ErrLoRATrainingBackendNotImplemented)",
		SimilarityThreshold: rt.clustering.SimilarityThreshold(),
		LastErrors:          lastErrors,
	}

	views := make([]drq.ClusterView, 0)
	for _, clusterID := range rt.clusterManager.TrackedClusterIDs() {
		cluster, ok := rt.clusterManager.Cluster(clusterID)
		if !ok || cluster == nil {
			continue
		}
		report := rt.clusterManager.EvaluateConvergence(cluster)
		view := drq.ClusterView{
			ClusterID:      clusterID,
			Status:         clusterStatusName(cluster.Status),
			Members:        len(cluster.Errors),
			Validated:      report.ValidatedCount,
			MeanSimilarity: report.MeanSimilarity,
			AgeSeconds:     report.Age.Seconds(),
			Ready:          report.Ready,
			ValidationTask: cluster.ValidationTaskID,
			MintedSkillID:  cluster.MintedSkillID,
			ResolvedErrors: drq.ResolvedErrorIDs(cluster),
		}
		if !report.Ready {
			view.NotReadyReason = report.Reason
		}
		status.ClustersByStatus[view.Status]++
		views = append(views, view)
	}
	status.Clusters = len(views)
	status.ClustersDetail = views
	return status
}

// clusterStatusName renders a ClusterStatus for the status endpoint.
func clusterStatusName(status drq.ClusterStatus) string {
	switch status {
	case drq.CLUSTER_ACTIVE:
		return "active"
	case drq.CLUSTER_TRAINING:
		return "training"
	case drq.CLUSTER_VALIDATING:
		return "validating"
	case drq.CLUSTER_RESOLVED:
		return "resolved"
	case drq.CLUSTER_ARCHIVED:
		return "archived"
	default:
		return fmt.Sprintf("unknown(%d)", int(status))
	}
}

// initDRQ constructs the DRQ loop from the app's already-initialised systems.
//
// A construction failure disables the loop and records why, rather than failing
// node startup: this node serves graph query/index traffic independently of
// mining, and DRQStatus() reports the disabled state so it is never a silent
// absence.
func (app *App) initDRQ() {
	if app == nil {
		return
	}
	rt, err := newDRQRuntime(app.nrvSystem, app.embeddingService, app.config, app.logger)
	if err != nil {
		app.drqDisabledReason = err.Error()
		app.logger.Error("DRQ loop disabled", zap.Error(err))
		return
	}
	app.drq = rt
}

// DRQStatus reports the knowledge-mining loop's state, including whether it is
// disabled and why. It is safe to call on an app whose loop failed to build.
func (app *App) DRQStatus() drqStatus {
	if app == nil || app.drq == nil {
		reason := "DRQ loop was not initialised"
		if app != nil && app.drqDisabledReason != "" {
			reason = app.drqDisabledReason
		}
		return drqStatus{
			Enabled:        false,
			DisabledReason: reason,
		}
	}
	return app.drq.Status()
}

// RecordGradedSolution hands a solution graded against its error node's
// sealed 8-test suite to the running DRQ loop (see
// drq.ClusterManager.RecordSolution). Implements the network package's
// solutionRecorder.
func (app *App) RecordGradedSolution(sol *drq.Solution) (string, error) {
	if app == nil || app.drq == nil {
		return "", fmt.Errorf("DRQ loop is not running on this node")
	}
	return app.drq.clusterManager.RecordSolution(sol)
}

// NewDRQRuntimeForTest exposes the loop constructor for tests in other packages.
// It exists only so tests can drive the loop without an App.
func NewDRQRuntimeForTest(nrvSystem *nrv.NRVSystem, embeddingService *embeddings.EmbeddingService, logger *zap.Logger) (*drqRuntime, error) {
	return newDRQRuntime(nrvSystem, embeddingService, nil, logger)
}

// drqEmbeddingService builds an embedding service for the mining loop from the
// app's EmbeddingConfig.
//
// Mining needs embeddings to cluster errors at all, so it must not depend on the
// document-processing pipeline being switched on. This uses the configured
// provider when one is set, and the module default otherwise.
func drqEmbeddingService(config *Config, logger *zap.Logger) (*embeddings.EmbeddingService, error) {
	cfg := embeddings.DefaultProviderConfig(types.EmbeddingProviderTextEmbedder)
	if config != nil {
		if config.Embedding.Provider != "" {
			cfg.Type = config.Embedding.Provider
		}
		if config.Embedding.Endpoint != "" {
			cfg.Endpoint = config.Embedding.Endpoint
		}
		if config.Embedding.Model != "" {
			cfg.Model = config.Embedding.Model
		}
		if config.Embedding.Dimension > 0 {
			cfg.Dimension = config.Embedding.Dimension
		}
	}
	service, err := embeddings.NewEmbeddingService(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("DRQ requires an embedding service and none could be built: %w", err)
	}
	if service == nil {
		return nil, fmt.Errorf("DRQ requires an embedding service but the embedding provider returned none")
	}
	return service, nil
}

// ============================================================================
// NRV -> DRQ error node conversion
// ============================================================================

// resolverContextKeys are the Context keys an ingested error may use to
// attribute the fix to an agent. Ownership and bounty are computed from this
// attribution, so a cluster whose errors carry none cannot mint: there would be
// no address to pay. The RPC error-creation routes strip them from submitted
// context (nrv.StripResolverClaims).
var resolverContextKeys = nrv.ResolverContextKeys

// toDRQErrorNode converts an NRV error node into DRQ's clustering node.
func toDRQErrorNode(node *nrv.ErrorNode) *drq.ErrorNode {
	if node == nil || strings.TrimSpace(node.Id) == "" {
		return nil
	}

	// The canonical node carries its context as a protobuf Struct; DRQ's
	// clustering node wants a Go map for embedding and attribution.
	nodeContext := nrv.ErrorContextMap(node)

	// The failure context is what gets embedded, so it must carry the error's
	// identity and description — embedding only the raw JSON map would make
	// two structurally similar errors indistinguishable.
	contextJSON, err := json.Marshal(map[string]interface{}{
		"error_type":  node.ErrorType,
		"description": node.Description,
		"context":     nodeContext,
	})
	if err != nil {
		contextJSON = []byte(node.ErrorType + " " + node.Description)
	}

	converted := &drq.ErrorNode{
		Id:             node.Id,
		Domain:         node.ErrorType,
		Description:    node.Description,
		FailureContext: contextJSON,
		Complexity:     node.Severity,
		Timestamp:      node.GetTimestamp(),
		ResolvedBy:     resolverFromContext(nodeContext),
	}
	return converted
}

// resolverFromContext extracts the resolving agent, if any, from an error's
// context map.
func resolverFromContext(context map[string]interface{}) string {
	for _, key := range resolverContextKeys {
		raw, ok := context[key]
		if !ok {
			continue
		}
		if value, ok := raw.(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// ============================================================================
// Platform validation client
// ============================================================================

// validationServiceTestTransport is an injectable RoundTripper used only by tests
// so the client can be exercised without a real unix socket. In production it is
// nil and only the internal http+unix:// transport is permitted.
var validationServiceTestTransport http.RoundTripper

// newValidationServiceClient builds a client for KNIRVSERVER's validation API,
// or returns (nil, false) when no endpoint/credential is configured.
//
// The validation surface is reached over KNIRV's internal transport only — the
// endpoint must be an http+unix:// socket so no public TCP listener is ever
// opened. Any other scheme (e.g. http://, tcp) is rejected and the client fails
// closed. The token is read from the handoff file on every request and rejected
// if empty, malformed, or expired, so a stale or rotated token can never be
// replayed.
func newValidationServiceClient() (drq.ValidationService, bool) {
	baseURL := strings.TrimSpace(os.Getenv(validationServiceURLEnv))
	if baseURL == "" {
		return nil, false
	}
	c := &validationServiceClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	if validationServiceTestTransport != nil {
		c.httpClient.Transport = validationServiceTestTransport
		c.requestBase = strings.TrimRight(baseURL, "/")
		if _, err := c.readToken(); err != nil {
			return nil, false
		}
		return c, true
	}
	// Fail closed: only the internal unix transport is permitted. A TCP endpoint
	// would expose the validation surface on a publicly reachable listener.
	if !strings.HasPrefix(baseURL, "http+unix://") {
		return nil, false
	}
	sp := strings.TrimPrefix(baseURL, "http+unix://")
	c.socketPath = sp
	c.requestBase = "http://unixlocal"
	c.httpClient.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", c.socketPath)
		},
	}
	if _, err := c.readToken(); err != nil {
		return nil, false
	}
	return c, true
}

// lateValidationService is a drq.ValidationService that finds its endpoint and
// credential when first called and keeps looking until it does. Until then every
// call fails with a clear error, and a single warning is logged the first time a
// cluster actually reaches the validation stage without one.
type lateValidationService struct {
	logger *zap.Logger

	mu     sync.Mutex
	client drq.ValidationService
	warned bool
}

func newLateValidationService(logger *zap.Logger) *lateValidationService {
	return &lateValidationService{logger: logger}
}

// resolve returns the client once the endpoint and credential exist.
func (l *lateValidationService) resolve() (drq.ValidationService, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.client != nil {
		return l.client, true
	}
	if c, ok := newValidationServiceClient(); ok {
		l.client = c
		l.logger.Info("DRQ: validation service resolved")
		return c, true
	}
	if !l.warned {
		l.warned = true
		l.logger.Warn("DRQ: a cluster reached the validation stage but no validation service is configured; it will wait",
			zap.String("set", validationServiceURLEnv+" and "+validationServiceTokenEnv+" (or "+validationServiceTokenFileEnv+")"))
	}
	return nil, false
}

// configured reports whether the service can be resolved now, without warning.
func (l *lateValidationService) configured() bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.client != nil {
		return true
	}
	if c, ok := newValidationServiceClient(); ok {
		l.client = c
		return true
	}
	return false
}

func (l *lateValidationService) SubmitClusterForValidation(ctx context.Context, cluster *drq.ErrorCluster) (*drq.ValidationTask, error) {
	c, ok := l.resolve()
	if !ok {
		return nil, errNoValidationService
	}
	return c.SubmitClusterForValidation(ctx, cluster)
}

func (l *lateValidationService) GetValidationProof(ctx context.Context, taskID string) (*drq.ValidationProof, error) {
	c, ok := l.resolve()
	if !ok {
		return nil, errNoValidationService
	}
	return c.GetValidationProof(ctx, taskID)
}

var errNoValidationService = errors.New("no validation service configured")

type validationServiceClient struct {
	// requestBase is the origin used for requests: "http://unixlocal" when
	// talking to the backend API unix socket, or the configured http origin.
	requestBase string
	// socketPath is non-empty when communicating over a unix socket.
	socketPath  string
	httpClient *http.Client
}

// readValidationToken reads the service token from its handoff file (preferred,
// so rotation takes effect without a restart) or the environment, and fails
// closed on an empty, malformed, or expired token. We cannot verify the HMAC
// secret locally (only the backend does that), but we do reject anything that is
// not a well-formed signed JWT with the expected algorithm header and the
// mandatory iss/aud claims, so a fabricated or corrupt handoff can never be sent.
func readValidationToken() (string, error) {
	if path := strings.TrimSpace(os.Getenv(validationServiceTokenFileEnv)); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("read validation token file: %w", err)
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, 1<<16))
		if err != nil {
			return "", fmt.Errorf("read validation token file: %w", err)
		}
		tok := strings.TrimSpace(string(b))
		if tok == "" {
			return "", fmt.Errorf("validation token file is empty")
		}
		if err := assertTokenWellFormed(tok); err != nil {
			return "", err
		}
		return tok, nil
	}
	if tok := strings.TrimSpace(os.Getenv(validationServiceTokenEnv)); tok != "" {
		if err := assertTokenWellFormed(tok); err != nil {
			return "", err
		}
		return tok, nil
	}
	return "", fmt.Errorf("no validation token available")
}

// assertTokenWellFormed checks that token is a structurally valid JWT carrying the
// expected HS256 alg header and non-empty iss and aud claims, and that it is not
// expired. It does NOT verify the HMAC signature — that is the backend's job. It
// exists to fail closed on fabricated, unsigned, or otherwise malformed content
// before a request is ever sent.
func assertTokenWellFormed(token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("malformed validation token")
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("malformed validation token header")
	}
	var hdr struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(header, &hdr); err != nil {
		return fmt.Errorf("malformed validation token header")
	}
	// The backend issues HS256 service tokens; accept only that algorithm locally.
	if hdr.Alg != "HS256" {
		return fmt.Errorf("validation token must use HS256, got %q", hdr.Alg)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("malformed validation token payload")
	}
	var claims struct {
		Exp *float64    `json:"exp"`
		Iss string      `json:"iss"`
		Aud interface{} `json:"aud"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fmt.Errorf("malformed validation token claims")
	}
	if claims.Iss == "" {
		return fmt.Errorf("validation token missing issuer")
	}
	if !jwtAudiencePresent(claims.Aud) {
		return fmt.Errorf("validation token missing audience")
	}
	if claims.Exp == nil {
		return fmt.Errorf("validation token missing expiry")
	}
	if time.Now().After(time.Unix(int64(*claims.Exp), 0)) {
		return fmt.Errorf("validation token expired")
	}
	return nil
}

// jwtAudiencePresent reports whether a JWT aud claim carries at least one value.
// aud may be a string or an array of strings.
func jwtAudiencePresent(aud interface{}) bool {
	switch v := aud.(type) {
	case string:
		return v != ""
	case []interface{}:
		for _, e := range v {
			if s, ok := e.(string); ok && s != "" {
				return true
			}
		}
	}
	return false
}

func (c *validationServiceClient) readToken() (string, error) {
	return readValidationToken()
}

// SubmitClusterForValidation creates a validation task for a cluster.
func (c *validationServiceClient) SubmitClusterForValidation(ctx context.Context, cluster *drq.ErrorCluster) (*drq.ValidationTask, error) {
	if cluster == nil {
		return nil, fmt.Errorf("cluster is required")
	}

	payload := map[string]interface{}{
		"type":     "skillnode",
		"priority": 5,
		"data": map[string]interface{}{
			"cluster_id":       cluster.ClusterID,
			"resolves_errors":  drq.ResolvedErrorIDs(cluster),
			"member_count":     len(cluster.Errors),
			"avg_complexity":   cluster.AvgComplexity,
			"generality_score": cluster.GeneralityScore,
		},
		"requested_by": cluster.OwnerAgent,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal validation task request: %w", err)
	}

	token, err := c.readToken()
	if err != nil {
		return nil, fmt.Errorf("read validation token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.requestBase+"/api/v1/validation/tasks", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build validation task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("submit validation task: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read validation task response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("validation service rejected the configured credential (%d): %s",
			resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("validation service returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var task struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &task); err != nil {
		return nil, fmt.Errorf("decode validation task response: %w", err)
	}
	if strings.TrimSpace(task.ID) == "" {
		return nil, fmt.Errorf("validation service returned a task with no id")
	}
	return &drq.ValidationTask{
		ID:        task.ID,
		ClusterID: cluster.ClusterID,
		Status:    task.Status,
	}, nil
}

// GetValidationProof fetches a validation task's result.
//
// A task with no result yet returns (nil, nil): "still pending" is not a
// failure. A result that carries no proof is likewise not treated as valid.
func (c *validationServiceClient) GetValidationProof(ctx context.Context, taskID string) (*drq.ValidationProof, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}

	token, err := c.readToken()
	if err != nil {
		return nil, fmt.Errorf("read validation token: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.requestBase+"/api/v1/validation/tasks/"+taskID+"/results", nil)
	if err != nil {
		return nil, fmt.Errorf("build validation result request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch validation result: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read validation result: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("validation service returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var envelope struct {
		TaskID string          `json:"task_id"`
		Status string          `json:"status"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode validation result: %w", err)
	}
	// The pending shape is {task_id, status, result: null}.
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return nil, nil
	}

	var result struct {
		Status      string          `json:"status"`
		Valid       *bool           `json:"valid"`
		IsValid     *bool           `json:"is_valid"`
		Proof       json.RawMessage `json:"proof"`
		Certificate json.RawMessage `json:"certificate"`
		Degraded    bool            `json:"degraded"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		return nil, fmt.Errorf("decode validation result body: %w", err)
	}

	// Canonical verdict is the result "status" string ("valid"/"invalid"/
	// "inconclusive"); fall back to explicit boolean flags only for legacy
	// servers that omit it.
	valid := false
	switch strings.ToLower(strings.TrimSpace(result.Status)) {
	case "valid":
		valid = true
	case "invalid":
		valid = false
	default:
		if result.Valid != nil {
			valid = *result.Valid
		} else if result.IsValid != nil {
			valid = *result.IsValid
		}
	}

	proofBytes := rawFieldBytes(result.Proof)
	return &drq.ValidationProof{
		TaskID:      taskID,
		Valid:       valid,
		Proof:       proofBytes,
		Certificate: rawFieldBytes(result.Certificate),
		IssuedAt:    time.Now().UTC(),
		Degraded:    result.Degraded,
	}, nil
}

// rawFieldBytes renders an optional JSON field to bytes, treating null/absent as
// empty. A string field is unwrapped from its JSON quoting so an opaque proof
// string round-trips as its own bytes rather than as a quoted literal.
func rawFieldBytes(raw json.RawMessage) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return []byte(asString)
	}
	return []byte(raw)
}

// newULoRAMintingConfig wires uLoRA adapter minting from the environment.
//
// Adapter minting is opt-in: a deployment that has not configured it still mints
// skill.md skills, it simply does not produce adapters. Turning it on requires
// KNIRV_ULORA_MINTING_ENABLED plus a target-model specification, because
// compilation needs each model's full architecture and guessing it would build an
// adapter for a model that does not exist.
//
// The validator is deliberately absent: no BundleValidator implementation exists
// yet (it needs the DVE run + cognitive-engine verdict). Enabling minting without
// it therefore stops at the gate with ErrDVEValidationUnavailable rather than
// publishing an unvalidated bundle — the intended failure.
func newULoRAMintingConfig() (compiler drq.ULoRACompiler, validator drq.BundleValidator, models map[string]drq.ULoRATargetModel, manifestVersion string) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KNIRV_ULORA_MINTING_ENABLED"))) {
	case "1", "true", "yes":
	default:
		return nil, nil, nil, ""
	}

	raw := strings.TrimSpace(os.Getenv("KNIRV_ULORA_TARGET_MODELS"))
	if raw == "" {
		// Refuse to enable minting we cannot compile for.
		return nil, nil, nil, ""
	}
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil, nil, nil, ""
	}
	if len(models) == 0 {
		return nil, nil, nil, ""
	}

	manifestVersion = strings.TrimSpace(os.Getenv("KNIRV_ULORA_MANIFEST_VERSION"))
	if manifestVersion == "" {
		manifestVersion = "1.0.0"
	}

	// The gate: the same validation-service endpoint the cluster validation path
	// uses. Without it minting still stops at the gate — a configured compiler
	// with no validator must refuse rather than publish unvalidated bytes.
	baseURL := strings.TrimSpace(os.Getenv(validationServiceURLEnv))
	if baseURL != "" {
		if tok, err := readValidationToken(); err == nil {
			validator = drq.NewDVEBundleValidator(baseURL, tok)
		}
	}
	return drq.NewKNIRVULORAClient(""), validator, models, manifestVersion
}
