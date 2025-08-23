package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"time"
)

// FileReference represents a reference to a file
type FileReference struct {
	OriginalPath string    `json:"original_path"`
	RelativePath string    `json:"relative_path"`
	ContentHash  string    `json:"content_hash"`
	LocationHint string    `json:"location_hint"`
	FileSize     int64     `json:"file_size"`
	LastModified time.Time `json:"last_modified"`
}

// FileReferenceStrategy defines the interface for file reference strategies
type FileReferenceStrategy interface {
	GenerateLocationHint(filePath string) (string, error)
	GetRelativePath(filePath string) (string, error)
	EnsureAccessibility(filePath string) error
}

// LocalFileServerStrategy implements FileReferenceStrategy for a local file server
type LocalFileServerStrategy struct {
	BaseDir    string
	ServerPort int
	LocalIP    string
	server     *http.Server
}

// WebServerStrategy implements FileReferenceStrategy for an existing web server
type WebServerStrategy struct {
	BaseDir string
	BaseURL string
}

// FileReferenceConfig represents the configuration for file references
type FileReferenceConfig struct {
	Strategy        string   `json:"strategy"`
	BaseDir         string   `json:"base_dir"`
	ServerPort      int      `json:"server_port"`
	BaseURL         string   `json:"base_url"`
	AdditionalHints []string `json:"additional_hints"`
	ValidateAccess  bool     `json:"validate_access"`
}

// FileManager handles file operations for plugins and manifests
type FileManager struct {
	baseDir    string
	strategy   FileReferenceStrategy
	fileServer *http.Server
}

// NewFileManager creates a new file manager
func NewFileManager(baseDir string) (*FileManager, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &FileManager{
		baseDir: baseDir,
	}, nil
}

// SetFileReferenceStrategy sets the file reference strategy
func (fm *FileManager) SetFileReferenceStrategy(config FileReferenceConfig) error {
	var strategy FileReferenceStrategy
	var err error

	switch config.Strategy {
	case "local":
		strategy, err = NewLocalFileServerStrategy(config.BaseDir, config.ServerPort)
	case "web":
		strategy = NewWebServerStrategy(config.BaseDir, config.BaseURL)
	default:
		return fmt.Errorf("unsupported file reference strategy: %s", config.Strategy)
	}

	if err != nil {
		return err
	}

	fm.strategy = strategy
	return nil
}

// NewLocalFileServerStrategy creates a new local file server strategy
func NewLocalFileServerStrategy(baseDir string, serverPort int) (*LocalFileServerStrategy, error) {
	// Get local IP address
	localIP, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP address: %w", err)
	}

	return &LocalFileServerStrategy{
		BaseDir:    baseDir,
		ServerPort: serverPort,
		LocalIP:    localIP,
	}, nil
}

// GenerateLocationHint generates a location hint for a file
func (s *LocalFileServerStrategy) GenerateLocationHint(filePath string) (string, error) {
	// Get relative path
	relPath, err := s.GetRelativePath(filePath)
	if err != nil {
		return "", err
	}

	// Generate URL
	return fmt.Sprintf("http://%s:%d/%s", s.LocalIP, s.ServerPort, relPath), nil
}

