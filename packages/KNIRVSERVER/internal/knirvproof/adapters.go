package knirvproof

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// LocalReplicator confirms the local encrypted CAS. It is appropriate only
// when requiredReplicas is one; production must install a replicated adapter.
type LocalReplicator struct {
	Location string
}

// FilesystemReplicator copies ciphertext CAS objects into independently
// configured stores. A production deployment should place these roots on
// distinct durable volumes or nodes; confirmations are emitted only after
// every object has been content-verified by each destination FileStore.
type FilesystemReplicator struct {
	PrimaryLocation string
	Replicas        []*FileStore
}

// ValidationChainMinter finalizes a validation proof by committing it to the
// embedded Validation Chain (pkg/embedded/validationchain, spawned and owned
// by KNIRVSERVER, bound to a Unix domain socket), not KNIRVCHAIN. KNIRVCHAIN
// stays sovereign, minting NFT bundles only; the Validation Chain is the
// merkle/checkpoint source of record for proof commits.
type ValidationChainMinter struct {
	client        *http.Client
	internalToken string
}

func NewValidationChainMinter(socketPath, internalToken string) (*ValidationChainMinter, error) {
	if strings.TrimSpace(socketPath) == "" {
		return nil, fmt.Errorf("validation chain socket path is required")
	}
	if strings.TrimSpace(internalToken) == "" {
		return nil, fmt.Errorf("internal chain token is required")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	return &ValidationChainMinter{
		client:        &http.Client{Transport: transport, Timeout: 10 * time.Second},
		internalToken: internalToken,
	}, nil
}

func (m *ValidationChainMinter) Mint(ctx context.Context, mint MintRequest) (*ChainReceipt, error) {
	body, err := json.Marshal(mint)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix/validation/commit", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-KNIRV-Internal-Token", m.internalToken)
	response, err := m.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("submit validation proof to Validation Chain: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("%w: Validation Chain rejected conflicting proof binding", ErrConflict)
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Validation Chain returned %d", response.StatusCode)
	}
	var receipt ChainReceipt
	if err := json.NewDecoder(response.Body).Decode(&receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

// ChainEventBundleFetcher re-fetches a KNIRVCHAIN event-bundle NFT by event
// ID over the same trusted unix-socket path ChainMinter uses, so
// NativeVerifier can independently confirm a bundle a submission references
// was actually minted (chain_refactor.md Open Decision #2) rather than
// trusting the CLI-supplied hash alone.
type ChainEventBundleFetcher struct {
	client *http.Client
}

func NewChainEventBundleFetcher(socketPath string) (*ChainEventBundleFetcher, error) {
	if strings.TrimSpace(socketPath) == "" {
		return nil, fmt.Errorf("KNIRVCHAIN socket is required")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	return &ChainEventBundleFetcher{client: &http.Client{Transport: transport, Timeout: 10 * time.Second}}, nil
}

// FetchBundleHash returns the bundle_hash KNIRVCHAIN has on record for
// eventID. ok is false when no bundle has been minted for that event yet
// (a 404) — the caller treats that as a retryable, not permanent, failure
// since minting is asynchronous relative to the commit that references it.
func (f *ChainEventBundleFetcher) FetchBundleHash(ctx context.Context, eventID string) (hash string, ok bool, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://knirvchain/api/v1/event-bundles/"+eventID, nil)
	if err != nil {
		return "", false, err
	}
	response, err := f.client.Do(request)
	if err != nil {
		return "", false, fmt.Errorf("fetch event bundle from KNIRVCHAIN: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAccepted {
		return "", false, fmt.Errorf("KNIRVCHAIN returned %d fetching event bundle", response.StatusCode)
	}
	var receipt struct {
		BundleHash string `json:"bundle_hash"`
	}
	if err := json.NewDecoder(response.Body).Decode(&receipt); err != nil {
		return "", false, err
	}
	if receipt.BundleHash == "" {
		return "", false, nil
	}
	return receipt.BundleHash, true, nil
}

func (r LocalReplicator) Replicate(_ context.Context, submission ProofSubmission, store *FileStore) ([]StorageConfirmation, error) {
	for _, object := range submission.Objects {
		exists, size, err := store.HasObject(object.CID)
		if err != nil {
			return nil, err
		}
		if !exists || size != object.Size {
			return nil, ErrNotFound
		}
	}
	location := r.Location
	if location == "" {
		location = "local-filesystem"
	}
	return []StorageConfirmation{{Location: location, Root: submission.StorageRoot, At: time.Now().UTC()}}, nil
}

func (r FilesystemReplicator) Replicate(ctx context.Context, submission ProofSubmission, primary *FileStore) ([]StorageConfirmation, error) {
	local := LocalReplicator{Location: r.PrimaryLocation}
	confirmations, err := local.Replicate(ctx, submission, primary)
	if err != nil {
		return nil, err
	}
	for index, replica := range r.Replicas {
		if replica == nil {
			return nil, fmt.Errorf("proof replica %d is not configured", index+1)
		}
		for _, object := range submission.Objects {
			file, _, err := primary.OpenObject(object.CID)
			if err != nil {
				return nil, err
			}
			size, _, putErr := replica.PutObject(ctx, object.CID, file)
			closeErr := file.Close()
			if putErr != nil {
				return nil, fmt.Errorf("replicate %s to store %d: %w", object.CID, index+1, putErr)
			}
			if closeErr != nil {
				return nil, closeErr
			}
			if size != object.Size {
				return nil, fmt.Errorf("replica %d size mismatch for %s", index+1, object.CID)
			}
		}
		confirmations = append(confirmations, StorageConfirmation{
			Location: fmt.Sprintf("filesystem-replica-%d", index+1), Root: submission.StorageRoot, At: time.Now().UTC(),
		})
	}
	return confirmations, nil
}
