package transformer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type CompileResult struct {
	WASMPath string
	WASMHash string
}

type WASMCompiler struct {
	tinyGoPath string
	outDir     string
}

func NewWASMCompiler(tinyGoPath, outDir string) *WASMCompiler {
	return &WASMCompiler{tinyGoPath: tinyGoPath, outDir: outDir}
}

func (c *WASMCompiler) Compile(ctx context.Context, source string, wasmType WASMType) (*CompileResult, error) {
	os.MkdirAll(c.outDir, 0755)

	srcFile, err := os.CreateTemp("", fmt.Sprintf("heart_%s_*.go", wasmType))
	if err != nil {
		return nil, err
	}
	defer os.Remove(srcFile.Name())
	if _, err := srcFile.WriteString(source); err != nil {
		return nil, err
	}
	srcFile.Close()

	outPath := filepath.Join(c.outDir, fmt.Sprintf("%s_%d.wasm", wasmType, time.Now().Unix()))

	cmd := exec.CommandContext(ctx, c.tinyGoPath, "build", "-o", outPath, "-target", "wasm", srcFile.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tinygo compile: %w\nstderr: %s", err, string(out))
	}

	data, _ := os.ReadFile(outPath)
	hash := sha256.Sum256(data)

	hashHex := hex.EncodeToString(hash[:])
	finalPath := filepath.Join(c.outDir, hashHex+".wasm")
	os.Rename(outPath, finalPath)

	return &CompileResult{WASMPath: finalPath, WASMHash: hashHex}, nil
}