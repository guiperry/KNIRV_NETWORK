// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package transaction_test

import (
	"context"
	"os"
	"testing"

	knirvchaintransactionsdk "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/internal/testutil"
	"github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction/option"
)

func TestUsage(t *testing.T) {
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
	chain, err := client.Chain.Get(context.TODO())
	if err != nil {
		t.Fatalf("err should be nil: %s", err.Error())
	}
	t.Logf("%+v\n", chain.ChainID)
}
