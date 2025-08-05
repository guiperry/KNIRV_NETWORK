

---

**Source**: KNIRVROOT/docs/pending_implementation_plans/Root_DNS_SSL_Implementation_Plan.md

# Root DNS and SSL Implementation Plan

This document outlines the implementation plan for adding SSL support to the reverse proxy via Certbot and Let's Encrypt, as well as configuring dynamic DNS management with Cloudflare for the Root node. This plan is designed to integrate with the future proxy utilization implementation described in `proxy_utilization_implementation.md`.

## Table of Contents

1. [Overview](#overview)
2. [SSL Implementation with Certbot](#ssl-implementation-with-certbot)
3. [Dynamic DNS Management with Cloudflare](#dynamic-dns-management-with-cloudflare)
4. [Graceful Shutdown and DNS Failover](#graceful-shutdown-and-dns-failover)
5. [Integration with Role-Based Proxy Implementation](#integration-with-role-based-proxy-implementation)
6. [Security Considerations](#security-considerations)
7. [Implementation Steps](#implementation-steps)

## Overview

The implementation will enable:

1. **SSL Support**: Automatic certificate issuance and renewal via Certbot and Let's Encrypt
2. **Dynamic DNS Management**: Automatic updating of DNS records in Cloudflare when the Root node's IP changes
3. **Graceful Failover**: Resetting DNS records to central server targets during Root node shutdown
4. **Role-Based Integration**: Seamless integration with the planned role-specific proxy optimizations

## SSL Implementation with Certbot

### Configuration Updates

1. Update the `ReverseProxyConfig` struct in `config/config.go` to include ACME challenge webroot, ensuring compatibility with the planned role-based proxy implementation:

```go
type ReverseProxyConfig struct {
    // Existing fields
    Enabled       bool     `mapstructure:"enabled" json:"enabled"`
    ListenAddr    string   `mapstructure:"listen_addr" json:"listen_addr"`
    CertFile      string   `mapstructure:"cert_file" json:"cert_file"`         // Path to SSL certificate
    KeyFile       string   `mapstructure:"key_file" json:"key_file"`           // Path to SSL key
    EnableCache   bool     `mapstructure:"enable_cache" json:"enable_cache"`   // Enable response caching
    CacheSize     int      `mapstructure:"cache_size" json:"cache_size"`       // Number of items to cache
    RemoteAPIURL  string   `mapstructure:"remote_api_url" json:"remote_api_url"` // For client-only mode
    TrustedIPs    []string `mapstructure:"trusted_ips" json:"trusted_ips"`     // List of trusted IPs
    
    // New SSL-related field
    ACMEChallengeWebroot string `mapstructure:"acme_challenge_webroot" json:"acme_challenge_webroot"` // Path for Certbot HTTP-01 challenge files
    
    // Role-specific settings (from proxy_utilization_implementation.md)
    RootSettings struct {
        EnableLoadBalancing bool `mapstructure:"enable_load_balancing" json:"enable_load_balancing"`
        EnableMetrics       bool `mapstructure:"enable_metrics" json:"enable_metrics"`
    } `mapstructure:"root_settings" json:"root_settings"`
    
    BootnodeSettings struct {
        EnableRateLimiting bool `mapstructure:"enable_rate_limiting" json:"enable_rate_limiting"`
        RequestsPerSecond  int  `mapstructure:"requests_per_second" json:"requests_per_second"`
    } `mapstructure:"bootnode_settings" json:"bootnode_settings"`
    
    PeerSettings struct {
        EnableCompression bool `mapstructure:"enable_compression" json:"enable_compression"`
    } `mapstructure:"dev_settings" json:"dev_settings"`
    
    ClientSettings struct {
        EnableOfflineMode bool `mapstructure:"enable_offline_mode" json:"enable_offline_mode"`
        OfflineCacheSize  int  `mapstructure:"offline_cache_size" json:"offline_cache_size"`
    } `mapstructure:"client_settings" json:"client_settings"`
}
```

### Enhanced GoReverseProxy Structure

Update the `GoReverseProxy` struct to support both SSL and role-based functionality:

```go
type GoReverseProxy struct {
    config          *config.ReverseProxyConfig
    frontendProxy   *httputil.ReverseProxy
    backendApiProxy *httputil.ReverseProxy
    frontendTarget  *url.URL
    backendTarget   *url.URL
    
    // Role-based fields
    nodeRole        config.Role
    cache           *lru.Cache
    offlineCache    *lru.Cache
    remoteAPIURL    *url.URL
    trustedIPs      map[string]bool
    clientOnly      bool
    
    // Load balancing
    backendServers  []*url.URL
    currentBackend  int
    backendMutex    sync.Mutex
    
    // Rate limiting
    rateLimiter     map[string]*rate.Limiter
    rateLimiterMutex sync.Mutex
    
    // Metrics
    metrics struct {
        requestCount      int64
        errorCount        int64
        avgResponseTime   float64
        lastRequestTime   time.Time
        mutex             sync.Mutex
    }
}
```

### Dynamic Certificate Reloading

Modify the `Start` method of `GoReverseProxy` to implement dynamic certificate reloading:

```go
func (p *GoReverseProxy) Start() error {
    server := &http.Server{
        Addr:    p.config.ListenAddr,
        Handler: p, // The GoReverseProxy itself implements http.Handler via ServeHTTP
    }

    if p.config.CertFile != "" && p.config.KeyFile != "" {
        log.Printf("ReverseProxy: Starting with HTTPS on %s", p.config.ListenAddr)
        // Use a TLS config with GetCertificate for dynamic reloading
        tlsConfig := &tls.Config{
            GetCertificate: p.getCertificate,
        }
        server.TLSConfig = tlsConfig
        return server.ListenAndServeTLS("", "") // CertFile and KeyFile are handled by GetCertificate
    } else {
        log.Printf("ReverseProxy: Starting with HTTP on %s", p.config.ListenAddr)
        return server.ListenAndServe()
    }
}

// getCertificate is called by the tls package whenever a new TLS connection is made.
// It loads the certificate and key from disk, allowing for dynamic updates if Certbot renews them.
func (p *GoReverseProxy) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
    // Load the certificate pair from the paths specified in the config
    cert, err := tls.LoadX509KeyPair(p.config.CertFile, p.config.KeyFile)
    if err != nil {
        log.Printf("ERROR: ReverseProxy: Failed to load key pair for TLS: %v", err)
        return nil, err
    }
    log.Printf("DEBUG: ReverseProxy: Successfully loaded TLS certificate for %s", hello.ServerName)
    return &cert, nil
}
```

### ACME Challenge Handling

Update the `ServeHTTP` method to handle ACME challenges while integrating with the role-based routing:

```go
func (p *GoReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()
    
    // Handle ACME HTTP-01 Challenge (highest priority)
    if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") && p.config.ACMEChallengeWebroot != "" {
        http.StripPrefix("/.well-known/acme-challenge/", 
            http.FileServer(http.Dir(p.config.ACMEChallengeWebroot))).ServeHTTP(w, r)
        log.Printf("DEBUG: ReverseProxy: Served ACME challenge request for %s", r.URL.Path)
        return
    }
    
    // Apply rate limiting for bootnode if enabled
    if p.nodeRole == config.RoleBootnode && p.rateLimiter != nil {
        if !p.checkRateLimit(r) {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
    }
    
    // Handle client-only mode restrictions
    if p.clientOnly {
        // Client-only mode logic from proxy_utilization_implementation.md
        // ...
    }
    
    // Root-specific routing for Node.js services
    if p.nodeRole == config.RoleRoot {
        // Route to Tunnel Registry service
        if strings.HasPrefix(r.URL.Path, "/tunnel/") {
            tunnelURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", p.config.NodeJSServices.TunnelRegistry.HTTPPort))
            proxy := httputil.NewSingleHostReverseProxy(tunnelURL)
            proxy.ServeHTTP(w, r)
            return
        }
        
        // Route to Payment Gateway service
        if strings.HasPrefix(r.URL.Path, "/payment/") {
            paymentURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", p.config.NodeJSServices.PaymentGateway.HTTPPort))
            proxy := httputil.NewSingleHostReverseProxy(paymentURL)
            proxy.ServeHTTP(w, r)
            return
        }
    }
    
    // Bootnode-specific routing for Tunnel Registry
    if p.nodeRole == config.RoleBootnode {
        // Bootnode-specific routing logic
        // ...
    }
    
    // Peer-specific routing for WebSocket tunnel client
    if p.nodeRole == config.RolePeer {
        // Peer-specific routing logic
        // ...
    }
    
    // Check for protected admin endpoints
    if strings.HasPrefix(r.URL.Path, "/admin/") || strings.HasPrefix(r.URL.Path, "/api/admin/") {
        clientIP := r.RemoteAddr
        if !p.isTrustedIP(clientIP) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
    }
    
    // Check cache for cacheable endpoints
    if p.cache != nil && (strings.HasPrefix(r.URL.Path, "/api/chain/info") || strings.HasPrefix(r.URL.Path, "/api/stats")) {
        if cachedResponse, found := p.cache.Get(r.URL.Path); found {
            w.Write(cachedResponse.([]byte))
            return
        }
    }
    
    // Standard routing logic
    log.Printf("ReverseProxy: Received request for %s %s", r.Method, r.URL.Path)
    
    // Route to backend API if path starts with /api/
    if strings.HasPrefix(r.URL.Path, "/api/") {
        log.Printf("ReverseProxy: Routing to backend API: %s", r.URL.Path)
        
        // Use load balancing if enabled for Root role
        if p.nodeRole == config.RoleRoot && len(p.backendServers) > 1 {
            backend := p.getNextBackend()
            proxy := httputil.NewSingleHostReverseProxy(backend)
            r.Host = backend.Host
            proxy.ServeHTTP(w, r)
        } else {
            // Standard backend routing
            r.Host = p.backendTarget.Host
            p.backendApiProxy.ServeHTTP(w, r)
        }
    } else {
        // Otherwise, route to frontend
        log.Printf("ReverseProxy: Routing to frontend: %s", r.URL.Path)
        r.Host = p.frontendTarget.Host
        p.frontendProxy.ServeHTTP(w, r)
    }
    
    // Track metrics if enabled for Root role
    if p.nodeRole == config.RoleRoot && p.config.RootSettings.EnableMetrics {
        p.trackRequest(startTime, getResponseStatus(w))
    }
}
```

### HTTP and HTTPS Server Setup

Implement both HTTP (for redirects and ACME challenges) and HTTPS servers:

```go
// In your main application setup
func main() {
    // ... load config and initialize components ...
    
    // Start HTTPS server
    go func() {
        log.Printf("Starting HTTPS server on %s", cfg.ReverseProxy.ListenAddr) // e.g., :443
        if err := proxy.Start(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("HTTPS server error: %v", err)
        }
    }()

    // Start HTTP server on port 80 for redirection and ACME challenges
    httpMux := http.NewServeMux()
    httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Handle ACME challenges
        if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") && cfg.ReverseProxy.ACMEChallengeWebroot != "" {
            fs := http.FileServer(http.Dir(cfg.ReverseProxy.ACMEChallengeWebroot))
            http.StripPrefix("/.well-known/acme-challenge/", fs).ServeHTTP(w, r)
            log.Printf("DEBUG: HTTP Server: Served ACME challenge request for %s", r.URL.Path)
            return
        }
        // Redirect other HTTP traffic to HTTPS
        targetURL := "https://" + r.Host + r.URL.Path
        if r.URL.RawQuery != "" {
            targetURL += "?" + r.URL.RawQuery
        }
        log.Printf("DEBUG: HTTP Server: Redirecting %s to %s", r.URL.String(), targetURL)
        http.Redirect(w, r, targetURL, http.StatusPermanentRedirect)
    })

    httpServer := &http.Server{
        Addr:    ":80", // Standard HTTP port
        Handler: httpMux,
    }
    log.Println("Starting HTTP server on :80 for ACME challenges and HTTPS redirection")
    if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("HTTP server error: %v", err)
    }
}
```

## Dynamic DNS Management with Cloudflare

### Configuration Updates

Add Cloudflare configuration to the `Config` struct in `config/config.go`:

```go
type CloudflareConfig struct {
    Enabled             bool              `json:"enabled" mapstructure:"enabled"`    // Enable/Disable Cloudflare integration
    APITokenEnvVar      string            `json:"-" mapstructure:"-"`                // Name of the ENV var holding the token
    ZoneID              string            `json:"zone_id" mapstructure:"zoneId"`     // Your Cloudflare Zone ID for agent.com
    Subdomains          []string          `json:"subdomains" mapstructure:"subdomains"` // e.g., ["rootchain", "tunnel", "payment"]
    CentralServerTargets map[string]string `json:"central_server_targets" mapstructure:"centralServerTargets"` // Failover targets
} `json:"cloudflare_config" mapstructure:"cloudflareConfig"`
```

### CloudflareManager Implementation

Create a new file `cloudflare_manager.go`:

```go
package main // Or your appropriate package

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "io/ioutil" // For Go 1.15 and earlier, use io/ioutil. For Go 1.16+, use io.
    "net" // For net.ParseIP
    "os" // For os.Getenv
    "strings"
    "time"

    "github.com/cloudflare/cloudflare-go"
    "KNIRVCHAIN_GO_ROOT_MCP_PROTO/config" // Adjust to your actual config import
)

// CloudflareManager handles DNS updates.
type CloudflareManager struct {
    cfg        *config.CloudflareConfig
    api        *cloudflare.API
    mainDomain string
}

// NewCloudflareManager creates a new manager.
func NewCloudflareManager(cfg *config.CloudflareConfig, domain string) (*CloudflareManager, error) {
    if !cfg.Enabled {
        return nil, fmt.Errorf("Cloudflare integration is not enabled in config")
    }
    
    apiToken := os.Getenv("CLOUDFLARE_API_TOKEN") // Default environment variable name
    if cfg.APITokenEnvVar != "" { // Allow overriding the env var name via config
        apiToken = os.Getenv(cfg.APITokenEnvVar)
    }
    
    if apiToken == "" || cfg.ZoneID == "" {
        return nil, fmt.Errorf("Cloudflare integration is misconfigured: missing API token or Zone ID")
    }
    
    api, err := cloudflare.NewWithAPIToken(apiToken)
    if err != nil {
        return nil, fmt.Errorf("failed to create Cloudflare API client: %w", err)
    }
    
    return &CloudflareManager{cfg: cfg, api: api, mainDomain: domain}, nil
}

// getPublicIPForCloudflare fetches the current public IP address.
func getPublicIPForCloudflare() (string, error) {
    resp, err := http.Get("https://api.ipify.org")
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    ip, err := ioutil.ReadAll(resp.Body) // For Go 1.16+, use io.ReadAll
    if err != nil {
        return "", err
    }
    return string(ip), nil
}

// UpdateDNSRecords ensures DNS records for configured subdomains point to the public IP.
func (cm *CloudflareManager) UpdateDNSRecords(ctx context.Context) error {
    publicIP, err := getPublicIPForCloudflare()
    if err != nil {
        return fmt.Errorf("failed to get public IP: %w", err)
    }
    log.Printf("[Cloudflare] Current public IP: %s", publicIP)

    for _, sub := range cm.cfg.Subdomains {
        fullDomain := fmt.Sprintf("%s.%s", sub, cm.mainDomain)
        log.Printf("[Cloudflare] Checking/Updating DNS for %s", fullDomain)

        // List existing A records for the subdomain
        records, _, err := cm.api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(cm.cfg.ZoneID), 
            cloudflare.ListDNSRecordsParams{Name: fullDomain, Type: "A"})
        if err != nil {
            log.Printf("[Cloudflare] Error listing DNS records for %s: %v", fullDomain, err)
            continue
        }

        found := false
        for _, r := range records {
            found = true
            if r.Content == publicIP {
                log.Printf("[Cloudflare] DNS record for %s is already up-to-date (IP: %s).", fullDomain, publicIP)
            } else {
                log.Printf("[Cloudflare] DNS record for %s (ID: %s) has old IP %s. Updating to %s.", 
                    fullDomain, r.ID, r.Content, publicIP)
                _, err := cm.api.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(cm.cfg.ZoneID), 
                    cloudflare.UpdateDNSRecordParams{
                        ID: r.ID, 
                        Content: publicIP, 
                        Type: "A", 
                        Name: fullDomain, 
                        Proxied: cloudflare.BoolPtr(false)
                    })
                if err != nil {
                    log.Printf("[Cloudflare] Error updating DNS record for %s: %v", fullDomain, err)
                } else {
                    log.Printf("[Cloudflare] Successfully updated DNS record for %s to IP %s.", fullDomain, publicIP)
                }
            }
            break // Assuming only one A record per subdomain
        }

        if !found {
            log.Printf("[Cloudflare] No A record found for %s. Creating new record pointing to %s.", fullDomain, publicIP)
            _, err := cm.api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(cm.cfg.ZoneID), 
                cloudflare.CreateDNSRecordParams{
                    Name:    fullDomain,
                    Type:    "A",
                    Content: publicIP,
                    TTL:     120, // Low TTL for potentially dynamic IPs, or 1 for "Auto"
                    Proxied: cloudflare.BoolPtr(false),
                })
            if err != nil {
                log.Printf("[Cloudflare] Error creating DNS record for %s: %v", fullDomain, err)
            } else {
                log.Printf("[Cloudflare] Successfully created DNS record for %s pointing to IP %s.", fullDomain, publicIP)
            }
        }
    }
    return nil
}

// ResetDNSRecordsToCentral resets the DNS records for managed subdomains to their central/fallback targets.
func (cm *CloudflareManager) ResetDNSRecordsToCentral(ctx context.Context) error {
    if len(cm.cfg.CentralServerTargets) == 0 {
        log.Println("[Cloudflare] No central server targets configured. Skipping DNS reset.")
        return nil
    }
    log.Println("[Cloudflare] Resetting DNS records to central server targets...")

    for subdomainPrefix, targetValue := range cm.cfg.CentralServerTargets {
        fullDomain := fmt.Sprintf("%s.%s", subdomainPrefix, cm.mainDomain)
        targetType := "A"
        if net.ParseIP(targetValue) == nil { // If it's not a valid IP, assume it's a hostname for CNAME
            targetType = "CNAME"
        }

        log.Printf("[Cloudflare] Processing reset for %s -> %s (Type: %s)", fullDomain, targetValue, targetType)

        // List all existing records for the domain to manage conflicts (e.g., A vs CNAME)
        existingRecords, _, err := cm.api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(cm.cfg.ZoneID), 
            cloudflare.ListDNSRecordsParams{Name: fullDomain})
        if err != nil {
            log.Printf("[Cloudflare] Error listing DNS records for %s during reset: %v", fullDomain, err)
            continue // Move to the next subdomain
        }

        var recordToUpdateID string
        var needsCreation = true

        for _, r := range existingRecords {
            if r.Type == targetType {
                if r.Content == targetValue {
                    log.Printf("[Cloudflare] DNS record for %s is already correctly set to central target %s.", 
                        fullDomain, targetValue)
                    needsCreation = false // Already exists and is correct
                    recordToUpdateID = "" // No update needed
                    break
                }
                recordToUpdateID = r.ID // Found a record of the correct type, will update it
                needsCreation = false
                // Don't break yet, in case we need to delete other conflicting types
            } else {
                // Found a record of a conflicting type (e.g., an A record exists but we want to set a CNAME)
                log.Printf("[Cloudflare] Deleting conflicting DNS record type %s for %s (ID: %s) before setting %s record.", 
                    r.Type, fullDomain, r.ID, targetType)
                err := cm.api.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(cm.cfg.ZoneID), r.ID)
                if err != nil {
                    log.Printf("[Cloudflare] Error deleting conflicting DNS record %s for %s: %v", 
                        r.ID, fullDomain, err)
                }
            }
        }

        if needsCreation {
            log.Printf("[Cloudflare] Creating new DNS record for %s -> %s (Type: %s)", fullDomain, targetValue, targetType)
            _, err := cm.api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(cm.cfg.ZoneID), 
                cloudflare.CreateDNSRecordParams{
                    Name:    fullDomain,
                    Type:    targetType,
                    Content: targetValue,
                    TTL:     120, // Or 1 for "Auto"
                    Proxied: cloudflare.BoolPtr(false),
                })
            if err != nil {
                log.Printf("[Cloudflare] Error creating DNS record for %s during reset: %v", fullDomain, err)
            } else {
                log.Printf("[Cloudflare] Successfully created DNS record for %s pointing to central target %s.", 
                    fullDomain, targetValue)
            }
        } else if recordToUpdateID != "" { // Needs update, and we have an ID for a record of the correct type
            log.Printf("[Cloudflare] Updating DNS record %s for %s -> %s (Type: %s)", 
                recordToUpdateID, fullDomain, targetValue, targetType)
            _, err := cm.api.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(cm.cfg.ZoneID), 
                cloudflare.UpdateDNSRecordParams{
                    ID:      recordToUpdateID,
                    Type:    targetType,
                    Name:    fullDomain,
                    Content: targetValue,
                    Proxied: cloudflare.BoolPtr(false),
                })
            if err != nil {
                log.Printf("[Cloudflare] Error updating DNS record %s for %s during reset: %v", 
                    recordToUpdateID, fullDomain, err)
            } else {
                log.Printf("[Cloudflare] Successfully updated DNS record for %s to central target %s.", 
                    fullDomain, targetValue)
            }
        }
    }
    return nil
}

// RunPeriodicUpdates starts a loop to periodically update DNS records.
func (cm *CloudflareManager) RunPeriodicUpdates(ctx context.Context, interval time.Duration) {
    log.Printf("[Cloudflare] Starting periodic DNS updates every %v", interval)
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    // Initial update
    if err := cm.UpdateDNSRecords(ctx); err != nil {
        log.Printf("[Cloudflare] Initial DNS update failed: %v", err)
    }

    for {
        select {
        case <-ticker.C:
            if err := cm.UpdateDNSRecords(ctx); err != nil {
                log.Printf("[Cloudflare] Periodic DNS update failed: %v", err)
            }
        case <-ctx.Done():
            log.Println("[Cloudflare] Stopping periodic DNS updates.")
            return
        }
    }
}
```

### Integration into Main Application

Update `main.go` to initialize and use the CloudflareManager:

```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    
    "github.com/joho/godotenv"
    // Other imports
)

var (
    mainCtx        context.Context
    mainCancel     context.CancelFunc
    nodeManager    *NodeManager
    cloudflareMgr  *CloudflareManager // Added for DNS management
)

func main() {
    // Load .env file if it exists
    if err := godotenv.Load(); err != nil {
        if !os.IsNotExist(err) {
            log.Printf("Warning: Error loading .env file: %v", err)
        } else {
            log.Println("No .env file found, relying on system environment variables.")
        }
    }
    
    // Initialize context and other components
    mainCtx, mainCancel = context.WithCancel(context.Background())
    defer mainCancel()
    
    // Load configuration
    cfg := loadAppConfig() // Your config loading logic
    
    // Determine node role
    nodeRole := determineNodeRole() // Your role determination logic
    
    // Initialize Cloudflare Manager for Root node
    if nodeRole == config.RoleRoot && cfg.Cloudflare.Enabled {
        var err error
        cloudflareMgr, err = NewCloudflareManager(&cfg.Cloudflare, "agent.com") // Assuming "agent.com" is your main domain
        if err != nil {
            log.Printf("[ERROR] Failed to initialize Cloudflare Manager: %v", err)
        } else {
            mainWg.Add(1)
            go func() {
                defer mainWg.Done()
                if cloudflareMgr != nil {
                    cloudflareMgr.RunPeriodicUpdates(mainCtx, 10*time.Minute) // Update every 10 minutes
                }
            }()
            log.Println("[INFO] Cloudflare DNS Manager (for dynamic IP) started for Root node.")
        }
    }
    
    // Initialize and start other components
    // ...
    
    // Wait for shutdown signal
    // ...
    
    // Graceful shutdown
    log.Println("Starting graceful shutdown...")
    mainCancel() // Signal all goroutines to stop
    
    // Reset Cloudflare DNS records if this is a Root node
    if nodeRole == config.RoleRoot && cfg.Cloudflare.Enabled && cloudflareMgr != nil {
        log.Println("[INFO] Root node shutting down. Attempting to reset DNS records to central targets...")
        if err := cloudflareMgr.ResetDNSRecordsToCentral(context.Background()); err != nil {
            log.Printf("[ERROR] Failed to reset Cloudflare DNS records: %v", err)
        }
    }
    
    // Wait for all goroutines to finish
    mainWg.Wait()
    log.Println("Graceful shutdown completed.")
}
```

## Graceful Shutdown and DNS Failover

The implementation includes a graceful shutdown process that:

1. Cancels the main context to stop all goroutines
2. Resets DNS records to point to central server targets
3. Waits for all goroutines to finish before exiting

## Integration with Role-Based Proxy Implementation

This SSL and DNS implementation is designed to integrate seamlessly with the role-based proxy implementation described in `proxy_utilization_implementation.md`. The integration focuses on:

### 1. Role-Specific SSL Configuration

Each node role (Root, Bootnode, Peer, Client-only) can have different SSL requirements:

- **Root Nodes**: Full SSL implementation with dynamic certificate reloading, supporting multiple subdomains
- **Bootnodes**: SSL implementation with rate limiting for public endpoints
- **Peer Nodes**: Optional SSL based on deployment requirements
- **Client-Only Nodes**: Typically minimal SSL requirements, often relying on the Root/Bootnode for secure connections

### 2. Initialization Process

The role-based proxy initialization will include SSL setup:

```go
func initReverseProxy(cfg *config.Config, role config.Role) (*GoReverseProxy, error) {
    if !cfg.ReverseProxy.Enabled {
        return nil, nil
    }
    
    // Create base proxy
    proxy, err := NewGoReverseProxy(&cfg.ReverseProxy, frontendURL, backendURL)
    if err != nil {
        return nil, err
    }
    
    // Set node role
    proxy.nodeRole = role
    
    // Apply role-specific configurations
    switch role {
    case config.RoleRoot:
        configureRootProxy(proxy, cfg)
        // For Root nodes, ensure SSL and Cloudflare DNS are properly configured
        if cfg.ReverseProxy.CertFile != "" && cfg.ReverseProxy.KeyFile != "" {
            log.Printf("Root node: SSL configuration detected, certificates will be loaded from %s and %s", 
                cfg.ReverseProxy.CertFile, cfg.ReverseProxy.KeyFile)
        }
    case config.RoleBootnode:
        configureBootnodeProxy(proxy, cfg)
    case config.RolePeer:
        configurePeerProxy(proxy, cfg)
    case config.RoleClient:
        configureClientProxy(proxy, cfg)
    }
    
    return proxy, nil
}
```

### 3. Root-Specific Configuration

The Root node configuration will include additional SSL and DNS-related settings:

```go
func configureRootProxy(proxy *GoReverseProxy, cfg *config.Config) {
    // Initialize caching if enabled
    if cfg.ReverseProxy.EnableCache {
        proxy.cache, _ = lru.New(cfg.ReverseProxy.CacheSize)
    }
    
    // Configure load balancing if enabled
    if cfg.ReverseProxy.RootSettings.EnableLoadBalancing {
        proxy.backendServers = []*url.URL{proxy.backendTarget}
        proxy.currentBackend = 0
    }
    
    // Configure metrics tracking
    if cfg.ReverseProxy.RootSettings.EnableMetrics {
        proxy.metrics.lastRequestTime = time.Now()
    }
    
    // Ensure ACME challenge directory exists
    if cfg.ReverseProxy.ACMEChallengeWebroot != "" {
        if _, err := os.Stat(cfg.ReverseProxy.ACMEChallengeWebroot); os.IsNotExist(err) {
            log.Printf("Creating ACME challenge directory: %s", cfg.ReverseProxy.ACMEChallengeWebroot)
            if err := os.MkdirAll(cfg.ReverseProxy.ACMEChallengeWebroot, 0755); err != nil {
                log.Printf("WARNING: Failed to create ACME challenge directory: %v", err)
            }
        }
    }
}
```

### 4. Default Configuration Examples

The default configuration for each node role will include appropriate SSL settings:

#### Root Node Configuration

```json
{
  "reverse_proxy": {
    "enabled": true,
    "listen_addr": ":443",
    "cert_file": "/etc/letsencrypt/live/rootchain.agent.com/fullchain.pem",
    "key_file": "/etc/letsencrypt/live/rootchain.agent.com/privkey.pem",
    "acme_challenge_webroot": "/var/www/letsencrypt",
    "enable_cache": true,
    "cache_size": 1000,
    "trusted_ips": ["192.168.1.100", "10.0.0.1"],
    "root_settings": {
      "enable_load_balancing": false,
      "enable_metrics": true
    }
  },
  "cloudflare_config": {
    "enabled": true,
    "zone_id": "YOUR_CLOUDFLARE_ZONE_ID",
    "subdomains": ["rootchain", "tunnel", "payment"],
    "central_server_targets": {
      "rootchain": "central.rootchain.agent.com",
      "tunnel": "central.tunnel.agent.com",
      "payment": "198.51.100.10"
    }
  }
}
```

## Security Considerations

1. **API Token Security**: The Cloudflare API token is stored in an environment variable, not in the configuration file
2. **Certificate Permissions**: The Go application needs read access to Certbot's certificate files
3. **ACME Challenge Security**: Proper path handling for serving ACME challenge files

## Implementation Steps

1. **Install Dependencies**:
   ```bash
   go get github.com/cloudflare/cloudflare-go
   go get github.com/joho/godotenv
   go get github.com/hashicorp/golang-lru  # For caching in role-based implementation
   go get golang.org/x/time/rate           # For rate limiting in role-based implementation
   ```

2. **Install Certbot**:
   ```bash
   # For Ubuntu/Debian
   sudo apt-get update
   sudo apt-get install certbot
   
   # For DNS-01 challenge with Cloudflare (recommended for dynamic IPs)
   sudo apt-get install python3-certbot-dns-cloudflare
   ```

3. **Configure Certbot**:
   ```bash
   # For HTTP-01 challenge
   sudo certbot certonly --webroot -w /var/www/letsencrypt -d rootchain.agent.com -d tunnel.agent.com -d payment.agent.com
   
   # For DNS-01 challenge with Cloudflare (recommended)
   # First, create a Cloudflare credentials file
   sudo mkdir -p /etc/letsencrypt/cloudflare
   sudo nano /etc/letsencrypt/cloudflare/credentials.ini
   # Add: dns_cloudflare_api_token = your_cloudflare_api_token
   sudo chmod 600 /etc/letsencrypt/cloudflare/credentials.ini
   
   # Then run Certbot
   sudo certbot certonly --dns-cloudflare --dns-cloudflare-credentials /etc/letsencrypt/cloudflare/credentials.ini -d rootchain.agent.com -d tunnel.agent.com -d payment.agent.com
   ```

4. **Create .env File**:
   ```bash
   echo "CLOUDFLARE_API_TOKEN=your_cloudflare_api_token" > .env
   echo ".env" >> .gitignore
   ```

5. **Update Configuration**:
   ```json
   {
     "reverse_proxy": {
       "enabled": true,
       "listen_addr": ":443",
       "cert_file": "/etc/letsencrypt/live/rootchain.agent.com/fullchain.pem",
       "key_file": "/etc/letsencrypt/live/rootchain.agent.com/privkey.pem",
       "acme_challenge_webroot": "/var/www/letsencrypt",
       "enable_cache": true,
       "cache_size": 1000,
       "trusted_ips": ["192.168.1.100", "10.0.0.1"],
       "root_settings": {
         "enable_load_balancing": false,
         "enable_metrics": true
       }
     },
     "cloudflare_config": {
       "enabled": true,
       "zone_id": "YOUR_CLOUDFLARE_ZONE_ID_FOR_agent.COM",
       "subdomains": ["rootchain", "tunnel", "payment"],
       "central_server_targets": {
         "rootchain": "central.rootchain.agent.com",
         "tunnel": "central.tunnel.agent.com",
         "payment": "198.51.100.10"
       }
     }
   }
   ```

6. **Implement Core SSL and DNS Changes**:
   - Update `config/config.go` with new configuration structs for SSL and Cloudflare
   - Create `cloudflare_manager.go` with the CloudflareManager implementation
   - Modify `proxy/reverse_proxy.go` to support dynamic certificate reloading
   - Update `main.go` to initialize and use the CloudflareManager

7. **Integrate with Role-Based Proxy Implementation**:
   - Enhance the `GoReverseProxy` struct with role-specific fields
   - Implement the role-specific configuration functions
   - Update the `ServeHTTP` method to handle ACME challenges with priority
   - Implement helper methods for the role-based functionality

8. **Implement HTTP and HTTPS Server Setup**:
   - Set up the HTTPS server with dynamic certificate reloading
   - Set up the HTTP server for redirects and ACME challenges
   - Ensure proper shutdown handling for both servers

9. **Test Implementation**:
   - Verify SSL certificate loading and renewal
   - Test DNS record updates when the IP changes
   - Test DNS failover during graceful shutdown
   - Test role-specific functionality (caching, rate limiting, etc.)
   - Test ACME challenge handling

10. **Set Up Automatic Renewal**:
    Certbot typically sets up a cron job or systemd timer for automatic renewal during installation. Verify with:
    ```bash
    sudo certbot renew --dry-run
    ```

11. **Documentation and Knowledge Transfer**:
    - Document the SSL and DNS configuration for each node role
    - Create a troubleshooting guide for common SSL and DNS issues
    - Provide examples of role-specific configurations

This implementation plan provides a comprehensive approach to adding SSL support and dynamic DNS management to the KNIRVCHAIN system, ensuring high availability, security, and seamless integration with the role-based proxy implementation.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
