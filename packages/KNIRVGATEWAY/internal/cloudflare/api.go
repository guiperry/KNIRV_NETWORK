// Package cloudflare provides a Cloudflare API client for tunnel management
// and a cloudflared lifecycle manager.  It follows the same HTTP client and
// tunnel lifecycle conventions demonstrated in proxy-penguin.
package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// API is a Cloudflare v4 API client scoped to tunnel and DNS operations.
type API struct {
	token      string
	accountID  string
	zoneID     string
	httpClient *http.Client
	baseURL    string
}

// Account holds a Cloudflare account summary.
type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Tunnel mirrors a Cloudflare Tunnel resource.
type Tunnel struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	AccountTag  string       `json:"account_tag"`
	TunType     string       `json:"tun_type"`
	Connections []TunnelConn `json:"connections"`
	Token       string       `json:"token,omitempty"`
}

// TunnelConn is one edge connection within a tunnel.
type TunnelConn struct {
	ColoName string `json:"colo_name"`
	ID       string `json:"id"`
	OriginIP string `json:"origin_ip"`
}

// TunnelConfig is the payload for creating / configuring a tunnel.
type TunnelConfig struct {
	Name      string `json:"name"`
	ConfigSrc string `json:"config_src"`
}

// IngressRule maps a hostname to an upstream service URL.
type IngressRule struct {
	Hostname string `json:"hostname"`
	Service  string `json:"service"`
}

// TunnelIngress is the full ingress configuration for a tunnel.
type TunnelIngress struct {
	Config struct {
		Ingress []interface{} `json:"ingress"`
	} `json:"config"`
}

