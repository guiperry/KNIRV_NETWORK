package transformer

import "path/filepath"

type HEARTConfig struct {
	Gorgonite GorgoniteConfig

	UseHashNetwork      bool
	UseCerebras         bool
	CerebrasProgramDir  string
	CerebrasWeightsPath string

	TinyGoPath  string
	WASMOutDir  string
	AuditLogDir string

	HashNetworkConfidenceThreshold float32
	EntropySpikethreshold          float64
	MaxTurns                       int

	InferenceMode string

	ExternalGeneratorURL string

	// AttestationLedgerDir is the 3_DATA_SEEDER frames directory containing
	// seed_writes.jsonl. The bridge uses its v2 span assertions only.
	AttestationLedgerDir     string
	AttestationSignalIndices []int
	AttestationQueueSize     int

	ExternalGenerateFn ExternalGenerateFn `json:"-" yaml:"-"`

	// ModelCheckpointPath explicitly opts into the legacy, memory-intensive GPT
	// path. It is considered only when no compatible semantic memory is found.
	ModelCheckpointPath string

	// SemanticMemoryPath is the bounded embedding memory produced by the
	// semantic data-trainer. When present, HEART uses it before the legacy GPT
	// path and does not allocate a transformer model.
	SemanticMemoryPath string
}

func DefaultHEARTConfig(useHashNetwork, useCerebras bool) *HEARTConfig {
	return &HEARTConfig{
		Gorgonite:                      *DefaultGorgoniteConfig(),
		UseHashNetwork:                 useHashNetwork,
		UseCerebras:                    useCerebras,
		TinyGoPath:                     "tinygo",
		WASMOutDir:                     "/var/heart/wasm",
		AuditLogDir:                    "/var/heart/audits",
		HashNetworkConfidenceThreshold: 0.85,
		EntropySpikethreshold:          3.0,
		MaxTurns:                       3,
		AttestationLedgerDir:           DefaultFramesDir,
		AttestationSignalIndices:       []int{0, 1, 2, 3},
		AttestationQueueSize:           128,
		SemanticMemoryPath:             filepath.Join(filepath.Dir(DefaultFramesDir), "trainer-checkpoints", "semantic_memory.json"),
	}
}
