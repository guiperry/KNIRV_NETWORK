package nrv

import "math"

// I/P-bracket policy shared by the FrameTicker (live writes) and the NRV3
// converter, so a bracket stream is delta-encoded the same way whichever path
// produced it.
const (
	// IFrameInterval forces a fresh I-bracket every N brackets.
	IFrameInterval = 50
	// DriftThreshold promotes a bracket to a new I-bracket when its projections
	// drift further than this from the current anchor.
	DriftThreshold = 0.25
)

// EuclideanDrift is the distance between the 16 little-endian uint16
// projection dimensions of two brackets, each normalised to [0,1].
func EuclideanDrift(a, b [32]byte) float64 {
	var sum float64
	for i := 0; i < 16; i++ {
		av := float64(uint16(a[i*2])|uint16(a[i*2+1])<<8) / 65535.0
		bv := float64(uint16(b[i*2])|uint16(b[i*2+1])<<8) / 65535.0
		diff := av - bv
		sum += diff * diff
	}
	return math.Sqrt(sum)
}
