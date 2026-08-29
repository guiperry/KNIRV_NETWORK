package api

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FilesystemConnection implements TargetSystemConnection for filesystem access
type FilesystemConnection struct {
	target       *TargetSystem
	basePath     string
	allowedPaths []string
	deniedPaths  []string
	readOnly     bool
	maxFileSize  int64
	connected    bool
	startTime    time.Time
	mutex        sync.RWMutex
}

// NewFilesystemConnection creates a new filesystem connection
func NewFilesystemConnection(target *TargetSystem) (TargetSystemConnection, error) {
	conn := &FilesystemConnection{
		target:      target,
		maxFileSize: 100 * 1024 * 1024, // 100MB default
	}

	// Configure from target config
	if basePath, ok := target.Config["basePath"].(string); ok {
		conn.basePath = basePath
	} else {
		conn.basePath = "/tmp" // Default safe path
	}

	if allowedPaths, ok := target.Config["allowedPaths"].([]interface{}); ok {
		for _, path := range allowedPaths {
			if pathStr, ok := path.(string); ok {
				conn.allowedPaths = append(conn.allowedPaths, pathStr)
			}
		}
	}

	if deniedPaths, ok := target.Config["deniedPaths"].([]interface{}); ok {
		for _, path := range deniedPaths {
			if pathStr, ok := path.(string); ok {
				conn.deniedPaths = append(conn.deniedPaths, pathStr)
			}
		}
	}

	if readOnly, ok := target.Config["readOnly"].(bool); ok {
		conn.readOnly = readOnly
	}

	if maxSize, ok := target.Config["maxFileSize"].(float64); ok {
		conn.maxFileSize = int64(maxSize)
	}

	return conn, nil
}

// Connect establishes the filesystem connection
func (c *FilesystemConnection) Connect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return nil
	}

	// Validate base path exists and is accessible
	if _, err := os.Stat(c.basePath); err != nil {
		return fmt.Errorf("base path not accessible: %v", err)
	}

	// Test read access
	testFile := filepath.Join(c.basePath, ".agentic_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		if !c.readOnly {
			return fmt.Errorf("write access test failed: %v", err)
		}
	} else {
		os.Remove(testFile) // Clean up test file
	}

	c.connected = true
	c.startTime = time.Now()
	return nil
}

// Disconnect closes the filesystem connection
func (c *FilesystemConnection) Disconnect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.connected = false
	return nil
}

// IsConnected returns the connection status
func (c *FilesystemConnection) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// GetCapabilities returns available filesystem capabilities
func (c *FilesystemConnection) GetCapabilities() []string {
	capabilities := []string{
		"read_file",
		"list_directory",
		"get_file_info",
		"search_files",
		"watch_directory",
	}

	if !c.readOnly {
		capabilities = append(capabilities, []string{
			"write_file",
			"create_directory",
			"delete_file",
			"delete_directory",
			"move_file",
			"copy_file",
			"set_permissions",
		}...)
	}

	return capabilities
}

// Execute executes a filesystem operation
func (c *FilesystemConnection) Execute(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("filesystem not connected")
	}

	switch operation {
	case "read_file":
		return c.readFile(ctx, params)
	case "write_file":
		return c.writeFile(ctx, params)
	case "list_directory":
		return c.listDirectory(ctx, params)
	case "create_directory":
		return c.createDirectory(ctx, params)
	case "delete_file":
		return c.deleteFile(ctx, params)
	case "delete_directory":
		return c.deleteDirectory(ctx, params)
	case "get_file_info":
		return c.getFileInfo(ctx, params)
	case "move_file":
		return c.moveFile(ctx, params)
	case "copy_file":
		return c.copyFile(ctx, params)
	case "search_files":
		return c.searchFiles(ctx, params)
	case "set_permissions":
		return c.setPermissions(ctx, params)
	case "watch_directory":
		return c.watchDirectory(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// GetStatus returns detailed filesystem status
func (c *FilesystemConnection) GetStatus() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	status := map[string]interface{}{
		"connected":   c.connected,
		"type":        "filesystem",
		"basePath":    c.basePath,
		"readOnly":    c.readOnly,
		"maxFileSize": c.maxFileSize,
		"uptime":      time.Since(c.startTime).String(),
	}

	if c.connected {
		// Get disk usage info
		if usage, err := c.getDiskUsage(); err == nil {
			status["diskUsage"] = usage
		}
	}

	return status
}

// GetType returns the target system type
func (c *FilesystemConnection) GetType() TargetSystemType {
	return TargetTypeFilesystem
}

// Helper methods

// validatePath checks if a path is allowed
func (c *FilesystemConnection) validatePath(path string) error {
	// Clean and resolve the path
	cleanPath := filepath.Clean(path)

	// Make it absolute if relative
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(c.basePath, cleanPath)
	}

	// Check if path is within base path
	if !strings.HasPrefix(cleanPath, c.basePath) {
		return fmt.Errorf("path outside base directory: %s", path)
	}

	// Check denied paths
	for _, denied := range c.deniedPaths {
		if strings.HasPrefix(cleanPath, denied) {
			return fmt.Errorf("path is denied: %s", path)
		}
	}

	// Check allowed paths (if specified)
	if len(c.allowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range c.allowedPaths {
			if strings.HasPrefix(cleanPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path not in allowed list: %s", path)
		}
	}

	return nil
}

// resolvePath resolves a path relative to base path
func (c *FilesystemConnection) resolvePath(path string) (string, error) {
	if err := c.validatePath(path); err != nil {
		return "", err
	}

	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(c.basePath, cleanPath)
	}

	return cleanPath, nil
}

