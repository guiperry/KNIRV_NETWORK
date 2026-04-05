package storage

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
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

func (r *NRVReader) GetFrame(id string) (*nrv.Frame, error) {
	for _, entry := range r.registry.Frames {
		if entry.ID == id {
			if entry.Tombstone != nil {
				return nil, nil
			}
			return r.decodeFrame(entry)
		}
	}
	return nil, fmt.Errorf("nrv: frame not found: %s", id)
}

func (r *NRVReader) GetModality(frameID string, modality nrv.ModalityType) ([]byte, error) {
	for _, entry := range r.registry.Frames {
		if entry.ID == frameID {
			if entry.Tombstone != nil {
				return nil, fmt.Errorf("nrv: frame tombstoned: %s", frameID)
			}
			modIndex, ok := entry.Modalities[modality]
			if !ok {
				return nil, fmt.Errorf("nrv: modality not found: %s", modality)
			}
			start := entry.Offset + int64(modIndex.Offset)
			end := start + int64(modIndex.Length)
			return r.data[start:end], nil
		}
	}
	return nil, fmt.Errorf("nrv: frame not found: %s", frameID)
}

func (r *NRVReader) StreamFrames(modalityFilter nrv.ModalityType) <-chan *nrv.Frame {
	ch := make(chan *nrv.Frame)
	go func() {
		defer close(ch)
		for _, entry := range r.registry.Frames {
			if entry.Tombstone != nil {
				continue
			}
			frame, err := r.decodeFrame(entry)
			if err != nil {
				continue
			}
			ch <- frame
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
			frameData := r.data[entry.Offset : entry.Offset+int64(entry.Length)]
			return keyPair.Verify(frameData, sig), nil
		}
	}
	return false, fmt.Errorf("nrv: frame not found: %s", id)
}

func (r *NRVReader) Close() error {
	return syscall.Munmap(r.data)
}

func (r *NRVReader) decodeFrame(entry nrv.FrameEntry) (*nrv.Frame, error) {
	frameData := r.data[entry.Offset : entry.Offset+int64(entry.Length)]

	frame := &nrv.Frame{
		ID: entry.ID,
	}

	if len(frameData) < 96 {
		return nil, fmt.Errorf("nrv: frame too small")
	}

	for i := 0; i < 12; i++ {
		bits := binary.LittleEndian.Uint32(frameData[i*4:])
		frame.Vector[i] = math.Float32frombits(bits)
	}

	copy(frame.Seed[:], frameData[48:80])

	frame.Thermo.TempCelsius = math.Float32frombits(binary.LittleEndian.Uint32(frameData[80:84]))
	frame.Thermo.VoltageV = math.Float32frombits(binary.LittleEndian.Uint32(frameData[84:88]))
	frame.Thermo.FreqMHz = math.Float32frombits(binary.LittleEndian.Uint32(frameData[88:92]))
	frame.Thermo.FanRPM = math.Float32frombits(binary.LittleEndian.Uint32(frameData[92:96]))

	if modIndex, ok := entry.Modalities[nrv.ModalityProof]; ok {
		start := modIndex.Offset
		end := start + modIndex.Length
		if end <= len(frameData) {
			frame.Proof = frameData[start:end]
		}
	}

	return frame, nil
}

func binaryDecodeRegistry(data []byte, registry *nrv.Registry) error {
	return nil
}
