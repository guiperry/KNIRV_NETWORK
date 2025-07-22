// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package transaction_test

import (
	"context"
	"errors"
	"os"
	"testing"

	knirvchaintransactionsdk "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/testutil"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
)

func TestBlockSubmitWithOptionalParams(t *testing.T) {
	t.Skip("skipped: tests are disabled for the time being")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := knirvchaintransactionsdk.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Block.Submit(context.TODO(), knirvchaintransactionsdk.BlockSubmitParams{
		Block: knirvchaintransactionsdk.BlockParam{
			BlockNumber:     knirvchaintransactionsdk.Int(0),
			Hash:            knirvchaintransactionsdk.String("U3RhaW5sZXNzIHJvY2tz"),
			Nonce:           knirvchaintransactionsdk.Int(0),
			PrevHash:        knirvchaintransactionsdk.String("U3RhaW5sZXNzIHJvY2tz"),
			ProposerAddress: knirvchaintransactionsdk.String("proposer_address"),
			Timestamp:       knirvchaintransactionsdk.Int(0),
			Transactions: []knirvchaintransactionsdk.TransactionParam{{
				ID:        "0xabcdef123456...",
				Fee:       "fee",
				From:      "from",
				PublicKey: "public_key",
				Signature: "U3RhaW5sZXNzIHJvY2tz",
				Timestamp: 0,
				Type:      "type",
				Version:   1,
				Data: map[string]any{
					"foo": "bar",
				},
				Status:          knirvchaintransactionsdk.String("status"),
				To:              knirvchaintransactionsdk.String("to"),
				TransactionHash: knirvchaintransactionsdk.String("transaction_hash"),
				Value:           knirvchaintransactionsdk.String("1000000000000000000"),
			}},
		},
	})
	if err != nil {
		var apierr *knirvchaintransactionsdk.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}