// getDiskUsage gets disk usage information
func (c *FilesystemConnection) getDiskUsage() (map[string]interface{}, error) {
	var stat fs.FileInfo
	var err error

	if stat, err = os.Stat(c.basePath); err != nil {
		return nil, err
	}

	// Basic file info (more detailed disk usage would require platform-specific code)
	return map[string]interface{}{
		"path":    c.basePath,
		"exists":  true,
		"isDir":   stat.IsDir(),
		"size":    stat.Size(),
		"modTime": stat.ModTime(),
	}, nil
}

// setPermissions sets file permissions
func (c *FilesystemConnection) setPermissions(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	if c.readOnly {
		return nil, fmt.Errorf("filesystem is read-only")
	}

	path := getStringParam(params, "path", "")
	if path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	mode := getStringParam(params, "mode", "")
	if mode == "" {
		return nil, fmt.Errorf("mode parameter is required")
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Parse mode string (e.g., "0755", "755")
	var fileMode os.FileMode
	if strings.HasPrefix(mode, "0") {
		// Octal mode
		if parsed, err := strconv.ParseUint(mode, 8, 32); err == nil {
			fileMode = os.FileMode(parsed)
		} else {
			return nil, fmt.Errorf("invalid octal mode: %s", mode)
		}
	} else {
		// Try parsing as decimal then convert to octal
		if parsed, err := strconv.ParseUint(mode, 8, 32); err == nil {
			fileMode = os.FileMode(parsed)
		} else {
			return nil, fmt.Errorf("invalid mode: %s", mode)
		}
	}

	if err := os.Chmod(resolvedPath, fileMode); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
		"mode":    fileMode.String(),
	}, nil
}

// watchDirectory watches a directory for changes (simplified implementation)
func (c *FilesystemConnection) watchDirectory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	path := getStringParam(params, "path", "")
	if path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Check if directory exists
	if stat, err := os.Stat(resolvedPath); err != nil {
		return nil, fmt.Errorf("directory not found: %v", err)
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	// For now, just return the current state
	// In a real implementation, this would set up a file watcher
	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, map[string]interface{}{
			"name":    entry.Name(),
			"isDir":   entry.IsDir(),
			"size":    info.Size(),
			"modTime": info.ModTime(),
		})
	}

	return map[string]interface{}{
		"success":   true,
		"path":      path,
		"watching":  true,
		"files":     files,
		"count":     len(files),
		"timestamp": time.Now(),
	}, nil
}

// Filesystem operation implementations

// readFile reads a file
func (c *FilesystemConnection) readFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	path := getStringParam(params, "path", "")
	if path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Check file size
	if stat, err := os.Stat(resolvedPath); err != nil {
		return nil, fmt.Errorf("file not found: %v", err)
	} else if stat.Size() > c.maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", stat.Size(), c.maxFileSize)
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"data":    string(data),
		"size":    len(data),
		"path":    path,
	}, nil
}

// writeFile writes a file
func (c *FilesystemConnection) writeFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	if c.readOnly {
		return nil, fmt.Errorf("filesystem is read-only")
	}

	path := getStringParam(params, "path", "")
	data := getStringParam(params, "data", "")
	if path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Check data size
	if int64(len(data)) > c.maxFileSize {
		return nil, fmt.Errorf("data too large: %d bytes (max: %d)", len(data), c.maxFileSize)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	// Write file
	if err := os.WriteFile(resolvedPath, []byte(data), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
		"size":    len(data),
	}, nil
}

// listDirectory lists directory contents
func (c *FilesystemConnection) listDirectory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	path := getStringParam(params, "path", "")
	if path == "" {
		path = "." // Current directory
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, map[string]interface{}{
			"name":    entry.Name(),
			"isDir":   entry.IsDir(),
			"size":    info.Size(),
			"modTime": info.ModTime(),
			"mode":    info.Mode().String(),
		})
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
		"files":   files,
		"count":   len(files),
	}, nil
}

