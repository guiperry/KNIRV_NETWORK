package tlsprovider

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	cfOriginCAEndpoint = "https://api.cloudflare.com/client/v4/certificates"
	cfZonesEndpoint    = "https://api.cloudflare.com/client/v4/zones/"

	// Maximum Cloudflare Origin CA validity (15 years).
	certValidityDays = 5475
	// Renew when fewer than this many days remain.
	renewThresholdDays = 30

	DefaultCertFile = "/etc/knirvserver/tls/origin.crt"
	DefaultKeyFile  = "/etc/knirvserver/tls/origin.key"

	// DefaultCheckInterval is how often the auto-renewal goroutine wakes up.
	DefaultCheckInterval = 12 * time.Hour
)

// needsProvisioning returns true when certFile is absent, invalid PEM, or
// expiring within renewThresholdDays.
func needsProvisioning(certFile string) bool {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return true
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return true
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true
	}
	return time.Until(cert.NotAfter) < time.Duration(renewThresholdDays)*24*time.Hour
}

// zoneName fetches the root domain for a Cloudflare zone.
func zoneName(ctx context.Context, apiToken, zoneID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfZonesEndpoint+zoneID, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloudflare zones API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Result  struct{ Name string `json:"name"` }                  `json:"result"`
		Success bool                                                   `json:"success"`
		Errors  []struct{ Message string `json:"message"` }           `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode zones response: %w", err)
	}
	if !result.Success || result.Result.Name == "" {
		msgs := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			msgs[i] = e.Message
		}
		return "", fmt.Errorf("cloudflare zones API failed for zone %s: %v", zoneID, msgs)
	}
	return result.Result.Name, nil
}

// generateCSR creates an ECDSA P-256 key and a CSR for the given hostnames.
// Returns (csrPEM, keyPEM). The private key never leaves the process.
func generateCSR(hostnames []string) (csrPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ecdsa key: %w", err)
	}

	tmpl := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: hostnames[0]},
		DNSNames: hostnames,
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, tmpl, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create CSR: %w", err)
	}
	csrPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal EC key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return csrPEM, keyPEM, nil
}

// requestOriginCACert submits a CSR to the Cloudflare Origin CA API and
// returns the signed certificate PEM.
func requestOriginCACert(ctx context.Context, apiToken string, hostnames []string, csrPEM []byte) (string, error) {
	body, err := json.Marshal(map[string]any{
		"hostnames":          hostnames,
		"requested_validity": certValidityDays,
		"request_type":       "origin-ecc",
		"csr":                string(csrPEM),
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfOriginCAEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloudflare origin CA request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var result struct {
		Result  struct{ Certificate string `json:"certificate"` }  `json:"result"`
		Success bool                                               `json:"success"`
		Errors  []struct{ Message string `json:"message"` }       `json:"errors"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse origin CA response: %w", err)
	}
	if !result.Success {
		msgs := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			msgs[i] = e.Message
		}
		return "", fmt.Errorf("cloudflare origin CA error: %v", msgs)
	}
	if result.Result.Certificate == "" {
		return "", fmt.Errorf("cloudflare origin CA returned empty certificate (HTTP %d)", resp.StatusCode)
	}
	return result.Result.Certificate, nil
}

// EnsureCertificate checks whether a valid TLS certificate exists at certFile/keyFile.
// If the certificate is absent or expiring within renewThresholdDays, it provisions a
// new 15-year Cloudflare Origin CA certificate.
//
// apiToken must have "SSL and Certificates: Edit" permission.
// zoneID identifies the Cloudflare zone; the zone's root domain becomes the cert's SANs
// (apex + wildcard).
//
// Returns nil immediately when apiToken or zoneID is empty — safe no-op for dev/testnet.
func EnsureCertificate(ctx context.Context, apiToken, zoneID, certFile, keyFile string) error {
	if apiToken == "" || zoneID == "" {
		return nil
	}
	if certFile == "" {
		certFile = DefaultCertFile
	}
	if keyFile == "" {
		keyFile = DefaultKeyFile
	}

	if !needsProvisioning(certFile) {
		log.Printf("[TLS] certificate valid at %s, skipping provisioning", certFile)
		return nil
	}

	log.Printf("[TLS] provisioning Cloudflare Origin CA certificate...")

	domain, err := zoneName(ctx, apiToken, zoneID)
	if err != nil {
		return fmt.Errorf("resolve domain for zone %s: %w", zoneID, err)
	}
	hostnames := []string{domain, "*." + domain}
	log.Printf("[TLS] requesting certificate for: %v", hostnames)

	csrPEM, keyPEM, err := generateCSR(hostnames)
	if err != nil {
		return err
	}

	certPEM, err := requestOriginCACert(ctx, apiToken, hostnames, csrPEM)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(certFile), 0750); err != nil {
		return fmt.Errorf("create TLS directory %s: %w", filepath.Dir(certFile), err)
	}
	if err := os.WriteFile(certFile, []byte(certPEM), 0640); err != nil {
		return fmt.Errorf("write cert to %s: %w", certFile, err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		_ = os.Remove(certFile)
		return fmt.Errorf("write key to %s: %w", keyFile, err)
	}

	log.Printf("[TLS] certificate provisioned → cert: %s  key: %s", certFile, keyFile)
	return nil
}

// StartAutoRenew starts a background goroutine that periodically calls
// EnsureCertificate. When the certificate nears expiry it is renewed
// automatically without a server restart.
//
// checkInterval controls how often the check runs; pass 0 to use
// DefaultCheckInterval (12 hours). The goroutine exits when ctx is cancelled.
//
// Returns immediately (no goroutine) when apiToken or zoneID is empty.
func StartAutoRenew(ctx context.Context, apiToken, zoneID, certFile, keyFile string, checkInterval time.Duration) {
	if apiToken == "" || zoneID == "" {
		return
	}
	if checkInterval <= 0 {
		checkInterval = DefaultCheckInterval
	}

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		log.Printf("[TLS] auto-renewal goroutine started (interval: %s, threshold: %d days)",
			checkInterval, renewThresholdDays)

		for {
			select {
			case <-ctx.Done():
				log.Printf("[TLS] auto-renewal goroutine stopped")
				return
			case <-ticker.C:
				renewCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
				if err := EnsureCertificate(renewCtx, apiToken, zoneID, certFile, keyFile); err != nil {
					log.Printf("[TLS] auto-renewal failed: %v", err)
				}
				cancel()
			}
		}
	}()
}
