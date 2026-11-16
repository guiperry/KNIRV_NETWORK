package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/guiperry/knirv-sdk-go/secrets"

	"knirv-server/internal/enroll"
)

// RunEnroll is the `knirvserver enroll` subcommand family:
//
//	knirvserver enroll issue --socket S --registration-id R --chain-id C
//	    asks the enrollment server to issue a single-use nonce for an approved
//	    registration (the "on approval" step) and prints it.
//	knirvserver enroll --socket S --registration-id R --nonce N --chain-id C --operator O
//	    performs the DEC-2 enrollment (transfer -> bond -> register) with the issued
//	    nonce, signing with the boot-node and root keys resolved from the secrets
//	    provider.
func RunEnroll(args []string) error {
	if len(args) > 0 && args[0] == "issue" {
		return runEnrollIssue(args[1:])
	}
	return runEnrollRegister(args)
}

func runEnrollRegister(args []string) error {
	fs := flag.NewFlagSet("enroll", flag.ContinueOnError)
	socket := fs.String("socket", "", "Enrollment server Unix socket path (e.g. /run/knirv/enroll.sock)")
	chainID := fs.String("chain-id", "knirv-testnet-1", "Chain ID the boot node joins")
	operator := fs.String("operator", "", "Operator identity claim (e.g. operator:alice)")
	nonce := fs.String("nonce", "", "Nonce issued by the enrollment server for this registration (required)")
	registrationID := fs.String("registration-id", "", "Approved onboarding registration ID bound to this boot-node key (required)")
	lifetime := fs.Uint64("lifetime", 600, "Enrollment validity window in seconds")
	backend := fs.String("secrets-backend", "", "Secrets provider for the enrollment keys: auto, file or bao")
	baoSocket := fs.String("bao-socket", "", "OpenBao Unix socket (with --secrets-backend bao or auto)")
	baoRoleID := fs.String("bao-role-id", "", "OpenBao AppRole role_id for the enrollment key read (with --secrets-backend bao)")
	baoSecretIDFile := fs.String("bao-secret-id-file", "", "File holding the single-use AppRole secret_id (mode 0600); exchanged for a short-lived token and revoked after the run")
	devRawKeys := fs.Bool("dev-raw-keys", false, "DEV/TEST ONLY: read raw 32-byte keys from --boot-key/--root-key instead of the secrets provider")
	bootKeyPath := fs.String("boot-key", "", "DEV/TEST ONLY: boot-node private key file (raw 32 bytes)")
	rootKeyPath := fs.String("root-key", "", "DEV/TEST ONLY: root owner private key file (raw 32 bytes)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *socket == "" {
		return fmt.Errorf("--socket is required")
	}
	if *operator == "" {
		return fmt.Errorf("--operator is required")
	}
	if *registrationID == "" {
		return fmt.Errorf("--registration-id is required for automatic admission")
	}
	if strings.TrimSpace(*nonce) == "" {
		return fmt.Errorf("--nonce is required: issue one with `knirvserver enroll issue` (the server never accepts a client-generated nonce)")
	}

	ctx := context.Background()
	keys, release, err := resolveKeySource(ctx, *backend, *baoSocket, *baoRoleID, *baoSecretIDFile, *devRawKeys, *bootKeyPath, *rootKeyPath)
	if err != nil {
		return err
	}
	defer release()
	bootKey, err := keys.BootKey(ctx)
	if err != nil {
		return err
	}
	rootKey, err := keys.RootKey(ctx)
	if err != nil {
		return err
	}

	claims := enroll.MarshalClaims(enroll.EnrollmentClaims{RegistrationID: *registrationID})
	req := enroll.StepRequest{ChainID: *chainID, Operator: *operator, Nonce: *nonce}
	steps := enroll.NewStepClientUnix(*socket)
	resp, err := enroll.ClientDrive(ctx, enroll.SDKSigner{}, steps, req, bootKey, rootKey, claims, time.Now(), *lifetime)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("enrollment failed at %s: %s", resp.Step, resp.Error)
	}
	fmt.Printf("enrollment accepted: sequence=%v\n", resp.Order)
	return nil
}

