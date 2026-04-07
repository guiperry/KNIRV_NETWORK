package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/google/uuid"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

type NRVWriter struct {
	path     string
	keyPair  *pqc.PQCKeyPair
	wal      *WAL
	mu       sync.Mutex
	registry *nrv.Registry
	file     *os.File
}

func NewNRVWriter(path string, keyPair *pqc.PQCKeyPair) (*NRVWriter, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	wal := NewWAL(path + ".wal")

	w := &NRVWriter{
		path:    path,
		keyPair: keyPair,
		wal:     wal,
		file:    file,
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() == 0 {
		if err := w.initFile(); err != nil {
			file.Close()
			return nil, err
		}
	} else {
		recoverLen, err := wal.Recover()
		if err != nil {
			return nil, err
		}
		if recoverLen > -1 {
			if err := os.Truncate(path, recoverLen); err != nil {
				return nil, err
			}
			file, err = os.OpenFile(path, os.O_RDWR, 0644)
			if err != nil {
				return nil, err
			}
			w.file = file
		}
		if err := w.loadRegistry(); err != nil {
			file.Close()
			return nil, err
		}
	}

	return w, nil
}

func (w *NRVWriter) AppendFrame(frame *nrv.Frame, verified bool, ergoRank float64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := flockAcquire(w.file); err != nil {
		return err
	}
	defer flockRelease(w.file)

	info, err := w.file.Stat()
	if err != nil {
		return err
	}
	lastGoodLength := info.Size()

	if err := w.wal.Begin(WALEntry{
		FrameID:        frame.ID,
		LastGoodLength: lastGoodLength,
		Committed:      false,
	}); err != nil {
		return err
	}

	frameBinary, modalities := nrv.EncodeFrame(frame)

	var sig []byte
	if w.keyPair != nil {
		sig, err = w.keyPair.Sign(frameBinary)
		if err != nil {
			return fmt.Errorf("nrv: sign frame: %w", err)
		}
		w.registry.PQCManifest.FrameSignatures[frame.ID] = base64.StdEncoding.EncodeToString(sig)
	}

	currentOffset := lastGoodLength
	if _, err := w.file.Seek(0, 2); err != nil {
		return err
	}
	if _, err := w.file.Write(frameBinary); err != nil {
		return err
	}

	entry := nrv.FrameEntry{
		ID:         frame.ID,
		Offset:     currentOffset,
		Length:     len(frameBinary),
		Tombstone:  nil,
		Verified:   verified,
		ERGORank:   ergoRank,
		Modalities: modalities,
	}

	w.registry.Frames = append(w.registry.Frames, entry)
	w.registry.FrameCount++
	if verified {
		w.registry.GlobalMetrics.VerifiedFrameCount++
	}
	w.registry.GlobalMetrics.ERGORankSum += ergoRank

	if err := w.saveRegistry(); err != nil {
		return err
	}

	if err := w.wal.Commit(frame.ID); err != nil {
		return err
	}

	return nil
}

func (w *NRVWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	err := w.file.Close()
	// Best-effort WAL cleanup; prevents TempDir removal failures in tests.
	_ = w.wal.Truncate()
	return err
}

func flockAcquire(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

func flockRelease(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

func (w *NRVWriter) initFile() error {
	w.registry = &nrv.Registry{
		Version:        1,
		DatasetID:      uuid.New().String(),
		DatasetVersion: make(map[string]int64),
		Chunk0Length:   nrv.RegistryPadding,
		FrameCount:     0,
		TombstoneCount: 0,
		GlobalMetrics:  nrv.GlobalMetrics{},
		Frames:         []nrv.FrameEntry{},
		PQCManifest: nrv.PQCManifest{
			Algorithm:       "Dilithium-3",
			FrameSignatures: make(map[string]string),
		},
	}

	if w.keyPair != nil {
		w.registry.PQCManifest.KeyID = w.keyPair.ID
	}

	header := nrv.Header{
		Magic:       nrv.Magic,
		Version:     nrv.Version,
		TotalLength: uint32(nrv.HeaderSize + nrv.RegistryPadding),
	}

	if err := nrv.EncodeHeader(w.file, header); err != nil {
		return err
	}

	registryJSON, err := json.MarshalIndent(w.registry, "", "  ")
	if err != nil {
		return err
	}

	if len(registryJSON) > nrv.RegistryPadding {
		return fmt.Errorf("nrv: initial registry exceeds padding")
	}

	padding := make([]byte, nrv.RegistryPadding-len(registryJSON))
	if _, err := w.file.Write(registryJSON); err != nil {
		return err
	}
	if _, err := w.file.Write(padding); err != nil {
		return err
	}

	return nil
}

func (w *NRVWriter) loadRegistry() error {
	buf := make([]byte, nrv.RegistryPadding)
	if _, err := w.file.ReadAt(buf, nrv.HeaderSize); err != nil {
		return err
	}

	end := 0
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] != 0 && buf[i] != ' ' && buf[i] != '\n' && buf[i] != '\r' && buf[i] != '\t' {
			end = i + 1
			break
		}
	}

	if end == 0 {
		return fmt.Errorf("nrv: empty registry")
	}

	return json.Unmarshal(buf[:end], &w.registry)
}

func (w *NRVWriter) saveRegistry() error {
	registryJSON, err := json.MarshalIndent(w.registry, "", "  ")
	if err != nil {
		return err
	}

	if len(registryJSON) > nrv.RegistryPadding {
		return fmt.Errorf("nrv: registry exceeds padding")
	}

	padding := make([]byte, nrv.RegistryPadding-len(registryJSON))

	if _, err := w.file.WriteAt(registryJSON, nrv.HeaderSize); err != nil {
		return err
	}
	if _, err := w.file.WriteAt(padding, nrv.HeaderSize+int64(len(registryJSON))); err != nil {
		return err
	}

	info, err := w.file.Stat()
	if err != nil {
		return err
	}

	header := nrv.Header{
		Magic:       nrv.Magic,
		Version:     nrv.Version,
		TotalLength: uint32(info.Size()),
	}

	if err := nrv.EncodeHeader(w.file, header); err != nil {
		return err
	}

	return w.file.Sync()
}
