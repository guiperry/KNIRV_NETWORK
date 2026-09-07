package compiler

import (
	"math"
	"strings"

	"ulora/internal/api"
)

const (
	DefaultTau = 0.82
)

func ComputeAffinity(src, tgt api.BaseModelSpec) float64 {
	sArch := archScore(src, tgt)
	sDim := dimScore(src, tgt)
	sLayer := layerScore(src, tgt)

	w1, w2, w3 := 1.0/3.0, 1.0/3.0, 1.0/3.0
	return w1*sArch + w2*sDim + w3*sLayer
}

func archScore(src, tgt api.BaseModelSpec) float64 {
	if strings.EqualFold(src.AttentionMechanism, tgt.AttentionMechanism) &&
		strings.EqualFold(src.ActivationFunc, tgt.ActivationFunc) {
		return 1.0
	}
	return 0.0
}

func dimScore(src, tgt api.BaseModelSpec) float64 {
	dSrc := float64(src.HiddenSize)
	dTgt := float64(tgt.HiddenSize)
	if dSrc == 0 || dTgt == 0 {
		return 0.0
	}
	return math.Min(dSrc, dTgt) / math.Max(dSrc, dTgt)
}

func layerScore(src, tgt api.BaseModelSpec) float64 {
	lSrc := float64(src.NumLayers)
	lTgt := float64(tgt.NumLayers)
	if lSrc == 0 || lTgt == 0 {
		return 0.0
	}
	maxLayers := math.Max(lSrc, lTgt)
	if maxLayers == 0 {
		return 1.0
	}
	return 1.0 - math.Abs(lSrc-lTgt)/maxLayers
}

func SelectPath(affinity float64, rp api.RoutingPolicy) string {
	if affinity >= rp.SimilarityThreshold {
		return "subspace_svd"
	}
	return "synthetic_distill"
}

func ComputePairwiseAffinities(src api.BaseModelSpec, targets []api.BaseModelSpec) map[string]float64 {
	result := make(map[string]float64)
	for _, t := range targets {
		affinity := ComputeAffinity(src, t)
		result[keyForModel(t)] = affinity
	}
	return result
}

func keyForModel(m api.BaseModelSpec) string {
	return strings.ToLower(m.Family) + ":" + m.ParamCount
}

func AllAboveThreshold(src api.BaseModelSpec, targets []api.BaseModelSpec, threshold float64) bool {
	for _, t := range targets {
		if ComputeAffinity(src, t) < threshold {
			return false
		}
	}
	return true
}
