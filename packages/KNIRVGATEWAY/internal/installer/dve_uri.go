package installer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha3"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type DVEURIProxy struct {
	oracleSocket string
	httpClient  *http.Client
	logger      *zap.Logger
}

type GenerateWalletRequest struct {
	OwnerID string `json:"owner_id,omitempty"`
}

type GenerateWalletResponse struct {
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
	WalletAddr string `json:"wallet_address"`
	DVEID       string `json:"dve_id,omitempty"`
}

type GenerateDVEURIRequest struct {
	DVEID      string `json:"dve_id"`
	DesiredURI string `json:"desired_uri,omitempty"`
}

type GenerateDVEURIResponse struct {
	DVEID    string `json:"dve_id"`
	FullURI string `json:"full_uri"`
	Hash   string `json:"hash"`
	Created int64  `json:"created"`
}

func NewDVEURIProxy(oracleSocket string, logger *zap.Logger) *DVEURIProxy {
	client := &http.Client{Timeout: 30 * time.Second}
	if oracleSocket != "" {
		client.Transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", oracleSocket)
			},
		}
	}
	return &DVEURIProxy{
		oracleSocket: oracleSocket,
		httpClient:   client,
		logger:       logger,
	}
}

func (p *DVEURIProxy) GenerateWallet(req GenerateWalletRequest) (*GenerateWalletResponse, error) {
	if p.oracleSocket == "" {
		p.logger.Warn("Oracle socket not configured — falling back to local wallet generation")
		return p.generateLocalWallet(req)
	}
	p.logger.Info("Generating wallet via oracle socket",
		zap.String("socket", p.oracleSocket))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "http://knirvoracle/oracle/v3/install/wallet", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.logger.Warn("Failed to connect to oracle socket, generating local wallet",
			zap.Error(err))
		return p.generateLocalWallet(req)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oracle returned status %d: %s", resp.StatusCode, string(body))
	}

	var result GenerateWalletResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (p *DVEURIProxy) generateLocalWallet(req GenerateWalletRequest) (*GenerateWalletResponse, error) {
	privateKey, err := generateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &GenerateWalletResponse{
		PrivateKey:  privateKey.privateKeyHex,
		PublicKey:   privateKey.publicKeyHex,
		WalletAddr: privateKey.address,
		DVEID:       req.OwnerID,
	}, nil
}

func (p *DVEURIProxy) GenerateDVEURI(req GenerateDVEURIRequest) (*GenerateDVEURIResponse, error) {
	if p.oracleSocket == "" {
		p.logger.Warn("Oracle socket not configured — falling back to local DVE URI generation")
		return p.generateLocalDVEURI(req)
	}
	p.logger.Info("Generating DVE URI",
		zap.String("dve_id", req.DVEID))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "http://knirvoracle/oracle/v3/install/dve_uri", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		p.logger.Warn("Failed to connect to oracle socket, generating local DVE URI",
			zap.Error(err))
		return p.generateLocalDVEURI(req)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oracle returned status %d: %s", resp.StatusCode, string(body))
	}

	var result GenerateDVEURIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (p *DVEURIProxy) generateLocalDVEURI(req GenerateDVEURIRequest) (*GenerateDVEURIResponse, error) {
	hash, err := generateDVEHash(req.DVEID, req.DesiredURI)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DVE hash: %w", err)
	}

	fullURI := fmt.Sprintf("knirv://%s/%s", hash[:12], req.DVEID)

	return &GenerateDVEURIResponse{
		DVEID:   req.DVEID,
		FullURI: fullURI,
		Hash:   hash,
		Created: time.Now().Unix(),
	}, nil
}

func (p *DVEURIProxy) CheckOracleHealth() error {
	if p.oracleSocket == "" {
		return fmt.Errorf("oracle socket not configured")
	}
	resp, err := p.httpClient.Get("http://knirvoracle/oracle/v3/health")
	if err != nil {
		return fmt.Errorf("oracle health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oracle unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

type keyPair struct {
	privateKeyHex string
	publicKeyHex string
	address     string
}

func generateKeyPair() (*keyPair, error) {
	privKeyBytes := make([]byte, 32)
	_, err := rand.Read(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	publicKeyBytes := make([]byte, 64)
	copy(publicKeyBytes[:32], privKeyBytes)

	return &keyPair{
		privateKeyHex: hex.EncodeToString(privKeyBytes),
		publicKeyHex:  hex.EncodeToString(publicKeyBytes),
		address:      hex.EncodeToString(publicKeyBytes[len(publicKeyBytes)-20:]),
	}, nil
}

func generateDVEHash(dveID, desiredURI string) (string, error) {
	data := fmt.Sprintf("dve:%s:%s:%d", dveID, desiredURI, time.Now().UnixNano())
	hash := sha3.New256()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil)), nil
}