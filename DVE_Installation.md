# DVE Installation: Integration with Existing Codebase

## Overview

This document maps the DVE Installation plan to existing codebase components, identifying where we can integrate without duplicating code.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        INSTALLER FLOW                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. KNIRVGATEWAY                                                    │
│     ├─ Registry/STUN Discovery (existing tunnel/registry.go)          │
│     └─ Port Discovery (existing tunnel/stun.go)                      │
│            │                                                         │
│            ▼                                                         │
│  2. KNIRVORACLE (core)                                              │
│     ├─ Wallet Generation (existing crypto/ecdsa.go)                   │
│     └─ DVE URI Generation (adapt generateTransferID pattern)            │
│            │                                                         │
│            ▼                                                         │
│  3. KNIRVSERVER                                                    │
│     ├─ Service Setup (reuse knirvchain.Manager pattern)                │
│     ├─ InstallComplete Tracking (extend DVENode in objects/)             │
│     └─ Validation Chain Submission (extend validationchain.Client)    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                       DVE ROUTING FLOW                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Browser: knirv://abc123/my-dve-id                                    │
│         │                                                             │
│         ▼                                                             │
│  KNIRVSERVER (Router)                                                │
│     ├─ Parse DVE ID from URI                                         │
│     ├─ Check BuntDB cache                                            │
│     │   └─ If miss → Query Validation Chain                          │
│     ├─ Validate (no auth for public pages)                           │
│     └─ Serve Go HTML Template (Public DVE Page)                      │
│                                                                      │
│  KNIRVARENA (interactive)                                            │
│     └─ WebSocket proxy for interactive DVE features                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Integration Map

### Phase 1: KNIRVGATEWAY - Reuse Existing Code

#### 1.1 Registry Handler → EXTEND `internal/tunnel/registry.go`

**Existing:**
```go
// internal/tunnel/registry.go
type RegistryManager struct {
    nodes map[string]*NodeInfo
}

func (rm *RegistryManager) RegisterNodeViaAPI(nodeID, chainID, ip string, port int) error
func (rm *RegistryManager) GetNodeByChainId(chainID string) (*NodeInfo, error)
```

**Plan:** Add new methods for installer registration:
```go
// internal/tunnel/registry.go - ADD
func (rm *RegistryManager) RegisterBootnode(nodeID, chainID, ip string, port int) error
func (rm *RegistryManager) GetBootnodes() ([]*NodeInfo, error)
```

**Files to Modify:**
- `packages/KNIRVGATEWAY/internal/tunnel/registry.go` - Add installer methods

#### 1.2 STUN/Port Discovery → REUSE `internal/tunnel/stun.go`

**Existing:**
```go
// internal/tunnel/stun.go
type STUNServer struct {
    UDPAddr *net.UDPAddr
    conn    *net.UDPConn
    logger  *zap.Logger
}

func NewSTUNServer(port int, logger *zap.Logger) *STUNServer
func (s *STUNServer) Start() error
func (s *STUNServer) Stop() error
func (s *STUNServer) handleConnections()
```

**Plan:** Create STUN client that uses existing server pattern:
```go
// internal/installer/stun_client.go - NEW (adapts existing pattern)
type STUNClient struct {
    servers []string
    client  *stun.Client
}
```

**Files:**
- New: `packages/KNIRVGATEWAY/internal/installer/stun_client.go`
- Modify: `packages/KNIRVGATEWAY/internal/server/server.go` - Register routes

#### 1.3 URI Proxy → REUSE `internal/oracle` pattern

**Plan:** Create proxy to KNIRVORACLE Unix socket
```go
// internal/installer/dve_uri.go - NEW
type DVEURIProxy struct {
    oracleSocket string
    httpClient *http.Client
}
```

**Files:**
- New: `packages/KNIRVGATEWAY/internal/installer/dve_uri.go`

---

### Phase 2: KNIRVORACLE - Reuse Existing Crypto

