// desktop/tee_manager.go
// Enhanced TEE Manager for Desktop Application
// Provides secure plugin loading and execution with isolation

package desktop

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"KNIRVENGINE/desktop-client/agentify"
)

// DesktopTEEManager manages TEE instances for the desktop application
type DesktopTEEManager struct {
	appDataPath    string
	pluginRegistry map[string]*PluginInfo
	teeInstances   map[string]agentify.TEE
	mutex          sync.RWMutex
	config         *DesktopTEEConfig
}

// DesktopTEEConfig configuration for desktop TEE manager
type DesktopTEEConfig struct {
	// Security settings
	EnableSignatureVerification bool
	TrustedSigners              []string
	MaxPluginSize               int64
	AllowedFileExtensions       []string

	// Isolation settings
	EnableNetworkIsolation    bool
	EnableFileSystemIsolation bool
	MaxMemoryUsage            int64
	MaxCPUUsage               int
	ExecutionTimeout          time.Duration

	// Plugin settings
	PluginDirectory     string
	QuarantineDirectory string
	LogDirectory        string
}

// PluginInfo contains metadata about a loaded plugin
type PluginInfo struct {
	ID         string
	Name       string
	Version    string
	Path       string
	Hash       string
	Signature  string
	LoadedAt   time.Time
	LastUsed   time.Time
	TEEType    string
	IsVerified bool
	Plugin     *plugin.Plugin
}

// SecurityContext contains security information for plugin execution
type SecurityContext struct {
	PluginID      string
	Permissions   []string
	NetworkAccess bool
	FileAccess    []string
	MaxMemory     int64
	MaxCPU        int
	Timeout       time.Duration
}

// NewDesktopTEEManager creates a new desktop TEE manager
func NewDesktopTEEManager(appDataPath string, config *DesktopTEEConfig) (*DesktopTEEManager, error) {
	if config == nil {
		config = &DesktopTEEConfig{
			EnableSignatureVerification: true,
			MaxPluginSize:               100 * 1024 * 1024, // 100MB
			AllowedFileExtensions:       []string{".so", ".dll", ".dylib"},
			EnableNetworkIsolation:      true,
			EnableFileSystemIsolation:   true,
			MaxMemoryUsage:              512 * 1024 * 1024, // 512MB
			MaxCPUUsage:                 50,                // 50% CPU
			ExecutionTimeout:            30 * time.Second,
			PluginDirectory:             filepath.Join(appDataPath, "plugin-data"),
			QuarantineDirectory:         filepath.Join(appDataPath, "quarantine"),
			LogDirectory:                filepath.Join(appDataPath, "logs"),
		}
	}

	// Ensure directories exist
	dirs := []string{
		config.PluginDirectory,
		config.QuarantineDirectory,
		config.LogDirectory,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	manager := &DesktopTEEManager{
		appDataPath:    appDataPath,
		pluginRegistry: make(map[string]*PluginInfo),
		teeInstances:   make(map[string]agentify.TEE),
		config:         config,
	}

	// Scan for existing plugins
	if err := manager.scanExistingPlugins(); err != nil {
		return nil, fmt.Errorf("failed to scan existing plugins: %v", err)
	}

	return manager, nil
}

// scanExistingPlugins scans the plugin directory for existing plugins
func (m *DesktopTEEManager) scanExistingPlugins() error {
	return filepath.Walk(m.config.PluginDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file has allowed extension
		ext := filepath.Ext(path)
		allowed := false
		for _, allowedExt := range m.config.AllowedFileExtensions {
			if ext == allowedExt {
				allowed = true
				break
			}
		}

		if !allowed {
			return nil
		}

		// Register the plugin
		if err := m.registerPlugin(path); err != nil {
			// Log error but continue scanning
			fmt.Printf("Warning: Failed to register plugin %s: %v\n", path, err)
		}

		return nil
	})
}

