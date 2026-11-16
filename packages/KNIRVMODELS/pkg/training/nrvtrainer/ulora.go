package nrvtrainer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Solution-adapter compilation. The ErrorNode -> Skill -> Workflow -> Solution
// chain is enforced in KNIRVGRAPH (skill minting + DVE gate); this package is
// the trainer-side half: it turns a Workflow .nrv corpus into the compile
// request KNIRVULORA already accepts, over KNIRVULORA's Unix socket.
//
// KNIRVULORA is deliberately KNIRV-free, so this mirrors its internal/api DTOs
// instead of importing them — exactly the pattern already used by
// KNIRVGRAPH/internal/drq/ulora_minting.go.

const (
	uloraDefaultSocket     = "./ulora.sock"
	uloraCompileRoute      = "/ulora/v1/compile-cluster"
	uloraCompileTimeout    = 30 * time.Minute
	uloraMaxRecords        = 100_000
	uloraMaxResponseBytes  = 1 << 20
)

// ULoRADatasetRecord mirrors ulora's DatasetRecord.
type ULoRADatasetRecord struct {
	Context             string `json:"context"`
	CorrectedCompletion string `json:"corrected_completion"`
	TargetModel         string `json:"target_model"`
	TargetModule        string `json:"target_module,omitempty"`
}

// ULoRATargetModel mirrors ulora's BaseModelSpec. AttentionMechanism and
// ActivationFunc are required: a bundle compiled without them cannot be bound,
// so a compile that "succeeds" would produce a dead artifact.
type ULoRATargetModel struct {
	Family             string `json:"family"`
	ParamCount         string `json:"param_count"`
	HiddenSize         int    `json:"hidden_size"`
	IntermediateSize   int    `json:"intermediate_size"`
	NumAttentionHeads  int    `json:"num_attention_heads"`
	NumKeyValueHeads   int    `json:"num_key_value_heads"`
	HeadDim            int    `json:"head_dim"`
	NumLayers          int    `json:"num_layers"`
	AttentionMechanism string `json:"attention_mechanism"`
	ActivationFunc     string `json:"activation_func"`
}

func (m ULoRATargetModel) validate() error {
	switch {
	case strings.TrimSpace(m.Family) == "":
		return errors.New("family is required")
	case strings.TrimSpace(m.AttentionMechanism) == "":
		return errors.New("attention_mechanism is required")
	case strings.TrimSpace(m.ActivationFunc) == "":
		return errors.New("activation_func is required")
	case m.HiddenSize <= 0:
		return fmt.Errorf("hidden_size must be positive, got %d", m.HiddenSize)
	case m.IntermediateSize <= 0:
		return fmt.Errorf("intermediate_size must be positive, got %d", m.IntermediateSize)
	case m.NumLayers <= 0:
		return fmt.Errorf("num_layers must be positive, got %d", m.NumLayers)
	}
	return nil
}

// ULoRAOptions configures one Workflow -> Solution compile.
type ULoRAOptions struct {
	WorkflowID       string
	TargetModel      string
	TargetModels     []ULoRATargetModel
	SourceDatasetIDs []string
	TargetModule     string
	Alpha            float64
	Rank             int
	NumEpochs        int
	LearningRate     float64
	SocketPath       string
	AuthToken        string
}

// ULoRAProvenance mirrors ulora's CompileProvenance.
type ULoRAProvenance struct {
	SourceID         string         `json:"source_id"`
	SourceDatasetIDs []string       `json:"source_dataset_ids"`
	Extensions       map[string]any `json:"extensions,omitempty"`
}

type uloraCompileRequest struct {
	Provenance   ULoRAProvenance      `json:"provenance"`
	Dataset      []ULoRADatasetRecord `json:"dataset"`
	TargetModels []ULoRATargetModel   `json:"target_models"`
	Alpha        float64              `json:"alpha,omitempty"`
	Rank         int                  `json:"rank,omitempty"`
	NumEpochs    int                  `json:"num_epochs,omitempty"`
	LearningRate float64              `json:"learning_rate,omitempty"`
}

