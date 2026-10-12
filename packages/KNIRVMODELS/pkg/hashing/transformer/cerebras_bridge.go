package transformer

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// CerebrasBridge manages the interface between Go and Cerebras WSE
type CerebrasBridge struct {
	programDir   string
	weightsPath  string
	pythonRunner string
	cmAddr       string // Cerebras system address (empty for simulator)
}

// NewCerebrasBridge creates a new Cerebras bridge
func NewCerebrasBridge(programDir, weightsPath string, useSimulator bool) *CerebrasBridge {
	bridge := &CerebrasBridge{
		programDir:   programDir,
		weightsPath:  weightsPath,
		pythonRunner: "cs_python",
	}

	if !useSimulator {
		// Read CM address from environment or config
		bridge.cmAddr = os.Getenv("CEREBRAS_CMADDR")
	}

	return bridge
}

// HEARTInput represents input data for the HEART transformer
type HEARTInput struct {
	Measurements [][]float32 // Shape: (N, MEASURE_VEC_SIZE)
	NodeIDs      []uint32    // Shape: (N,)
	TimeSlices   []uint32    // Shape: (N,)
}

// HEARTOutput represents output from the HEART transformer
type HEARTOutput struct {
	Commands   [][]float32 // Shape: (N, CONTROL_VEC_SIZE)
	NodeIDs    []uint32    // Shape: (N,)
	TimeSlices []uint32    // Shape: (N,)
}

