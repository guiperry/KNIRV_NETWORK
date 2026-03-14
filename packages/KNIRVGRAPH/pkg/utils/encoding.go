package utils

import (
    "encoding/hex"
    "encoding/json"
    "fmt"
)

func EncodeHex(data []byte) string {
    return "0x" + hex.EncodeToString(data)
}

func DecodeHex(hexStr string) ([]byte, error) {
    if len(hexStr) >= 2 && hexStr[:2] == "0x" {
        hexStr = hexStr[2:]
    }
    return hex.DecodeString(hexStr)
}

func ToJSON(v interface{}) ([]byte, error) {
    return json.MarshalIndent(v, "", "  ")
}

func FromJSON(data []byte, v interface{}) error {
    return json.Unmarshal(data, v)
}

func PrettyPrint(v interface{}) {
    data, err := ToJSON(v)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Println(string(data))
}