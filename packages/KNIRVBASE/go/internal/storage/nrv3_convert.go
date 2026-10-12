package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
)

// MaxBracketsPerFile caps one converted .nrv. KNIRVBASE keeps every bracket's
// index entry in a fixed 5 MiB registry (indented JSON, ~200 B per bracket), so
// a models batch with hundreds of thousands of brackets must be sharded.
const MaxBracketsPerFile = 16000

// ConvertNRV3File converts a models NRV3 file into one or more KNIRVBASE .nrv
// files. See nrv.ConvertFrame for the encoding. The first output is written to
// outPath; further shards go to "<outPath minus .nrv>.1.nrv", ".2.nrv", ... The
// written paths are returned in order.
//
// With a key pair every frame is signed (ML-DSA-65) into the registry's PQC
// manifest, so a converted dataset is verifiable (E18); with nil it is unsigned.
func ConvertNRV3File(inPath, outPath string, kp *pqc.PQCKeyPair) ([]string, error) {
	src, err := nrv.OpenNRV3(inPath)
	if err != nil {
		return nil, err
	}

	shardPath := func(n int) string {
		if n == 0 {
			return outPath
		}
		return fmt.Sprintf("%s.%d.nrv", strings.TrimSuffix(outPath, ".nrv"), n)
	}

	var (
		written []string
		w       *NRVWriter
		inFile  int
	)
	closeWriter := func() error {
		if w == nil {
			return nil
		}
		err := w.Close()
		w = nil
		return err
	}
	fail := func(err error) ([]string, error) {
		_ = closeWriter()
		for _, p := range written { // leave no half-converted output behind
			_ = os.Remove(p)
			_ = os.Remove(p + ".wal")
		}
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, err
	}
	for _, frame := range src.Registry.Frames {
		raw := src.RawBrackets(frame)
		if raw == nil && frame.Count > 0 {
			return fail(fmt.Errorf("nrv3: frame %s points outside the bracket section (offset %d, count %d)", frame.ID, frame.Offset, frame.Count))
		}
		converted, err := nrv.ConvertFrame(frame, raw)
		if err != nil {
			return fail(err)
		}
		for _, cf := range converted {
			n := len(cf.Index)
			if n == 0 {
				continue
			}
			if w == nil || inFile+n > MaxBracketsPerFile {
				if err := closeWriter(); err != nil {
					return fail(err)
				}
				p := shardPath(len(written))
				if w, err = NewNRVWriter(p, kp); err != nil {
					return fail(err)
				}
				written = append(written, p)
				inFile = 0
			}
			if err := w.AppendFrame(cf.ID, cf.Brackets, cf.Index, cf.Thermo, cf.Ling); err != nil {
				return fail(err)
			}
			inFile += n
		}
	}
	if err := closeWriter(); err != nil {
		return fail(err)
	}
	return written, nil
}
