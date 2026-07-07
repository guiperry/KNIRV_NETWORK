package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

func TestRecoverAddressFromCosmJSSignature(t *testing.T) {
	message := []byte("transfer:0x7e5f4552091a69125d5dfcb7b8c2659029395bdf:0x0000000000000000000000000000000000000002:5:0")
	signature, err := hex.DecodeString("61f3726f9c26522c833cba2e13438c4fbcf89b23b5d2e95df78169dfd7172e7254cf50ac93dcf5ab3d070e5a703ae5ad9cb48fe9d422a8bc06e13427fefb1ad901")
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	want, err := types.AddressFromString("0x7e5f4552091a69125d5dfcb7b8c2659029395bdf")
	if err != nil {
		t.Fatalf("parse address: %v", err)
	}

	got, err := RecoverAddress(message, signature)
	if err != nil {
		t.Fatalf("recover address: %v", err)
	}
	if got != want {
		t.Fatalf("recovered address = %s, want %s", got.String(), want.String())
	}
}
