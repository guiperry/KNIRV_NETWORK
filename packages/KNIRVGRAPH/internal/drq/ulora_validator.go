package drq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DVEBundleValidator submits a compiled `.ulora` bundle to the platform's
// validation service, where a DVE runs it against held-out cases, and gates the
// mint on the result.
//
// # Why this is strict
//
// ulora_implementation.md §2.2 forbids publishing a bundle on the strength of a
// placeholder validator, and a gate that accepts any "valid" result is a
// placeholder in all but name: a generic task type would happily return success
// for something that never exercised the bundle. So a verdict is only accepted
// when it demonstrably refers to *this* bundle:
//
//   - result.task_id equals the task we created for this bundle,
//   - result.status is "success",
//   - result.proof is non-empty,
//   - result.score meets the configured floor,
//   - result.results echoes our content hash under the key below.
//
// # The DVE-side contract this depends on
//
// The platform's validation and inference services have no uLoRA support today —
// nothing under backend_server references `.ulora`, and no inference path loads
// an adapter bundle. So the last condition is currently unsatisfiable and every
// bundle is refused with ErrBundleValidationUnsupported. That is the intended
// state: the gate is closed until the DVE can actually execute a bundle and
// report the hash it ran, at which point this validator accepts it unchanged.
const (
	// BundleRevalidationKey is the Results key the DVE must echo the bundle's
	// content hash under, so a verdict can be attributed to a specific bundle.
	BundleRevalidationKey = "ulora_content_hash"
	// BundleAnchorsKey carries the held-out calibration anchors the DVE should
	// evaluate against.
	BundleAnchorsKey = "ulora_anchors"

	// bundleValidationTaskType is the task type this submits. It is deliberately
	// distinct from the skill-node type so a deployment can route it separately.
	bundleValidationTaskType = "ulora_bundle"

	defaultBundlePollInterval = 5 * time.Second
	defaultBundlePollTimeout  = 30 * time.Minute
)

// ErrBundleValidationUnsupported means the validation service accepted the task
// but produced no verdict attributable to this bundle. In practice this is what
// happens until the DVE gains uLoRA execution.
var ErrBundleValidationUnsupported = errors.New("no validation service can execute a uLoRA bundle and report its content hash")

// DVEBundleValidator is the Option A gate: submit, poll, then require a verdict
// that provably refers to the submitted bundle.
type DVEBundleValidator struct {
	BaseURL      string
	Token        string
	Client       *http.Client
	PollInterval time.Duration
	PollTimeout  time.Duration
	// MinScore is the score floor a verdict must clear. Zero accepts any
	// successful, proven verdict.
	MinScore float64
}

// NewDVEBundleValidator builds a validator for a validation-service endpoint.
func NewDVEBundleValidator(baseURL, token string) *DVEBundleValidator {
	return &DVEBundleValidator{
		BaseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:        strings.TrimSpace(token),
		Client:       &http.Client{Timeout: 60 * time.Second},
		PollInterval: defaultBundlePollInterval,
		PollTimeout:  defaultBundlePollTimeout,
	}
}

// ValidateBundle implements BundleValidator.
func (v *DVEBundleValidator) ValidateBundle(ctx context.Context, cluster *ErrorCluster, result *ULoRACompileResult, dataset []ULoRADatasetRecord) error {
	if v == nil || v.BaseURL == "" || v.Token == "" {
		return ErrDVEValidationUnavailable
	}
	if cluster == nil {
		return errors.New("cluster is required to validate a bundle")
	}
	if result == nil {
		return errors.New("compile result is required to validate a bundle")
	}
	contentHash := strings.TrimSpace(result.ContentHash)
	if contentHash == "" {
		return errors.New("compile result has no content hash; cannot attribute a verdict to it")
	}

	taskID, err := v.submit(ctx, cluster, result, dataset)
	if err != nil {
		return err
	}

	verdict, err := v.await(ctx, taskID)
	if err != nil {
		return err
	}
	return v.accept(verdict, taskID, contentHash)
}

