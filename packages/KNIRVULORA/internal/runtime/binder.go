package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"ulora/internal/api"
	"ulora/internal/bundling"
	"ulora/internal/safetensors"
)

type LayerWeight struct {
	A [][]float32 `json:"a"`
	B [][]float32 `json:"b"`
}

type Binder struct {
	connectorProvider ConnectorProvider

	bundlePath string
	manifest   *api.Manifest
}

type BinderOption func(*Binder)

func WithBundlePath(path string) BinderOption {
	return func(b *Binder) {
		b.bundlePath = path
	}
}

func NewBinder(opts ...BinderOption) (*Binder, error) {
	b := &Binder{}
	for _, opt := range opts {
		opt(b)
	}
	if b.bundlePath == "" {
		return nil, fmt.Errorf("bundle path is required")
	}
	manifestData, err := bundling.ReadFile(b.bundlePath, bundling.ManifestFile)
	if err != nil {
		return nil, fmt.Errorf("read manifest from bundle: %w", err)
	}
	manifest, err := api.ManifestFromJSON(manifestData)
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	b.manifest = manifest
	return b, nil
}

func (b *Binder) Manifest() *api.Manifest {
	return b.manifest
}

func (b *Binder) Bind(targetModel api.BaseModelSpec) (map[string]LayerWeight, error) {
	if err := b.validateTopology(targetModel); err != nil {
		return nil, err
	}

	// The connector is what makes a canonical core usable on this model: without
	// one there is no projection from the shared space into the target's frame,
	// and binding would have to be invented. It comes from the runtime because it
	// belongs to the model, not to this bundle.
	if b.connectorProvider == nil {
		return nil, fmt.Errorf("no connector provider is configured: a canonical core cannot be projected into a model's frame without one (target %s/%s)",
			targetModel.Family, targetModel.ParamCount)
	}
	pIn, pOut, err := b.connectorProvider.ConnectorFor(targetModel, b.manifest.CanonicalCore.CanonicalDim)
	if err != nil {
		return nil, fmt.Errorf("resolve connector for %s/%s: %w", targetModel.Family, targetModel.ParamCount, err)
	}
	if pIn == nil || pOut == nil {
		return nil, fmt.Errorf("no connector available for %s/%s (canonical_dim=%d): the runtime has not derived one for this model yet. Deriving it needs the model's weight matrices — a connector is a projection built from the model's own principal subspaces, and an invented basis would bind without aligning anything",
			targetModel.Family, targetModel.ParamCount, b.manifest.CanonicalCore.CanonicalDim)
	}

	weightsData, err := bundling.ReadFile(b.bundlePath, bundling.WeightsFile)
	if err != nil {
		return nil, fmt.Errorf("read weights from bundle: %w", err)
	}

	// The compiler merges weights with safetensors.numpy.save_file
	// (internal/compiler/engines/merge_safetensors.py), so this file is real
	// binary safetensors. Parsing it as JSON — as this function previously
	// did — cannot work on any compiler-produced bundle.
	weights, err := safetensors.Parse(weightsData)
	if err != nil {
		return nil, fmt.Errorf("parse weights: %w", err)
	}

	k := b.manifest.CanonicalCore.CanonicalDim

	// Shapes are checked against the architecture and K rather than trusted: a
	// connector that disagrees would otherwise multiply into a result of the
	// right shape carrying the wrong meaning.
	if len(pIn) != k || (len(pIn) > 0 && len(pIn[0]) != targetModel.HiddenSize) {
		return nil, fmt.Errorf("connector P_in is %dx%d, expected %dx%d (canonical_dim × hidden_size)",
			rows(pIn), cols(pIn), k, targetModel.HiddenSize)
	}
	if len(pOut) != targetModel.HiddenSize || (len(pOut) > 0 && len(pOut[0]) != k) {
		return nil, fmt.Errorf("connector P_out is %dx%d, expected %dx%d (hidden_size × canonical_dim)",
			rows(pOut), cols(pOut), targetModel.HiddenSize, k)
	}

	result := make(map[string]LayerWeight, len(b.manifest.CanonicalCore.TargetModules))
	for _, module := range b.manifest.CanonicalCore.TargetModules {
		// Canonical core tensors live in K-space and are per semantic module:
		// lora_A is r × K, lora_B is K × r.
		aCore, err := tensorMatrix(weights, module+"/lora_A")
		if err != nil {
			return nil, err
		}
		if c := cols(aCore); c != k {
			return nil, fmt.Errorf("core %s/lora_A has %d columns, expected canonical_dim %d", module, c, k)
		}
		bCore, err := tensorMatrix(weights, module+"/lora_B")
		if err != nil {
			return nil, err
		}
		if r := rows(bCore); r != k {
			return nil, fmt.Errorf("core %s/lora_B has %d rows, expected canonical_dim %d", module, r, k)
		}

		// Project into the target's native frame.
		aTarget, err := matMul(aCore, pIn, module+"/lora_A · P_in")
		if err != nil {
			return nil, err
		}
		bTarget, err := matMul(pOut, bCore, "P_out · "+module+"/lora_B")
		if err != nil {
			return nil, err
		}

		result[module] = LayerWeight{A: aTarget, B: bTarget}
	}

	return result, nil
}

