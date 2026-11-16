package vlmserver

import (
	"fmt"
	"strconv"
)

// ModelShape is what the planner needs to know about the weights. The MoE split comes
// from the GGUF tensor table (see measurements.md, 2026-11-11), not from the file name.
type ModelShape struct {
	Layers int // transformer blocks; each has one set of expert tensors
	// ExpertBytesPerLayer is the size of one block's ffn_*_exps tensors. Expert tensors
	// are the part that can stay in system RAM (llama.cpp --n-cpu-moe).
	ExpertBytesPerLayer uint64
	// OtherBytes is everything that is not an expert tensor (attention, shared
	// experts, embeddings, output). It always goes to the GPU when there is one.
	OtherBytes uint64
	// ProjectorBytes is the vision projector (mmproj).
	ProjectorBytes uint64
	// GPUOverheadBytes is the CUDA context, compute buffers and KV cache at the planned
	// context size, measured at 2048 tokens.
	GPUOverheadBytes uint64
}

// MiniCPMShape is MiniCPM-V-4.7-35B-A3B at IQ4_XS, measured on this project's RTX 3060.
var MiniCPMShape = ModelShape{
	Layers:              40,
	ExpertBytesPerLayer: 427819008,
	OtherBytes:          1604541184,
	ProjectorBytes:      1119575840,
	GPUOverheadBytes:    1181116006, // 1.1 GiB
}

const (
	// gpuMarginBytes is kept free on the GPU for the desktop and driver.
	gpuMarginBytes = 256 * mib
	// ramHeadroomBytes is kept free after the CPU-side experts are loaded.
	ramHeadroomBytes = 3 << 30
	// maxProjectorLayerPenalty is how many more CPU-resident expert layers the planner
	// accepts to keep the vision projector on the GPU. Vision on a CPU projector runs at
	// 0.2 tokens/s on this host, so the GPU projector is worth a few layers.
	maxProjectorLayerPenalty = 4
	// planCtx is the context size the overhead figure was measured at.
	planCtx = 2048
)

// Plan is one concrete llama-server launch, derived from a Host.
type Plan struct {
	NGPULayers     int    // -ngl: 0 means no GPU offload
	NCPUMoE        int    // --n-cpu-moe: expert tensors of this many blocks stay on the CPU
	ProjectorOnGPU bool   // false adds --no-mmproj-offload
	LoadMode       string // "none" (weights read into RAM) or "mmap"
	Threads        int
	Ctx            int
	Mode           string   // "gpu" or "cpu"
	Notes          []string // what the planner decided and why
	Warnings       []string // things the operator should look at
}

// Args returns the llama-server arguments for this plan (without the model paths and
// the listen address, which the supervisor adds).
func (p Plan) Args() []string {
	args := []string{
		"-ngl", strconv.Itoa(p.NGPULayers),
		"--n-cpu-moe", strconv.Itoa(p.NCPUMoE),
		"-fa", "on",
		"-lm", p.LoadMode,
		"-t", strconv.Itoa(p.Threads),
		"-c", strconv.Itoa(p.Ctx),
		"--parallel", "1",
	}
	if !p.ProjectorOnGPU {
		args = append(args, "--no-mmproj-offload")
	}
	return args
}

// gpuNeed is the VRAM a launch with nCPU expert layers and the given projector
// placement needs.
func gpuNeed(s ModelShape, nCPU int, projectorOnGPU bool) uint64 {
	onGPU := s.Layers - nCPU
	if onGPU < 0 {
		onGPU = 0
	}
	need := s.OtherBytes + uint64(onGPU)*s.ExpertBytesPerLayer + s.GPUOverheadBytes
	if projectorOnGPU {
		need += s.ProjectorBytes
	}
	return need
}

// minCPUExperts returns the fewest CPU-resident expert layers that fit in budget, or -1.
func minCPUExperts(s ModelShape, budget uint64, projectorOnGPU bool) int {
	for n := 0; n <= s.Layers; n++ {
		if gpuNeed(s, n, projectorOnGPU) <= budget {
			return n
		}
	}
	return -1
}

// PlanFor decides how to run the model on h. hasCUDA says whether the llama-server
// binary carries a CUDA backend; without it a GPU is not used even if one is present.
func PlanFor(h Host, s ModelShape, hasCUDA bool) Plan {
	p := Plan{Ctx: planCtx, Threads: threadsFor(h.CPUCores)}
	if !h.HasAVX2 && h.CPUCores > 0 {
		p.Notes = append(p.Notes, "CPU has no AVX2: CPU expert math will be slow")
	}

	budget := uint64(0)
	if h.VRAMFree > gpuMarginBytes {
		budget = h.VRAMFree - gpuMarginBytes
	}

	switch {
	case h.VRAMFree == 0 || !hasCUDA:
		p.Mode = "cpu"
		p.NGPULayers = 0
		p.NCPUMoE = s.Layers
		p.ProjectorOnGPU = false
		if h.GPUName != "" && !hasCUDA {
			p.Warnings = append(p.Warnings, fmt.Sprintf("GPU %q present but the llama-server binary has no CUDA backend; planning CPU-only", h.GPUName))
		} else {
			p.Notes = append(p.Notes, "no usable GPU: all experts on the CPU, expect very slow generation")
		}
	default:
		p.Mode = "gpu"
		p.NGPULayers = 99
		nGPUProj := minCPUExperts(s, budget, true)
		nCPUProj := minCPUExperts(s, budget, false)
		switch {
		case nGPUProj >= 0 && (nCPUProj < 0 || nGPUProj-nCPUProj <= maxProjectorLayerPenalty):
			p.NCPUMoE = nGPUProj
			p.ProjectorOnGPU = true
			p.Notes = append(p.Notes, fmt.Sprintf("projector on GPU; %d of %d expert layers on the CPU", nGPUProj, s.Layers))
		case nCPUProj >= 0:
			p.NCPUMoE = nCPUProj
			p.ProjectorOnGPU = false
			p.Warnings = append(p.Warnings, fmt.Sprintf("projector kept on CPU to fit: %d expert layers on the CPU; image requests will be slow", nCPUProj))
		default:
			// Even the non-expert tensors do not fit. Put the whole model on the CPU
			// rather than fail at load.
			p.Mode = "cpu"
			p.NGPULayers = 0
			p.NCPUMoE = s.Layers
			p.ProjectorOnGPU = false
			p.Warnings = append(p.Warnings, fmt.Sprintf("only %d MiB of VRAM is free: model does not fit the GPU, running on the CPU", h.VRAMFree/mib))
		}
	}

	expertRAM := uint64(p.NCPUMoE) * s.ExpertBytesPerLayer
	if h.RAMAvail >= expertRAM+ramHeadroomBytes {
		p.LoadMode = "none"
		p.Notes = append(p.Notes, fmt.Sprintf("weights loaded into RAM (%d MiB needed, %d MiB available)", (expertRAM+ramHeadroomBytes)/mib, h.RAMAvail/mib))
	} else {
		p.LoadMode = "mmap"
		p.Warnings = append(p.Warnings, fmt.Sprintf("only %d MiB of RAM available for %d MiB of CPU-side experts: memory-mapped weights, prompt speed drops sharply", h.RAMAvail/mib, expertRAM/mib))
	}

	if h.DiskFree > 0 && h.DiskFree < 2*(expertRAM+s.OtherBytes) {
		p.Warnings = append(p.Warnings, "low disk space for the model directory")
	}
	return p
}

func threadsFor(cores int) int {
	switch {
	case cores <= 0:
		return 4
	case cores > 8:
		return 8
	default:
		return cores
	}
}