// ULoRACompileResult mirrors ulora's CompileResponse.
type ULoRACompileResult struct {
	BundleID     string   `json:"bundle_id"`
	ContentHash  string   `json:"content_hash"`
	ManifestPath string   `json:"manifest_path"`
	WeightsPath  string   `json:"weights_path"`
	AnchorsPath  string   `json:"anchors_path,omitempty"`
	PathUsed     string   `json:"path_used"`
	TargetModels []string `json:"target_models"`
}

// ULoRAClient talks to ulorad over its Unix socket.
type ULoRAClient struct {
	socketPath string
	authToken  string
	http       *http.Client
}

// NewULoRAClient builds a compile client. Empty values fall back to
// ULORA_SOCKET_PATH and ULORA_AUTH_TOKEN (then to the daemon's defaults).
func NewULoRAClient(socketPath, authToken string) *ULoRAClient {
	path := strings.TrimSpace(socketPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("ULORA_SOCKET_PATH"))
	}
	if path == "" {
		path = uloraDefaultSocket
	}
	token := strings.TrimSpace(authToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("ULORA_AUTH_TOKEN"))
	}
	return &ULoRAClient{
		socketPath: path,
		authToken:  token,
		http: &http.Client{
			Timeout: uloraCompileTimeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var dialer net.Dialer
					return dialer.DialContext(ctx, "unix", path)
				},
			},
		},
	}
}

// CompileCluster submits a corpus and returns the compiled bundle.
func (c *ULoRAClient) CompileCluster(ctx context.Context, req uloraCompileRequest) (*ULoRACompileResult, error) {
	if c == nil {
		return nil, errors.New("no uLoRA compiler configured")
	}
	if strings.TrimSpace(c.authToken) == "" {
		return nil, errors.New("ULORA_AUTH_TOKEN is required to compile a solution adapter")
	}
	if len(req.Dataset) == 0 {
		return nil, errors.New("uLoRA compile requires at least one dataset record")
	}
	if len(req.TargetModels) == 0 {
		return nil, errors.New("uLoRA compile requires at least one target model")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal uLoRA compile request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://ulora"+uloraCompileRoute, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build uLoRA compile request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.authToken)

	response, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("contact KNIRVULORA at %s: %w", c.socketPath, err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, uloraMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read uLoRA compile response: %w", err)
	}
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uLoRA compile failed (HTTP %d): %s", response.StatusCode, strings.TrimSpace(string(payload)))
	}

	var result ULoRACompileResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode uLoRA compile response: %w", err)
	}
	if strings.TrimSpace(result.ContentHash) == "" {
		return nil, errors.New("uLoRA compile returned no content hash")
	}
	if strings.TrimSpace(result.WeightsPath) == "" || strings.TrimSpace(result.ManifestPath) == "" {
		return nil, fmt.Errorf("uLoRA compile returned incomplete output paths (manifest=%q weights=%q)",
			result.ManifestPath, result.WeightsPath)
	}
	return &result, nil
}

// ULoRATrainer compiles a Solution Adapter from Workflow frames.
type ULoRATrainer struct {
	Client *ULoRAClient
}

func (t *ULoRATrainer) Target() Target { return TargetULORA }

