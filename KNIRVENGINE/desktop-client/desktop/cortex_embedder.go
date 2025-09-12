// cortex_embedder.go - Desktop executable embedding for cortex.wasm
package desktop

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Note: load cortex.wasm at runtime from the local file path so tests can copy the file
var embeddedCortexWasm []byte

// CortexEmbedder handles embedding cortex.wasm into desktop executables
type CortexEmbedder struct {
	runtime        wazero.Runtime
	cortexModule   wazero.CompiledModule
	cortexInstance api.Module
	initialized    bool
}

// NewCortexEmbedder creates a new cortex embedder
func NewCortexEmbedder() *CortexEmbedder {
	return &CortexEmbedder{}
}

// Initialize sets up the embedded cortex.wasm runtime
func (ce *CortexEmbedder) Initialize() error {
	log.Println("Initializing embedded cortex.wasm runtime...")

	// Create WASM runtime
	ce.runtime = wazero.NewRuntime(context.Background())

	// Instantiate WASI
	_, err := wasi_snapshot_preview1.Instantiate(context.Background(), ce.runtime)
	if err != nil {
		return fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	// Compile embedded cortex.wasm
	wasmBytes := embeddedCortexWasm
	if len(wasmBytes) == 0 {
		// Attempt to load from disk (useful for tests where file is copied)
		wasmPath := filepath.Join(".", "cortex.wasm")
		wasmBytes, err = os.ReadFile(wasmPath)
		if err != nil {
			return fmt.Errorf("failed to read cortex.wasm from disk: %w", err)
		}
	}

	ce.cortexModule, err = ce.runtime.CompileModule(context.Background(), wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to compile embedded cortex.wasm: %w", err)
	}

	// Instantiate cortex module
	ce.cortexInstance, err = ce.runtime.InstantiateModule(context.Background(), ce.cortexModule, wazero.NewModuleConfig())
	if err != nil {
		return fmt.Errorf("failed to instantiate cortex module: %w", err)
	}

	ce.initialized = true
	log.Println("Embedded cortex.wasm runtime initialized successfully")
	return nil
}

// RunCognitiveTask executes a cognitive task using the embedded cortex
func (ce *CortexEmbedder) RunCognitiveTask(prompt string) (string, error) {
	if !ce.initialized {
		return "", fmt.Errorf("cortex embedder not initialized")
	}

	// Get the run_cognitive_task function
	runCognitiveTask := ce.cortexInstance.ExportedFunction("run_cognitive_task")
	if runCognitiveTask == nil {
		return "", fmt.Errorf("run_cognitive_task function not found")
	}

	// Allocate memory for the prompt
	allocateBuffer := ce.cortexInstance.ExportedFunction("allocate_buffer")
	if allocateBuffer == nil {
		return "", fmt.Errorf("allocate_buffer function not found")
	}

	promptBytes := []byte(prompt)
	results, err := allocateBuffer.Call(context.Background(), uint64(len(promptBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to allocate buffer: %w", err)
	}

	promptPtr := uint32(results[0])
	ce.writeMemory(promptPtr, promptBytes)

	// Call the cognitive task function
	results, err = runCognitiveTask.Call(context.Background(), uint64(promptPtr), uint64(len(promptBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to run cognitive task: %w", err)
	}

	// Unpack result pointer and length
	packed := results[0]
	outputPtr, outputLen := ce.unpackPtrLen(packed)

	// Read the output
	outputBytes := ce.readMemory(outputPtr, outputLen)
	return string(outputBytes), nil
}

// CompileLoRAAdapter compiles a LoRA adapter using the embedded cortex
func (ce *CortexEmbedder) CompileLoRAAdapter(skillDataJSON string) (string, error) {
	if !ce.initialized {
		return "", fmt.Errorf("cortex embedder not initialized")
	}

	// Get the compile_lora_adapter function
	compileLoRAAdapter := ce.cortexInstance.ExportedFunction("compile_lora_adapter")
	if compileLoRAAdapter == nil {
		return "", fmt.Errorf("compile_lora_adapter function not found")
	}

	// Allocate memory for the skill data
	allocateBuffer := ce.cortexInstance.ExportedFunction("allocate_buffer")
	if allocateBuffer == nil {
		return "", fmt.Errorf("allocate_buffer function not found")
	}

	skillDataBytes := []byte(skillDataJSON)
	results, err := allocateBuffer.Call(context.Background(), uint64(len(skillDataBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to allocate buffer: %w", err)
	}

	skillDataPtr := uint32(results[0])
	ce.writeMemory(skillDataPtr, skillDataBytes)

	// Call the LoRA compilation function
	results, err = compileLoRAAdapter.Call(context.Background(), uint64(skillDataPtr), uint64(len(skillDataBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to compile LoRA adapter: %w", err)
	}

	// Unpack result pointer and length
	packed := results[0]
	outputPtr, outputLen := ce.unpackPtrLen(packed)

	// Read the output
	outputBytes := ce.readMemory(outputPtr, outputLen)
	return string(outputBytes), nil
}

// InvokeLoRASkill invokes a LoRA skill using the embedded cortex
func (ce *CortexEmbedder) InvokeLoRASkill(skillRequestJSON string) (string, error) {
	if !ce.initialized {
		return "", fmt.Errorf("cortex embedder not initialized")
	}

	// Get the invoke_lora_skill function
	invokeLoRASkill := ce.cortexInstance.ExportedFunction("invoke_lora_skill")
	if invokeLoRASkill == nil {
		return "", fmt.Errorf("invoke_lora_skill function not found")
	}

	// Allocate memory for the skill request
	allocateBuffer := ce.cortexInstance.ExportedFunction("allocate_buffer")
	if allocateBuffer == nil {
		return "", fmt.Errorf("allocate_buffer function not found")
	}

	requestBytes := []byte(skillRequestJSON)
	results, err := allocateBuffer.Call(context.Background(), uint64(len(requestBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to allocate buffer: %w", err)
	}

	requestPtr := uint32(results[0])
	ce.writeMemory(requestPtr, requestBytes)

	// Call the skill invocation function
	results, err = invokeLoRASkill.Call(context.Background(), uint64(requestPtr), uint64(len(requestBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to invoke LoRA skill: %w", err)
	}

	// Unpack result pointer and length
	packed := results[0]
	outputPtr, outputLen := ce.unpackPtrLen(packed)

	// Read the output
	outputBytes := ce.readMemory(outputPtr, outputLen)
	return string(outputBytes), nil
}

// GetAvailableLoRAAdapters gets the list of available LoRA adapters
func (ce *CortexEmbedder) GetAvailableLoRAAdapters() (string, error) {
	if !ce.initialized {
		return "", fmt.Errorf("cortex embedder not initialized")
	}

	// Get the get_lora_adapters function
	getLoRAAdapters := ce.cortexInstance.ExportedFunction("get_lora_adapters")
	if getLoRAAdapters == nil {
		return "", fmt.Errorf("get_lora_adapters function not found")
	}

	// Call the function (no parameters needed)
	results, err := getLoRAAdapters.Call(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get LoRA adapters: %w", err)
	}

	// The function returns a packed pointer/length in results[0]
	packed := results[0]
	outPtr, outLen := ce.unpackPtrLen(packed)
	outBytes := ce.readMemory(outPtr, outLen)
	return string(outBytes), nil
}

// Memory management utilities
func (ce *CortexEmbedder) writeMemory(ptr uint32, data []byte) {
	memory := ce.cortexInstance.Memory()
	if memory != nil {
		// Use Memory.Write to safely write into wasm memory
		// api.Memory.Write(offset, data []byte) bool
		_ = memory.Write(uint32(ptr), data)
	}
}

func (ce *CortexEmbedder) readMemory(ptr uint32, length uint32) []byte {
	memory := ce.cortexInstance.Memory()
	if memory != nil {
		// api.Memory.Read(offset, length uint32) []byte
		out, ok := memory.Read(uint32(ptr), length)
		if ok && out != nil {
			data := make([]byte, len(out))
			copy(data, out)
			return data
		}
	}
	return nil
}

func (ce *CortexEmbedder) unpackPtrLen(packed uint64) (uint32, uint32) {
	ptr := uint32(packed >> 32)
	length := uint32(packed & 0xFFFFFFFF)
	return ptr, length
}

// Close shuts down the embedded cortex runtime
func (ce *CortexEmbedder) Close() error {
	if ce.runtime != nil {
		return ce.runtime.Close(context.Background())
	}
	return nil
}

// GetSystemInfo returns information about the embedded cortex system
func (ce *CortexEmbedder) GetSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"platform":         runtime.GOOS,
		"architecture":     runtime.GOARCH,
		"cortex_embedded":  ce.initialized,
		"cortex_wasm_size": len(embeddedCortexWasm),
		"runtime_version":  "wazero",
		"go_version":       runtime.Version(),
	}
}

// ExtractCortexWasm extracts the embedded cortex.wasm to a file (for debugging)
func (ce *CortexEmbedder) ExtractCortexWasm(outputPath string) error {
	// Ensure the directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the embedded WASM to file
	if err := os.WriteFile(outputPath, embeddedCortexWasm, 0644); err != nil {
		return fmt.Errorf("failed to write cortex.wasm: %w", err)
	}

	log.Printf("Extracted embedded cortex.wasm to: %s", outputPath)
	return nil
}

// Global embedded cortex instance
var globalCortexEmbedder *CortexEmbedder

// InitializeEmbeddedCortex initializes the global embedded cortex instance
func InitializeEmbeddedCortex() error {
	globalCortexEmbedder = NewCortexEmbedder()
	return globalCortexEmbedder.Initialize()
}

// GetEmbeddedCortex returns the global embedded cortex instance
func GetEmbeddedCortex() *CortexEmbedder {
	return globalCortexEmbedder
}
