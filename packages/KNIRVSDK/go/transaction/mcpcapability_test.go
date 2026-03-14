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

func TestMcpCapabilityGet(t *testing.T) {
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
	_, err := client.Mcp.Capability.Get(context.TODO(), "capability_id")
	if err != nil {
		var apierr *knirvchaintransactionsdk.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}

func TestMcpCapabilityUpdateWithOptionalParams(t *testing.T) {
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
	_, err := client.Mcp.Capability.Update(context.TODO(), knirvchaintransactionsdk.McpCapabilityUpdateParams{
		SignedTransaction: knirvchaintransactionsdk.TransactionParam{
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

func TestMcpCapabilityInvokeWithOptionalParams(t *testing.T) {
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
	_, err := client.Mcp.Capability.Invoke(context.TODO(), knirvchaintransactionsdk.McpCapabilityInvokeParams{
		SignedTransaction: knirvchaintransactionsdk.TransactionParam{
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

func TestMcpCapabilityPrepareRegistrationWithOptionalParams(t *testing.T) {
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
	_, err := client.Mcp.Capability.PrepareRegistration(context.TODO(), knirvchaintransactionsdk.McpCapabilityPrepareRegistrationParams{
		Fee:          0,
		OwnerAddress: "ownerAddress",
		Descriptor: knirvchaintransactionsdk.CapabilityDescriptorUnionParam{
			OfCapabilityDescriptorResourceDescriptor: &knirvchaintransactionsdk.CapabilityDescriptorResourceDescriptorParam{
				BaseDescriptorParam: knirvchaintransactionsdk.BaseDescriptorParam{
					ID:             knirvchaintransactionsdk.String("id"),
					CapabilityType: knirvchaintransactionsdk.BaseDescriptorCapabilityTypeResource,
					CustomMetadata: map[string]any{
						"foo": "bar",
					},
					Description: knirvchaintransactionsdk.String("description"),
					GasFeeNrn:   knirvchaintransactionsdk.Int(0),
					Name:        knirvchaintransactionsdk.String("name"),
					Owner:       knirvchaintransactionsdk.String("owner"),
					Timestamp:   knirvchaintransactionsdk.Int(0),
					Version:     knirvchaintransactionsdk.String("version"),
				},
				ContentHash:  knirvchaintransactionsdk.String("contentHash"),
				ResourceType: "FILE",
				Schema: knirvchaintransactionsdk.CapabilityDescriptorResourceDescriptorSchemaParam{
					AccessInfo: map[string]any{
						"foo": "bar",
					},
					ExecutableFile:      knirvchaintransactionsdk.String("executableFile"),
					LocationHints:       []string{"string"},
					ManifestFile:        knirvchaintransactionsdk.String("manifestFile"),
					OutputDirectoryHint: knirvchaintransactionsdk.String("outputDirectoryHint"),
					Summary:             knirvchaintransactionsdk.String("summary"),
				},
			},
		},
		Message: knirvchaintransactionsdk.String("message"),
	})
	if err != nil {
		var apierr *knirvchaintransactionsdk.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}

func TestMcpCapabilityGetInvocations(t *testing.T) {
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
	_, err := client.Mcp.Capability.GetInvocations(context.TODO(), "capability_id")
	if err != nil {
		var apierr *knirvchaintransactionsdk.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}