// createDirectory creates a directory
func (c *FilesystemConnection) createDirectory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	if c.readOnly {
		return nil, fmt.Errorf("filesystem is read-only")
	}

	path := getStringParam(params, "path", "")
	if path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	recursive := getBoolParam(params, "recursive", true)

	var createErr error
	if recursive {
		createErr = os.MkdirAll(resolvedPath, 0755)
	} else {
		createErr = os.Mkdir(resolvedPath, 0755)
	}

	if createErr != nil {
		return nil, fmt.Errorf("failed to create directory: %v", createErr)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

// deleteFile deletes a file
func (c *FilesystemConnection) deleteFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	if c.readOnly {
		return nil, fmt.Errorf("filesystem is read-only")
	}

	path := getStringParam(params, "path", "")
	if path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Check if it's a file
	if stat, err := os.Stat(resolvedPath); err != nil {
		return nil, fmt.Errorf("file not found: %v", err)
	} else if stat.IsDir() {
		return nil, fmt.Errorf("path is a directory, use delete_directory instead")
	}

	if err := os.Remove(resolvedPath); err != nil {
		return nil, fmt.Errorf("failed to delete file: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

// deleteDirectory deletes a directory
func (c *FilesystemConnection) deleteDirectory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	if c.readOnly {
		return nil, fmt.Errorf("filesystem is read-only")
	}

	path := getStringParam(params, "path", "")
	if path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Check if it's a directory
	if stat, err := os.Stat(resolvedPath); err != nil {
		return nil, fmt.Errorf("directory not found: %v", err)
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	recursive := getBoolParam(params, "recursive", false)

	var deleteErr error
	if recursive {
		deleteErr = os.RemoveAll(resolvedPath)
	} else {
		deleteErr = os.Remove(resolvedPath)
	}

	if deleteErr != nil {
		return nil, fmt.Errorf("failed to delete directory: %v", deleteErr)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

// getFileInfo gets file information
func (c *FilesystemConnection) getFileInfo(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	path := getStringParam(params, "path", "")
	if path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
		"name":    stat.Name(),
		"size":    stat.Size(),
		"isDir":   stat.IsDir(),
		"mode":    stat.Mode().String(),
		"modTime": stat.ModTime(),
	}, nil
}

// moveFile moves/renames a file
func (c *FilesystemConnection) moveFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	if c.readOnly {
		return nil, fmt.Errorf("filesystem is read-only")
	}

	srcPath := getStringParam(params, "src", "")
	dstPath := getStringParam(params, "dst", "")
	if srcPath == "" || dstPath == "" {
		return nil, fmt.Errorf("src and dst parameters are required")
	}

	resolvedSrc, err := c.resolvePath(srcPath)
	if err != nil {
		return nil, fmt.Errorf("invalid src path: %v", err)
	}

	resolvedDst, err := c.resolvePath(dstPath)
	if err != nil {
		return nil, fmt.Errorf("invalid dst path: %v", err)
	}

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(resolvedDst), 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %v", err)
	}

	if err := os.Rename(resolvedSrc, resolvedDst); err != nil {
		return nil, fmt.Errorf("failed to move file: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"src":     srcPath,
		"dst":     dstPath,
	}, nil
}

// copyFile copies a file
func (c *FilesystemConnection) copyFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	if c.readOnly {
		return nil, fmt.Errorf("filesystem is read-only")
	}

	srcPath := getStringParam(params, "src", "")
	dstPath := getStringParam(params, "dst", "")
	if srcPath == "" || dstPath == "" {
		return nil, fmt.Errorf("src and dst parameters are required")
	}

	resolvedSrc, err := c.resolvePath(srcPath)
	if err != nil {
		return nil, fmt.Errorf("invalid src path: %v", err)
	}

	resolvedDst, err := c.resolvePath(dstPath)
	if err != nil {
		return nil, fmt.Errorf("invalid dst path: %v", err)
	}

	// Open source file
	srcFile, err := os.Open(resolvedSrc)
	if err != nil {
		return nil, fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(resolvedDst), 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Create destination file
	dstFile, err := os.Create(resolvedDst)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	// Copy data
	copied, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"src":     srcPath,
		"dst":     dstPath,
		"size":    copied,
	}, nil
}

// searchFiles searches for files
func (c *FilesystemConnection) searchFiles(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	path := getStringParam(params, "path", ".")
	pattern := getStringParam(params, "pattern", "*")
	recursive := getBoolParam(params, "recursive", false)

	resolvedPath, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	var matches []map[string]interface{}

	if recursive {
		_ = filepath.WalkDir(resolvedPath, func(walkPath string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			// Check if path is still within allowed bounds
			if validateErr := c.validatePath(walkPath); validateErr != nil {
				return nil // Skip invalid paths
			}

			matched, matchErr := filepath.Match(pattern, d.Name())
			if matchErr != nil {
				return nil // Skip match errors
			}

			if matched {
				info, infoErr := d.Info()
				if infoErr == nil {
					relPath, _ := filepath.Rel(resolvedPath, walkPath)
					matches = append(matches, map[string]interface{}{
						"path":    relPath,
						"name":    d.Name(),
						"isDir":   d.IsDir(),
						"size":    info.Size(),
						"modTime": info.ModTime(),
					})
				}
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %v", err)
		}

		for _, entry := range entries {
			matched, matchErr := filepath.Match(pattern, entry.Name())
			if matchErr != nil {
				continue
			}

			if matched {
				info, infoErr := entry.Info()
				if infoErr == nil {
					matches = append(matches, map[string]interface{}{
						"path":    entry.Name(),
						"name":    entry.Name(),
						"isDir":   entry.IsDir(),
						"size":    info.Size(),
						"modTime": info.ModTime(),
					})
				}
			}
		}
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
		"pattern": pattern,
		"matches": matches,
		"count":   len(matches),
	}, nil
}
