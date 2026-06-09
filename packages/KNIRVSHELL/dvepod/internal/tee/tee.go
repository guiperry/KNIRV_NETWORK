package tee

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"
)

type Attestation struct {
	Payload   map[string]interface{} `json:"payload"`
	Signature string                 `json:"signature"`
}

type Context struct {
	NodeID      string    `json:"node_id"`
	TEEType     string    `json:"tee_type"`
	WASMHash    string    `json:"wasm_hash"`
	PublicKey   []byte    `json:"public_key"`
	Measurement string    `json:"measurement"`
	CreatedAt   time.Time `json:"created_at"`
	privateKey  *ecdsa.PrivateKey
}

func detectTEEType() string {
	if _, err := os.Stat("/dev/sev-guest"); err == nil {
		return "sev-snp"
	}
	if _, err := os.Stat("/dev/sgx_enclave"); err == nil {
		_, err2 := os.Stat("/dev/sgx_provision")
		if err2 == nil {
			return "sgx"
		}
	}
	if _, err := os.Stat("/dev/tdx-guest"); err == nil {
		return "tdx"
	}
	return "wasmer"
}

func deriveMeasurement(hash []byte) string {
	h := sha256.Sum256(append(hash, []byte("dvepod-measurement-v1")...))
	return hex.EncodeToString(h[:16])
}

func generateOrLoadKey(nodeID string) ([]byte, *ecdsa.PrivateKey) {
	keyDir := "/home/dvepod/.knirv"
	keyPath := keyDir + "/identity.key"

	if _, err := os.Stat(keyPath); err == nil {
		keyBytes, err := os.ReadFile(keyPath)
		if err == nil {
			key, err := x509.ParseECPrivateKey(keyBytes)
			if err == nil {
				pub, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
				if err != nil {
					return nil, key
				}
				return pub, key
			}
		}
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil
	}

	os.MkdirAll(keyDir, 0700)
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err == nil {
		os.WriteFile(keyPath, keyBytes, 0600)
	}

	pub, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, privateKey
	}

	return pub, privateKey
}

func NewContext() (*Context, error) {
	wasmBytes, err := os.ReadFile("/proc/self/exe")
	if err != nil {
		wasmBytes = []byte("dvepod-unknown-binary")
	}
	hash := sha256.Sum256(wasmBytes)

	nodeIDBytes := make([]byte, 16)
	_, err = rand.Read(nodeIDBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate node ID: %w", err)
	}

	pubKey, privateKey := generateOrLoadKey("")
	if privateKey == nil {
		return nil, fmt.Errorf("failed to generate identity keypair")
	}

	ctx := &Context{
		NodeID:      "dvepod-" + hex.EncodeToString(nodeIDBytes[:8]),
		TEEType:     detectTEEType(),
		WASMHash:    hex.EncodeToString(hash[:]),
		Measurement: deriveMeasurement(hash[:]),
		CreatedAt:   time.Now(),
		privateKey:  privateKey,
	}
	ctx.PublicKey = pubKey

	return ctx, nil
}

func sign(data []byte, ctx *Context) (string, error) {
	if ctx.privateKey == nil {
		return "", fmt.Errorf("no private key available for signing")
	}
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, ctx.privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}
	sig := append(r.Bytes(), s.Bytes()...)
	return hex.EncodeToString(sig), nil
}

func (c *Context) Attest() (Attestation, error) {
	payload := map[string]interface{}{
		"node_id":     c.NodeID,
		"tee_type":    c.TEEType,
		"wasm_hash":   c.WASMHash,
		"measurement": c.Measurement,
		"public_key":  hex.EncodeToString(c.PublicKey),
		"timestamp":   time.Now().Unix(),
		"version":     "dvepod/1.0",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return Attestation{}, fmt.Errorf("failed to marshal attestation: %w", err)
	}

	sig, err := sign(data, c)
	if err != nil {
		return Attestation{}, fmt.Errorf("failed to sign attestation: %w", err)
	}

	return Attestation{Payload: payload, Signature: sig}, nil
}

func VerifyPodAttestation(att Attestation) error {
	pubKeyHex, ok := att.Payload["public_key"].(string)
	if !ok {
		return fmt.Errorf("invalid attestation: missing public_key")
	}

	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return fmt.Errorf("invalid attestation: invalid public_key encoding: %w", err)
	}

	pubKeyIface, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("invalid attestation: failed to parse public key: %w", err)
	}

	pubKey, ok := pubKeyIface.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("invalid attestation: not an ECDSA key")
	}

	data, err := json.Marshal(att.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for verification: %w", err)
	}

	hash := sha256.Sum256(data)

	sigBytes, err := hex.DecodeString(att.Signature)
	if err != nil {
		return fmt.Errorf("invalid attestation: invalid signature encoding: %w", err)
	}

	r := new(big.Int).SetBytes(sigBytes[:len(sigBytes)/2])
	s := new(big.Int).SetBytes(sigBytes[len(sigBytes)/2:])

	valid := ecdsa.Verify(pubKey, hash[:], r, s)
	if !valid {
		return fmt.Errorf("attestation signature verification failed")
	}

	return nil
}
