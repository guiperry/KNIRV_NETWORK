package distributed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/knirv/knirvbase/internal/types"
)

// Storage interface for persistence
type Storage interface {
	Insert(collection string, doc map[string]interface{}) error
	Update(collection, id string, update map[string]interface{}) error
	Delete(collection, id string) error
	Find(collection, id string) (map[string]interface{}, error)
	FindAll(collection string) ([]map[string]interface{}, error)
}

// FileStorage implements Storage using files
type FileStorage struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileStorage(baseDir string) *FileStorage {
	os.MkdirAll(baseDir, 0755)
	return &FileStorage{baseDir: baseDir}
}

func (fs *FileStorage) getCollectionDir(collection string) string {
	return filepath.Join(fs.baseDir, collection)
}

func (fs *FileStorage) getDocPath(collection, id string) string {
	return filepath.Join(fs.getCollectionDir(collection), id+".json")
}

func (fs *FileStorage) Insert(collection string, doc map[string]interface{}) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	os.MkdirAll(fs.getCollectionDir(collection), 0755)
	path := fs.getDocPath(collection, doc["id"].(string))

	// Handle MEMORY blob
	if entryType, ok := doc["entryType"].(types.EntryType); ok && entryType == types.EntryTypeMemory {
		if payload, ok := doc["payload"].(map[string]interface{}); ok {
			if blob, hasBlob := payload["blob"]; hasBlob {
				blobPath := fs.saveBlob(collection, doc["id"].(string), blob)
				payload["blobRef"] = blobPath
				delete(payload, "blob")
				doc["payload"] = payload
			}
		}
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (fs *FileStorage) Update(collection, id string, update map[string]interface{}) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	doc, err := fs.Find(collection, id)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("not found")
	}

	for k, v := range update {
		doc[k] = v
	}

	return fs.Insert(collection, doc)
}

func (fs *FileStorage) Delete(collection, id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.getDocPath(collection, id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove blob if exists
	blobDir := filepath.Join(fs.getCollectionDir(collection), "blobs")
	blobPath := filepath.Join(blobDir, id)
	os.Remove(blobPath)

	return nil
}

func (fs *FileStorage) Find(collection, id string) (map[string]interface{}, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := fs.getDocPath(collection, id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// Load blob for MEMORY
	if entryType, ok := doc["entryType"].(string); ok && entryType == string(types.EntryTypeMemory) {
		if payload, ok := doc["payload"].(map[string]interface{}); ok {
			if blobRef, hasRef := payload["blobRef"].(string); hasRef {
				blob, err := fs.loadBlob(blobRef)
				if err == nil {
					payload["blob"] = blob
					delete(payload, "blobRef")
				}
			}
		}
	}

	return doc, nil
}

func (fs *FileStorage) FindAll(collection string) ([]map[string]interface{}, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	dir := fs.getCollectionDir(collection)
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, err
	}

	var docs []map[string]interface{}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			id := file.Name()[:len(file.Name())-5]
			doc, err := fs.Find(collection, id)
			if err != nil {
				continue
			}
			if doc != nil {
				docs = append(docs, doc)
			}
		}
	}
	return docs, nil
}

func (fs *FileStorage) saveBlob(collection, id string, blob interface{}) string {
	blobDir := filepath.Join(fs.getCollectionDir(collection), "blobs")
	os.MkdirAll(blobDir, 0755)

	blobPath := filepath.Join(blobDir, id)
	data, _ := json.Marshal(blob)
	os.WriteFile(blobPath, data, 0644)
	return blobPath
}

func (fs *FileStorage) loadBlob(blobRef string) (interface{}, error) {
	data, err := os.ReadFile(blobRef)
	if err != nil {
		return nil, err
	}
	var blob interface{}
	json.Unmarshal(data, &blob)
	return blob, nil
}
