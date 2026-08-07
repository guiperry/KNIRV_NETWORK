package crypto

import (
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestPublicKeyToCanonicalAddress(t *testing.T) {
	key, err := ethcrypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	got := PublicKeyToAddress(&key.PublicKey).String()
	if got != "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e" {
		t.Fatalf("address = %s", got)
	}
}
