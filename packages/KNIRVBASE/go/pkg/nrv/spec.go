package nrv

const (
	Magic           uint32 = 0x4E525621
	Version         uint32 = 1
	Alignment              = 8
	RegistryPadding        = 5 * 1024 * 1024
	HeaderSize             = 12
)

type ModalityType string

const (
	ModalityVector ModalityType = "vector"
	ModalitySeed   ModalityType = "seed"
	ModalityThermo ModalityType = "thermo"
	ModalityProof  ModalityType = "proof"
)

func Align8(n int) int {
	return (n + 7) &^ 7
}
