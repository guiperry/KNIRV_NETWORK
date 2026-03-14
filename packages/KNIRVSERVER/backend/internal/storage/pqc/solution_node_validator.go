// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package pqc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cloudflare/circl/sign"
)

type SolutionNodeValidator struct {
	mu                    sync.RWMutex
	keyPair               *PQCKeyPair
	signedNodes           map[string]*SignedNodeRecord
	requirePQC签名          bool
	minVerificationLevel  VerificationLevel
	attestationCache      map[string]*AttestationResult
	verificationCallbacks []VerificationCallback
}

type SignedNodeRecord struct {
	NodeID         string             `json:"node_id"`
	PublicKey      string             `json:"public_key"`
	Signature      string             `json:"signature"`
	SignedAt       time.Time          `json:"signed_at"`
	ExpiresAt      time.Time          `json:"expires_at"`
	Verified       bool               `json:"verified"`
	TEEAttestation *AttestationResult `json:"tee_attestation,omitempty"`
}

type VerificationLevel string

const (
	VerificationLevelNone     VerificationLevel = "none"
	VerificationLevelBasic    VerificationLevel = "basic"
	VerificationLevelStandard VerificationLevel = "standard"
	VerificationLevelStrict   VerificationLevel = "strict"
)

type VerificationCallback func(nodeID string, result *VerificationResult) error

type VerificationResult struct {
	NodeID         string            `json:"node_id"`
	Valid          bool              `json:"valid"`
	SignatureValid bool              `json:"signature_valid"`
	TEEValid       bool              `json:"tee_valid"`
	Errors         []string          `json:"errors"`
	Warnings       []string          `json:"warnings"`
	VerifiedAt     time.Time         `json:"verified_at"`
	Level          VerificationLevel `json:"level"`
}

