package transformer

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
	}
}
