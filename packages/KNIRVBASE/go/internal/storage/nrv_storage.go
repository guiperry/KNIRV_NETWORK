package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/internal/types"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

type NRVStorage struct {
	baseDir   string
	keyPair   *pqc.PQCKeyPair
	writers   map[string]*NRVWriter
	readers   map[string]*NRVReader
	compactor map[string]*Compactor
	mu        sync.RWMutex
	fileStore *FileStorage
}

func NewNRVStorage(baseDir string, keyPair *pqc.PQCKeyPair) *NRVStorage {
	return &NRVStorage{
		baseDir:   baseDir,
		keyPair:   keyPair,
		writers:   make(map[string]*NRVWriter),
		readers:   make(map[string]*NRVReader),
		compactor: make(map[string]*Compactor),
		fileStore: NewFileStorage(baseDir),
	}
}

func (s *NRVStorage) getNRVPath(collection string) string {
	return s.baseDir + "/" + collection + ".nrv"
}

func (s *NRVStorage) getOrCreateWriter(collection string) (*NRVWriter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := s.writers[collection]; ok {
		return w, nil
	}

	path := s.getNRVPath(collection)
	w, err := NewNRVWriter(path, s.keyPair)
	if err != nil {
		return nil, err
	}

	s.writers[collection] = w

	comp := NewCompactor(path, s.keyPair)
	s.compactor[collection] = comp

	return w, nil
}

func (s *NRVStorage) getOrCreateReader(collection string) (*NRVReader, error) {
	s.mu.RLock()
	if r, ok := s.readers[collection]; ok {
		s.mu.RUnlock()
		return r, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if r, ok := s.readers[collection]; ok {
		return r, nil
	}

	path := s.getNRVPath(collection)
	r, err := NewNRVReader(path)
	if err != nil {
		return nil, err
	}

	s.readers[collection] = r
	return r, nil
}

func (s *NRVStorage) Insert(ctx context.Context, collection string, doc map[string]interface{}) error {
	frame := &nrv.Frame{
		ID: doc["id"].(string),
	}

	if payload, ok := doc["payload"].(map[string]interface{}); ok {
		switch vec := payload["vector"].(type) {
		case []interface{}:
			if len(vec) == 12 {
				for i, v := range vec {
					if f, ok := v.(float64); ok {
						frame.Vector[i] = float32(f)
					}
				}
			}
		case []float64:
			if len(vec) == 12 {
				for i, f := range vec {
					frame.Vector[i] = float32(f)
				}
			}
		case []float32:
			if len(vec) == 12 {
				copy(frame.Vector[:], vec)
			}
		}

		if seedBytes, ok := payload["seed"].([]byte); ok && len(seedBytes) == 32 {
			copy(frame.Seed[:], seedBytes)
		}

		extractThermo := func(m map[string]float64) {
			frame.Thermo.TempCelsius = float32(m["temp_celsius"])
			frame.Thermo.VoltageV = float32(m["voltage_v"])
			frame.Thermo.FreqMHz = float32(m["freq_mhz"])
			frame.Thermo.FanRPM = float32(m["fan_rpm"])
		}
		switch thermo := payload["thermo"].(type) {
		case map[string]interface{}:
			if v, ok := thermo["temp_celsius"].(float64); ok {
				frame.Thermo.TempCelsius = float32(v)
			}
			if v, ok := thermo["voltage_v"].(float64); ok {
				frame.Thermo.VoltageV = float32(v)
			}
			if v, ok := thermo["freq_mhz"].(float64); ok {
				frame.Thermo.FreqMHz = float32(v)
			}
			if v, ok := thermo["fan_rpm"].(float64); ok {
				frame.Thermo.FanRPM = float32(v)
			}
		case map[string]float64:
			extractThermo(thermo)
		case map[string]float32:
			frame.Thermo.TempCelsius = thermo["temp_celsius"]
			frame.Thermo.VoltageV = thermo["voltage_v"]
			frame.Thermo.FreqMHz = thermo["freq_mhz"]
			frame.Thermo.FanRPM = thermo["fan_rpm"]
		}

		if proofStr, ok := payload["proof"].(string); ok {
			frame.Proof = []byte(proofStr)
		} else if proofBytes, ok := payload["proof"].([]byte); ok {
			frame.Proof = proofBytes
		}
	}

	if frame.Proof == nil {
		proofJSON, _ := json.Marshal(doc)
		frame.Proof = proofJSON
	}

	verified := false
	if v, ok := doc["verified"].(bool); ok {
		verified = v
	}

	ergoRank := 0.0
	if er, ok := doc["ergo_rank"].(float64); ok {
		ergoRank = er
	}

	writer, err := s.getOrCreateWriter(collection)
	if err != nil {
		return err
	}

	return writer.AppendFrame(frame, verified, ergoRank)
}

func (s *NRVStorage) Find(ctx context.Context, collection, id string) (map[string]interface{}, error) {
	reader, err := s.getOrCreateReader(collection)
	if err != nil {
		return nil, err
	}

	frame, err := reader.GetFrame(id)
	if err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, nil
	}

	doc := map[string]interface{}{
		"id":        frame.ID,
		"entryType": string(types.EntryTypeMemory),
		"verified":  false,
		"ergo_rank": 0.0,
		"payload": map[string]interface{}{
			"vector": frame.Vector[:],
			"seed":   frame.Seed[:],
			"thermo": map[string]float32{
				"temp_celsius": frame.Thermo.TempCelsius,
				"voltage_v":    frame.Thermo.VoltageV,
				"freq_mhz":     frame.Thermo.FreqMHz,
				"fan_rpm":      frame.Thermo.FanRPM,
			},
			"proof": string(frame.Proof),
		},
	}

	for _, entry := range reader.registry.Frames {
		if entry.ID == id {
			doc["verified"] = entry.Verified
			doc["ergo_rank"] = entry.ERGORank
			break
		}
	}

	return doc, nil
}

