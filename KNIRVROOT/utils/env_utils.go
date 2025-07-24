package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UpdateEnvVariable updates or adds a specific key-value pair in a .env file.
// It reads the file line by line, updates the line if the key is found,
// and writes the modified content back. If the key is not found, it adds
// the new key-value pair at the end of the file.
func UpdateEnvVariable(filePath, key, newValue string) error {
	// Construct the new line
	newLine := fmt.Sprintf("%s=%s", key, newValue)
	keyPrefix := key + "="

	// Read the file content
	file, err := os.Open(filePath)
	if err != nil {
		// If the file doesn't exist, create it with the new line
		if os.IsNotExist(err) {
			return os.WriteFile(filePath, []byte(newLine+"\n"), 0644)
		}
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	var lines []string
	foundKey := false
	scanner := bufio.NewScanner(file)

	// Read line by line and update if the key is found
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, keyPrefix) {
			lines = append(lines, newLine)
			foundKey = true
		} else {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// If the key was not found, add the new line at the end
	if !foundKey {
		lines = append(lines, newLine)
	}

	// Write the modified content back to the file
	output := strings.Join(lines, "\n") + "\n" // Add a newline at the end

	// Use a temporary file for atomic write
	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), "env-update-") // Create temp in same dir
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file

	if _, err := io.WriteString(tmpFile, output); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temporary file %s: %w", tmpPath, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file %s: %w", tmpPath, err)
	}

	// Replace the original file with the temporary file
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temporary file %s to %s: %w", tmpPath, filePath, err)
	}

	return nil
}
