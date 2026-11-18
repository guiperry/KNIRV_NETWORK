package pqc_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/knirvcorp/knirvbase/internal/crypto/pqc"
)

// genKeyPairJSON returns a freshly generated key pair serialised with private
// keys, as a generic JSON object. Test-only; no key files are read.
func genKeyPairJSON(t *testing.T) (*pqc.PQCKeyPair, map[string]json.RawMessage) {
	t.Helper()
	kp, err := pqc.GeneratePQCKeyPair("json-test", "signature")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair: %v", err)
	}
	raw, err := kp.MarshalWithPrivateKeys()
	if err != nil {
		t.Fatalf("MarshalWithPrivateKeys: %v", err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return kp, obj
}

func renameKey(t *testing.T, obj map[string]json.RawMessage, from, to string) {
	t.Helper()
	v, ok := obj[from]
	if !ok {
		t.Fatalf("fixture missing %q", from)
	}
	delete(obj, from)
	obj[to] = v
}

func encode(t *testing.T, obj map[string]json.RawMessage) []byte {
	t.Helper()
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return b
}

func TestPQCKeyJSON_RoundTripWritesMLDSAOnly(t *testing.T) {
	kp, _ := genKeyPairJSON(t)
	raw, err := kp.MarshalWithPrivateKeys()
	if err != nil {
		t.Fatalf("MarshalWithPrivateKeys: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, `"mldsa_public_key"`) || !strings.Contains(s, `"mldsa_private_key"`) {
		t.Fatalf("writer must emit mldsa_* names")
	}
	if strings.Contains(s, "dilithium_") {
		t.Fatalf("writer must not emit legacy dilithium_* names")
	}

	loaded, err := pqc.LoadPQCKeyPair(raw)
	if err != nil {
		t.Fatalf("LoadPQCKeyPair(round trip): %v", err)
	}
	if !loaded.Verify(signedMsg, mustSign(t, kp)) {
		t.Fatalf("round-tripped key failed to verify a signature from the original")
	}
}

func TestPQCKeyJSON_PublicMarshalWritesMLDSAOnly(t *testing.T) {
	kp, _ := genKeyPairJSON(t)
	raw, err := kp.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(raw), "dilithium_") || !strings.Contains(string(raw), `"mldsa_public_key"`) {
		t.Fatalf("public marshal must emit only mldsa_public_key")
	}
}

func TestPQCKeyJSON_LegacyDilithiumNamesStillLoad(t *testing.T) {
	kp, obj := genKeyPairJSON(t)
	renameKey(t, obj, "mldsa_public_key", "dilithium_public_key")
	renameKey(t, obj, "mldsa_private_key", "dilithium_private_key")

	loaded, err := pqc.LoadPQCKeyPair(encode(t, obj))
	if err != nil {
		t.Fatalf("legacy dilithium_* JSON must still load: %v", err)
	}
	if loaded.DilithiumPrivateKey == nil || loaded.DilithiumPublicKey == nil {
		t.Fatalf("legacy ML-DSA keys were not loaded")
	}
	if !loaded.Verify(signedMsg, mustSign(t, kp)) {
		t.Fatalf("legacy-loaded public key failed to verify")
	}
}

func TestPQCKeyJSON_BothPresentEqualLoads(t *testing.T) {
	_, obj := genKeyPairJSON(t)
	// Duplicate each mldsa_* value under the legacy name with identical bytes.
	obj["dilithium_public_key"] = obj["mldsa_public_key"]
	obj["dilithium_private_key"] = obj["mldsa_private_key"]

	if _, err := pqc.LoadPQCKeyPair(encode(t, obj)); err != nil {
		t.Fatalf("identical mldsa_* and dilithium_* values must load: %v", err)
	}
}

func TestPQCKeyJSON_BothPresentDifferentRefuses(t *testing.T) {
	_, obj := genKeyPairJSON(t)
	other, _ := pqc.GeneratePQCKeyPair("other", "signature")
	otherRaw, err := other.MarshalWithPrivateKeys()
	if err != nil {
		t.Fatalf("marshal other: %v", err)
	}
	var otherObj map[string]json.RawMessage
	if err := json.Unmarshal(otherRaw, &otherObj); err != nil {
		t.Fatalf("unmarshal other: %v", err)
	}
	obj["dilithium_private_key"] = otherObj["mldsa_private_key"]

	if _, err := pqc.LoadPQCKeyPair(encode(t, obj)); err == nil {
		t.Fatalf("conflicting mldsa_private_key / dilithium_private_key must be refused")
	}
}

func TestPQCKeyJSON_UnknownNamesIgnored(t *testing.T) {
	_, obj := genKeyPairJSON(t)
	obj["totally_unknown_field"] = json.RawMessage(`"noise"`)
	obj["mldsa_unknown_extra"] = json.RawMessage(`[1,2,3]`)

	if _, err := pqc.LoadPQCKeyPair(encode(t, obj)); err != nil {
		t.Fatalf("unknown fields must be ignored: %v", err)
	}
}

var signedMsg = []byte("pi_q31 json test message")

func mustSign(t *testing.T, kp *pqc.PQCKeyPair) []byte {
	t.Helper()
	sig, err := kp.Sign(signedMsg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return sig
}