// DNSRecord is a Cloudflare DNS record.
type DNSRecord struct {
	Type    string `json:"type"`
	Proxied bool   `json:"proxied"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// apiResponse is the standard v4 API envelope.
// Messages can be either strings or objects depending on the endpoint,
// so we use interface{} and decode if needed.
type apiResponse struct {
	Success  bool                     `json:"success"`
	Errors   []map[string]interface{} `json:"errors"`
	Messages []interface{}            `json:"messages"`
	Result   interface{}              `json:"result"`
}

var tokenValidationCache sync.Map

type tunnelListCacheEntry struct {
	tunnels   []Tunnel
	expiresAt time.Time
}

var tunnelListCache sync.Map

func (a *API) cacheKey() string {
	return strings.TrimSpace(a.accountID) + "|" + strings.TrimSpace(a.zoneID) + "|" + strings.TrimSpace(a.token)
}

// NewAPI creates a Cloudflare API client. accountID may be empty, in
// which case it will be fetched from the API on first use.
func NewAPI(token, zoneID, accountID string) *API {
	return &API{
		token:     strings.TrimSpace(token),
		zoneID:    zoneID,
		accountID: strings.TrimSpace(accountID),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.cloudflare.com/client/v4",
	}
}

// ValidateToken checks that the provided API token is active.
func (a *API) ValidateToken() error {
	cacheKey := strings.TrimSpace(a.token)
	if _, ok := tokenValidationCache.Load(cacheKey); ok {
		log.Println("[cloudflare] API token validation cache hit")
		return nil
	}
	_, err := a.do("GET", "/user/tokens/verify", nil)
	if err == nil {
		tokenValidationCache.Store(cacheKey, struct{}{})
	}
	return err
}

// AccountID returns the account ID. If it was provided explicitly via
// NewAPI, that value is used. Otherwise it falls back to fetching the
// first account from the API (requires Account:Read permission).
func (a *API) AccountID() (string, error) {
	if a.accountID != "" {
		return a.accountID, nil
	}
	accts, err := a.listAccounts()
	if err != nil {
		return "", err
	}
	if len(accts) == 0 {
		return "", fmt.Errorf("no accounts associated with this token -- provide account ID explicitly")
	}
	a.accountID = accts[0].ID
	return a.accountID, nil
}

func (a *API) listAccounts() ([]Account, error) {
	resp, err := a.do("GET", "/accounts", nil)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(resp.Result)
	var accounts []Account
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, fmt.Errorf("failed to parse accounts: %w", err)
	}
	return accounts, nil
}

// CreateTunnel creates a named tunnel and returns it.
func (a *API) CreateTunnel(name string) (*Tunnel, error) {
	accountID, err := a.AccountID()
	if err != nil {
		return nil, fmt.Errorf("account ID required: %w", err)
	}

	endpoint := fmt.Sprintf("/accounts/%s/cfd_tunnel", accountID)
	resp, err := a.do("POST", endpoint, TunnelConfig{Name: name, ConfigSrc: "cloudflare"})
	if err != nil {
		return nil, err
	}
	tunnelListCache.Delete(a.cacheKey())

	return parseTunnel(resp.Result)
}

// GetTunnel retrieves a single tunnel by ID.
func (a *API) GetTunnel(tunnelID string) (*Tunnel, error) {
	accountID, err := a.AccountID()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", accountID, tunnelID)
	resp, err := a.do("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	return parseTunnel(resp.Result)
}

// ListTunnels returns all tunnels for the account.
func (a *API) ListTunnels() ([]Tunnel, error) {
	accountID, err := a.AccountID()
	if err != nil {
		return nil, err
	}
	cacheKey := a.cacheKey()
	if cached, ok := tunnelListCache.Load(cacheKey); ok {
		entry := cached.(tunnelListCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			log.Println("[cloudflare] tunnel list cache hit")
			return entry.tunnels, nil
		}
		tunnelListCache.Delete(cacheKey)
	}

	endpoint := fmt.Sprintf("/accounts/%s/cfd_tunnel", accountID)
	resp, err := a.do("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(resp.Result)
	var tunnels []Tunnel
	if err := json.Unmarshal(data, &tunnels); err != nil {
		return nil, fmt.Errorf("failed to parse tunnel list: %w", err)
	}
	tunnelListCache.Store(cacheKey, tunnelListCacheEntry{
		tunnels:   tunnels,
		expiresAt: time.Now().Add(30 * time.Second),
	})
	return tunnels, nil
}

// FindTunnelByName returns the first tunnel whose name matches.
func (a *API) FindTunnelByName(name string) (*Tunnel, error) {
	tunnels, err := a.ListTunnels()
	if err != nil {
		return nil, err
	}
	for i := range tunnels {
		if tunnels[i].Name == name {
			return &tunnels[i], nil
		}
	}
	return nil, nil
}

// ConfigureTunnel sets the ingress rules for a tunnel.
func (a *API) ConfigureTunnel(tunnelID string, rules []IngressRule) error {
	accountID, err := a.AccountID()
	if err != nil {
		return err
	}

	ingress := make([]interface{}, len(rules)+1)
	for i, r := range rules {
		ingress[i] = r
	}
	catchAll := map[string]interface{}{"service": "http_status:404"}
	ingress[len(rules)] = catchAll

	body := TunnelIngress{}
	body.Config.Ingress = ingress

	endpoint := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", accountID, tunnelID)
	_, err = a.do("PUT", endpoint, body)
	return err
}

// CreateDNSRecord creates (or updates) a proxied CNAME record for the tunnel.
func (a *API) CreateDNSRecord(hostname, tunnelID string) error {
	if a.zoneID == "" {
		return fmt.Errorf("zone ID not configured")
	}

	endpoint := fmt.Sprintf("/zones/%s/dns_records", a.zoneID)
	record := DNSRecord{
		Type:    "CNAME",
		Proxied: true,
		Name:    hostname,
		Content: fmt.Sprintf("%s.cfargotunnel.com", tunnelID),
	}
	_, err := a.do("POST", endpoint, record)
	return err
}

// GetTunnelToken retrieves the JWT used to authenticate a cloudflared instance.
func (a *API) GetTunnelToken(tunnelID string) (string, error) {
	accountID, err := a.AccountID()
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/token", accountID, tunnelID)
	resp, err := a.do("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	token, ok := resp.Result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected token response format")
	}
	return token, nil
}

// D1ResultSet is one result set returned by the D1 SQL query API.
type D1ResultSet struct {
	Results []map[string]interface{} `json:"results"`
	Success bool                     `json:"success"`
}

// QueryD1 executes a SQL statement against a Cloudflare D1 database and
// returns the rows from the first result set.
func (a *API) QueryD1(databaseID, sql string, params []interface{}) ([]map[string]interface{}, error) {
	accountID, err := a.AccountID()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("/accounts/%s/d1/database/%s/query", accountID, databaseID)
	body := map[string]interface{}{"sql": sql, "params": params}
	resp, err := a.do("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(resp.Result)
	var sets []D1ResultSet
	if err := json.Unmarshal(data, &sets); err != nil {
		return nil, fmt.Errorf("parse D1 response: %w", err)
	}
	if len(sets) == 0 || len(sets[0].Results) == 0 {
		return nil, nil
	}
	return sets[0].Results, nil
}

// GetNodeRegistrationID queries the D1 onboarding database for this node's
// registration ID, using hostname as the lookup key.
func (a *API) GetNodeRegistrationID(databaseID, hostname string) (string, error) {
	rows, err := a.QueryD1(databaseID,
		"SELECT registration_id FROM node_registrations WHERE hostname = ? AND status = 'active' LIMIT 1",
		[]interface{}{hostname},
	)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no active registration found for hostname %q", hostname)
	}
	id, ok := rows[0]["registration_id"]
	if !ok {
		return "", fmt.Errorf("registration_id column missing from D1 result")
	}
	return fmt.Sprintf("%v", id), nil
}

// --- internals ---

func (a *API) do(method, endpoint string, body interface{}) (*apiResponse, error) {
	url := a.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	tok := strings.TrimSpace(a.token)
	if strings.HasPrefix(strings.ToLower(tok), "bearer ") {
		tok = tok[7:]
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "knirvgateway/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("cloudflare %s %s → %d: %s", method, endpoint, resp.StatusCode, string(respBody))

	var r apiResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if !r.Success {
		var msgs []string
		for _, e := range r.Errors {
			if code, ok := e["code"]; ok {
				msgs = append(msgs, fmt.Sprintf("[%v] %v", code, e["message"]))
			}
		}
		return nil, fmt.Errorf("API error: %s", strings.Join(msgs, ", "))
	}
	return &r, nil
}

func parseTunnel(v interface{}) (*Tunnel, error) {
	// The API nests the tunnel under "result" inside the top-level "result".
	if m, ok := v.(map[string]interface{}); ok {
		if inner, ok := m["result"]; ok {
			v = inner
		}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal tunnel: %w", err)
	}
	var t Tunnel
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("unmarshal tunnel: %w", err)
	}
	return &t, nil
}
