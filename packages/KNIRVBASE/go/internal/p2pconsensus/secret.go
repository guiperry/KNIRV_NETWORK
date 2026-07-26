package p2pconsensus

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// deriveNetworkKey mixes the NetworkID with the NetworkSecret (PSK) to produce a
// stable 32-byte key. This key is used both to derive the DHT service ID /
// pubsub topic (so an unknown secret cannot discover or join the topic) and as
// the HMAC key for authenticating gateway-sourced messages.
//
// With an empty secret the result degrades to a plain SHA-256 of the NetworkID,
// preserving backwards-compatible open-network behaviour.
func deriveNetworkKey(networkID, secret string) [32]byte {
	h := sha256.New()
	h.Write([]byte("knirvbase-network-v1"))
	h.Write([]byte{0})
	h.Write([]byte(networkID))
	h.Write([]byte{0})
	h.Write([]byte(secret))
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// ServiceID returns a stable identifier derived from the NetworkID and secret.
// It is suitable for use as a libp2p DHT service ID or rendezvous point.
func ServiceID(networkID, secret string) string {
	key := deriveNetworkKey(networkID, secret)
	return fmt.Sprintf("knirv-%s", hex.EncodeToString(key[:8]))
}

// Topic returns the pubsub topic for a network, bound to the secret so that two
// networks with the same human-readable NetworkID but different secrets do not
// collide or cross-talk.
func Topic(networkID, secret string) string {
	key := deriveNetworkKey(networkID, secret)
	return fmt.Sprintf("knirvbase.%s.ops", hex.EncodeToString(key[:8]))
}

// SignMessage produces an HMAC-SHA256 hex signature over the canonical bytes of
// a message, authenticated by the network secret. Returns an empty string when
// no secret is configured (open network).
func SignMessage(networkID, secret string, payload []byte) string {
	if secret == "" {
		return ""
	}
	key := deriveNetworkKey(networkID, secret)
	mac := hmac.New(sha256.New, key[:])
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyMessage checks a gateway-supplied HMAC signature against the expected
// value derived from the network secret. When no secret is configured (open
// network) signatures are not required and the function returns true.
func VerifyMessage(networkID, secret, signature string, payload []byte) bool {
	if secret == "" {
		// Open network: no authentication enforced.
		return true
	}
	if signature == "" {
		return false
	}
	expected := SignMessage(networkID, secret, payload)
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}
