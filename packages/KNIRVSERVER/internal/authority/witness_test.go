package authority

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

func TestSoftwareWitnesserMatchesSharedWitnessVector(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "KNIRVSDK", "testvectors", "authority", "witness.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		PrevHash, Merkle, Header, Hash string
		NTime, NBits, Nonce            uint32
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	v.PrevHash, v.Merkle, v.Header, v.Hash = raw["prev_hash"].(string), raw["merkle"].(string), raw["header"].(string), raw["hash"].(string)
	v.NTime, v.NBits, v.Nonce = uint32(raw["ntime"].(float64)), uint32(raw["nbits"].(float64)), uint32(raw["nonce"].(float64))

	var prev, merkle [32]byte
	p, _ := hex.DecodeString(v.PrevHash)
	m, _ := hex.DecodeString(v.Merkle)
	copy(prev[:], p)
	copy(merkle[:], m)
	header := sdk.NewHeader(prev, merkle, v.NTime, v.NBits, 0, sdk.HeaderVersion, sdk.TierT1)

	nonce, hash, err := SoftwareWitnesser{}.Mine(context.Background(), [80]byte(header), v.NBits)
	if err != nil {
		t.Fatal(err)
	}
	if nonce != v.Nonce || hex.EncodeToString(hash[:]) != v.Hash {
		t.Fatalf("mined nonce %d hash %x; the shared vector has nonce %d hash %s", nonce, hash, v.Nonce, v.Hash)
	}
	if got := hex.EncodeToString(sdk.NewHeader(prev, merkle, v.NTime, v.NBits, nonce, sdk.HeaderVersion, sdk.TierT1).Bytes()); got != v.Header {
		t.Fatalf("header %s differs from the vector %s", got, v.Header)
	}
}

func TestSoftwareWitnesserMeetsEveryEnabledTier(t *testing.T) {
	for _, tier := range []sdk.Tier{sdk.TierT1, sdk.TierT2} {
		bits := sdk.TierTargetBits(tier)
		var prev, merkle [32]byte
		prev[0], merkle[0] = byte(tier), 7
		header := sdk.NewHeader(prev, merkle, 1_750_000_000, bits, 0, sdk.HeaderVersion, tier)
		nonce, hash, err := SoftwareWitnesser{}.Mine(context.Background(), [80]byte(header), bits)
		if err != nil {
			t.Fatalf("%s: %v", tier, err)
		}
		mined := sdk.NewHeader(prev, merkle, 1_750_000_000, bits, nonce, sdk.HeaderVersion, tier)
		if mined.Witness() != hash || !sdk.MeetsTarget(hash, bits) {
			t.Fatalf("%s: witness does not meet its target", tier)
		}
	}
}

func TestSoftwareWitnesserRefusesBadInput(t *testing.T) {
	bits := sdk.TierTargetBits(sdk.TierT1)
	header := sdk.NewHeader([32]byte{}, [32]byte{}, 1, bits, 0, sdk.HeaderVersion, sdk.TierT1)

	if _, _, err := (SoftwareWitnesser{}).Mine(context.Background(), [80]byte(header), bits+1); err == nil {
		t.Error("a target different from the header's nbits was accepted")
	}
	t0 := header
	t0[3] = 0
	if _, _, err := (SoftwareWitnesser{}).Mine(context.Background(), [80]byte(t0), bits); err == nil {
		t.Error("a T0 header was mined")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := (SoftwareWitnesser{}).Mine(ctx, [80]byte(header), bits); err == nil {
		t.Error("a cancelled context was ignored")
	}
	if (SoftwareWitnesser{}).Kind() != sdk.WitnessSoftware {
		t.Error("wrong witness kind")
	}
}
