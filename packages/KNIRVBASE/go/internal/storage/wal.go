package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
)

type WALEntry struct {
	FrameID        string `json:"frame_id"`
	LastGoodLength int64  `json:"last_good_length"`
	Committed      bool   `json:"committed"`
}

type WAL struct {
	path string
	mu   sync.Mutex
}

func NewWAL(path string) *WAL {
	return &WAL{path: path}
}

func (w *WAL) Begin(entry WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

func (w *WAL) Commit(frameID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	entries, err := w.readEntries()
	if err != nil {
		return err
	}

	f, err := os.Create(w.path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, entry := range entries {
		if entry.FrameID == frameID {
			entry.Committed = true
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		_, err = writer.Write(append(data, '\n'))
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

func (w *WAL) Recover() (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	entries, err := w.readEntries()
	if err != nil {
		return -1, err
	}

	var minUncommitted int64 = -1
	for _, entry := range entries {
		if !entry.Committed {
			if minUncommitted == -1 || entry.LastGoodLength < minUncommitted {
				minUncommitted = entry.LastGoodLength
			}
		}
	}

	if minUncommitted == -1 && len(entries) > 0 {
		return -1, nil
	}
	return minUncommitted, nil
}

func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return os.Remove(w.path)
}

func (w *WAL) readEntries() ([]WALEntry, error) {
	f, err := os.Open(w.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []WALEntry
	decoder := json.NewDecoder(f)
	for {
		var entry WALEntry
		if err := decoder.Decode(&entry); err != nil {
			if err.Error() == "EOF" {
				break
			}
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
