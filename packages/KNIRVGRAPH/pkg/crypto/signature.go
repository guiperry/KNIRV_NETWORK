package crypto

import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "math/big"
)

func SignData(data []byte, privateKey *ecdsa.PrivateKey) (string, error) {
    hash := sha256.Sum256(data)
    r, s, err := ecdsa.Sign(nil, privateKey, hash[:])
    if err != nil {
        return "", fmt.Errorf("failed to sign data: %w", err)
    }
    
    signature := append(r.Bytes(), s.Bytes()...)
    return hex.EncodeToString(signature), nil
}

func VerifySignature(data []byte, signatureHex string, publicKeyHex string) bool {
    signature, err := hex.DecodeString(signatureHex)
    if err != nil || len(signature) != 64 {
        return false
    }
    
    pubKeyBytes, err := hex.DecodeString(publicKeyHex)
    if err != nil || len(pubKeyBytes) != 64 {
        return false
    }
    
    x := new(big.Int).SetBytes(pubKeyBytes[:32])
    y := new(big.Int).SetBytes(pubKeyBytes[32:])
    
    publicKey := &ecdsa.PublicKey{
        Curve: elliptic.P256(),
        X:     x,
        Y:     y,
    }
    
    r := new(big.Int).SetBytes(signature[:32])
    s := new(big.Int).SetBytes(signature[32:])
    
    hash := sha256.Sum256(data)
    return ecdsa.Verify(publicKey, hash[:], r, s)
}

func RecoverAddress(data []byte, signatureHex string) (string, error) {
    // This is a simplified version - in production, use proper key recovery
    hash := sha256.Sum256(data)
    return "0x" + hex.EncodeToString(hash[:20]), nil
}