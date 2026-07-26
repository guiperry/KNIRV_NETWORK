package nrv

const (
	Magic           uint32 = 0x4E525621
	Version         uint32 = 1
	Alignment              = 8
	RegistryPadding        = 5 * 1024 * 1024
	HeaderSize             = 12
)

type NRVModalityType string

const (
	NRVModalityVector NRVModalityType = "vector"
	NRVModalitySeed   NRVModalityType = "seed"
	NRVModalityThermo NRVModalityType = "thermo"
	NRVModalityProof  NRVModalityType = "proof"
)

type ModalityType = NRVModalityType

var (
	ModalityVector = NRVModalityVector
	ModalitySeed   = NRVModalitySeed
	ModalityThermo = NRVModalityThermo
	ModalityProof  = NRVModalityProof
)

func Align8(n int) int {
	return (n + 7) &^ 7
}
