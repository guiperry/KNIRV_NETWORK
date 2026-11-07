package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	"knirv-server/internal/enroll"
)

// RunEnroll is the `knirvserver enroll` subcommand. It performs a DEC-2 boot-node
// enrollment against a running D14 enrollment server, following the order
// transfer -> bond -> register.
func RunEnroll(args []string) error {
	fs := flag.NewFlagSet("enroll", flag.ContinueOnError)
	socket := fs.String("socket", "", "Enrollment server Unix socket path (e.g. /run/knirv/enroll.sock)")
	chainID := fs.String("chain-id", "knirv-testnet-1", "Chain ID the boot node joins")
	operator := fs.String("operator", "", "Operator identity claim (e.g. operator:alice)")
	nonce := fs.String("nonce", "", "Enrollment nonce; generated if empty")
	bootKeyPath := fs.String("boot-key", "", "Path to the boot-node private key file (raw 32 bytes)")
	rootKeyPath := fs.String("root-key", "", "Path to the root owner private key file (raw 32 bytes)")
	claimsPath := fs.String("claims", "", "Optional path to opaque boot-node claims file")
	lifetime := fs.Uint64("lifetime", 600, "Enrollment validity window in seconds")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *socket == "" {
		return fmt.Errorf("--socket is required")
	}
	if *operator == "" {
		return fmt.Errorf("--operator is required")
	}
	if *bootKeyPath == "" || *rootKeyPath == "" {
		return fmt.Errorf("--boot-key and --root-key are required")
	}

	bootKey, err := os.ReadFile(*bootKeyPath)
	if err != nil {
		return fmt.Errorf("read boot key: %w", err)
	}
	rootKey, err := os.ReadFile(*rootKeyPath)
	if err != nil {
		return fmt.Errorf("read root key: %w", err)
	}
	var claims []byte
	if *claimsPath != "" {
		claims, err = os.ReadFile(*claimsPath)
		if err != nil {
			return fmt.Errorf("read claims: %w", err)
		}
	}

	n := *nonce
	if n == "" {
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return fmt.Errorf("generate nonce: %w", err)
		}
		n = "enroll:" + hex.EncodeToString(buf)
	}

	req := enroll.StepRequest{ChainID: *chainID, Operator: *operator, Nonce: n}
	steps := enroll.NewStepClientUnix(*socket)
	resp, err := enroll.ClientDrive(context.Background(), enroll.SDKSigner{}, steps, req, bootKey, rootKey, claims, time.Now(), *lifetime)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("enrollment failed at %s: %s", resp.Step, resp.Error)
	}
	fmt.Printf("enrollment accepted: sequence=%v\n", resp.Order)
	return nil
}