#### 2.1 Wallet Generation → REUSE `internal/oracle/crypto/ecdsa.go`

**Existing:**
```go
// internal/oracle/crypto/ecdsa.go
func GenerateKeyPair() (*KeyPair, error)
func PrivateKeyFromHex(privateKeyHex string) (*KeyPair, error)
func SignMessage(privateKeyHex string, message []byte) (string, error)

// KeyPair methods
func (kp *KeyPair) PrivateKeyHex() string
func (kp *KeyPair) PublicKeyHex() string
func (kp *KeyPair) Address() types.Address
```

**Plan:** Add HTTP routes that use existing crypto
```go
// internal/oracle/routes/wallet.go - NEW (uses existing crypto)
func (r *OracleRoutes) handleGenerateWallet(w http.ResponseWriter, req *http.Request) {
    kp, err := crypto.GenerateKeyPair()  // ← REUSE EXISTING
    // ... return JSON response
}
```

**Files:**
- New: `packages/KNIRVORACLE/internal/oracle/routes/wallet.go`

#### 2.2 DVE URI Generation → ADAPT `internal/oracle/crosschain/router.go` pattern

**Existing pattern:**
```go
// internal/oracle/crosschain/router.go line 300
func generateTransferID(req *TransferRequest) string {
    data := fmt.Sprintf("%s:%s:%s:%s:%d:%d",
        req.SourceChain.String(), req.DestChain.String(),
        req.Sender, req.Recipient,
        req.Amount, time.Now().UnixNano(),
    )
    return crypto.Keccak256HashWithPrefix([]byte(data))
}
```

**Plan:** Adapt for DVE URI:
```go
// internal/oracle/routes/dve_uri.go - NEW
func generateDVEURI(dveID, desiredID string) string {
    data := fmt.Sprintf("dve:%s:%s:%d", dveID, desiredID, time.Now().UnixNano())
    return "knirv://" + crypto.Keccak256HashWithPrefix([]byte(data))[:12] + "/" + dveID
}
```

**Files:**
- New: `packages/KNIRVORACLE/internal/oracle/routes/dve_uri.go`

---

### Phase 3: KNIRVSERVER - Reuse Manager Patterns

#### 3.1 InstallComplete Tracking → EXTEND `internal/objects/dve.go`

**Existing:**
```go
// internal/objects/dve.go
type DVENode struct {
    ID              string    `json:"id"`
    IPAddress       string    `json:"ip_address"`
    SSHPort         int       `json:"ssh_port"`
    Status          string    `json:"status"`  // "online", "offline", etc.
    CreatedAt       time.Time `json:"created_at"`
    // ... many more fields
}
```

**Plan:** Add new fields:
```go
// internal/objects/dve.go - ADD
type DVENode struct {
    // ... existing fields ...
    
    // NEW INSTALLER FIELDS
    InstallComplete   bool      `json:"install_complete"`
    InstallCompleteAt time.Time `json:"install_complete_at"`
    InstallPhase    string    `json:"install_phase"` // "registry", "stun", "wallet", "dve_uri", "service", "complete"
    DVEURI         string    `json:"dve_uri"`
    WalletAddress string    `json:"wallet_address"`
}
```

**Files:**
- Modify: `packages/KNIRVSERVER/backend/internal/objects/dve.go`

#### 3.2 Service Setup → REUSE `pkg/knirvchain/manager.go` pattern

**Existing:**
```go
// pkg/knirvchain/manager.go
type Manager struct {
    config  *ManagerConfig
    cmd     *exec.Cmd
    logger  *zap.Logger
    client  *http.Client
    baseURL string
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager
func (m *Manager) Start(ctx context.Context) error
func (m *Manager) Stop(ctx context.Context) error
func (m *Manager) GetBaseURL() string
```

