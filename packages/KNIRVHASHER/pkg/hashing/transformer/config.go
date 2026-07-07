package transformer

type HEARTConfig struct {
	Gorgonite GorgoniteConfig

	UseHashNetwork bool
	UseCerebras    bool
	CerebrasProgramDir  string
	CerebrasWeightsPath string

	TinyGoPath   string
	WASMOutDir   string
	AuditLogDir  string

	HashNetworkConfidenceThreshold float32
	EntropySpikethreshold          float64
	MaxTurns                       int

	// ExternalGeneratorURL is the HTTP endpoint for the bootstrap LLM
	// generator.  When set, the HEARTService routes WASM generation to
	// this URL while the internal Gorgonite model is being pre-trained.
	// The endpoint receives a WASMGenerationRequest and must return a
	// WASMGenerationResponse with the compiled TinyGo source.
	ExternalGeneratorURL string

	// ExternalGenerateFn is an optional function that directly generates
	// WASM source from a prompt.  If set, it takes precedence over
	// ExternalGeneratorURL.  This allows in-process injection of the
	// LLM callback without an HTTP round-trip.
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
	}
}