func rows(m [][]float32) int {
	if len(m) == 0 {
		return 0
	}
	return len(m)
}

func cols(m [][]float32) int {
	if len(m) == 0 {
		return 0
	}
	return len(m[0])
}

// matMul multiplies two matrices, refusing a dimension mismatch.
//
// A silent shape mismatch is the failure mode this whole path is exposed to: the
// projection is the one operation that turns a portable core into something a
// specific model can use, so getting it wrong would produce plausible numbers of
// the right shape carrying the wrong meaning.
func matMul(a, b [][]float32, label string) ([][]float32, error) {
	aRows, aCols := rows(a), cols(a)
	bRows, bCols := rows(b), cols(b)
	if aCols != bRows {
		return nil, fmt.Errorf("%s: cannot multiply %dx%d by %dx%d (inner dimensions differ)",
			label, aRows, aCols, bRows, bCols)
	}
	// Ragged input would read out of bounds below.
	for i := range a {
		if len(a[i]) != aCols {
			return nil, fmt.Errorf("%s: left operand row %d has %d columns, expected %d", label, i, len(a[i]), aCols)
		}
	}
	for i := range b {
		if len(b[i]) != bCols {
			return nil, fmt.Errorf("%s: right operand row %d has %d columns, expected %d", label, i, len(b[i]), bCols)
		}
	}

	out := make([][]float32, aRows)
	for i := 0; i < aRows; i++ {
		out[i] = make([]float32, bCols)
		for j := 0; j < bCols; j++ {
			var sum float32
			for k := 0; k < aCols; k++ {
				sum += a[i][k] * b[k][j]
			}
			out[i][j] = sum
		}
	}
	return out, nil
}

// ConnectorProvider supplies the projection pair that maps a canonical core into
// a specific model's native frame.
//
// Connectors are a property of the model rather than of a skill, so they are not
// carried in the bundle: the same skill binding to three models would otherwise
// embed three connectors in every artifact, and a model that appeared later could
// not bind an otherwise perfectly good core. The runtime owns them and hands one
// in here.
type ConnectorProvider interface {
	// ConnectorFor returns P_in (K × d_in) and P_out (d_out × K) for the target
	// model. It returns (nil, nil, nil) when no connector is available, which is
	// distinct from an error: absent means "derive one", error means "the lookup
	// itself failed".
	ConnectorFor(spec api.BaseModelSpec, canonicalDim int) (pIn, pOut [][]float32, err error)
}

// WithConnectorProvider supplies the projection source used by Bind.
func WithConnectorProvider(p ConnectorProvider) BinderOption {
	return func(b *Binder) { b.connectorProvider = p }
}

// tensorMatrix reads a tensor as a matrix, failing when it is absent.
//
// A missing tensor used to yield a fabricated zero matrix. That made an adapter
// which was never actually bound look like a working one — a silently wrong
// answer instead of an error, and precisely the outcome a validation gate exists
// to prevent.
func tensorMatrix(weights *safetensors.File, name string) ([][]float32, error) {
	tensor, ok := weights.Tensor(name)
	if !ok {
		return nil, fmt.Errorf("bundle has no tensor %q; present tensors: %s",
			name, strings.Join(weights.Names(), ", "))
	}
	matrix, err := tensor.Float32Matrix()
	if err != nil {
		return nil, fmt.Errorf("tensor %q: %w", name, err)
	}
	return matrix, nil
}

func (b *Binder) BindBytes(targetModel api.BaseModelSpec) (map[string][][]float32, error) {
	return nil, fmt.Errorf("BindBytes not implemented; use Bind instead")
}

func (b *Binder) validateTopology(target api.BaseModelSpec) error {
	src := b.manifest.BaseSourceModel
	if src.AttentionMechanism != target.AttentionMechanism ||
		src.ActivationFunc != target.ActivationFunc {
		return fmt.Errorf("topology mismatch: attention=%s/%s, activation=%s/%s",
			src.AttentionMechanism, target.AttentionMechanism,
			src.ActivationFunc, target.ActivationFunc)
	}
	return nil
}

type BindResult struct {
	TargetModel  api.BaseModelSpec      `json:"target_model"`
	LayerWeights map[string]LayerWeight `json:"layer_weights"`
}

func (br *BindResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(br, "", " ")
}
