package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

type NRVReader struct {
	path     string
	data     []byte
	registry *nrv.Registry
}

func NewNRVReader(path string) (*NRVReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, int(info.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	registry := &nrv.Registry{}
	buf := data[nrv.HeaderSize : nrv.HeaderSize+nrv.RegistryPadding]

	end := 0
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] != 0 && buf[i] != ' ' && buf[i] != '\n' && buf[i] != '\r' && buf[i] != '\t' {
			end = i + 1
			break
		}
	}

	if end > 0 {
		if err := binaryDecodeRegistry(buf[:end], registry); err != nil {
			syscall.Munmap(data)
			return nil, err
		}
	}

	return &NRVReader{
		path:     path,
		data:     data,
		registry: registry,
	}, nil
}

func (r *NRVReader) GetFrame(id string) (*nrv.FrameEntry, []*nrv.Bracket, error) {
	for _, entry := range r.registry.Frames {
		if entry.ID == id {
			if entry.Tombstone != nil {
				return nil, nil, nil
			}
			brackets := r.decodeBrackets(entry)
			return &entry, brackets, nil
		}
	}
	return nil, nil, fmt.Errorf("nrv: frame not found: %s", id)
}

func (r *NRVReader) GetFrameEntry(id string) (*nrv.FrameEntry, error) {
	for _, entry := range r.registry.Frames {
		if entry.ID == id {
			if entry.Tombstone != nil {
				return nil, nil
			}
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("nrv: frame not found: %s", id)
}

func (r *NRVReader) StreamBrackets(goldOnly bool) <-chan *nrv.Bracket {
	ch := make(chan *nrv.Bracket, 256)
	go func() {
		defer close(ch)
		for _, entry := range r.registry.Frames {
			if entry.Tombstone != nil {
				continue
			}
			if goldOnly && entry.Z3.Status != "VALID" {
				continue
			}
			brackets := r.decodeBrackets(entry)
			for _, b := range brackets {
				ch <- b
			}
		}
	}()
	return ch
}

func (r *NRVReader) VerifyFrame(id string, keyPair *pqc.PQCKeyPair) (bool, error) {
	for _, entry := range r.registry.Frames {
		if entry.ID == id {
			sigStr, ok := r.registry.PQCManifest.FrameSignatures[id]
			if !ok {
				return false, fmt.Errorf("nrv: no signature for frame")
			}
			sig, err := base64.StdEncoding.DecodeString(sigStr)
			if err != nil {
				return false, err
			}
			bracketData := r.data[entry.Brackets.Offset : entry.Brackets.Offset+int64(entry.Brackets.Length)]
			return keyPair.Verify(bracketData, sig), nil
		}
	}
	return false, fmt.Errorf("nrv: frame not found: %s", id)
}

func (r *NRVReader) Close() error {
	return syscall.Munmap(r.data)
}

func (r *NRVReader) decodeBrackets(entry nrv.FrameEntry) []*nrv.Bracket {
	buf := r.data[entry.Brackets.Offset : entry.Brackets.Offset+int64(entry.Brackets.Length)]
	anchors := make(map[string]*nrv.Bracket)
	brackets := make([]*nrv.Bracket, len(entry.BracketIndex))

	for i, meta := range entry.BracketIndex {
		var raw [nrv.BracketSize]byte
		copy(raw[:], buf[meta.Offset:meta.Offset+nrv.BracketSize])
		b := nrv.DecodeBracket(raw)
		b.ID = meta.ID

		if meta.Type == nrv.DeltaTypeP && meta.AnchorID != nil {
			if anchor, ok := anchors[*meta.AnchorID]; ok {
				b.Projections = nrv.ApplyProjectionDelta(b.Projections, anchor.Projections)
			}
		}

		anchors[meta.ID] = &b
		brackets[i] = &b
	}
	return brackets
}

func binaryDecodeRegistry(data []byte, registry *nrv.Registry) error {
	return json.Unmarshal(data, registry)
}

func (r *NRVReader) GetRegistry() *nrv.Registry {
	return r.registry
}
