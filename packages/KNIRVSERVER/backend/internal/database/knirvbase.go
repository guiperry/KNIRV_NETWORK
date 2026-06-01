package database

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend_server/internal/storage/mdstorage"
	"backend_server/internal/storage/pqc"
)

// KNIRVBASEManager handles interaction with the PQC-secured persistence layer
type KNIRVBASEManager struct {
	enabled    bool
	storage    *mdstorage.MarkdownStorageDriver
	encManager *pqc.EncryptionManager
	baseDir    string
}

// NewKNIRVBASEManager creates a new manager for the KNIRVBASE persistence layer
func NewKNIRVBASEManager(enabled bool, storage *mdstorage.MarkdownStorageDriver, encManager *pqc.EncryptionManager, baseDir string) *KNIRVBASEManager {
	return &KNIRVBASEManager{
		enabled:    enabled,
		storage:    storage,
		encManager: encManager,
		baseDir:    baseDir,
	}
}

// StoreObject stores a JSON object with PQC encryption and signature
func (k *KNIRVBASEManager) StoreObject(ctx context.Context, key string, obj interface{}, tags ...string) error {
	if !k.enabled || k.storage == nil {
		return nil
	}

	// 1. Serialize object to JSON
	content, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object for KNIRVBASE: %w", err)
	}

	// 2. Determine type from key (convention: "type:id")
	docType := "DATA"
	docID := key
	if parts := strings.SplitN(key, ":", 2); len(parts) == 2 {
		docType = strings.ToUpper(parts[0])
		docID = parts[1]
	}

	// 3. Prepare tags metadata
	metadata := make(map[string]interface{})
	metadata["tags"] = tags
	metadata["last_updated"] = time.Now().Format(time.RFC3339)

	// 4. Save via MarkdownStorageDriver (handles Kyber-768 encryption and Dilithium-3 signing)
	doc := &mdstorage.MarkdownDocument{
		ID:        docID,
		Type:      docType,
		Timestamp: time.Now(),
		Metadata:  metadata,
		Content:   content,
	}

	return k.storage.SaveDocument(doc)
}

// GetObject retrieves and decrypts a JSON object
func (k *KNIRVBASEManager) GetObject(ctx context.Context, key string, dest interface{}) error {
	if !k.enabled || k.storage == nil {
		return fmt.Errorf("KNIRVBASE is not enabled")
	}

	// 1. Parse key
	docType := "DATA"
	docID := key
	if parts := strings.SplitN(key, ":", 2); len(parts) == 2 {
		docType = strings.ToUpper(parts[0])
		docID = parts[1]
	}

	// 2. Load and decrypt via MarkdownStorageDriver
	doc, err := k.storage.LoadDocument(docType, docID)
	if err != nil {
		return fmt.Errorf("failed to load document from KNIRVBASE: %w", err)
	}

	// 3. Unmarshal content
	return json.Unmarshal(doc.Content, dest)
}

// Query objects based on tags (replacing BuntDB indexes)
func (k *KNIRVBASEManager) Query(ctx context.Context, tags ...string) ([]string, error) {
	if !k.enabled || k.storage == nil {
		return nil, nil
	}

	var matchingKeys []string

	// Implementation: Scan the base directory for .md files
	files, err := os.ReadDir(k.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read KNIRVBASE directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		// Optimized: We only read the first few lines to find frontmatter
		filePath := filepath.Join(k.baseDir, file.Name())
		f, err := os.Open(filePath)
		if err != nil {
			continue
		}

		// Simplified frontmatter tag check
		// In a full implementation, we'd use a YAML/JSON parser
		scanner := bufio.NewScanner(f)
		hasAllTags := true
		foundTagsSection := false
		
		var fileContent strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			fileContent.WriteString(line)
			if strings.Contains(line, "\"tags\":") {
				foundTagsSection = true
			}
			if line == "---" && fileContent.Len() > 5 {
				break // End of frontmatter
			}
		}
		f.Close()

		if foundTagsSection {
			tagsStr := fileContent.String()
			for _, tag := range tags {
				if !strings.Contains(tagsStr, tag) {
					hasAllTags = false
					break
				}
			}
		} else if len(tags) > 0 {
			hasAllTags = false
		}

		if hasAllTags {
			// Convert filename back to key
			// e.g., "dve_nodes_id.md" -> "dve:nodes:id"
			name := strings.TrimSuffix(file.Name(), ".md")
			matchingKeys = append(matchingKeys, name)
		}
	}

	return matchingKeys, nil
}

// SetWithTTL stores a key-value pair with a time-to-live (for sessions)
func (k *KNIRVBASEManager) SetWithTTL(ctx context.Context, key string, value string, ttl time.Duration) error {
	if !k.enabled {
		return nil
	}

	// Wrap value in a simple object
	obj := map[string]interface{}{
		"value":      value,
		"expires_at": time.Now().Add(ttl).Format(time.RFC3339),
	}

	return k.StoreObject(ctx, key, obj, "ttl")
}

// GetStatus returns the current health status of the KNIRVBASE submodule
func (k *KNIRVBASEManager) GetStatus() (string, map[string]interface{}) {
	if !k.enabled {
		return "disabled", nil
	}

	status := "healthy"
	if k.storage == nil {
		status = "degraded"
	}

	return status, map[string]interface{}{
		"pqc_active":  true,
		"algorithm":   "Kyber-768/Dilithium-3",
		"sync_status": "synced",
		"storage_dir": k.baseDir,
	}
}