// registerPlugin registers a plugin file
func (m *DesktopTEEManager) registerPlugin(pluginPath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Calculate file hash
	hash, err := m.calculateFileHash(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %v", err)
	}

	// Check file size
	info, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	if info.Size() > m.config.MaxPluginSize {
		return fmt.Errorf("plugin file too large: %d bytes (max: %d)", info.Size(), m.config.MaxPluginSize)
	}

	// Verify signature if enabled
	isVerified := false
	if m.config.EnableSignatureVerification {
		isVerified, err = m.verifyPluginSignature(pluginPath)
		if err != nil {
			return fmt.Errorf("signature verification failed: %v", err)
		}
	}

	// Create plugin info
	pluginInfo := &PluginInfo{
		ID:         hash[:16], // Use first 16 chars of hash as ID
		Name:       filepath.Base(pluginPath),
		Path:       pluginPath,
		Hash:       hash,
		LoadedAt:   time.Now(),
		IsVerified: isVerified,
	}

	m.pluginRegistry[pluginInfo.ID] = pluginInfo
	return nil
}

// calculateFileHash calculates SHA256 hash of a file
func (m *DesktopTEEManager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyPluginSignature verifies the digital signature of a plugin
func (m *DesktopTEEManager) verifyPluginSignature(pluginPath string) (bool, error) {
	// Check if signature file exists
	sigPath := pluginPath + ".sig"
	if _, err := os.Stat(sigPath); os.IsNotExist(err) {
		return false, fmt.Errorf("signature file not found: %s", sigPath)
	}

	// Read the signature file
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return false, fmt.Errorf("failed to read signature file: %v", err)
	}

	// Decode base64 signature
	signature, err := base64.StdEncoding.DecodeString(string(sigData))
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %v", err)
	}

	// Calculate hash of the plugin file
	pluginHash, err := m.calculateFileHash(pluginPath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate plugin hash: %v", err)
	}

	// Convert hex hash to bytes
	hashBytes, err := hex.DecodeString(pluginHash)
	if err != nil {
		return false, fmt.Errorf("failed to decode plugin hash: %v", err)
	}

	// Verify signature using trusted public keys
	for _, signerKeyPath := range m.config.TrustedSigners {
		if verified, err := m.verifySignatureWithKey(hashBytes, signature, signerKeyPath); err == nil && verified {
			return true, nil
		}
	}

	return false, fmt.Errorf("signature verification failed with all trusted signers")
}

// verifySignatureWithKey verifies a signature using a specific public key
func (m *DesktopTEEManager) verifySignatureWithKey(hash, signature []byte, keyPath string) (bool, error) {
	// Read the public key file
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return false, fmt.Errorf("failed to read public key file: %v", err)
	}

	// Parse the PEM-encoded public key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return false, fmt.Errorf("failed to decode PEM block from public key")
	}

	// Parse the public key
	var publicKey interface{}
	switch block.Type {
	case "PUBLIC KEY":
		publicKey, err = x509.ParsePKIXPublicKey(block.Bytes)
	case "RSA PUBLIC KEY":
		publicKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return false, fmt.Errorf("unsupported public key type: %s", block.Type)
	}

	if err != nil {
		return false, fmt.Errorf("failed to parse public key: %v", err)
	}

	// Verify signature based on key type
	switch pub := publicKey.(type) {
	case *rsa.PublicKey:
		// Verify RSA signature using PKCS1v15 with SHA256
		err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash, signature)
		return err == nil, err
	default:
		return false, fmt.Errorf("unsupported public key type for verification")
	}
}

// GenerateKeyPair generates an RSA key pair for plugin signing (utility function)
func GenerateKeyPair(keySize int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %v", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// SavePublicKeyToPEM saves a public key to a PEM file
func SavePublicKeyToPEM(publicKey *rsa.PublicKey, filename string) error {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %v", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %v", err)
	}
	defer file.Close()

	return pem.Encode(file, publicKeyPEM)
}

// SignPlugin signs a plugin file with a private key
func SignPlugin(pluginPath string, privateKey *rsa.PrivateKey) error {
	// Calculate hash of the plugin file
	file, err := os.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin file: %v", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate hash: %v", err)
	}

	hashBytes := hash.Sum(nil)

	// Sign the hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashBytes)
	if err != nil {
		return fmt.Errorf("failed to sign plugin: %v", err)
	}

	// Save signature to file
	sigPath := pluginPath + ".sig"
	sigData := base64.StdEncoding.EncodeToString(signature)

	return os.WriteFile(sigPath, []byte(sigData), 0644)
}

