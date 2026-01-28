package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	APIURL        = "http://localhost:8000/completion"
	ConfigFile    = ".tinyllm-cli-config"
	Model         = "tinyllama-1.1b-chat-v1.0"
	MaxTokens     = 64 // Keep small for CPU inference speed
	ServerTimeout = 10 * time.Second
)

type Config struct {
	ModelPath  string `json:"model_path"`
	ServerPath string `json:"server_path"`
}

type APIRequest struct {
	Prompt    string  `json:"prompt"`
	NPredict  int     `json:"n_predict"`
	Temperature float64 `json:"temperature,omitempty"`
	Stop      []string `json:"stop,omitempty"`
}

type APIResponse struct {
	Content string `json:"content"`
	Stop    bool   `json:"stop"`
	// For error responses
	Error string `json:"error,omitempty"`
}

func CleanupExistingServer() {
	cmd := exec.Command("pkill", "-9", "-f", "llama-server")
	cmd.Run() // Ignore errors, just try to clean up
	time.Sleep(500 * time.Millisecond)
}

func ShutdownServer(cmd *exec.Cmd, serverStarted bool) {
	if serverStarted && cmd != nil {
		fmt.Println("Shutting down Tiny-LLM server...")
		// First try to kill the process
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Error killing server process: %v\n", err)
			// Fallback to pkill
			exec.Command("pkill", "-9", "-f", "llama-server").Run()
		}
		// Wait for process to terminate with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case <-done:
			// Process exited normally
		case <-time.After(5 * time.Second):
			// Timeout, force kill
			fmt.Println("Server shutdown timeout, force killing...")
			exec.Command("pkill", "-9", "-f", "llama-server").Run()
		}
		// Verify no remaining processes
		time.Sleep(500 * time.Millisecond)
		fmt.Println("Server shut down successfully")
	}
}

func FindModelFile() string {
	home, _ := os.UserHomeDir()
	possibleModelPaths := []string{
		filepath.Join(home, "models", "tinyllama-1.1b-chat-v1.0.Q4_0.gguf"),
		filepath.Join(home, "tinyllama-1.1b-chat-v1.0.Q4_0.gguf"),
		filepath.Join(home, "models", "llama.cpp", "models", "tinyllama-1.1b-chat-v1.0.Q4_0.gguf"),
	}

	for _, path := range possibleModelPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func FindServerExecutable() string {
	home, _ := os.UserHomeDir()
	possibleServerPaths := []string{
		filepath.Join(home, "llama.cpp", "build", "bin", "llama-server"),
		filepath.Join(home, "models", "llama.cpp", "build", "bin", "llama-server"),
		filepath.Join(home, "llama.cpp", "llama-server"),
		filepath.Join(home, "models", "llama.cpp", "llama-server"),
	}

	for _, path := range possibleServerPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func IsServerRunning() bool {
	resp, err := http.Get("http://localhost:8000/health")
	if err != nil {
		// If we get an error (e.g., connection refused), server is not running
		return false
	}
	defer resp.Body.Close()

	// If we get a response (even non-200), server is running
	return true
}

func InstallTinyLLM() error {
	// Step 1: Create models directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	modelsDir := filepath.Join(home, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return err
	}

	// Step 2: Download the model
	modelPath := filepath.Join(modelsDir, "tinyllama-1.1b-chat-v1.0.Q4_0.gguf")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Println("Downloading Tiny-LLM model (this may take a few minutes)...")
		cmd := exec.Command("wget", "https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_0.gguf", "-O", modelPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to download model: %v\nOutput: %s", err, string(output))
		}
	}

	// Step 3: Clone and compile llama.cpp
	llamaCppDir := filepath.Join(home, "llama.cpp")
	if _, err := os.Stat(llamaCppDir); os.IsNotExist(err) {
		fmt.Println("Cloning llama.cpp repository...")
		cmd := exec.Command("git", "clone", "https://github.com/ggerganov/llama.cpp.git", llamaCppDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clone llama.cpp: %v\nOutput: %s", err, string(output))
		}
	}

	// Compile llama.cpp
	fmt.Println("Compiling llama.cpp server...")
	cmd := exec.Command("make")
	cmd.Dir = llamaCppDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to compile llama.cpp: %v\nOutput: %s", err, string(output))
	}

	// Save configuration
	serverPath := filepath.Join(llamaCppDir, "build", "bin", "llama-server")
	config := &Config{ModelPath: modelPath, ServerPath: serverPath}
	if err := SaveConfig(config); err != nil {
		return err
	}

	// Start the server
	fmt.Println("Starting Tiny-LLM server...")
	StartServerWithPath(modelPath, serverPath, make(chan string))
	return nil
}