func runEnrollIssue(args []string) error {
	fs := flag.NewFlagSet("enroll issue", flag.ContinueOnError)
	socket := fs.String("socket", "", "Enrollment server Unix socket path (required)")
	chainID := fs.String("chain-id", "knirv-testnet-1", "Chain ID the registration joins")
	registrationID := fs.String("registration-id", "", "Approved onboarding registration ID (required)")
	ttl := fs.Duration("ttl", enroll.DefaultIssuanceTTL, "How long the issued nonce stays redeemable (max 1h)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *socket == "" {
		return fmt.Errorf("--socket is required")
	}
	if *registrationID == "" {
		return fmt.Errorf("--registration-id is required")
	}
	issued, err := enroll.RequestIssuedChallenge(context.Background(), *socket, *registrationID, *chainID, *ttl)
	if err != nil {
		return err
	}
	fmt.Printf("nonce=%s\nregistration_id=%s\nchain_id=%s\nvalidator_id=%s\nexpires_at=%s\n",
		issued.Nonce, issued.RegistrationID, issued.ChainID, issued.ValidatorID, issued.ExpiresAt.UTC().Format(time.RFC3339))
	return nil
}

// resolveKeySource chooses where the enrollment keys come from. The secrets
// provider is the production path. Raw key files are reachable only through the
// explicitly named --dev-raw-keys flag and never by default. The returned release
// function revokes any AppRole token it obtained; call it once the run is over.
func resolveKeySource(ctx context.Context, backend, baoSocket, roleID, secretIDFile string, devRaw bool, bootPath, rootPath string) (enroll.KeySource, func(), error) {
	noop := func() {}
	if devRaw {
		if backend != "" {
			return nil, noop, fmt.Errorf("--dev-raw-keys and --secrets-backend are mutually exclusive")
		}
		fmt.Fprintln(os.Stderr, "enroll: WARNING --dev-raw-keys reads raw key files; dev/test use only")
		return enroll.RawFileKeySource{BootPath: bootPath, RootPath: rootPath}, noop, nil
	}
	if backend == "" {
		return nil, noop, fmt.Errorf("--secrets-backend (auto, file or bao) is required; --dev-raw-keys is the test-only file fallback")
	}
	token := ""
	release := noop
	if backend == string(secrets.BackendBao) {
		if baoSocket == "" {
			return nil, noop, fmt.Errorf("--bao-socket is required with --secrets-backend bao")
		}
		if roleID == "" {
			return nil, noop, fmt.Errorf("--bao-role-id is required with --secrets-backend bao")
		}
		if secretIDFile == "" {
			return nil, noop, fmt.Errorf("--bao-secret-id-file is required with --secrets-backend bao")
		}
		secretID, err := readSecretIDFile(secretIDFile)
		if err != nil {
			return nil, noop, err
		}
		session, err := loginBaoAppRole(ctx, baoSocket, roleID, secretID)
		if err != nil {
			return nil, noop, err
		}
		// Q47: the secret_id is single-use. Once login has consumed it, delete the file
		// immediately so no reusable copy stays on disk. If the delete fails, the run is
		// aborted: the session is revoked and the error surfaces, rather than continuing
		// with a credential file still present.
		if err := os.Remove(secretIDFile); err != nil {
			revokeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if rerr := session.Close(revokeCtx); rerr != nil {
				fmt.Fprintf(os.Stderr, "enroll: WARNING %v\n", rerr)
			}
			return nil, noop, fmt.Errorf("remove secret-id file after login: %w", err)
		}
		token = session.Token()
		release = func() {
			if err := session.Close(context.Background()); err != nil {
				fmt.Fprintf(os.Stderr, "enroll: WARNING %v\n", err)
			}
		}
	}
	provider, err := secrets.Select(backend, baoSocket, token)
	if err != nil {
		release()
		return nil, noop, fmt.Errorf("select secrets provider: %w", err)
	}
	return enroll.ProviderKeySource{Provider: provider}, release, nil
}