// GetRelativePath gets the path of a file relative to the base directory
func (s *LocalFileServerStrategy) GetRelativePath(filePath string) (string, error) {
	// Get absolute paths
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute file path: %w", err)
	}

	absBasePath, err := filepath.Abs(s.BaseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	// Get relative path
	relPath, err := filepath.Rel(absBasePath, absFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Convert to forward slashes for consistency
	return filepath.ToSlash(relPath), nil
}

// EnsureAccessibility ensures that a file is accessible
func (s *LocalFileServerStrategy) EnsureAccessibility(filePath string) error {
	// Check if file exists
	if err := ValidateFileExists(filePath); err != nil {
		return err
	}

	// Check if file is in the base directory
	relPath, err := s.GetRelativePath(filePath)
	if err != nil {
		return err
	}

	// Check if the relative path starts with ".." (outside base directory)
	if relPath == ".." || filepath.HasPrefix(relPath, "../") {
		return fmt.Errorf("file is outside the base directory: %s", filePath)
	}

	// Start file server if not already running
	if s.server == nil {
		// Create file server
		fileServer := http.FileServer(http.Dir(s.BaseDir))

		// Create server
		s.server = &http.Server{
			Addr:    fmt.Sprintf(":%d", s.ServerPort),
			Handler: fileServer,
		}

		// Start server in a goroutine
		go func() {
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Error starting file server: %v\n", err)
			}
		}()

		// Wait for server to start
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// NewWebServerStrategy creates a new web server strategy
func NewWebServerStrategy(baseDir string, baseURL string) *WebServerStrategy {
	return &WebServerStrategy{
		BaseDir: baseDir,
		BaseURL: baseURL,
	}
}

// GenerateLocationHint generates a location hint for a file
func (s *WebServerStrategy) GenerateLocationHint(filePath string) (string, error) {
	// Get relative path
	relPath, err := s.GetRelativePath(filePath)
	if err != nil {
		return "", err
	}

	// Generate URL
	return fmt.Sprintf("%s/%s", s.BaseURL, relPath), nil
}

// GetRelativePath gets the path of a file relative to the base directory
func (s *WebServerStrategy) GetRelativePath(filePath string) (string, error) {
	// Get absolute paths
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute file path: %w", err)
	}

	absBasePath, err := filepath.Abs(s.BaseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	// Get relative path
	relPath, err := filepath.Rel(absBasePath, absFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Convert to forward slashes for consistency
	return filepath.ToSlash(relPath), nil
}

// EnsureAccessibility ensures that a file is accessible
func (s *WebServerStrategy) EnsureAccessibility(filePath string) error {
	// Check if file exists
	if err := ValidateFileExists(filePath); err != nil {
		return err
	}

	// Check if file is in the base directory
	relPath, err := s.GetRelativePath(filePath)
	if err != nil {
		return err
	}

	// Check if the relative path starts with ".." (outside base directory)
	if relPath == ".." || filepath.HasPrefix(relPath, "../") {
		return fmt.Errorf("file is outside the base directory: %s", filePath)
	}

	// Try to access the file via the web server
	url := fmt.Sprintf("%s/%s", s.BaseURL, relPath)
	resp, err := http.Head(url)
	if err != nil {
		return fmt.Errorf("failed to access file via web server: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("file is not accessible via web server: %s (status: %s)", url, resp.Status)
	}

	return nil
}

// getLocalIP gets the local IP address
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "127.0.0.1", nil
}

// ValidatePluginFile validates a plugin .so file
func (fm *FileManager) ValidatePluginFile(filePath string) error {
	// Check if file exists and is readable
	if err := ValidateFileExists(filePath); err != nil {
		return err
	}

	// Check file extension
	ext := filepath.Ext(filePath)
	if ext != ".so" {
		return fmt.Errorf("invalid plugin file extension: %s (expected .so)", ext)
	}

	// Try to load the plugin to verify it's a valid shared library
	// This is a basic validation and might not catch all issues
	_, err := plugin.Open(filePath)
	if err != nil {
		return fmt.Errorf("invalid plugin file: %w", err)
	}

	return nil
}

// ValidateManifestFile validates a manifest file
func (fm *FileManager) ValidateManifestFile(filePath string) error {
	// Check if file exists and is readable
	if err := ValidateFileExists(filePath); err != nil {
		return err
	}

	// Check file extension
	ext := filepath.Ext(filePath)
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("invalid manifest file extension: %s (expected .json, .yaml, or .yml)", ext)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}

	// For JSON files, try to parse it
	if ext == ".json" {
		var manifest map[string]interface{}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("invalid JSON in manifest file: %w", err)
		}

		// Check for required fields
		requiredFields := []string{"name", "version", "description"}
		for _, field := range requiredFields {
			if _, ok := manifest[field]; !ok {
				return fmt.Errorf("missing required field in manifest: %s", field)
			}
		}
	}

	// For YAML files, we would need a YAML parser
	// This is left as a TODO for now

	return nil
}

// GenerateFileReference generates a file reference for a file
func (fm *FileManager) GenerateFileReference(filePath string) (*FileReference, error) {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get file info
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate content hash
	contentHash, err := calculateContentHash(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate content hash: %w", err)
	}

	// Get relative path and location hint
	var relPath, locationHint string
	if fm.strategy != nil {
		// Use strategy to get relative path
		relPath, err = fm.strategy.GetRelativePath(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path: %w", err)
		}

		// Ensure file is accessible
		if err := fm.strategy.EnsureAccessibility(absPath); err != nil {
			return nil, fmt.Errorf("failed to ensure file accessibility: %w", err)
		}

		// Generate location hint
		locationHint, err = fm.strategy.GenerateLocationHint(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to generate location hint: %w", err)
		}
	} else {
		// Fallback to basic relative path
		relPath, err = filepath.Rel(fm.baseDir, absPath)
		if err != nil {
			// If we can't get a relative path, use the base name
			relPath = filepath.Base(absPath)
		}
		// Convert to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		// Create basic location hint
		locationHint = fmt.Sprintf("file://%s", relPath)
	}

	return &FileReference{
		OriginalPath: absPath,
		RelativePath: relPath,
		ContentHash:  contentHash,
		LocationHint: locationHint,
		FileSize:     info.Size(),
		LastModified: info.ModTime(),
	}, nil
}

// calculateContentHash calculates the SHA-256 hash of a file's contents
func calculateContentHash(filePath string) (string, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create hasher
	hasher := sha256.New()

	// Copy file contents to hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Get hash
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// CopyFileToBaseDir copies a file to the base directory
func (fm *FileManager) CopyFileToBaseDir(filePath string) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get base name
	baseName := filepath.Base(absPath)

	// Create destination path
	destPath := filepath.Join(fm.baseDir, baseName)

	// Copy file
	if err := copyFile(absPath, destPath); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return destPath, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Sync to ensure write is complete
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

// GetRelativePath gets the path of a file relative to the base directory
func (fm *FileManager) GetRelativePath(filePath string) (string, error) {
	// Get absolute paths
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute file path: %w", err)
	}

	absBasePath, err := filepath.Abs(fm.baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	// Get relative path
	relPath, err := filepath.Rel(absBasePath, absFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// Convert to forward slashes for consistency
	return filepath.ToSlash(relPath), nil
}

// ValidateFileExists checks if a file exists and is readable
func ValidateFileExists(filePath string) error {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		return fmt.Errorf("failed to access file: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", filePath)
	}

	// Try to open the file to verify it's readable
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return nil
}
