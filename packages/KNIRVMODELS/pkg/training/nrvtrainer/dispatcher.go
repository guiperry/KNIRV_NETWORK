package nrvtrainer

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

// Target names one training handoff fed by .nrv assets.
type Target string

const (
	TargetHASHER Target = "hasher"
	TargetHEART  Target = "heart"
	TargetHANDS  Target = "hands"
	TargetHEAD   Target = "head"
	TargetULORA  Target = "ulora"
)

// Base training for HASHER, HEART and HEAD stays owned by the existing pipeline
// stages (4_DATA_TRAINER, 5_HEAD_TRAINER, the HEAD trainer). They register a
// FuncTrainer here so the dispatcher can route a corpus without owning their
// model code. HANDS and ULORA have concrete implementations in this package.

// Options selects targets and bounds the work. Every field is optional; the
// dispatcher applies documented defaults and hard caps so a caller cannot turn
// a route into an unbounded training run.
type Options struct {
	Targets   []Target
	MaxFrames int
	Hands     HandsOptions
	ULoRA     ULoRAOptions
	// Signature is the optional trusted-key check for frame signatures. Nil means
	// no trusted key, so signed frames report verify-error. It only reports:
	// signature status never drops a frame from training.
	Signature *SignatureCheck
}

const (
	defaultMaxFrames = 100_000
	hardMaxFrames    = 1_000_000
)

// MaxFramesDefault is the default frame ceiling a dispatcher applies when
// Options.MaxFrames is unset. Callers with their own flag defaults use this so
// the CLI and the library agree.
func MaxFramesDefault() int { return defaultMaxFrames }

// Result is the dispatcher's report: what was read, what each target did.
type Result struct {
	Asset   AssetSummary   `json:"asset"`
	Targets []TargetResult `json:"targets"`
}

// AssetSummary describes the asset without dumping every frame.
type AssetSummary struct {
	Path                string         `json:"path"`
	Format              Format         `json:"format"`
	Dataset             string         `json:"dataset,omitempty"`
	Frames              int            `json:"frames"`
	ValidFrames         int            `json:"valid_frames"`
	SkippedInvalid      int            `json:"skipped_invalid"`
	PayloadFrames       int            `json:"payload_frames"`
	PayloadCounts       map[string]int `json:"payload_counts,omitempty"`
	Signed              bool           `json:"signed"`
	KeyID               string         `json:"key_id,omitempty"`
	Algorithm           string         `json:"algorithm,omitempty"`
	VerificationSkipped bool           `json:"verification_skipped"`
	// SignatureCounts counts every live frame by SignatureStatus. Frames are
	// still trained on regardless of status (report-only).
	SignatureCounts map[SignatureStatus]int `json:"signature_counts"`
	// SignatureFlagged lists frames whose status is not verified.
	SignatureFlagged []SignatureFlag `json:"signature_flagged,omitempty"`
}

// SignatureFlag names one frame that did not verify, with the reason.
type SignatureFlag struct {
	Frame  string          `json:"frame"`
	Status SignatureStatus `json:"status"`
	Detail string          `json:"detail,omitempty"`
}

// TargetResult is one trainer's outcome.
type TargetResult struct {
	Target  Target   `json:"target"`
	Frames  int      `json:"frames"`
	Windows int      `json:"windows,omitempty"`
	Steps   int      `json:"steps,omitempty"`
	Detail  string   `json:"detail,omitempty"`
	Skipped []string `json:"skipped,omitempty"`
}

// Trainer is the port each handoff implements. Train receives only frames whose
// Z3 assertion validated.
type Trainer interface {
	Target() Target
	Train(ctx context.Context, asset *Asset, opts Options) (TargetResult, error)
}

// TrainFunc is the function form used by the base trainers.
type TrainFunc func(ctx context.Context, asset *Asset, opts Options) (TargetResult, error)

// FuncTrainer adapts a function to Trainer.
type FuncTrainer struct {
	Name Target
	Fn   TrainFunc
}

func (t *FuncTrainer) Target() Target { return t.Name }

func (t *FuncTrainer) Train(ctx context.Context, asset *Asset, opts Options) (TargetResult, error) {
	if t.Fn == nil {
		return TargetResult{}, fmt.Errorf("%s trainer is not configured", t.Name)
	}
	res, err := t.Fn(ctx, asset, opts)
	res.Target = t.Name
	return res, err
}

// Dispatcher owns the target registry and the single load+validate path.
type Dispatcher struct {
	trainers map[Target]Trainer
}

// New builds a dispatcher. Later trainers for the same target replace earlier
// ones, which lets a caller start from Default and override one handoff.
func New(trainers ...Trainer) *Dispatcher {
	d := &Dispatcher{trainers: map[Target]Trainer{}}
	for _, trainer := range trainers {
		d.Register(trainer)
	}
	return d
}

func (d *Dispatcher) Register(trainer Trainer) {
	if trainer == nil {
		return
	}
	d.trainers[trainer.Target()] = trainer
}

