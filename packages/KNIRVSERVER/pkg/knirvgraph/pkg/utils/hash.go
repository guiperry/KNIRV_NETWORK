package utils

import (
    "crypto/sha256"
    "encoding/hex"
)

func Hash(data []byte) []byte {
    hash := sha256.Sum256(data)
    return hash[:]
}

func HashHex(data []byte) string {
    return hex.EncodeToString(Hash(data))
}

func DoubleHash(data []byte) []byte {
    first := Hash(data)
    return Hash(first)
}

func DoubleHashHex(data []byte) string {
    return hex.EncodeToString(DoubleHash(data))
}

func MerkleRoot(hashes [][]byte) []byte {
    if len(hashes) == 0 {
        return nil
    }
    
    if len(hashes) == 1 {
        return hashes[0]
    }
    
    var nextLevel [][]byte
    for i := 0; i < len(hashes); i += 2 {
        if i+1 < len(hashes) {
            combined := append(hashes[i], hashes[i+1]...)
            nextLevel = append(nextLevel, Hash(combined))
        } else {
            nextLevel = append(nextLevel, hashes[i])
        }
    }
    
    return MerkleRoot(nextLevel)
}