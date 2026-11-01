package replica

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/guiperry/knirv-sdk-go/replication"
	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// HTTPSource uses the existing Gateway /oracle proxy (or a direct Oracle URL
// in a local test). Public head/MMR calls are unsigned; the state-log pull is
// authenticated with the bootnode's RemoteSigner and never with raw key bytes.
type HTTPSource struct {
	baseURL string
	chainID string
	client  *http.Client
	signer  signing.RemoteSigner
}

func NewHTTPSource(baseURL, chainID string, signer signing.RemoteSigner, client *http.Client) (*HTTPSource, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("replica: invalid source URL %q", baseURL)
	}
	if chainID == "" {
		return nil, fmt.Errorf("replica: source chain ID is required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPSource{baseURL: strings.TrimRight(baseURL, "/"), chainID: chainID, signer: signer, client: client}, nil
}

func (s *HTTPSource) Head(ctx context.Context) (replication.SignedHead, error) {
	var head replication.SignedHead
	return head, s.get(ctx, "/oracle/v3/replication/head", nil, false, &head)
}

func (s *HTTPSource) HeadsAfter(ctx context.Context, after uint64) ([]replication.SignedHead, error) {
	var result struct {
		Heads []replication.SignedHead `json:"heads"`
	}
	return result.Heads, s.get(ctx, "/oracle/v3/replication/heads", url.Values{"after_seq": {strconv.FormatUint(after, 10)}}, false, &result)
}

func (s *HTTPSource) Leaves(ctx context.Context, from uint64, limit int) ([]string, error) {
	var result struct {
		Leaves []string `json:"leaves"`
	}
	return result.Leaves, s.get(ctx, "/oracle/v3/replication/leaves", url.Values{"from": {strconv.FormatUint(from, 10)}, "limit": {strconv.Itoa(limit)}}, false, &result)
}

func (s *HTTPSource) State(ctx context.Context, from uint64, limit int) ([]StateRecord, error) {
	var result struct {
		Records []StateRecord `json:"records"`
	}
	return result.Records, s.get(ctx, "/oracle/v3/replication/state", url.Values{"from": {strconv.FormatUint(from, 10)}, "limit": {strconv.Itoa(limit)}}, true, &result)
}

func (s *HTTPSource) Registry(ctx context.Context) (RegistrySnapshot, error) {
	var result RegistrySnapshot
	return result, s.get(ctx, "/oracle/v3/replication/registry", nil, true, &result)
}

func (s *HTTPSource) Aux(ctx context.Context) (AuxSnapshot, error) {
	var result AuxSnapshot
	return result, s.get(ctx, "/oracle/v3/replication/aux", nil, true, &result)
}

func (s *HTTPSource) Checkpoints(ctx context.Context, chain string, after uint64, limit int) ([]*types.CheckpointRecord, error) {
	var result struct {
		Records []*types.CheckpointRecord `json:"records"`
	}
	query := url.Values{"chain": {chain}, "after_height": {strconv.FormatUint(after, 10)}, "limit": {strconv.Itoa(limit)}}
	return result.Records, s.get(ctx, "/oracle/v3/replication/checkpoints", query, false, &result)
}

func (s *HTTPSource) get(ctx context.Context, path string, query url.Values, authenticated bool, out any) error {
	endpoint := s.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if authenticated {
		if s.signer == nil {
			return fmt.Errorf("%w: standby signer unavailable", replication.ErrPullAuth)
		}
		nonce, err := randomNonce()
		if err != nil {
			return err
		}
		signed, err := replication.SignPull(s.signer, replication.PullRequest{ChainID: s.chainID, Nonce: nonce})
		if err != nil {
			return err
		}
		body, err := json.Marshal(signed)
		if err != nil {
			return err
		}
		req.Header.Set("X-Knirv-Replica-Auth", base64.StdEncoding.EncodeToString(body))
	}
	response, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("replica: source %s returned %s", path, response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("replica: decode %s: %w", path, err)
	}
	return nil
}

func randomNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("replica: nonce: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