func (s *NRVStorage) FindAll(ctx context.Context, collection string) ([]map[string]interface{}, error) {
	reader, err := s.getOrCreateReader(collection)
	if err != nil {
		return nil, err
	}

	var docs []map[string]interface{}
	for _, entry := range reader.registry.Frames {
		if entry.Tombstone != nil {
			continue
		}

		frame, err := reader.decodeFrame(entry)
		if err != nil {
			continue
		}

		doc := map[string]interface{}{
			"id":        frame.ID,
			"entryType": string(types.EntryTypeMemory),
			"verified":  entry.Verified,
			"ergo_rank": entry.ERGORank,
			"payload": map[string]interface{}{
				"vector": frame.Vector[:],
				"seed":   frame.Seed[:],
				"thermo": map[string]float32{
					"temp_celsius": frame.Thermo.TempCelsius,
					"voltage_v":    frame.Thermo.VoltageV,
					"freq_mhz":     frame.Thermo.FreqMHz,
					"fan_rpm":      frame.Thermo.FanRPM,
				},
				"proof": string(frame.Proof),
			},
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func (s *NRVStorage) Delete(ctx context.Context, collection, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	writer, ok := s.writers[collection]
	if !ok {
		return fmt.Errorf("nrv: collection not open: %s", collection)
	}

	nowNano := time.Now().UnixNano()
	found := false
	for i, entry := range writer.registry.Frames {
		if entry.ID == id {
			writer.registry.Frames[i].Tombstone = &nowNano
			writer.registry.TombstoneCount++
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("nrv: frame not found: %s", id)
	}

	if err := writer.saveRegistry(); err != nil {
		return err
	}

	if comp, ok := s.compactor[collection]; ok {
		comp.MaybeCompact(writer.registry)
	}

	return nil
}

func (s *NRVStorage) Update(ctx context.Context, collection, id string, update map[string]interface{}) error {
	if err := s.Delete(ctx, collection, id); err != nil {
		return err
	}

	doc, err := s.Find(ctx, collection, id)
	if err != nil {
		return err
	}
	if doc == nil {
		doc = make(map[string]interface{})
	}

	for k, v := range update {
		doc[k] = v
	}

	return s.Insert(ctx, collection, doc)
}

func (s *NRVStorage) GetModality(ctx context.Context, collection, frameID string, mod nrv.ModalityType) ([]byte, error) {
	reader, err := s.getOrCreateReader(collection)
	if err != nil {
		return nil, err
	}
	return reader.GetModality(frameID, mod)
}

func (s *NRVStorage) StreamFrames(ctx context.Context, collection string, modalityFilter nrv.ModalityType) (<-chan *nrv.Frame, error) {
	reader, err := s.getOrCreateReader(collection)
	if err != nil {
		return nil, err
	}
	return reader.StreamFrames(modalityFilter), nil
}

func (s *NRVStorage) Put(ctx context.Context, key string, value []byte) error {
	return s.fileStore.Put(ctx, key, value)
}

func (s *NRVStorage) Get(ctx context.Context, key string) ([]byte, error) {
	return s.fileStore.Get(ctx, key)
}

func (s *NRVStorage) DeleteKey(ctx context.Context, key string) error {
	return s.fileStore.DeleteKey(ctx, key)
}

func (s *NRVStorage) StoreObject(ctx context.Context, key string, obj interface{}) error {
	return s.fileStore.StoreObject(ctx, key, obj)
}

func (s *NRVStorage) GetObject(ctx context.Context, key string, dest interface{}) error {
	return s.fileStore.GetObject(ctx, key, dest)
}

func (s *NRVStorage) ProjectToMarkdown(ctx context.Context, key string, targetPath string) error {
	return s.fileStore.ProjectToMarkdown(ctx, key, targetPath)
}

func (s *NRVStorage) CreateIndex(ctx context.Context, collection, name string, indexType IndexType, fields []string, unique bool, partialExpr string, options map[string]interface{}) error {
	return s.fileStore.CreateIndex(ctx, collection, name, indexType, fields, unique, partialExpr, options)
}

func (s *NRVStorage) DropIndex(ctx context.Context, collection, name string) error {
	return s.fileStore.DropIndex(ctx, collection, name)
}

func (s *NRVStorage) GetIndex(ctx context.Context, collection, name string) *Index {
	return s.fileStore.GetIndex(ctx, collection, name)
}

func (s *NRVStorage) GetIndexesForCollection(ctx context.Context, collection string) []*Index {
	return s.fileStore.GetIndexesForCollection(ctx, collection)
}

func (s *NRVStorage) QueryIndex(ctx context.Context, collection, indexName string, query map[string]interface{}) ([]string, error) {
	return s.fileStore.QueryIndex(ctx, collection, indexName, query)
}

func (s *NRVStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, w := range s.writers {
		w.Close()
	}
	for _, r := range s.readers {
		r.Close()
	}
	for _, c := range s.compactor {
		c.Stop()
	}

	return nil
}
