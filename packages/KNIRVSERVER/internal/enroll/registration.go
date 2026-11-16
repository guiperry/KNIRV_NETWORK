package enroll

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// EnrollmentClaims is the operator claim carried inside a DEC-2 boot-node
// enrollment challenge. The registration ID binds the challenge to the approved
// onboarding record; validator_id is optional and, when present, must equal the
// address derived from the challenge public key.
type EnrollmentClaims struct {
	RegistrationID string `json:"registration_id"`
	ValidatorID    string `json:"validator_id,omitempty"`
}

// MarshalClaims returns the canonical JSON claim payload for a challenge.
func MarshalClaims(claims EnrollmentClaims) []byte {
	encoded, _ := json.Marshal(claims)
	return encoded
}

// ParseClaims decodes the claim payload embedded in a boot-node challenge.
func ParseClaims(payload []byte) (EnrollmentClaims, error) {
	var claims EnrollmentClaims
	if len(payload) == 0 {
		return claims, fmt.Errorf("enrollment challenge has no claim payload")
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return EnrollmentClaims{}, fmt.Errorf("parse enrollment claims: %w", err)
	}
	claims.RegistrationID = strings.TrimSpace(claims.RegistrationID)
	claims.ValidatorID = strings.TrimSpace(claims.ValidatorID)
	if claims.RegistrationID == "" {
		return EnrollmentClaims{}, fmt.Errorf("enrollment claim is missing registration_id")
	}
	return claims, nil
}

// ParseEnrollmentClaims decodes the DEC-2 challenge payload. The Rust core wraps
// caller-supplied claim bytes in a BootnodeEnrollClaim {operator, claims} object
// (with claims serialized as a byte array), so unwrap that form before decoding
// the registration claim. A bare claim object is also accepted for forwards
// compatibility and unit tests.
func ParseEnrollmentClaims(payload []byte) (EnrollmentClaims, error) {
	var wrapper struct {
		Operator string          `json:"operator"`
		Claims   json.RawMessage `json:"claims"`
	}
	if err := json.Unmarshal(payload, &wrapper); err == nil && len(wrapper.Claims) > 0 && string(wrapper.Claims) != "null" {
		claimBytes := []byte(wrapper.Claims)
		var raw []byte
		if err := json.Unmarshal(wrapper.Claims, &raw); err == nil {
			claimBytes = raw
		}
		return ParseClaims(claimBytes)
	}
	return ParseClaims(payload)
}

// Registration is an approved onboarding registration bound to a validator
// identity. The root/approver writes this record only after reviewing the
// onboarding approval; a registration ID alone is never proof of key ownership.
type Registration struct {
	RegistrationID string `json:"registration_id"`
	ValidatorID    string `json:"validator_id"`
	ChainID        string `json:"chain_id,omitempty"`
}

// RegistrationStore resolves an approved onboarding registration ID to the
// validator identity it is allowed to enroll.
type RegistrationStore interface {
	Lookup(ctx context.Context, registrationID string) (Registration, bool, error)
}

// FileRegistrationStore reads approved registrations from a JSON file. It
// reloads on every lookup so the root/approver can add an approved registration
// without restarting the node. Both the object form
// `{"registrations":[...]}` and a bare array are accepted.
type FileRegistrationStore struct {
	path string
	mu   sync.RWMutex
}

// NewFileRegistrationStore returns a store backed by path. A missing file is an
// empty approval set, not an error: every registration is then unknown and
// rejected until the approver writes one.
func NewFileRegistrationStore(path string) *FileRegistrationStore {
	return &FileRegistrationStore{path: path}
}

// Lookup returns the approved registration for registrationID.
func (s *FileRegistrationStore) Lookup(_ context.Context, registrationID string) (Registration, bool, error) {
	registrationID = strings.TrimSpace(registrationID)
	if registrationID == "" {
		return Registration{}, false, fmt.Errorf("registration_id is required")
	}

	registrations, err := s.load()
	if err != nil {
		return Registration{}, false, err
	}
	reg, ok := registrations[registrationID]
	return reg, ok, nil
}

func (s *FileRegistrationStore) load() (map[string]Registration, error) {
	if s.path == "" {
		return nil, fmt.Errorf("approved registrations path is not configured")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Registration{}, nil
		}
		return nil, fmt.Errorf("read approved registrations: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]Registration{}, nil
	}

	var list []Registration
	if err := json.Unmarshal(data, &list); err != nil {
		var wrapper struct {
			Registrations []Registration `json:"registrations"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return nil, fmt.Errorf("decode approved registrations: %w", err)
		}
		list = wrapper.Registrations
	}

	out := make(map[string]Registration, len(list))
	for _, reg := range list {
		reg.RegistrationID = strings.TrimSpace(reg.RegistrationID)
		reg.ValidatorID = strings.TrimSpace(reg.ValidatorID)
		reg.ChainID = strings.TrimSpace(reg.ChainID)
		if reg.RegistrationID == "" || reg.ValidatorID == "" {
			return nil, fmt.Errorf("approved registration entry requires registration_id and validator_id")
		}
		if _, exists := out[reg.RegistrationID]; exists {
			return nil, fmt.Errorf("duplicate approved registration %q", reg.RegistrationID)
		}
		out[reg.RegistrationID] = reg
	}
	return out, nil
}
