package crypto

import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "math/big"
)

type KeyPair struct {
    PrivateKey *ecdsa.PrivateKey
    PublicKey  *ecdsa.PublicKey
}

func GenerateKeyPair() (*KeyPair, error) {
    privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, fmt.Errorf("failed to generate private key: %w", err)
    }
    
    return &KeyPair{
        PrivateKey: privateKey,
        PublicKey:  &privateKey.PublicKey,
    }, nil
}

func (kp *KeyPair) Sign(data []byte) ([]byte, error) {
    hash := sha256.Sum256(data)
    r, s, err := ecdsa.Sign(rand.Reader, kp.PrivateKey, hash[:])
    if err != nil {
        return nil, fmt.Errorf("failed to sign data: %w", err)
    }
    
    signature := append(r.Bytes(), s.Bytes()...)
    return signature, nil
}

func (kp *KeyPair) Verify(data, signature []byte) bool {
    if len(signature) != 64 {
        return false
    }
    
    r := new(big.Int).SetBytes(signature[:32])
    s := new(big.Int).SetBytes(signature[32:])
    
    hash := sha256.Sum256(data)
    return ecdsa.Verify(kp.PublicKey, hash[:], r, s)
}

func (kp *KeyPair) Address() string {
    pubKeyBytes := append(kp.PublicKey.X.Bytes(), kp.PublicKey.Y.Bytes()...)
    hash := sha256.Sum256(pubKeyBytes)
    return "0x" + hex.EncodeToString(hash[:20])
}

func (kp *KeyPair) PrivateKeyHex() string {
    return hex.EncodeToString(kp.PrivateKey.D.Bytes())
}

func (kp *KeyPair) PublicKeyHex() string {
    pubKeyBytes := append(kp.PublicKey.X.Bytes(), kp.PublicKey.Y.Bytes()...)
    return hex.EncodeToString(pubKeyBytes)
}