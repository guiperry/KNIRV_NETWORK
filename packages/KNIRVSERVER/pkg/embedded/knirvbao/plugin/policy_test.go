package main

import (
	"testing"

	"github.com/guiperry/knirv-sdk-go/signing"
)

func TestPolicyDeniesUnlistedSigningAndFees(t *testing.T) {
	p := Policy{TypeURLs: []string{signing.ActionTypeURL}, ChainIDs: []string{"chain"}, Messages: []MessagePair{{"knirv.scenes", "scene-manifest"}}, MaxFee: "10"}
	tx := signing.SignRequest{ChainID: "chain", Fee: signing.Fee{Amount: "10"}}
	if err := p.allowTransaction(tx); err != nil {
		t.Fatal(err)
	}
	for _, amount := range []string{"11", "-1", "not-a-number"} {
		tx.Fee.Amount = amount
		if p.allowTransaction(tx) == nil {
			t.Errorf("fee %q allowed", amount)
		}
	}
	tx.Fee.Amount = ""
	if err := p.allowTransaction(tx); err != nil {
		t.Fatal("empty SDK fee should mean zero")
	}
	tx.Fee.Amount, tx.ChainID = "1", "other"
	if p.allowTransaction(tx) == nil {
		t.Fatal("other chain allowed")
	}
	tx.ChainID = "chain"
	wrongType := p
	wrongType.TypeURLs = []string{"/other.type"}
	if wrongType.allowTransaction(tx) == nil {
		t.Fatal("disallowed type URL accepted")
	}
	if (Policy{}).allowTransaction(tx) == nil {
		t.Fatal("empty policy allowed transaction")
	}
	msg := signing.MessageEnvelope{ChainID: "chain", Domain: "knirv.scenes", Purpose: "scene-manifest"}
	if err := p.allowMessage(msg); err != nil {
		t.Fatal(err)
	}
	msg.Purpose = "other"
	if p.allowMessage(msg) == nil {
		t.Fatal("other purpose allowed")
	}
}
