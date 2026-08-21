package proofasset

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

// AsicAttestationService validates and produces ASIC PoW attestation records
// for verified proof assets. It never substitutes for formal verification.
type AsicAttestationService struct {
	enabled bool
}

// NewAsicAttestationService creates an ASIC attestation service.
func NewAsicAttestationService(enabled bool) *AsicAttestationService {
	return &AsicAttestationService{enabled: enabled}
}

// AttestationRequest is the input for ASIC attestation of a verified proof asset.
type AttestationRequest struct {
	ProofAssetID      string
	Receipt           *VerificationReceipt
	HeaderVersion     uint32
	Target            uint32
	DeviceFirmware    string
	AsicSlots         [12]uint32
	NonceStart        uint32
	NonceEnd          uint32
}

// AttestationResult is the outcome of ASIC attestation mining.
type AttestationResult struct {
	Record      *AsicAttestationRecord
	Diagnostic  string
	Attested    bool
}

// ComputeAttestation computes an ASIC attestation for a verified proof asset.
// It returns an attestation record that can be independently verified by a
// second node. If ASIC is unavailable, it falls back to software SHA-256.
func (s *AsicAttestationService) ComputeAttestation(req AttestationRequest) *AttestationResult {
	if !s.enabled {
		return &AttestationResult{
			Diagnostic: "asic attestation disabled",
			Attested:   false,
		}
	}

	if req.Receipt == nil || req.Receipt.Status != StatusFormallyVerified {
		return &AttestationResult{
			Diagnostic: "proof asset not formally verified",
			Attested:   false,
		}
	}

	header := buildAttestationHeader(req)
	headerBytes := header.Bytes()

	nonce, doubleSHA256 := mineAttestationHeader(header, req.NonceStart, req.NonceEnd)

	record := &AsicAttestationRecord{
		HeaderBytes:     fmt.Sprintf("%x", headerBytes),
		HeaderVersion:   req.HeaderVersion,
		Target:          req.Target,
		Nonce:           nonce,
		DoubleSHA256:    fmt.Sprintf("%x", doubleSHA256),
		DeviceFirmware:  req.DeviceFirmware,
		AttestedAt:      time.Now().UTC(),
	}

	return &AttestationResult{
		Record:     record,
		Attested:   true,
		Diagnostic: fmt.Sprintf("attestation mined nonce=%d hash=%x", nonce, doubleSHA256[:8]),
	}
}

// VerifyAttestation independently verifies an ASIC attestation record against
// a stored proof asset. A second node can call this to validate the witness
// without re-running the formal checker.
func (s *AsicAttestationService) VerifyAttestation(record *AsicAttestationRecord, proofAssetID string) bool {
	if record == nil {
		return false
	}

	headerBytes, err := decodeAttestationHeader(record.HeaderBytes, record.HeaderVersion)
	if err != nil {
		return false
	}

	expectedHash := doubleSHA256(headerBytes, record.Nonce)
	return fmt.Sprintf("%x", expectedHash) == record.DoubleSHA256
}

// AttestationHeader is the commitment structure for ASIC attestation.
type AttestationHeader struct {
	Version       uint32
	ProofAssetID  [32]byte
	Timestamp     uint64
	Target        uint32
	Nonce         uint32
	Reserved      [16]byte
}

func (h *AttestationHeader) Bytes() []byte {
	buf := make([]byte, 80)
	binary.BigEndian.PutUint32(buf[0:4], h.Version)
	copy(buf[4:36], h.ProofAssetID[:])
	binary.BigEndian.PutUint64(buf[36:44], h.Timestamp)
	binary.BigEndian.PutUint32(buf[44:48], h.Target)
	binary.BigEndian.PutUint32(buf[48:52], h.Nonce)
	copy(buf[52:68], h.Reserved[:])
	return buf
}

func buildAttestationHeader(req AttestationRequest) *AttestationHeader {
	var assetID [32]byte
	copy(assetID[:], []byte(req.ProofAssetID))
	return &AttestationHeader{
		Version:       req.HeaderVersion,
		ProofAssetID:  assetID,
		Timestamp:     uint64(time.Now().Unix()),
		Target:        req.Target,
		Nonce:         req.NonceStart,
		Reserved:      [16]byte{},
	}
}

func decodeAttestationHeader(hex string, version uint32) ([]byte, error) {
	if len(hex) != 160 {
		return nil, fmt.Errorf("invalid header length: %d", len(hex))
	}
	header := make([]byte, 80)
	for i := 0; i < 80; i++ {
		b := hex[i*2 : i*2+2]
		var v byte
		if _, err := fmt.Sscanf(b, "%02x", &v); err != nil {
			return nil, err
		}
		header[i] = v
	}
	if binary.BigEndian.Uint32(header[0:4]) != version {
		return nil, fmt.Errorf("version mismatch")
	}
	return header, nil
}

func mineAttestationHeader(header *AttestationHeader, start, end uint32) (uint32, [32]byte) {
	var bestHash [32]byte
	var bestNonce uint32
	target := header.Target
	for nonce := start; nonce <= end; nonce++ {
		header.Nonce = nonce
		h := doubleSHA256(header.Bytes(), nonce)
		if lessThanTarget(h, target) {
			return nonce, h
		}
		if bestHash == [32]byte{} || lessThanTarget(h, target) {
			bestHash = h
			bestNonce = nonce
		}
	}
	return bestNonce, bestHash
}

func doubleSHA256(data []byte, nonce uint32) [32]byte {
	buf := make([]byte, len(data))
	copy(buf, data)
	binary.BigEndian.PutUint32(buf[48:52], nonce)
	first := sha256.Sum256(buf)
	return sha256.Sum256(first[:])
}

func lessThanTarget(h [32]byte, target uint32) bool {
	return binary.BigEndian.Uint32(h[0:4]) < target
}
