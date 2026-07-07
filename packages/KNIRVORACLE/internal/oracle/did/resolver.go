package did

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type Method string

const (
	MethodKNIRV Method = "knirv"
	MethodKey   Method = "key"
)

var (
	ErrDIDNotFound      = fmt.Errorf("did not found")
	ErrDIDDeactivated   = fmt.Errorf("did is deactivated")
	ErrUnsupportedMethod = fmt.Errorf("unsupported did method")
	ErrInvalidEncoding  = fmt.Errorf("invalid did encoding")
)

type DIDStore interface {
	Get(did string) (*DIDDocument, error)
	Put(doc *DIDDocument) error
	Deactivate(did string) error
}

const ed25519Multicodec byte = 0xed

var base58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

var base58Index = func() map[byte]int {
	idx := make(map[byte]int, len(base58Alphabet))
	for i, c := range base58Alphabet {
		idx[c] = i
	}
	return idx
}()

func base58Decode(input string) ([]byte, error) {
	if len(input) == 0 {
		return nil, ErrInvalidEncoding
	}

	radix := big.NewInt(58)

	leadingZeros := 0
	for _, c := range input {
		if c == '1' {
			leadingZeros++
		} else {
			break
		}
	}

	total := big.NewInt(0)
	for _, c := range input {
		val, ok := base58Index[byte(c)]
		if !ok {
			return nil, fmt.Errorf("%w: invalid character %c", ErrInvalidEncoding, c)
		}
		total.Mul(total, radix)
		total.Add(total, big.NewInt(int64(val)))
	}

	decoded := total.Bytes()
	result := make([]byte, leadingZeros+len(decoded))
	copy(result[leadingZeros:], decoded)
	return result, nil
}

type Resolver struct {
	store DIDStore
}

func NewResolver(store DIDStore) *Resolver {
	return &Resolver{store: store}
}

func parseDID(did string) (method Method, id string, err error) {
	if !strings.HasPrefix(did, "did:") {
		return "", "", fmt.Errorf("%w: missing did: prefix", ErrInvalidEncoding)
	}
	rest := strings.TrimPrefix(did, "did:")
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("%w: malformed did", ErrInvalidEncoding)
	}

	switch parts[0] {
	case "knirv":
		return MethodKNIRV, parts[1], nil
	case "key":
		return MethodKey, parts[1], nil
	default:
		return "", "", fmt.Errorf("%w: %s", ErrUnsupportedMethod, parts[0])
	}
}

func (r *Resolver) Resolve(did string) (*DIDDocument, error) {
	method, id, err := parseDID(did)
	if err != nil {
		return nil, err
	}

	switch method {
	case MethodKNIRV:
		return r.resolveKNIRV(id)
	case MethodKey:
		return r.resolveKey(id)
	default:
		return nil, ErrUnsupportedMethod
	}
}

func (r *Resolver) Register(doc *DIDDocument) error {
	if doc.ID == "" {
		return fmt.Errorf("did document must have an id")
	}
	if !strings.HasPrefix(doc.ID, "did:knirv:") {
		return fmt.Errorf("only did:knirv: documents can be registered")
	}

	doc.Created = time.Now().UTC()
	doc.Updated = time.Now().UTC()

	return r.store.Put(doc)
}

func (r *Resolver) Deactivate(did string) error {
	method, id, err := parseDID(did)
	if err != nil {
		return err
	}
	if method != MethodKNIRV {
		return fmt.Errorf("only did:knirv: documents can be deactivated")
	}
	fullDID := "did:knirv:" + id
	return r.store.Deactivate(fullDID)
}

func (r *Resolver) resolveKNIRV(id string) (*DIDDocument, error) {
	did := "did:knirv:" + id
	doc, err := r.store.Get(did)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, ErrDIDNotFound
	}
	return doc, nil
}

func (r *Resolver) resolveKey(encoded string) (*DIDDocument, error) {
	if len(encoded) < 2 {
		return nil, ErrInvalidEncoding
	}

	prefix := encoded[0]
	data := encoded[1:]

	var raw []byte
	switch prefix {
	case 'z':
		var err error
		raw, err = base58Decode(data)
		if err != nil {
			return nil, fmt.Errorf("base58 decode: %w", err)
		}
	case 'u':
		var err error
		raw, err = base64.URLEncoding.DecodeString(data)
		if err != nil {
			return nil, fmt.Errorf("base64url decode: %w", err)
		}
	case 'f':
		var err error
		raw, err = hex.DecodeString(data)
		if err != nil {
			return nil, fmt.Errorf("hex decode: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w: unsupported multibase prefix %c", ErrInvalidEncoding, prefix)
	}

	if len(raw) < 1 {
		return nil, ErrInvalidEncoding
	}

	var pubKey []byte
	if raw[0] == ed25519Multicodec {
		if len(raw) != 33 {
			return nil, fmt.Errorf("%w: expected 33 bytes for ed25519 key (1+32), got %d", ErrInvalidEncoding, len(raw))
		}
		pubKey = raw[1:]
	} else {
		pubKey = raw
	}

	if len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: expected %d byte public key, got %d", ErrInvalidEncoding, ed25519.PublicKeySize, len(pubKey))
	}

	multibaseEncoded := string(prefix) + data

	did := "did:key:" + multibaseEncoded
	vmID := did + "#" + encoded[len(encoded)-8:]

	now := time.Now().UTC()
	return &DIDDocument{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		ID:      did,
		VerificationMethod: []VerificationMethod{
			{
				ID:                 vmID,
				Type:               "Ed25519VerificationKey2020",
				Controller:         did,
				PublicKeyMultibase: multibaseEncoded,
			},
		},
		Authentication:  []string{vmID},
		AssertionMethod: []string{vmID},
		Created:         now,
		Updated:         now,
	}, nil
}
