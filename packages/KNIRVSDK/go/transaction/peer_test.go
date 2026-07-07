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

func TestPeerList(t *testing.T) {
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
	_, err := client.Peers.List(context.TODO())
	if err != nil {
		var apierr *knirvchaintransactionsdk.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}