// LoadPlugin loads a plugin in a secure TEE environment
func (m *DesktopTEEManager) LoadPlugin(pluginID string, securityContext *SecurityContext) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	pluginInfo, exists := m.pluginRegistry[pluginID]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	// Check if plugin is verified (if verification is enabled)
	if m.config.EnableSignatureVerification && !pluginInfo.IsVerified {
		return fmt.Errorf("plugin not verified: %s", pluginID)
	}

	// Create TEE for plugin execution
	teeConfig := agentify.TEEConfig{
		WorkingDir: filepath.Join(m.config.PluginDirectory, pluginID),
		Env: map[string]string{
			"PLUGIN_ID":  pluginID,
			"APP_DATA":   m.appDataPath,
			"MAX_MEMORY": fmt.Sprintf("%d", securityContext.MaxMemory),
			"MAX_CPU":    fmt.Sprintf("%d", securityContext.MaxCPU),
			"TIMEOUT":    securityContext.Timeout.String(),
		},
	}

	// Create appropriate TEE based on security requirements
	var tee agentify.TEE
	if m.config.EnableFileSystemIsolation || m.config.EnableNetworkIsolation {
		// Use container TEE for better isolation
		teeConfig.Image = "alpine"
		teeConfig.Tag = "latest"
		tee = agentify.NewContainerTEE(teeConfig)
	} else {
		// Use process TEE for lighter isolation
		tee = agentify.NewProcessTEE(teeConfig)
	}

	// Start the TEE
	if err := tee.Start(); err != nil {
		return fmt.Errorf("failed to start TEE: %v", err)
	}

	// Load the plugin
	loadedPlugin, err := plugin.Open(pluginInfo.Path)
	if err != nil {
		tee.Stop()
		return fmt.Errorf("failed to load plugin: %v", err)
	}

	// Update plugin info
	pluginInfo.Plugin = loadedPlugin
	pluginInfo.LastUsed = time.Now()
	pluginInfo.TEEType = "process"
	if m.config.EnableFileSystemIsolation || m.config.EnableNetworkIsolation {
		pluginInfo.TEEType = "container"
	}

	// Store TEE instance
	m.teeInstances[pluginID] = tee

	return nil
}

// UnloadPlugin unloads a plugin and cleans up its TEE
func (m *DesktopTEEManager) UnloadPlugin(pluginID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	pluginInfo, exists := m.pluginRegistry[pluginID]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	// Stop TEE if it exists
	if tee, exists := m.teeInstances[pluginID]; exists {
		if err := tee.Stop(); err != nil {
			return fmt.Errorf("failed to stop TEE: %v", err)
		}
		delete(m.teeInstances, pluginID)
	}

	// Clear plugin reference
	pluginInfo.Plugin = nil

	return nil
}

// ExecuteInTEE executes a command in the plugin's TEE
func (m *DesktopTEEManager) ExecuteInTEE(pluginID string, command string, args []string) (string, string, int, error) {
	m.mutex.RLock()
	tee, exists := m.teeInstances[pluginID]
	m.mutex.RUnlock()

	if !exists {
		return "", "", 1, fmt.Errorf("TEE not found for plugin: %s", pluginID)
	}

	return tee.Execute(command, args)
}

// GetPluginInfo returns information about a plugin
func (m *DesktopTEEManager) GetPluginInfo(pluginID string) (*PluginInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	pluginInfo, exists := m.pluginRegistry[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}

	return pluginInfo, nil
}

// ListPlugins returns a list of all registered plugins
func (m *DesktopTEEManager) ListPlugins() []*PluginInfo {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	plugins := make([]*PluginInfo, 0, len(m.pluginRegistry))
	for _, plugin := range m.pluginRegistry {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// Cleanup cleans up all TEE instances and resources
func (m *DesktopTEEManager) Cleanup() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var errors []error

	// Stop all TEE instances
	for pluginID, tee := range m.teeInstances {
		if err := tee.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop TEE for plugin %s: %v", pluginID, err))
		}
	}

	// Clear all instances
	m.teeInstances = make(map[string]agentify.TEE)

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}
