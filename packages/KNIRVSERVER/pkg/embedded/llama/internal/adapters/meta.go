// Package adapters is the llama wrapper's GGUF LoRA adapter registry and
// lifecycle: which adapters exist, which base model each was bound to, and how a
// change reaches the running llama-server.
//
// llama-server cannot load an adapter file after it has started (verified against
// the provisioned build: POST /lora-adapters only changes the scale of adapters
// loaded at startup). So adding or removing an adapter restarts llama-server with
// the new set, after draining in-flight requests. See Manager.
package adapters

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Info is the part of a GGUF file's metadata the wrapper checks.
type Info struct {
	Type         string // general.type ("adapter" for a LoRA)
	AdapterType  string // adapter.type ("lora")
	Architecture string // general.architecture
	Alpha        float64
	BaseSHA256   string // knirv.base_sha256 (written by ULoRA)
	BundleHash   string // knirv.bundle_content_hash
	Skill        string // knirv.skill_name
}

// ReadInfo reads GGUF metadata only (no tensors). It is deliberately small and
// independent of ULoRA's reader: the two modules ship separately.
func ReadInfo(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<20)
	le := binary.LittleEndian
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil || string(hdr[:]) != "GGUF" {
		return Info{}, fmt.Errorf("%s is not a GGUF file", path)
	}
	var version uint32
	var nTensors, nKV uint64
	for _, v := range []any{&version, &nTensors, &nKV} {
		if err := binary.Read(r, le, v); err != nil {
			return Info{}, err
		}
	}
	if version < 2 || version > 3 || nKV > 1<<20 {
		return Info{}, fmt.Errorf("%s: unsupported GGUF header (version %d, %d keys)", path, version, nKV)
	}
	kv := map[string]any{}
	for i := uint64(0); i < nKV; i++ {
		k, err := str(r)
		if err != nil {
			return Info{}, err
		}
		var vt uint32
		if err := binary.Read(r, le, &vt); err != nil {
			return Info{}, err
		}
		v, err := value(r, vt)
		if err != nil {
			return Info{}, fmt.Errorf("%s: value of %q: %w", path, k, err)
		}
		kv[k] = v
	}
	s := func(k string) string { v, _ := kv[k].(string); return v }
	info := Info{
		Type: s("general.type"), AdapterType: s("adapter.type"), Architecture: s("general.architecture"),
		BaseSHA256: s("knirv.base_sha256"), BundleHash: s("knirv.bundle_content_hash"), Skill: s("knirv.skill_name"),
	}
	switch a := kv["adapter.lora.alpha"].(type) {
	case float32:
		info.Alpha = float64(a)
	case float64:
		info.Alpha = a
	}
	return info, nil
}

func str(r io.Reader) (string, error) {
	var n uint64
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return "", err
	}
	if n > 1<<24 {
		return "", fmt.Errorf("string of %d bytes", n)
	}
	b := make([]byte, n)
	_, err := io.ReadFull(r, b)
	return string(b), err
}

func value(r io.Reader, vt uint32) (any, error) {
	le := binary.LittleEndian
	switch vt {
	case 0, 1, 7:
		var v uint8
		return v, binary.Read(r, le, &v)
	case 2, 3:
		var v uint16
		return v, binary.Read(r, le, &v)
	case 4, 5:
		var v uint32
		return v, binary.Read(r, le, &v)
	case 6:
		var v float32
		return v, binary.Read(r, le, &v)
	case 8:
		return str(r)
	case 10, 11:
		var v uint64
		return v, binary.Read(r, le, &v)
	case 12:
		var v float64
		return v, binary.Read(r, le, &v)
	case 9:
		var et uint32
		var n uint64
		if err := binary.Read(r, le, &et); err != nil {
			return nil, err
		}
		if err := binary.Read(r, le, &n); err != nil {
			return nil, err
		}
		for i := uint64(0); i < n; i++ {
			if _, err := value(r, et); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	return nil, fmt.Errorf("unknown metadata type %d", vt)
}
