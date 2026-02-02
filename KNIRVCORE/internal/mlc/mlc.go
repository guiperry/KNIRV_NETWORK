package mlc

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	LibName    = "libmlc_llm.so"
	SourceRepo = "https://github.com/mlc-ai/mlc-llm"
	InstallDir = "/usr/local/lib" // Standard location for .so files
)

// EnsureEngine verifies or builds the C++ engine
func EnsureEngine() error {
	libPath := filepath.Join(InstallDir, LibName)

	if _, err := os.Stat(libPath); err == nil {
		log.Printf("[NEXUS] AI Engine %s found. Ready.", LibName)
		return nil
	}

	log.Println("[NEXUS] AI Engine not found. Starting automated build process...")
	return buildFromSource()
}

func buildFromSource() error {
	buildDir := "/tmp/mlc_build"
	os.MkdirAll(buildDir, 0755)
	defer os.RemoveAll(buildDir)

	steps := [][]string{
		{"git", "clone", "--recursive", SourceRepo, buildDir},
	}

	// Hardware Detection Logic
	cmakeFlags := "-DCMAKE_BUILD_TYPE=Release"
	if hasCUDA() {
		log.Println("[NEXUS] NVIDIA GPU detected. Compiling with CUDA support.")
		cmakeFlags += " -DUSE_CUDA=ON"
	} else {
		log.Println("[NEXUS] No NVIDIA GPU found. Falling back to Vulkan/CPU.")
		cmakeFlags += " -DUSE_VULKAN=ON"
	}

	// Build Commands
	buildSteps := [][]string{
		{"mkdir", "-p", filepath.Join(buildDir, "build")},
		{"sh", "-c", fmt.Sprintf("cd %s/build && cmake .. %s && make -j%d", buildDir, cmakeFlags, runtime.NumCPU())},
		{"cp", filepath.Join(buildDir, "build", LibName), InstallDir},
		{"ldconfig"}, // Refresh system library cache
	}
	steps = append(steps, buildSteps...)

	for _, args := range steps {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed at step %v: %w", args, err)
		}
	}

	return nil
}

func hasCUDA() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}