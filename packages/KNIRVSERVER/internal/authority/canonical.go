package authority

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Canonical JSON for canonical_action_hash.
//
// AUTHORITY_SPEC.md section 2.1 says canonical_action_hash is "the SHA-256 of the
// canonical JSON of the action" but does not pin the canonicalisation. OPEN
// (recorded as D9 in the spec, not decided): until the owner pins one, this file
// implements the most conservative profile, a strict subset of RFC 8785 (JCS)
// that any other language can reproduce byte for byte:
//
//   - the input must be valid UTF-8 and exactly one JSON value;
//   - object keys are sorted bytewise and must be ASCII (so bytewise order equals
//     JCS's UTF-16 order); duplicate keys are refused;
//   - no insignificant whitespace;
//   - strings use the RFC 8785 escapes: \" \\ \b \f \n \r \t, other control
//     characters as lower-case \u00xx, everything else literal UTF-8;
//   - numbers must be integers in canonical decimal form within +-(2^53-1):
//     no fraction, no exponent, no leading zeros or plus sign, no -0. Anything
//     else is refused, because float formatting is where implementations drift.
//     Carry other numbers as strings.
//
// A refused action never produces a record.

// ErrNotCanonical is returned for an action the profile above cannot represent.
var ErrNotCanonical = errors.New("authority: action is not representable as canonical JSON")

const (
	maxActionBytes = 1 << 20
	maxActionDepth = 64
	maxSafeInt     = 1<<53 - 1
)

var integerPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// CanonicalJSON returns the canonical encoding of one JSON value.
func CanonicalJSON(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > maxActionBytes {
		return nil, fmt.Errorf("%w: size %d is outside 1..%d bytes", ErrNotCanonical, len(raw), maxActionBytes)
	}
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("%w: invalid UTF-8", ErrNotCanonical)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out bytes.Buffer
	if err := encodeValue(dec, &out, 0); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: trailing data after the value", ErrNotCanonical)
	}
	return out.Bytes(), nil
}

// ActionHash is SHA-256 of CanonicalJSON(raw).
func ActionHash(raw []byte) ([32]byte, error) {
	canonical, err := CanonicalJSON(raw)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(canonical), nil
}

// encodeValue streams one value from dec into out in canonical form. A streaming
// walk (Token) is used instead of decoding into interface{} so duplicate object
// keys, which a map would silently collapse, can be detected.
func encodeValue(dec *json.Decoder, out *bytes.Buffer, depth int) error {
	if depth > maxActionDepth {
		return fmt.Errorf("%w: nesting deeper than %d", ErrNotCanonical, maxActionDepth)
	}
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotCanonical, err)
	}
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			return encodeObject(dec, out, depth)
		case '[':
			return encodeArray(dec, out, depth)
		}
		return fmt.Errorf("%w: unexpected delimiter %q", ErrNotCanonical, v)
	case string:
		writeString(out, v)
	case bool:
		out.WriteString(strconv.FormatBool(v))
	case nil:
		out.WriteString("null")
	case json.Number:
		return writeInteger(out, v.String())
	default:
		return fmt.Errorf("%w: unexpected token %T", ErrNotCanonical, tok)
	}
	return nil
}

func encodeObject(dec *json.Decoder, out *bytes.Buffer, depth int) error {
	type member struct {
		key   string
		value []byte
	}
	var members []member
	seen := map[string]bool{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrNotCanonical, err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return fmt.Errorf("%w: object key is not a string", ErrNotCanonical)
		}
		for i := 0; i < len(key); i++ {
			if key[i] >= 0x80 {
				return fmt.Errorf("%w: object key %q is not ASCII", ErrNotCanonical, key)
			}
		}
		if seen[key] {
			return fmt.Errorf("%w: duplicate object key %q", ErrNotCanonical, key)
		}
		seen[key] = true
		var value bytes.Buffer
		if err := encodeValue(dec, &value, depth+1); err != nil {
			return err
		}
		members = append(members, member{key: key, value: value.Bytes()})
	}
	if _, err := dec.Token(); err != nil { // closing brace
		return fmt.Errorf("%w: %v", ErrNotCanonical, err)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].key < members[j].key })
	out.WriteByte('{')
	for i, m := range members {
		if i > 0 {
			out.WriteByte(',')
		}
		writeString(out, m.key)
		out.WriteByte(':')
		out.Write(m.value)
	}
	out.WriteByte('}')
	return nil
}

func encodeArray(dec *json.Decoder, out *bytes.Buffer, depth int) error {
	out.WriteByte('[')
	first := true
	for dec.More() {
		if !first {
			out.WriteByte(',')
		}
		first = false
		if err := encodeValue(dec, out, depth+1); err != nil {
			return err
		}
	}
	if _, err := dec.Token(); err != nil { // closing bracket
		return fmt.Errorf("%w: %v", ErrNotCanonical, err)
	}
	out.WriteByte(']')
	return nil
}

func writeInteger(out *bytes.Buffer, literal string) error {
	if !integerPattern.MatchString(literal) || literal == "-0" {
		return fmt.Errorf("%w: number %q is not a canonical integer (carry other numbers as strings)", ErrNotCanonical, literal)
	}
	n, err := strconv.ParseInt(literal, 10, 64)
	if err != nil || n > maxSafeInt || n < -maxSafeInt {
		return fmt.Errorf("%w: integer %q is outside +-(2^53-1)", ErrNotCanonical, literal)
	}
	out.WriteString(literal)
	return nil
}

// writeString writes s with the RFC 8785 escaping rules.
func writeString(out *bytes.Buffer, s string) {
	const hexDigits = "0123456789abcdef"
	out.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			out.WriteString(`\"`)
		case '\\':
			out.WriteString(`\\`)
		case '\b':
			out.WriteString(`\b`)
		case '\f':
			out.WriteString(`\f`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			if r < 0x20 {
				out.WriteString(`\u00`)
				out.WriteByte(hexDigits[r>>4])
				out.WriteByte(hexDigits[r&0xf])
			} else {
				out.WriteString(string(r))
			}
		}
	}
	out.WriteByte('"')
}

// trimmedLower is a small helper shared by the policy and request validators.
func trimmedLower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