type AttestationResult struct {
	NodeID     string    `json:"node_id"`
	TEEType    string    `json:"tee_type"`
	Quote      []byte    `json:"quote"`
	ReportData []byte    `json:"report_data"`
	MRENCLAVE  string    `json:"mrenclave"`
	MRSIGNER   string    `json:"mrsigner"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Valid      bool      `json:"valid"`
}

type NodeSignatureRequest struct {
	NodeID      string             `json:"node_id"`
	PublicKey   string             `json:"public_key"`
	Attestation *AttestationResult `json:"attestation,omitempty"`
}

type NodeSignatureResponse struct {
	NodeID    string `json:"node_id"`
	Signature string `json:"signature"`
	SignedAt  string `json:"signed_at"`
	ExpiresAt string `json:"expires_at"`
	Valid     bool   `json:"valid"`
}

func NewSolutionNodeValidator(requirePQC bool) (*SolutionNodeValidator, error) {
	keyPair, err := GeneratePQCKeyPair("solution-node-validator", "signature")
	if err != nil {
		return nil, fmt.Errorf("generate validator key pair: %w", err)
	}

	return &SolutionNodeValidator{
		keyPair:              keyPair,
		signedNodes:          make(map[string]*SignedNodeRecord),
		requirePQC签名:         requirePQC,
		minVerificationLevel: VerificationLevelStandard,
		attestationCache:     make(map[string]*AttestationResult),
	}, nil
}

func (snv *SolutionNodeValidator) SignNode(req *NodeSignatureRequest) (*NodeSignatureResponse, error) {
	snv.mu.Lock()
	defer snv.mu.Unlock()

	if snv.keyPair == nil {
		return nil, fmt.Errorf("validator key not initialized")
	}

	payload := fmt.Sprintf("%s:%s:%d", req.NodeID, req.PublicKey, time.Now().UnixNano())
	payloadBytes := []byte(payload)

	signature, err := snv.keyPair.Sign(payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("sign node: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	record := &SignedNodeRecord{
		NodeID:         req.NodeID,
		PublicKey:      req.PublicKey,
		Signature:      base64.StdEncoding.EncodeToString(signature),
		SignedAt:       time.Now(),
		ExpiresAt:      expiresAt,
		Verified:       false,
		TEEAttestation: req.Attestation,
	}

	snv.signedNodes[req.NodeID] = record

	log.Printf("SolutionNodeValidator: Signed node %s", req.NodeID)

	return &NodeSignatureResponse{
		NodeID:    req.NodeID,
		Signature: record.Signature,
		SignedAt:  record.SignedAt.Format(time.RFC3339),
		ExpiresAt: expiresAt.Format(time.RFC3339),
		Valid:     true,
	}, nil
}

func (snv *SolutionNodeValidator) VerifyNode(nodeID string) (*VerificationResult, error) {
	snv.mu.RLock()
	record, exists := snv.signedNodes[nodeID]
	snv.mu.RUnlock()

	result := &VerificationResult{
		NodeID:     nodeID,
		VerifiedAt: time.Now(),
		Level:      snv.minVerificationLevel,
	}

	if !exists {
		result.Errors = append(result.Errors, "node not found")
		return result, nil
	}

	if time.Now().After(record.ExpiresAt) {
		result.Errors = append(result.Errors, "signature expired")
		return result, nil
	}

	signature, err := base64.StdEncoding.DecodeString(record.Signature)
	if err != nil {
		result.Errors = append(result.Errors, "invalid signature encoding")
		return result, nil
	}

	payload := fmt.Sprintf("%s:%s:%d", record.NodeID, record.PublicKey, record.SignedAt.UnixNano())
	payloadBytes := []byte(payload)

	if snv.keyPair.Verify(payloadBytes, signature) {
		result.SignatureValid = true
		result.Valid = true
	} else {
		result.Errors = append(result.Errors, "signature verification failed")
	}

	if snv.minVerificationLevel == VerificationLevelStrict || snv.minVerificationLevel == VerificationLevelStandard {
		if record.TEEAttestation != nil {
			if record.TEEAttestation.Valid {
				result.TEEValid = true
			} else {
				result.Errors = append(result.Errors, "TEE attestation invalid")
			}
		} else if snv.minVerificationLevel == VerificationLevelStrict {
			result.Errors = append(result.Errors, "TEE attestation required for strict verification")
			result.Valid = false
		}
	}

	snv.mu.Lock()
	record.Verified = result.Valid
	snv.mu.Unlock()

	for _, callback := range snv.verificationCallbacks {
		if err := callback(nodeID, result); err != nil {
			log.Printf("SolutionNodeValidator: Verification callback error: %v", err)
		}
	}

	return result, nil
}

func (snv *SolutionNodeValidator) EnforcePQCSigning() bool {
	snv.mu.RLock()
	defer snv.mu.RUnlock()
	return snv.requirePQC签名
}

func (snv *SolutionNodeValidator) SetRequirePQC(require bool) {
	snv.mu.Lock()
	defer snv.mu.Unlock()
	snv.requirePQC签名 = require
	log.Printf("SolutionNodeValidator: PQC signing requirement set to %v", require)
}

func (snv *SolutionNodeValidator) SetVerificationLevel(level VerificationLevel) {
	snv.mu.Lock()
	defer snv.mu.Unlock()
	snv.minVerificationLevel = level
	log.Printf("SolutionNodeValidator: Verification level set to %s", level)
}

func (snv *SolutionNodeValidator) GetVerificationLevel() VerificationLevel {
	snv.mu.RLock()
	defer snv.mu.RUnlock()
	return snv.minVerificationLevel
}

func (snv *SolutionNodeValidator) RegisterVerificationCallback(callback VerificationCallback) {
	snv.mu.Lock()
	defer snv.mu.Unlock()
	snv.verificationCallbacks = append(snv.verificationCallbacks, callback)
}

func (snv *SolutionNodeValidator) ValidateAndEnforce(nodeID string) error {
	snv.mu.RLock()
	requirePQC := snv.requirePQC签名
	snv.mu.RUnlock()

	if !requirePQC {
		return nil
	}

	snv.mu.RLock()
	_, exists := snv.signedNodes[nodeID]
	snv.mu.RUnlock()

	if !exists {
		return fmt.Errorf("node %s requires PQC signature but is not signed", nodeID)
	}

	result, err := snv.VerifyNode(nodeID)
	if err != nil {
		return fmt.Errorf("verification error: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("node %s failed verification: %v", nodeID, result.Errors)
	}

	log.Printf("SolutionNodeValidator: Node %s passed PQC enforcement", nodeID)
	return nil
}

func (snv *SolutionNodeValidator) RevokeNodeSignature(nodeID string) error {
	snv.mu.Lock()
	defer snv.mu.Unlock()

	_, exists := snv.signedNodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	delete(snv.signedNodes, nodeID)
	log.Printf("SolutionNodeValidator: Revoked signature for node %s", nodeID)
	return nil
}

func (snv *SolutionNodeValidator) GetSignedNodes() []*SignedNodeRecord {
	snv.mu.RLock()
	defer snv.mu.RUnlock()

	records := make([]*SignedNodeRecord, 0, len(snv.signedNodes))
	for _, record := range snv.signedNodes {
		records = append(records, record)
	}
	return records
}

func (snv *SolutionNodeValidator) GetValidatorPublicKey() string {
	snv.mu.RLock()
	defer snv.mu.RUnlock()

	if snv.keyPair == nil {
		return ""
	}

	pubKeyBytes, err := snv.keyPair.DilithiumPublicKey.MarshalBinary()
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(pubKeyBytes)
}

func (snv *SolutionNodeValidator) IsNodeSigned(nodeID string) bool {
	snv.mu.RLock()
	defer snv.mu.RUnlock()

	record, exists := snv.signedNodes[nodeID]
	if !exists {
		return false
	}

	return time.Now().Before(record.ExpiresAt)
}

type TEESecurityValidator struct {
	mu           sync.RWMutex
	attestations map[string]*AttestationResult
	validators   []TEEValidator
}

type TEEValidator interface {
	ValidateAttestation(attestation *AttestationResult) bool
}

func NewTEESecurityValidator() *TEESecurityValidator {
	return &TEESecurityValidator{
		attestations: make(map[string]*AttestationResult),
	}
}

func (tsv *TEESecurityValidator) ValidateAndStoreAttestation(ctx context.Context, nodeID string, attestation *AttestationResult) error {
	if attestation == nil {
		return fmt.Errorf("nil attestation")
	}

	tsv.mu.Lock()
	defer tsv.mu.Unlock()

	attestation.NodeID = nodeID
	attestation.IssuedAt = time.Now()
	attestation.ExpiresAt = time.Now().Add(24 * time.Hour)

	for _, validator := range tsv.validators {
		if !validator.ValidateAttestation(attestation) {
			attestation.Valid = false
			return fmt.Errorf("TEE attestation validation failed")
		}
	}

	attestation.Valid = true
	tsv.attestations[nodeID] = attestation

	log.Printf("TEESecurityValidator: Stored attestation for node %s", nodeID)
	return nil
}

func (tsv *TEESecurityValidator) GetAttestation(nodeID string) (*AttestationResult, bool) {
	tsv.mu.RLock()
	defer tsv.mu.RUnlock()

	att, exists := tsv.attestations[nodeID]
	return att, exists
}

func (tsv *TEESecurityValidator) IsAttestationValid(nodeID string) bool {
	tsv.mu.RLock()
	defer tsv.mu.RUnlock()

	att, exists := tsv.attestations[nodeID]
	if !exists {
		return false
	}

	if !att.Valid {
		return false
	}

	return time.Now().Before(att.ExpiresAt)
}

func (tsv *TEESecurityValidator) RegisterValidator(validator TEEValidator) {
	tsv.mu.Lock()
	defer tsv.mu.Unlock()
	tsv.validators = append(tsv.validators, validator)
}

type BasicTEEValidator struct{}

func (btv *BasicTEEValidator) ValidateAttestation(attestation *AttestationResult) bool {
	if attestation == nil {
		return false
	}

	if len(attestation.Quote) == 0 {
		return false
	}

	if len(attestation.ReportData) == 0 {
		return false
	}

	return true
}

func GenerateSecureRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func CreateNodeSignaturePayload(nodeID, publicKey string, timestamp int64) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d", nodeID, publicKey, timestamp))
}

func VerifyDilithiumSignature(publicKey sign.PublicKey, message, signature []byte) bool {
	return DilithiumVerify(publicKey, message, signature)
}
