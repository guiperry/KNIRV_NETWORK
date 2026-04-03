package trainer

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

type HashNetwork struct{}

func SHA256Hash(input interface{}) [32]byte {
	var data []byte
	switch v := input.(type) {
	case string:
		data = []byte(v)
	case []float64:
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		enc.Encode(v)
		data = buf.Bytes()
	case []byte:
		data = v
	default:
		data = []byte{}
	}
	return sha256.Sum256(data)
}
