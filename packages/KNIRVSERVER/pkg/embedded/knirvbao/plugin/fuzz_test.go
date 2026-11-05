package main

import (
	"encoding/json"
	"testing"

	"github.com/guiperry/knirv-sdk-go/signing"
)

func FuzzDecodeUntrustedRequest(f *testing.F) {
	for _, seed := range []string{`{}`, `{"ChainID":"dev","Fee":{"Amount":"1"}}`, `{"messages":[{"domain":"x","purpose":"y"}]}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		var data map[string]any
		if json.Unmarshal([]byte(input), &data) != nil || data == nil {
			return
		}
		var tx signing.SignRequest
		if decodeRequest(data, &tx) == nil {
			_ = (Policy{}).allowTransaction(tx)
		}
		var msg signing.MessageEnvelope
		if decodeRequest(data, &msg) == nil {
			_ = (Policy{}).allowMessage(msg)
		}
		var policy Policy
		if decodeRequest(data, &policy) == nil {
			_ = policy.validate()
		}
	})
}
