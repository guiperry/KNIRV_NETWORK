package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
)

const (
	oracleBootstrapURL = "https://oracle.knirv.network"
)

func resolveBootstrapWalletAddress(ctx context.Context) (string, error) {
	store, err := core.NewNetworkWalletStore(log)
	if err != nil {
		return "", err
	}

	return store.Ensure(ctx, oracleBootstrapURL)
}

func resolveWalletAddress(ctx context.Context, walletFlag string) (string, error) {
	if trimmed := strings.TrimSpace(walletFlag); trimmed != "" {
		return trimmed, nil
	}

	return resolveBootstrapWalletAddress(ctx)
}

func resolveWalletAddressWithError(ctx context.Context, walletFlag string) (string, error) {
	address, err := resolveWalletAddress(ctx, walletFlag)
	if err != nil {
		return "", fmt.Errorf("failed to resolve wallet address: %w", err)
	}
	return address, nil
}

func resolveFaucetNetwork(environment string) string {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "mainnet", "production", "public-production":
		return "mainnet"
	case "public-testnet", "testnet", "development", "dev", "local", "":
		return "public-testnet"
	default:
		return "public-testnet"
	}
}
