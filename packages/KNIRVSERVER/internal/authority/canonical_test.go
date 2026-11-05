package authority

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestCanonicalJSONOrdersKeysAndDropsWhitespace(t *testing.T) {
	a := `{ "z": [1, -2, 0], "a": {"y": "x\ty", "b": null}, "m": true }`
	b := `{"m":true,"a":{"b":null,"y":"x\u0009y"},"z":[1,-2,0]}`
	want := `{"a":{"b":null,"y":"x\ty"},"m":true,"z":[1,-2,0]}`
	for _, in := range []string{a, b} {
		got, err := CanonicalJSON([]byte(in))
		if err != nil || string(got) != want {
			t.Fatalf("CanonicalJSON(%s) = %q, %v; want %q", in, got, err, want)
		}
	}
	// Expected hashes computed independently (Python sha256 of the canonical bytes).
	h, err := ActionHash([]byte(a))
	if err != nil || hex.EncodeToString(h[:]) != "f8d46522a212de5d0347b108848400dc84998b5ea9626cf43cd16e34e5d35100" {
		t.Fatalf("ActionHash = %x, %v", h, err)
	}
	h, _ = ActionHash([]byte(`{}`))
	if hex.EncodeToString(h[:]) != "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a" {
		t.Fatalf("hash of {} = %x", h)
	}
}

func TestCanonicalJSONStringEscapes(t *testing.T) {
	// RFC 8785: short escapes for " \ \b \f \n \r \t, lower-case \u00xx for the
	// other controls, everything else (including U+2028, DEL and non-ASCII) literal.
	in := "\"\\u0007\\u001f\\b\\f\\n\\r\\t\\\"\\\\\\/é \u007f\""
	want := "\"\\u0007\\u001f\\b\\f\\n\\r\\t\\\"\\\\/é \u007f\""
	got, err := CanonicalJSON([]byte(in))
	if err != nil || string(got) != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

func TestCanonicalJSONAcceptsBoundaryIntegers(t *testing.T) {
	for _, in := range []string{`0`, `-9007199254740991`, `9007199254740991`, `[]`, `{}`, `"x"`, `null`, `true`} {
		got, err := CanonicalJSON([]byte(in))
		if err != nil || string(got) != in {
			t.Errorf("CanonicalJSON(%s) = %q, %v", in, got, err)
		}
	}
}

func TestCanonicalJSONRefusesWhatItCannotPin(t *testing.T) {
	deep := strings.Repeat("[", 100) + strings.Repeat("]", 100)
	cases := map[string]string{
		"fraction":       `1.5`,
		"exponent":       `1e3`,
		"negative zero":  `-0`,
		"unsafe integer": `9007199254740993`,
		"duplicate key":  `{"a":1,"a":2}`,
		"non-ASCII key":  `{"é":1}`,
		"trailing value": `1 2`,
		"invalid UTF-8":  "\"\xff\"",
		"empty":          ``,
		"truncated":      `{"a":`,
		"too deep":       deep,
		"too large":      `"` + strings.Repeat("a", maxActionBytes) + `"`,
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if got, err := CanonicalJSON([]byte(in)); !errors.Is(err, ErrNotCanonical) {
				t.Fatalf("want ErrNotCanonical, got %q, %v", got, err)
			}
		})
	}
}

func TestCanonicalJSONDoesNotMutateInput(t *testing.T) {
	in := []byte(`{"b":1,"a":2}`)
	keep := bytes.Clone(in)
	if _, err := CanonicalJSON(in); err != nil || !bytes.Equal(in, keep) {
		t.Fatal("input changed")
	}
}