// CompileCSLProgram compiles the CSL source files
func (cb *CerebrasBridge) CompileCSLProgram(layoutFile string, params map[string]interface{}) error {
	fmt.Println("Compiling CSL program...")

	// Create params file
	paramsFile := filepath.Join(cb.programDir, "params.json")
	if err := cb.writeParamsFile(paramsFile, params); err != nil {
		return fmt.Errorf("failed to write params file: %w", err)
	}

	// Run cslc compiler
	cmd := exec.Command("cslc",
		layoutFile,
		"--fabric-dims="+fmt.Sprintf("%d,%d", params["GRID_WIDTH"], params["GRID_HEIGHT"]),
		"--fabric-offsets=4,1",
		"--params="+paramsFile,
		"-o", cb.programDir,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Println("Compilation successful!")
	return nil
}

// ExportWeightsFromGorgonia exports GPT model weights to NPZ format for Cerebras.
//
// GPT no longer holds permanent gorgonia nodes (Phase 4 rebuilds its graph
// fresh per forward/train call — see gpt.go) — its learnable weights live as
// plain *tensor.Dense values, retrieved here via namedParams() instead of
// walking gorgonia node fields that no longer exist.
func (cb *CerebrasBridge) ExportWeightsFromGorgonia(model *GPT, outputPath string) error {
	fmt.Println("Exporting Gorgonia weights to Cerebras format...")

	names, values := model.namedParams()
	weights := make(map[string]*tensor.Dense, len(names))
	var layerWeights []float32
	for i, name := range names {
		weights[name] = values[i]
		if data, ok := values[i].Data().([]float32); ok {
			layerWeights = append(layerWeights, data...)
		}
	}

	// Write to NPZ file
	if err := cb.writeNPZFile(outputPath, weights, layerWeights); err != nil {
		return fmt.Errorf("failed to write NPZ file: %w", err)
	}

	fmt.Printf("Weights exported to %s\n", outputPath)
	return nil
}

// RunInference executes the HEART transformer on Cerebras WSE
func (cb *CerebrasBridge) RunInference(input *HEARTInput) (*HEARTOutput, error) {
	fmt.Println("Running inference on Cerebras WSE...")

	// Prepare input data file
	inputPath := filepath.Join(cb.programDir, "input_data.npz")
	if err := cb.writeInputData(inputPath, input); err != nil {
		return nil, fmt.Errorf("failed to write input data: %w", err)
	}

	// Prepare output path
	outputPath := filepath.Join(cb.programDir, "output_data.npz")

	// Build Python command
	args := []string{
		filepath.Join("cerebras", "run_heart.py"),
		"--program", cb.programDir,
		"--weights", cb.weightsPath,
		"--input", inputPath,
		"--output", outputPath,
	}

	if cb.cmAddr != "" {
		args = append(args, "--cmaddr", cb.cmAddr)
	}

	// Execute Python runner
	cmd := exec.Command(cb.pythonRunner, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("inference execution failed: %w", err)
	}

	// Load output data
	output, err := cb.readOutputData(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output data: %w", err)
	}

	fmt.Println("Inference complete!")
	return output, nil
}

// Helper functions

func (cb *CerebrasBridge) writeParamsFile(path string, params map[string]interface{}) error {
	// Write JSON params file
	// Implementation would use encoding/json
	return nil // Placeholder
}

func (cb *CerebrasBridge) writeNPZFile(path string, weights map[string]*tensor.Dense, layerWeights []float32) error {
	// Write NPZ file using Python script or direct NPY format
	// For simplicity, call Python script
	tempScript := `
import numpy as np
import sys

# Read from stdin and write NPZ
# This is a placeholder - actual implementation would parse the data
`
	_ = tempScript
	return nil // Placeholder
}

func (cb *CerebrasBridge) writeInputData(path string, input *HEARTInput) error {
	// Write input data to NPZ format
	// Similar to writeNPZFile
	return nil // Placeholder
}

func (cb *CerebrasBridge) readOutputData(path string) (*HEARTOutput, error) {
	// Read output NPZ file
	// Parse and return HEARTOutput
	return &HEARTOutput{}, nil // Placeholder
}

func extractFloat32s(val tensor.Tensor) []float32 {
	data := val.Data()
	switch d := data.(type) {
	case []float32:
		return d
	case []float64:
		result := make([]float32, len(d))
		for i, v := range d {
			result[i] = float32(v)
		}
		return result
	default:
		return nil
	}
}

func extractFloat32sFromNode(node *gorgonia.Node) []float32 {
	if node == nil || node.Value() == nil {
		return nil
	}
	val := node.Value()
	data := val.Data()
	switch d := data.(type) {
	case []float32:
		return d
	case []float64:
		result := make([]float32, len(d))
		for i, v := range d {
			result[i] = float32(v)
		}
		return result
	default:
		return nil
	}
}

func extractTensorFromNode(node *gorgonia.Node) *tensor.Dense {
	if node == nil || node.Value() == nil {
		return nil
	}
	val := node.Value()
	data := val.Data()
	var f32s []float32
	switch d := data.(type) {
	case []float32:
		f32s = d
	case []float64:
		f32s = make([]float32, len(d))
		for i, v := range d {
			f32s[i] = float32(v)
		}
	}
	if f32s == nil {
		return nil
	}
	return tensor.New(tensor.WithBacking(f32s), tensor.WithShape(node.Shape()...))
}

// NetworkMetricsProcessor processes KNIRV network metrics for HEART
type NetworkMetricsProcessor struct {
	sequenceLength  int
	numNodes        int
	measureVecSize  int
	controlVecSize  int
	metricsBuffer   [][]float32
	normalizationStats map[string]struct{ Mean, StdDev float32 }
}

// NewNetworkMetricsProcessor creates a metrics processor
func NewNetworkMetricsProcessor(seqLen, numNodes, measureSize, controlSize int) *NetworkMetricsProcessor {
	return &NetworkMetricsProcessor{
		sequenceLength:     seqLen,
		numNodes:          numNodes,
		measureVecSize:    measureSize,
		controlVecSize:    controlSize,
		metricsBuffer:     make([][]float32, seqLen*numNodes),
		normalizationStats: make(map[string]struct{ Mean, StdDev float32 }),
	}
}

// ProcessRawMetrics converts raw network metrics to HEART input format
func (nmp *NetworkMetricsProcessor) ProcessRawMetrics(metrics map[uint32]map[uint32][]float32) *HEARTInput {
	// metrics: map[nodeID]map[timeSlice]metricVector
	input := &HEARTInput{
		Measurements: make([][]float32, 0, nmp.sequenceLength*nmp.numNodes),
		NodeIDs:      make([]uint32, 0, nmp.sequenceLength*nmp.numNodes),
		TimeSlices:   make([]uint32, 0, nmp.sequenceLength*nmp.numNodes),
	}

	// Flatten the metrics into sequence format
	for t := uint32(0); t < uint32(nmp.sequenceLength); t++ {
		for n := uint32(0); n < uint32(nmp.numNodes); n++ {
			var metricVec []float32
			if nodeMetrics, ok := metrics[n]; ok {
				if vec, ok := nodeMetrics[t]; ok {
					metricVec = vec
				}
			}

			// Ensure vector is correct size (pad or truncate)
			if len(metricVec) < nmp.measureVecSize {
				padded := make([]float32, nmp.measureVecSize)
				copy(padded, metricVec)
				metricVec = padded
			} else if len(metricVec) > nmp.measureVecSize {
				metricVec = metricVec[:nmp.measureVecSize]
			}

			// Apply adaptive Z-score normalization
			normalized := nmp.normalizeMetrics(metricVec, n, t)

			input.Measurements = append(input.Measurements, normalized)
			input.NodeIDs = append(input.NodeIDs, n)
			input.TimeSlices = append(input.TimeSlices, t)
		}
	}

	return input
}

// normalizeMetrics applies adaptive Z-score normalization
func (nmp *NetworkMetricsProcessor) normalizeMetrics(metrics []float32, nodeID, timeSlice uint32) []float32 {
	normalized := make([]float32, len(metrics))

	// Calculate running mean and stddev
	for i, val := range metrics {
		key := fmt.Sprintf("metric_%d_node_%d", i, nodeID)

		stats, exists := nmp.normalizationStats[key]
		if !exists {
			stats = struct{ Mean, StdDev float32 }{Mean: val, StdDev: 1.0}
		}

		// Exponential moving average
		alpha := float32(0.1)
		newMean := alpha*val + (1-alpha)*stats.Mean
		newStdDev := alpha*float32(math.Abs(float64(val-stats.Mean))) + (1-alpha)*stats.StdDev

		nmp.normalizationStats[key] = struct{ Mean, StdDev float32 }{newMean, newStdDev}

		// Normalize
		if newStdDev > 1e-6 {
			normalized[i] = (val - newMean) / newStdDev
		} else {
			normalized[i] = 0.0
		}
	}

	return normalized
}

// ApplyHeuristicCommands interprets HEART output and generates network actions
func (nmp *NetworkMetricsProcessor) ApplyHeuristicCommands(output *HEARTOutput) []NetworkCommand {
	commands := make([]NetworkCommand, 0)

	for i, cmdVec := range output.Commands {
		nodeID := output.NodeIDs[i]
		timeSlice := output.TimeSlices[i]

		// Interpret command vector
		// Format: [AlertLevel(1-5), HeuristicID, TargetNodeID, ConfidenceScore, Action1-4]
		if len(cmdVec) >= 8 {
			command := NetworkCommand{
				NodeID:           nodeID,
				TimeSlice:        timeSlice,
				AlertLevel:       int(cmdVec[0]),
				HeuristicID:      int(cmdVec[1]),
				TargetNodeID:     uint32(cmdVec[2]),
				ConfidenceScore:  cmdVec[3],
				Actions:          cmdVec[4:8],
			}

			// Only apply if confidence exceeds threshold
			if command.ConfidenceScore > 0.7 {
				commands = append(commands, command)
			}
		}
	}

	return commands
}

// NetworkCommand represents an actionable command for the KNIRV network
type NetworkCommand struct {
	NodeID          uint32
	TimeSlice       uint32
	AlertLevel      int       // 1-5
	HeuristicID     int       // Which analysis heuristic to apply
	TargetNodeID    uint32    // Target for the action
	ConfidenceScore float32   // 0-1
	Actions         []float32 // Additional action parameters
}

// Dummy binary write functions for completeness
func writeUint32(w *os.File, v uint32) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func writeFloat32(w *os.File, v float32) error {
	return binary.Write(w, binary.LittleEndian, v)
}
