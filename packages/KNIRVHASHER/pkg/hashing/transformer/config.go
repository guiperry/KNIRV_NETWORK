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