// Targets lists the registered targets in stable order.
func (d *Dispatcher) Targets() []Target {
	out := make([]Target, 0, len(d.trainers))
	for target := range d.trainers {
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Train reads path once, keeps VALID frames, then routes them to each requested
// target. It fails closed: an unknown target or an empty valid corpus is an
// error, not a partial success.
func (d *Dispatcher) Train(ctx context.Context, path string, opts Options) (*Result, error) {
	asset, err := LoadAssetWithCheck(path, opts.Signature)
	if err != nil {
		return nil, err
	}

	maxFrames := opts.MaxFrames
	if maxFrames <= 0 {
		maxFrames = defaultMaxFrames
	}
	if maxFrames > hardMaxFrames {
		maxFrames = hardMaxFrames
	}

	valid := make([]Frame, 0, len(asset.Frames))
	for _, frame := range asset.Frames {
		if frame.Valid() {
			valid = append(valid, frame)
		}
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("asset %s has no Z3-valid frames (%d frame(s) total); refusing to train", path, len(asset.Frames))
	}
	if len(valid) > maxFrames {
		return nil, fmt.Errorf("asset %s has %d Z3-valid frames, exceeds limit %d", path, len(valid), maxFrames)
	}

	payloadFrames := 0
	for _, frame := range valid {
		if frame.HasPayload {
			payloadFrames++
		}
	}

	working := *asset
	working.Frames = valid

	var flagged []SignatureFlag
	for _, frame := range asset.Frames {
		if frame.Signature != SignatureVerified && frame.Signature != SignatureUnsigned {
			flagged = append(flagged, SignatureFlag{Frame: frame.ID, Status: frame.Signature, Detail: frame.SignatureDetail})
		}
	}
	// Frames whose status is unsigned are not flagged individually; they are
	// counted. Invalid and verify-error frames are listed.
	log.Printf("nrvtrainer: asset=%s format=%s frames=%d signatures verified=%d unsigned=%d invalid=%d verify_error=%d (report-only; no frame dropped for signature)",
		asset.Path, asset.Format, len(asset.Frames),
		asset.SignatureCounts[SignatureVerified], asset.SignatureCounts[SignatureUnsigned],
		asset.SignatureCounts[SignatureInvalid], asset.SignatureCounts[SignatureVerifyError])

	result := &Result{
		Asset: AssetSummary{
			Path:                asset.Path,
			Format:              asset.Format,
			Dataset:             asset.Dataset,
			Frames:              len(asset.Frames),
			ValidFrames:         len(valid),
			SkippedInvalid:      len(asset.Frames) - len(valid),
			PayloadFrames:       payloadFrames,
			PayloadCounts:       working.PayloadCounts(),
			Signed:              asset.Signed,
			KeyID:               asset.KeyID,
			Algorithm:           asset.Algorithm,
			VerificationSkipped: asset.VerificationSkipped,
			SignatureCounts:     asset.SignatureCounts,
			SignatureFlagged:    flagged,
		},
	}

	targets := opts.Targets
	if len(targets) == 0 {
		targets = d.Targets()
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no trainers registered")
	}
	targets = uniqueTargets(targets)

	for _, target := range targets {
		trainer, ok := d.trainers[target]
		if !ok {
			return nil, fmt.Errorf("target %q is not registered (available: %s)", target, joinTargets(d.Targets()))
		}
		tres, err := trainer.Train(ctx, &working, opts)
		if err != nil {
			return nil, fmt.Errorf("%s trainer: %w", target, err)
		}
		tres.Target = target
		result.Targets = append(result.Targets, tres)
	}
	return result, nil
}

// Default builds the dispatcher with the concrete HANDS and ULORA trainers and
// the base-target placeholders wired to their existing pipeline stages when a
// caller supplies them. Hands and ulora may be nil to omit those targets.
func Default(hands *HandsTrainer, ulora *ULoRATrainer, base ...Trainer) *Dispatcher {
	trainers := make([]Trainer, 0, len(base)+2)
	trainers = append(trainers, base...)
	if hands != nil {
		trainers = append(trainers, hands)
	}
	if ulora != nil {
		trainers = append(trainers, ulora)
	}
	return New(trainers...)
}

func uniqueTargets(targets []Target) []Target {
	seen := make(map[Target]bool, len(targets))
	out := make([]Target, 0, len(targets))
	for _, target := range targets {
		trimmed := Target(strings.ToLower(strings.TrimSpace(string(target))))
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func joinTargets(targets []Target) string {
	parts := make([]string, len(targets))
	for i, target := range targets {
		parts[i] = string(target)
	}
	return strings.Join(parts, ", ")
}

// AssetName is the fallback workflow/source identifier for an asset.
func AssetName(asset *Asset) string {
	if asset == nil {
		return ""
	}
	if asset.Dataset != "" {
		return asset.Dataset
	}
	base := filepath.Base(asset.Path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