// submit creates the validation task, pinning the bundle by content hash and
// carrying the held-out anchors.
func (v *DVEBundleValidator) submit(ctx context.Context, cluster *ErrorCluster, result *ULoRACompileResult, dataset []ULoRADatasetRecord) (string, error) {
	anchors := make([]map[string]string, 0, len(dataset))
	for _, record := range dataset {
		anchors = append(anchors, map[string]string{
			"context":              record.Context,
			"corrected_completion": record.CorrectedCompletion,
			"target_model":         record.TargetModel,
		})
	}

	payload := map[string]any{
		"type":     bundleValidationTaskType,
		"priority": 5,
		"data": map[string]any{
			"cluster_id":          cluster.ClusterID,
			BundleRevalidationKey: result.ContentHash,
			"manifest_path":       result.ManifestPath,
			"weights_path":        result.WeightsPath,
			"bundle_id":           result.BundleID,
			"target_models":       result.TargetModels,
			BundleAnchorsKey:      anchors,
			"path_used":           result.PathUsed,
		},
		"requested_by": cluster.OwnerAgent,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal bundle validation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.BaseURL+"/validation/tasks", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build bundle validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.Token)

	resp, err := v.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("submit bundle validation task: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read bundle validation response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("validation service rejected the configured credential (%d): %s",
			resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode/100 != 2 {
		// A service that does not know this task type lands here, which is the
		// honest outcome rather than a silently skipped gate.
		return "", fmt.Errorf("validation service refused bundle task (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var task struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &task); err != nil {
		return "", fmt.Errorf("decode bundle validation task: %w", err)
	}
	if strings.TrimSpace(task.ID) == "" {
		return "", errors.New("validation service returned a bundle task with no id")
	}
	return task.ID, nil
}

// bundleVerdict is the subset of objects.ValidationResult this gate reads.
type bundleVerdict struct {
	TaskID       string                 `json:"task_id"`
	Status       string                 `json:"status"`
	Score        float64                `json:"score"`
	Proof        string                 `json:"proof"`
	Results      map[string]interface{} `json:"results"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// await polls the task's results until a verdict appears or the budget expires.
func (v *DVEBundleValidator) await(ctx context.Context, taskID string) (*bundleVerdict, error) {
	interval := v.PollInterval
	if interval <= 0 {
		interval = defaultBundlePollInterval
	}
	timeout := v.PollTimeout
	if timeout <= 0 {
		timeout = defaultBundlePollTimeout
	}
	deadline := time.Now().Add(timeout)

	for {
		verdict, pending, err := v.fetchVerdict(ctx, taskID)
		if err != nil {
			return nil, err
		}
		if !pending {
			return verdict, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("bundle validation task %s did not produce a verdict within %s", taskID, timeout)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("bundle validation cancelled while waiting on task %s: %w", taskID, ctx.Err())
		case <-time.After(interval):
		}
	}
}

// fetchVerdict reads the task's result. A task with no result yet is "pending",
// which is not an error.
func (v *DVEBundleValidator) fetchVerdict(ctx context.Context, taskID string) (*bundleVerdict, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.BaseURL+"/validation/tasks/"+taskID+"/results", nil)
	if err != nil {
		return nil, false, fmt.Errorf("build bundle result request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+v.Token)

	resp, err := v.Client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("fetch bundle validation result: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, false, fmt.Errorf("read bundle validation result: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, false, fmt.Errorf("validation service returned %d for bundle task %s: %s",
			resp.StatusCode, taskID, strings.TrimSpace(string(raw)))
	}

	// The route returns either a bare result or an envelope with the result
	// nested, so accept both rather than guessing one.
	var envelope struct {
		TaskID string         `json:"task_id"`
		Status string         `json:"status"`
		Result *bundleVerdict `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, false, fmt.Errorf("decode bundle validation result: %w", err)
	}
	if envelope.Result == nil {
		// No verdict yet. If the task itself failed, say so rather than waiting
		// out the whole timeout.
		if status := strings.ToLower(strings.TrimSpace(envelope.Status)); status == "failed" || status == "error" {
			return nil, false, fmt.Errorf("bundle validation task %s ended in status %q", taskID, envelope.Status)
		}
		return nil, true, nil
	}
	if envelope.Result.TaskID == "" {
		envelope.Result.TaskID = envelope.TaskID
	}
	return envelope.Result, false, nil
}

// accept applies the acceptance rules. Every rule here exists so that a verdict
// about something other than this bundle cannot be mistaken for a pass.
func (v *DVEBundleValidator) accept(verdict *bundleVerdict, taskID, contentHash string) error {
	if verdict == nil {
		return fmt.Errorf("bundle validation task %s produced no verdict", taskID)
	}
	if strings.TrimSpace(verdict.TaskID) != "" && verdict.TaskID != taskID {
		return fmt.Errorf("bundle verdict belongs to task %s, not %s", verdict.TaskID, taskID)
	}
	if status := strings.ToLower(strings.TrimSpace(verdict.Status)); status != "success" {
		return fmt.Errorf("bundle %s validation status %q (score %.4f): %s",
			contentHash, verdict.Status, verdict.Score, strings.TrimSpace(verdict.ErrorMessage))
	}
	if strings.TrimSpace(verdict.Proof) == "" {
		return fmt.Errorf("bundle %s validation returned no proof", contentHash)
	}
	if verdict.Score < v.MinScore {
		return fmt.Errorf("bundle %s scored %.4f, below the %.4f floor", contentHash, verdict.Score, v.MinScore)
	}

	echoed, ok := verdict.Results[BundleRevalidationKey]
	if !ok {
		return fmt.Errorf("%w: verdict for task %s carries no %q entry", ErrBundleValidationUnsupported, taskID, BundleRevalidationKey)
	}
	echoedHash, _ := echoed.(string)
	if !strings.EqualFold(strings.TrimSpace(echoedHash), contentHash) {
		return fmt.Errorf("bundle verdict ran %q but this bundle is %s: refusing a verdict for different content",
			echoedHash, contentHash)
	}
	return nil
}