**Plan:** Create new InstallerService that follows same pattern
```go
// internal/services/installer/ installer.go - NEW
type InstallerService struct {
    config  *ServiceConfig
    manager *knirvchain.Manager  // ← REUSE EXISTING
    logger  *zap.Logger
}

// Start/Stop follow exact same pattern as knirvchain.Manager
```

**Files:**
- New: `packages/KNIRVSERVER/backend/internal/services/installer/`

#### 3.3 Validation Chain Submission → EXTEND `internal/services/blockchain/validationchain/client.go`

**Existing:**
```go
// internal/services/blockchain/validationchain/client.go
type Client struct {
    baseURL string
    client *http.Client
}

func (c *Client) CommitValidationResult(req CommitValidationResultRequest) (string, error)
func (c *Client) CommitPolicy(req PolicyCommitRequest) (string, error)
func (c *Client) AnchorEvidencePack(req EvidenceAnchorRequest) (string, error)
```

**Plan:** Add new DVE URI methods:
```go
// internal/services/blockchain/validationchain/client.go - ADD
type DVEURIRecord struct {
    DVEID      string `json:"dve_id"`
    FullURI    string `json:"full_uri"`
    WalletAddr string `json:"wallet_address"`
    CreatedAt  int64  `json:"created_at"`
    TxHash     string `json:"tx_hash"`
}

func (c *Client) SubmitDVEURI(req DVEURIRecord) (string, error)
func (c *Client) GetDVEURI(dveID string) (*DVEURIRecord, error)
func (c *Client) GetDVEURIByWallet(walletAddr string) ([]*DVEURIRecord, error)
```

**Files:**
- Modify: `packages/KNIRVSERVER/backend/internal/services/blockchain/validationchain/client.go`

---

### Phase 4: DVE Browser Routing - Reuse Components

#### 4.1 DVE URI Registry → NEW (BuntDB)

**New:** No existing registry for DVEs - need new implementation
```go
// internal/services/dve_uri_registry.go - NEW
type DVEURI struct {
    ID         string `json:"id"`
    DVEID      string `json:"dve_id"`
    FullURI    string `json:"full_uri"`
    WalletAddr string `json:"wallet_address"`
    Endpoint  string `json:"endpoint"`
    Status    string `json:"status"`
    CreatedAt  int64  `json:"created_at"`
}

type DVEURIRegistry struct {
    db    *buntdb.BuntDB
    cache *ring.Ring
    chain *validationchain.Client
}
```

**Files:**
- New: `packages/KNIRVSERVER/backend/internal/services/dve_uri_registry.go`

#### 4.2 DVE Proxy Handler → REUSE `internal/runtime/viewport_proxy.go`

**Existing:**
```go
// internal/runtime/viewport_proxy.go
type ViewportProxyImpl struct {
    container  *UnifiedContainer
    targetURL  string
    proxy      *httputil.ReverseProxy
}

func (vp *ViewportProxyImpl) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

**Plan:** Create DVE-specific proxy:
```go
// internal/web/dve_proxy_handlers.go - NEW (reuse ReverseProxy)
type DVEProxyHandler struct {
    uriRegistry *dve_uri_registry.DVEURIRegistry
    viewport   *viewport_proxy.ViewportProxyImpl  // ← REUSE EXISTING
}
```

**Files:**
- New: `packages/KNIRVSERVER/backend/internal/web/dve_proxy_handlers.go`
- Modify: `packages/KNIRVSERVER/backend/internal/web/api_router.go` - Register routes

#### 4.3 Public DVE Page → NEW (Go HTML Templates)

**No existing templates** - need new:
```go
// internal/templates/dve/public_page.gohtml - NEW
{{ define "header" }}
<title>DVE {{ .DVEID }} - Public Validation Page</title>
{{ end }}

{{ define "content" }}
<div class="dve-public-page">
    <h1>DVE: {{ .DVEID }}</h1>
    <!-- Validation Records Search -->
    <!-- Validation Records Table -->
    <!-- Metrics Panel -->
