package storage

import (
	"context"
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
	tickers   map[string]*FrameTicker
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
		tickers:   make(map[string]*FrameTicker),
		compactor: make(map[string]*Compactor),
		fileStore: NewFileStorage(baseDir),
	}
}

func (s *NRVStorage) getNRVPath(collection string) string {
	return s.baseDir + "/" + collection + ".nrv"
}

func (s *NRVStorage) getOrCreateWriterLocked(collection string) (*NRVWriter, error) {
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

func (s *NRVStorage) getOrCreateTicker(collection string) (*FrameTicker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.tickers[collection]; ok {
		return t, nil
	}

	w, err := s.getOrCreateWriterLocked(collection)
	if err != nil {
		return nil, err
	}

	t := NewFrameTicker(w, time.Second)
	s.tickers[collection] = t
	return t, nil
}

func (s *NRVStorage) Insert(ctx context.Context, collection string, doc map[string]interface{}) error {
	bracket := &nrv.Bracket{
		ID:          doc["id"].(string),
		LSHSalt:     1,
		SubSecondUS: uint32(time.Now().UnixNano() / 1000),
		ASICLoops:   1,
	}

	if payload, ok := doc["payload"].(map[string]interface{}); ok {
		if projBytes, ok := payload["projections"].([]byte); ok && len(projBytes) == 64 {
			copy(bracket.Projections[:], projBytes)
		}
		if seedBytes, ok := payload["seed"].([]byte); ok && len(seedBytes) >= 4 {
			br := []byte(seedBytes)
			bracket.GoldenSeed = uint32(br[0]) | uint32(br[1])<<8 | uint32(br[2])<<16 | uint32(br[3])<<24
		}
	}

	thermo := nrv.ThermoAtmosphere{}
	if payload, ok := doc["payload"].(map[string]interface{}); ok {
		if t, ok := payload["thermo"].(map[string]interface{}); ok {
			if v, ok := t["temp_celsius"].(float64); ok {
				thermo.AvgTempC = float32(v)
			}
			if v, ok := t["voltage_v"].(float64); ok {
				thermo.PeakVoltV = float32(v)
			}
			if v, ok := t["freq_mhz"].(float64); ok {
				thermo.ClockMHz = float32(v)
			}
		}
	}

	return s.AppendBracketDirect(collection, bracket, thermo)
}

func (s *NRVStorage) AppendBracketDirect(collection string, b *nrv.Bracket, thermo nrv.ThermoAtmosphere) error {
	ticker, err := s.getOrCreateTicker(collection)
	if err != nil {
		return err
	}
	return ticker.AppendBracket(context.Background(), b, thermo)
}

func (s *NRVStorage) Find(ctx context.Context, collection, id string) (map[string]interface{}, error) {
	entry, brackets, err := s.GetFrame(ctx, collection, id)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	return nrvFrameToDocument(entry, brackets), nil
}

func nrvFrameToDocument(entry *nrv.FrameEntry, brackets []*nrv.Bracket) map[string]interface{} {
	doc := map[string]interface{}{
		"id":         entry.ID,
		"entryType":  string(types.EntryTypeMemory),
		"verified":   entry.Z3.Status == "VALID",
		"ergo_rank":  entry.Z3.Relevance,
		"timestamp":  entry.TimestampUnix,
		"linguistic": entry.Linguistic,
		"payload": map[string]interface{}{
			"thermo": map[string]float32{
				"temp_celsius": entry.Thermo.AvgTempC,
				"voltage_v":    entry.Thermo.PeakVoltV,
				"freq_mhz":     entry.Thermo.ClockMHz,
			},
			"z3": map[string]interface{}{
				"status":    entry.Z3.Status,
				"relevance": entry.Z3.Relevance,
			},
		},
	}

	if len(brackets) > 0 {
		bracketTypes := make([]string, len(brackets))
		var totalDrift float64
		var maxDrift float64
		for i, b := range brackets {
			if b.Meta != nil {
				bracketTypes[i] = string(b.Meta.Type)
				totalDrift += b.Meta.DriftScore
				if b.Meta.DriftScore > maxDrift {
					maxDrift = b.Meta.DriftScore
				}
			}
		}
		avgDrift := float64(0)
		if len(brackets) > 0 {
			avgDrift = totalDrift / float64(len(brackets))
		}
		doc["brackets"] = map[string]interface{}{
			"count":     len(brackets),
			"types":     bracketTypes,
			"avg_drift": avgDrift,
			"max_drift": maxDrift,
		}
	}

	return doc
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
		frame := entry
		brackets := reader.decodeBrackets(frame)
		docs = append(docs, nrvFrameToDocument(&frame, brackets))
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

func (s *NRVStorage) GetFrame(ctx context.Context, collection, frameID string) (*nrv.FrameEntry, []*nrv.Bracket, error) {
	reader, err := s.getOrCreateReader(collection)
	if err != nil {
		return nil, nil, err
	}
	return reader.GetFrame(frameID)
}

func (s *NRVStorage) StreamBrackets(ctx context.Context, collection string, goldOnly bool) (<-chan *nrv.Bracket, error) {
	reader, err := s.getOrCreateReader(collection)
	if err != nil {
		return nil, err
	}
	return reader.StreamBrackets(goldOnly), nil
}

func (s *NRVStorage) SetLinguistic(collection, token, unit string) error {
	ticker, err := s.getOrCreateTicker(collection)
	if err != nil {
		return err
	}
	ticker.SetLinguistic(token, unit)
	return nil
}

func (s *NRVStorage) GetReader(collection string) (*NRVReader, error) {
	return s.getOrCreateReader(collection)
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

	for _, t := range s.tickers {
		t.Stop()
	}
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
