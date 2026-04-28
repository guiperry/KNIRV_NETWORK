package mdstorage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend_server/internal/storage/pqc"
)

// MarkdownDocument represents a persistent memory node in Markdown format
type MarkdownDocument struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // ERROR, SOLUTION, TRACE, etc.
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
	Content   []byte                 `json:"-"` // Plaintext content (not saved directly)
}

// MarkdownStorageDriver handles the encryption and persistence of .md memory files
type MarkdownStorageDriver struct {
	baseDir    string
	encManager *pqc.EncryptionManager
	keyID      string
}

// NewMarkdownStorageDriver creates a new driver instance
func NewMarkdownStorageDriver(baseDir string, em *pqc.EncryptionManager, keyID string) (*MarkdownStorageDriver, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &MarkdownStorageDriver{
		baseDir:    baseDir,
		encManager: em,
		keyID:      keyID,
	}, nil
}

// SaveDocument encrypts and saves a MarkdownDocument to disk
func (d *MarkdownStorageDriver) SaveDocument(doc *MarkdownDocument) error {
	// 1. Encrypt the content (if encryption manager is available)
	var encryptedContent string
	if d.encManager != nil {
		var err error
		encryptedContent, err = d.encManager.EncryptData(doc.Content, d.keyID)
		if err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}
	} else {
		encryptedContent = string(doc.Content)
	}

	// 2. Prepare the .md file content
	var buf bytes.Buffer
	buf.WriteString("---\n")
	
	// Frontmatter metadata
	metaData := map[string]interface{}{
		"id":        doc.ID,
		"type":      doc.Type,
		"timestamp": doc.Timestamp.Format(time.RFC3339),
		"metadata":  doc.Metadata,
		"pqc_key":   d.keyID,
	}
	
	metaJSON, _ := json.MarshalIndent(metaData, "", "  ")
	buf.Write(metaJSON)
	buf.WriteString("\n---\n\n")
	
	buf.WriteString("# PQC Encrypted Payload\n\n")
	buf.WriteString("```pqc\n")
	buf.WriteString(encryptedContent)
	buf.WriteString("\n```\n")

	// 3. Write to disk
	fileName := fmt.Sprintf("%s_%s.md", strings.ToLower(doc.Type), doc.ID)
	filePath := filepath.Join(d.baseDir, fileName)
	
	return os.WriteFile(filePath, buf.Bytes(), 0644)
}

// ListDocuments scans the storage directory and returns all loadable MarkdownDocuments.
// Documents that cannot be decrypted (e.g. wrong key, corrupt file) are silently skipped.
func (d *MarkdownStorageDriver) ListDocuments() ([]*MarkdownDocument, error) {
	entries, err := os.ReadDir(d.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var docs []*MarkdownDocument
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		// Filename format: {type}_{id}.md
		name := strings.TrimSuffix(entry.Name(), ".md")
		idx := strings.Index(name, "_")
		if idx < 0 {
			continue
		}
		docType := name[:idx]
		id := name[idx+1:]
		doc, err := d.LoadDocument(docType, id)
		if err != nil {
			continue // skip unreadable/unencryptable files
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// LoadDocument reads and decrypts a .md document from disk
func (d *MarkdownStorageDriver) LoadDocument(docType, id string) (*MarkdownDocument, error) {
	fileName := fmt.Sprintf("%s_%s.md", strings.ToLower(docType), id)
	filePath := filepath.Join(d.baseDir, fileName)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Simple parser for frontmatter and PQC block
	doc := &MarkdownDocument{ID: id, Type: docType}
	var encryptedBlock string
	
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inFrontmatter := false
	inPQC := false
	var frontmatterJSON bytes.Buffer
	
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if !inFrontmatter && frontmatterJSON.Len() == 0 {
				inFrontmatter = true
				continue
			} else {
				inFrontmatter = false
				continue
			}
		}
		
		if inFrontmatter {
			frontmatterJSON.WriteString(line)
			continue
		}
		
		if line == "```pqc" {
			inPQC = true
			continue
		}
		if line == "```" && inPQC {
			inPQC = false
			continue
		}
		
		if inPQC {
			encryptedBlock += line
		}
	}

	// Unmarshal metadata
	var meta map[string]interface{}
	if err := json.Unmarshal(frontmatterJSON.Bytes(), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}
	
	if m, ok := meta["metadata"].(map[string]interface{}); ok {
		doc.Metadata = m
	}
	if ts, ok := meta["timestamp"].(string); ok {
		doc.Timestamp, _ = time.Parse(time.RFC3339, ts)
	}

	// Decrypt content
	plaintext, err := d.encManager.DecryptData(encryptedBlock)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	doc.Content = plaintext

	return doc, nil
}
