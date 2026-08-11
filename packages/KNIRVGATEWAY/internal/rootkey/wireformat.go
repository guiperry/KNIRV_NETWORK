package rootkey

import "fmt"

// readVarint and extractBytesField implement just enough of the protobuf
// wire format to pull specific length-delimited (wire type 2) fields out of
// a message by field number, without pulling in a full protobuf runtime or
// the generated .pb.go types (which live in backend_server, a separate
// repo/module). This mirrors the same technique already used in
// packages/KNIRVSERVER/pkg/knirvoracle/rootkey.go's
// validateEncryptedRootKeyEnvelope — that one only checks field presence;
// this one returns the actual bytes.
//
// Protobuf's wire format is forward/backward compatible by field number, so
// this correctly extracts field N from a message even though every other
// field defined in root_key.proto is silently skipped — it does not need to
// know the full schema, just the field numbers it cares about.

func readVarint(data []byte, start int) (uint64, int, error) {
	var value uint64
	var shift uint

	for i := start; i < len(data); i++ {
		b := data[i]
		value |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return value, i + 1, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, fmt.Errorf("varint overflow")
		}
	}

	return 0, 0, fmt.Errorf("truncated varint")
}

// extractBytesField returns the raw bytes of the first length-delimited
// (wire type 2) occurrence of fieldNum in data. ok is false if the field is
// absent.
func extractBytesField(data []byte, fieldNum int) (value []byte, ok bool, err error) {
	for i := 0; i < len(data); {
		key, next, err := readVarint(data, i)
		if err != nil {
			return nil, false, err
		}
		i = next

		fn := int(key >> 3)
		wireType := int(key & 0x7)

		switch wireType {
		case 0: // varint
			_, next, err := readVarint(data, i)
			if err != nil {
				return nil, false, err
			}
			i = next
		case 1: // fixed64
			if i+8 > len(data) {
				return nil, false, fmt.Errorf("truncated fixed64 field")
			}
			i += 8
		case 2: // length-delimited (bytes/string/embedded message)
			length, next, err := readVarint(data, i)
			if err != nil {
				return nil, false, err
			}
			i = next
			end := i + int(length)
			if end > len(data) {
				return nil, false, fmt.Errorf("truncated length-delimited field")
			}
			if fn == fieldNum {
				return data[i:end], true, nil
			}
			i = end
		case 5: // fixed32
			if i+4 > len(data) {
				return nil, false, fmt.Errorf("truncated fixed32 field")
			}
			i += 4
		default:
			return nil, false, fmt.Errorf("unsupported wire type %d", wireType)
		}
	}

	return nil, false, nil
}