func StartServerWithPath(modelPath, serverPath string, logChan chan string) *exec.Cmd {
	// Check if server executable exists
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		logChan <- fmt.Sprintf("Error: llama-server executable not found at %s", serverPath)
		return nil
	}

	// Check if model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		logChan <- fmt.Sprintf("Error: model file not found at %s", modelPath)
		return nil
	}

	// Start the server in background
	cmd := exec.Command(serverPath, "-m", modelPath, "--host", "0.0.0.0", "--port", "8000", "-n", "512")

	// Create pipes to capture output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logChan <- fmt.Sprintf("Error creating stdout pipe: %v", err)
		return nil
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logChan <- fmt.Sprintf("Error creating stderr pipe: %v", err)
		return nil
	}

	if err := cmd.Start(); err != nil {
		logChan <- fmt.Sprintf("Error starting server: %v", err)
		return nil
	}

	logChan <- fmt.Sprintf("Server started with PID %d", cmd.Process.Pid)

	// Capture stdout using bufio.Scanner for line-by-line reading
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				logChan <- line
			}
		}
	}()

	// Capture stderr using bufio.Scanner for line-by-line reading
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				logChan <- line
			}
		}
	}()

	return cmd
}

func LoadConfig() (*Config, error) {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveConfig(config *Config) error {
	configPath := getConfigPath()
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigFile)
}

func CallTinyLLMAPI(message string) (string, error) {
	// Format message according to Tiny-LLM prompt format
	prompt := fmt.Sprintf("### Instruction:\n%s\n\n### Response:", message)

	reqBody := APIRequest{
		Prompt:      prompt,
		NPredict:    MaxTokens,
		Temperature: 0.7,
		Stop:        []string{"###", "\n\n\n"}, // Stop at next instruction or triple newline
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Create HTTP client with timeout (5 min for slow CPU inference)
	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	resp, err := client.Post(APIURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %v", err)
	}

	if apiResp.Error != "" {
		return "", fmt.Errorf("API error: %s", apiResp.Error)
	}

	if apiResp.Content != "" {
		// Clean up the response - remove common artifacts
		text := cleanLLMResponse(apiResp.Content)
		return text, nil
	}

	return "", fmt.Errorf("no response from model")
}

// cleanLLMResponse removes common artifacts from LLM output
func cleanLLMResponse(text string) string {
	// Trim whitespace
	text = strings.TrimSpace(text)

	// Remove markdown code block markers
	text = strings.ReplaceAll(text, "```", "")

	// Remove repeated prompt artifacts
	artifactsToRemove := []string{
		"### Response:",
		"### Instruction:",
		"### Expected Output:",
		"Response:",
		"Expected output:",
	}
	for _, artifact := range artifactsToRemove {
		text = strings.ReplaceAll(text, artifact, "")
	}

	// Remove leading/trailing ### markers
	for strings.HasPrefix(text, "###") {
		text = strings.TrimPrefix(text, "###")
		text = strings.TrimSpace(text)
	}

	// Trim again after cleanup
	text = strings.TrimSpace(text)

	// If empty after cleanup, return a default message
	if text == "" {
		return "(No response generated)"
	}

	return text
}