func (t *ULoRATrainer) Train(ctx context.Context, asset *Asset, opts Options) (TargetResult, error) {
	ulo := opts.ULoRA
	if strings.TrimSpace(ulo.TargetModel) == "" {
		return TargetResult{}, errors.New("ulora target_model is required (ulora compiles per architecture)")
	}
	if len(ulo.TargetModels) == 0 {
		return TargetResult{}, errors.New("ulora target_models specs are required")
	}
	for _, spec := range ulo.TargetModels {
		if err := spec.validate(); err != nil {
			return TargetResult{}, fmt.Errorf("target model %q has an invalid architecture spec: %w", spec.Family, err)
		}
	}

	records, skipped := buildULoRACorpus(asset, ulo)
	if len(records) == 0 {
		return TargetResult{}, fmt.Errorf("asset has no usable Workflow records (error_test_case/trl_sample/supervision_episode)")
	}
	if len(records) > uloraMaxRecords {
		return TargetResult{}, fmt.Errorf("asset yields %d Workflow records, exceeds limit %d", len(records), uloraMaxRecords)
	}

	workflowID := strings.TrimSpace(ulo.WorkflowID)
	if workflowID == "" {
		workflowID = AssetName(asset)
	}
	if workflowID == "" {
		return TargetResult{}, errors.New("cannot derive a Workflow ID from the asset; set workflow_id")
	}
	datasetIDs := ulo.SourceDatasetIDs
	if len(datasetIDs) == 0 && asset.Dataset != "" {
		datasetIDs = []string{asset.Dataset}
	}

	client := t.Client
	if client == nil {
		client = NewULoRAClient(ulo.SocketPath, ulo.AuthToken)
	}
	frameIDs := make([]string, 0, len(records))
	for _, frame := range asset.Frames {
		if frame.HasPayload {
			frameIDs = append(frameIDs, frame.ID)
		}
	}
	result, err := client.CompileCluster(ctx, uloraCompileRequest{
		Provenance: ULoRAProvenance{
			SourceID:         workflowID,
			SourceDatasetIDs: datasetIDs,
			Extensions: map[string]any{
				"nrv_asset": asset.Path,
				"nrv_format": string(asset.Format),
				"frame_ids": frameIDs,
				"z3_status": Z3Valid,
			},
		},
		Dataset:      records,
		TargetModels: ulo.TargetModels,
		Alpha:        ulo.Alpha,
		Rank:         ulo.Rank,
		NumEpochs:    ulo.NumEpochs,
		LearningRate: ulo.LearningRate,
	})
	if err != nil {
		return TargetResult{}, err
	}
	return TargetResult{
		Frames:  len(records),
		Detail:  fmt.Sprintf("compiled %s (%s) path_used=%s targets=%s", result.BundleID, result.ContentHash, result.PathUsed, strings.Join(result.TargetModels, ",")),
		Skipped: skipped,
	}, nil
}

func buildULoRACorpus(asset *Asset, ulo ULoRAOptions) ([]ULoRADatasetRecord, []string) {
	records := make([]ULoRADatasetRecord, 0, len(asset.Frames))
	var skipped []string
	for _, frame := range asset.Frames {
		if !frame.Valid() || !frame.HasPayload {
			continue
		}
		targetModel := strings.TrimSpace(frameTargetModel(frame, ulo))
		if targetModel == "" {
			skipped = append(skipped, frame.ID+": no target model")
			continue
		}
		record := ULoRADatasetRecord{TargetModel: targetModel, TargetModule: ulo.TargetModule}

		switch {
		case frame.Payload.ErrorTestCase != nil:
			tc := frame.Payload.ErrorTestCase
			if strings.TrimSpace(tc.Input) == "" || strings.TrimSpace(tc.Expected) == "" {
				skipped = append(skipped, frame.ID+": error_test_case needs input and expected")
				continue
			}
			record.Context = tc.Input
			record.CorrectedCompletion = tc.Expected
		case frame.Payload.TRLSample != nil:
			sample := frame.Payload.TRLSample
			completion := sample.Chosen
			if strings.TrimSpace(completion) == "" {
				completion = sample.Completion
			}
			if strings.TrimSpace(sample.Prompt) == "" || strings.TrimSpace(completion) == "" {
				skipped = append(skipped, frame.ID+": trl_sample needs prompt and completion/chosen")
				continue
			}
			record.Context = sample.Prompt
			record.CorrectedCompletion = completion
		case frame.Payload.SupervisionEpisode != nil:
			episode := frame.Payload.SupervisionEpisode
			if strings.TrimSpace(episode.State) == "" || strings.TrimSpace(episode.Decision) == "" {
				skipped = append(skipped, frame.ID+": supervision_episode needs state and decision")
				continue
			}
			record.Context = episode.State
			record.CorrectedCompletion = episode.Decision
		default:
			skipped = append(skipped, frame.ID+": payload has no Workflow supervision")
			continue
		}
		records = append(records, record)
	}

	// Stable order keeps the compiler's content hash reproducible.
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Context != records[j].Context {
			return records[i].Context < records[j].Context
		}
		return records[i].CorrectedCompletion < records[j].CorrectedCompletion
	})
	return records, skipped
}

func frameTargetModel(frame Frame, ulo ULoRAOptions) string {
	// FramePayload has no architecture field: a frame says what to correct, not
	// which base it projects onto. The caller supplies the target explicitly.
	return ulo.TargetModel
}