</div>
{{ end }}
```

**Files:**
- NEW DIRECTORY: `packages/KNIRVSERVER/backend/templates/dve/`
- `public_page.gohtml`
- `validation_records.gohtml`
- `metrics_panel.gohtml`
- `search_form.gohtml`

#### 4.4 Interactive DVE (WebSocket) → EXTEND KNIRVARENA

**Existing in KNIRVARENA:**
```typescript
// networking/ArenaClient.ts
class ArenaClient {
    connect(): Promise<void>
    onMessage(callback: (msg) => void)
    send(type: string, payload: any): void
}
```

**Plan:** Add DVE-specific methods:
```typescript
// networking/DVEClient.ts - NEW (extend ArenaClient pattern)
class DVEClient {
    connect(dveId: string): WebSocket
    onValidationRecord(callback: (record) => void)
    sendValidationRequest(req: ValidationRequest): void
}
```

**Files:**
- New: `packages/KNIRVARENA/packages/ts_client_2/src/networking/DVEClient.ts`
- Modify: `packages/KNIRVARENA/packages/ts_client_2/src/App.tsx` - Add route

---

## Files Summary

### Existing Files to MODIFY

| Phase | File | Change |
|-------|------|--------|
| 1 | `packages/KNIRVGATEWAY/internal/tunnel/registry.go` | Add installer methods |
| 1 | `packages/KNIRVGATEWAY/internal/server/server.go` | Register routes |
| 3 | `packages/KNIRVSERVER/backend/internal/objects/dve.go` | Add InstallComplete fields |
| 3 | `packages/KNIRVSERVER/backend/internal/services/blockchain/validationchain/client.go` | Add DVE URI methods |
| 4 | `packages/KNIRVSERVER/backend/internal/web/api_router.go` | Register DVE proxy routes |
| 4 | `packages/KNIRVARENA/packages/ts_client_2/src/App.tsx` | Add DVE route |

### New Files to CREATE

| Phase | File |
|-------|------|
| 1 | `packages/KNIRVGATEWAY/internal/installer/stun_client.go` |
| 1 | `packages/KNIRVGATEWAY/internal/installer/dve_uri.go` |
| 2 | `packages/KNIRVORACLE/internal/oracle/routes/wallet.go` |
| 2 | `packages/KNIRVORACLE/internal/oracle/routes/dve_uri.go` |
| 3 | `packages/KNIRVSERVER/backend/internal/services/installer/installer.go` |
| 4 | `packages/KNIRVSERVER/backend/internal/services/dve_uri_registry.go` |
| 4 | `packages/KNIRVSERVER/backend/internal/web/dve_proxy_handlers.go` |
| 4 | `packages/KNIRVSERVER/backend/templates/dve/public_page.gohtml` |
| 4 | `packages/KNIRVSERVER/backend/templates/dve/validation_records.gohtml` |
| 4 | `packages/KNIRVSERVER/backend/templates/dve/metrics_panel.gohtml` |
| 4 | `packages/KNIRVSERVER/backend/templates/dve/search_form.gohtml` |
| 4 | `packages/KNIRVARENA/packages/ts_client_2/src/networking/DVEClient.ts` |

---

## Implementation Order

```
Phase 1: KNIRVGATEWAY
├── 1.1 extend registry.go (installer methods)
└── 1.2 add server.go routes + new installer files

Phase 2: KNIRVORACLE  
├── 2.1 new wallet.go (reuse crypto)
└── 2.2 new dve_uri.go (adapt pattern)

Phase 3: KNIRVSERVER
├── 3.1 extend objects/dve.go (InstallComplete)
├── 3.2 new installer/installer.go (reuse Manager pattern)
└── 3.3 extend validationchain/client.go

Phase 4: DVE Routing
├── 4.1 new dve_uri_registry.go
├── 4.2 new dve_proxy_handlers.go
├── 4.3 new templates/dve/
├── 4.4 new DVEClient.ts
└── 4.5 register routes
```