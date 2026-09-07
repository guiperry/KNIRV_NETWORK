package runtime

import (
	"encoding/json"
	"fmt"
	"os"

	"ulora/internal/api"
	"ulora/internal/bundling"
)

type LayerWeight struct {
	A    [][]float32 `json:"a"`
	B    [][]float32 `json:"b"`
}

type Binder struct {
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

	weightsData, err := bundling.ReadFile(b.bundlePath, bundling.WeightsFile)
	if err != nil {
		return nil, fmt.Errorf("read weights from bundle: %w", err)
	}

	var tensors map[string]interface{}
	if err := json.Unmarshal(weightsData, &tensors); err != nil {
		return nil, fmt.Errorf("parse weights: %w", err)
	}

	rank := b.manifest.CanonicalCore.AdapterRank
	alpha := b.manifest.CanonicalCore.ScalingFactorAlpha

	result := make(map[string]LayerWeight)
	for _, module := range b.manifest.CanonicalCore.TargetModules {
		prefix := determinePrefix(module, targetModel)
		loraA, loraB := extractTensor(tensors, prefix+"/lora_A", prefix+"/lora_B")

		if len(loraA) == 0 {
			loraA = make([][]float32, rank)
			for i := range loraA {
				loraA[i] = make([]float32, targetModel.HiddenSize)
			}
		}
		if len(loraB) == 0 {
			loraB = make([][]float32, targetModel.HiddenSize)
			for i := range loraB {
				loraB[i] = make([]float32, rank)
			}
		}

		result[module] = LayerWeight{
			A: projectMatrix(loraA, targetModel.HiddenSize, "input"),
			B: projectMatrix(loraB, targetModel.HiddenSize, "output"),
		}
	}

	_ = alpha
	_ = os.Stdout
	return result, nil
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

func determinePrefix(module string, target api.BaseModelSpec) string {
	return fmt.Sprintf("%s-%s", target.Family, target.ParamCount)
}

func extractTensor(tensors map[string]interface{}, aKey, bKey string) ([][]float32, [][]float32) {
	var a, b [][]float32
	if v, ok := tensors[aKey]; ok {
		if arr, ok := v.([][]float32); ok {
			a = arr
		}
	}
	if v, ok := tensors[bKey]; ok {
		if arr, ok := v.([][]float32); ok {
			b = arr
		}
	}
	return a, b
}

func projectMatrix(mat [][]float32, dim int, direction string) [][]float32 {
	if len(mat) == 0 || len(mat[0]) == 0 {
		if direction == "input" {
			return make([][]float32, len(mat))
		}
		return make([][]float32, dim)
	}
	rows := len(mat)
	cols := len(mat[0])
	if rows != dim {
		if rows > dim {
			newMat := make([][]float32, dim)
			for i := 0; i < dim; i++ {
				src := i * rows / dim
				if src >= rows {
					src = rows - 1
				}
				newMat[i] = mat[src]
			}
			mat = newMat
			rows = dim
		} else {
			newMat := make([][]float32, dim)
			for i := 0; i < rows; i++ {
				newMat[i] = mat[i]
			}
			for i := rows; i < dim; i++ {
				newMat[i] = make([]float32, cols)
			}
			mat = newMat
			rows = dim
		}
	}
	return mat
}

type BindResult struct {
	TargetModel   api.BaseModelSpec         `json:"target_model"`
	LayerWeights  map[string]LayerWeight    `json:"layer_weights"`
}

func (br *BindResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(br, "", " ")
}
