                                                                                                                                                                
  KNIRVBRIDGE → Browser DVE: Implementation Plan                  
                                                                                                                                                                         
  Objective                                                       
                                                                                                                                                                         
  Transform each KNIRVBRIDGE wallet into a self-sovereign, agentic DVE node that lives in the user's browser extension. The wallet address becomes the DVE's identity;   
  NFT badges held by the wallet define its capabilities and policy set; the extension's background service worker is the runtime.                                        
                                                                                                                                                                         
  ---                                                             
  Architecture Overview

  KNIRVSERVER DVE Fleet
    ├── Hardware DVE nodes  (sgx / sev-snp / tdx)                                                                                                                        
    ├── Software DVE nodes  (existing "software" TEE)                                                                                                                    
    └── Browser DVE nodes   ← NEW (TEE type: "browser-extension")                                                                                                        
             ↕  WebSocket relay                                                                                                                                          
        KNIRVBRIDGE (Chrome extension)                                                                                                                                   
          ├── background.ts  ← DVE runtime + WS client                                                                                                                   
          ├── DVEIdentityService  ← derives node ID from wallet key                                                                                                      
          ├── DVERegistryService  ← registers/heartbeats with KNIRVSERVER                                                                                                
          ├── BadgeManager        ← reads NFT badges → capabilities                                                                                                      
          ├── ValidationRuntime   ← executes tasks in Web Worker sandbox                                                                                                 
          └── Popup DVE UI        ← status, badge inventory, task feed                                                                                                   
                                                                                                                                                                         
  Core invariant: DVE node ID = sha256(walletAddress + extensionID). The DVEURI becomes knirv://dve/{walletAddress}/browser. Both are deterministic per user, stable     
  across sessions.                                                                                                                                                       
                                                                                                                                                                         
  ---                                                             
  Phase 1 — Browser DVE Identity & Registration
                                               
  1.1 KNIRVSERVER: New TEE type + browser DVE fields
                                                                                                                                                                         
  File: packages/KNIRVSERVER/backend/internal/objects/dve.go
                                                                                                                                                                         
  Add to DVENode:                                                                                                                                                        
  // Browser DVE fields
  ExtensionID     string `json:"extension_id,omitempty"`     // Chrome extension fingerprint                                                                             
  BrowserVersion  string `json:"browser_version,omitempty"`                                 
  WSConnectionID  string `json:"ws_connection_id,omitempty"` // active WS session                                                                                        
  BadgeNFTIDs     []string `json:"badge_nft_ids,omitempty"`  // held badge NFT token IDs
                                                                                                                                                                         
  The TEEType field already accepts strings — add "browser-extension" as a valid value. No schema change needed; update validation logic in dve_manager.go:RegisterNode  
  to allow it.                                                                                                                                                           
                                                                                                                                                                         
  File: packages/KNIRVSERVER/backend/internal/services/dvemanager/dve_manager.go                                                                                         
                                                                  
  Update NodeFilter.Matches to treat "browser-extension" as a valid TEE type. Update the load balancer to give browser nodes a lower priority weight (appropriate for    
  non-hardware tasks).
                                                                                                                                                                         
  1.2 KNIRVBRIDGE: DVE Identity Service                           

  New file: packages/KNIRVBRIDGE/src/services/dve/dve-identity.ts                                                                                                        
  
  export interface DVEIdentity {                                                                                                                                         
    nodeID: string;          // sha256(walletAddress + extensionID)
    dveURI: string;          // "knirv://dve/{walletAddress}/browser"                                                                                                    
    walletAddress: string;                                                                                                                                               
    extensionID: string;                                                                                                                                                 
  }                                                                                                                                                                      
                                                                  
  export async function deriveDVEIdentity(walletAddress: string): Promise<DVEIdentity>                                                                                   
  
  Uses crypto.subtle.digest available in extension service worker context. Deterministic — same wallet always produces the same DVE node ID.                             
                                                                  
  1.3 KNIRVBRIDGE: DVE Registry Service                                                                                                                                  
                                                                  
  New file: packages/KNIRVBRIDGE/src/services/dve/dve-registry.ts

  Responsibilities:
  - register(identity, capabilities, badgeNFTIDs) → POST /api/dve/nodes with is_remote: true, tee_type: "browser-extension", wallet_address, badge_nft_ids[]
  - heartbeat(nodeID) → PUT /api/dve/nodes/{id}/heartbeat every 45 seconds                                                                                               
  - deregister(nodeID) → PUT /api/dve/nodes/{id}/status to "offline" on wallet lock
  - Reuses the existing repositories/common/request pattern                                                                                                              
                                                                                                                                                                         
  1.4 KNIRVBRIDGE: Background Script Update                                                                                                                              
                                                                                                                                                                         
  File: packages/KNIRVBRIDGE/src/background.ts                                                                                                                           
                                                                  
  On wallet unlock (existing chrome.runtime.onConnect listener): trigger DVE registration + WebSocket connect. On wallet lock / inactivity expire: mark node offline,    
  close WS.                                                       
                                                                                                                                                                         
  Add alarm key DVE_HEARTBEAT (45s interval) to existing SCHEDULE_ALARMS in src/common/constants/alarm-key.constant.ts.                                                  
  
  ---                                                                                                                                                                    
  Phase 2 — WebSocket Communication Channel                       
                                           
  The extension cannot expose an HTTP server. The pattern used here mirrors DVEClient (existing dve_client.go) but inverted: the extension opens a persistent WebSocket
  to KNIRVSERVER; the server pushes task assignments down.                                                                                                               
  
  2.1 KNIRVSERVER: Browser DVE WebSocket Handler                                                                                                                         
                                                                  
  New file: packages/KNIRVSERVER/backend/internal/web/dve_browser_ws_handler.go                                                                                          
                                                                  
  type BrowserDVEHub struct {                                                                                                                                            
      mu          sync.RWMutex
      connections map[string]*BrowserDVEConn  // walletAddress → conn                                                                                                    
      dveManager  *dvemanager.DVEManager                                                                                                                                 
  }                                                                                                                                                                                                                                                                                                                                            
  // WS message types:                                                                                                                                                   
  // Server → Client: "task_assigned", "policy_sync", "badge_refresh", "heartbeat_ack"
  // Client → Server: "task_result", "heartbeat", "capability_update", "badge_sync"                                                                                                                                                                                                                                                          
  Register route: GET /api/dve/browser/ws — authenticates via wallet JWT (existing AuthMiddleware). On connect: sets DVENode.WSConnectionID. On disconnect: marks node   
  offline.                                                                                                                                                                                                                                                                                                                                  
  Update packages/KNIRVSERVER/backend/internal/web/api_router.go to register the WS handler.                                                                             
  2.2 KNIRVBRIDGE: WebSocket Client                                                                                                                                                                                               
  New file: packages/KNIRVBRIDGE/src/services/dve/dve-ws-client.ts                                                                                                                                                                      
  class DVEWebSocketClient {                                                                                                                                             
    connect(serverURL: string, authToken: string): void           
    onTaskAssigned(handler: (task: ValidationTask) => void): void                                                                                                        
    sendTaskResult(result: ValidationResult): void
    sendHeartbeat(): void                                                                                                                                                
    onPolicySynced(handler: (policies: PolicyRule[]) => void): void
  }                                                                                                                                                                                                                                      
  Runs in background service worker (MV3 service workers support WebSocket). Reconnects with exponential backoff. Heartbeat keeps the WS alive and the extension's       
  service worker from being terminated by Chrome.                                                                                                                                                                                  
  ---                                                             
  Phase 3 — NFT Badge → DVE Capability Mapping
                                                                                                                                                                         
  3.1 Badge NFT Schema
                                                                                                                                                                         
  Badge NFTs require a recognized metadata structure. Extend the existing NFT display (already in src/components/pages/nft/ and src/hooks/nft/) with DVE-specific        
  metadata parsing.
                                                                                                                                                                         
  Badge metadata schema (added to NFT metadata field on-chain):                                                                                                          
  {
    "dve_badge": true,                                                                                                                                                   
    "dve_capabilities": ["validation", "reasoning", "policy-check"],
    "supported_tags": ["skillnode", "error-resolution"],                                                                                                                 
    "attached_policies": ["policy_id_abc"],                                                                                                                              
    "stake_requirement": 1000,                                                                                                                                           
    "trust_tier": "standard"                                                                                                                                             
  }                                                               
                                                                                                                                                                         
  3.2 KNIRVBRIDGE: Badge Manager                                  
                                                                                                                                                                         
  New file: packages/KNIRVBRIDGE/src/services/dve/dve-badge-manager.ts                                                                                                   
  
  export interface DVEBadge {                                                                                                                                            
    nftTokenID: string;                                           
    collectionPath: string;
    capabilities: string[];
    supportedTags: string[];                                                                                                                                             
    attachedPolicies: string[];
    stakeRequirement: number;                                                                                                                                            
    trustTier: "standard" | "verified" | "root";                  
  }

  export class DVEBadgeManager {                                                                                                                                         
    getBadgesFromWallet(walletAddress: string): Promise<DVEBadge[]>
    computeAggregateCapabilities(badges: DVEBadge[]): string[]                                                                                                           
    computeAggregateStake(badges: DVEBadge[]): number             
    watchBadgeChanges(callback: (badges: DVEBadge[]) => void): void                                                                                                      
  }                                                               
                                                                                                                                                                         
  Queries the existing NFT repository (already in src/repositories/wallet/) and filters for dve_badge: true metadata.                                                    
  
  3.3 KNIRVBRIDGE: Badge Hooks                                                                                                                                           
                                                                  
  New file: packages/KNIRVBRIDGE/src/hooks/dve/use-dve-badges.ts                                                                                                         
                                                                  
  React hook that wraps DVEBadgeManager. Triggers capability re-sync to KNIRVSERVER when badges change (NFT transfer in or out).                                         
  
  ---                                                                                                                                                                    
  Phase 4 — Validation Task Execution                             
                                     
  4.1 Browser Validation Runtime
                                                                                                                                                                         
  New file: packages/KNIRVBRIDGE/src/services/dve/validation-runtime.ts
                                                                                                                                                                         
  Executes tasks inside a Web Worker (isolated JS sandbox — no DOM, no storage access):                                                                                  
  
  export type BrowserTaskType =                                                                                                                                          
    | "policy-check"       // Evaluate input against badge policy rules
    | "signature-verify"   // Verify transaction/message signature                                                                                                       
    | "reasoning-simple"   // Text-based reasoning (no ML, uses heuristics)                                                                                              
    | "skill-lint"         // Lint skill.md files against schema                                                                                                         
                                                                                                                                                                         
  export class ValidationRuntime {                                                                                                                                       
    executeTask(task: ValidationTask): Promise<ValidationResult>                                                                                                         
  }                                                               

  Constraints: Browser DVEs are not ML inference nodes. They handle lightweight validation tasks suited to a JavaScript runtime. Heavy tasks ("base_llm", "ml-inference")
   are declined with "insufficient_capabilities" error, prompting KNIRVSERVER to requeue to a hardware node.
                                                                                                                                                                         
  4.2 Result Signing                                              

  ValidationResult.Signature = wallet private key signs sha256(taskID + score + JSON.stringify(results)). This maps to the existing Signature field in                   
  objects/dve.go:ValidationResult. The TEEAttestation field = the wallet's public key (browser nodes lack hardware attestation; chain verification substitutes).
                                                                                                                                                                         
  4.3 Task Message Flow                                           

  KNIRVSERVER DVEManager.AllocateTask()
    → BrowserDVEHub.DispatchTask(walletAddress, task)                                                                                                                    
      → WebSocket push to extension                                                                                                                                      
        → background.ts receives "task_assigned"                                                                                                                         
          → ValidationRuntime.executeTask(task)                                                                                                                          
            → background.ts sends "task_result" with signed ValidationResult                                                                                             
              → BrowserDVEHub receives result                                                                                                                            
                → DVEManager stores result, updates node reputation
                                                                                                                                                                         
  ---                                                             
  Phase 5 — DVE UI in the Extension Popup                                                                                                                                
                                                                                                                                                                         
  5.1 DVE Status Page
                                                                                                                                                                         
  New file: packages/KNIRVBRIDGE/src/pages/popup/dve/dve-status.tsx                                                                                                      
  
  - DVE node ID (truncated + copy button — reuse CopyButton atom)                                                                                                        
  - Online/offline status dot (reuse StatusDot atom)              
  - DVEURI display                                                                                                                                                       
  - Active badge count + total capabilities                       
  - Task counters: pending / completed / failed                                                                                                                          
  - NRN stake amount with reputation score                        
                                                                                                                                                                         
  5.2 Badge Inventory Page                                        
                                                                                                                                                                         
  New file: packages/KNIRVBRIDGE/src/pages/popup/dve/badge-inventory.tsx                                                                                                 
  
  - Grid of held NFT badge cards (extend existing NftCollectionAssetCard)                                                                                                
  - Each card shows: badge name, capability chips, policy count, trust tier
  - Toggle to activate/deactivate a badge (deactivated badge excluded from capability sync)                                                                              
  - "Mint Badge" button → opens KNIRVSERVER badge lab in a new tab                                                                                                       
                                                                                                                                                                         
  5.3 Router Integration                                                                                                                                                 
                                                                                                                                                                         
  Update packages/KNIRVBRIDGE/src/router/popup/navigation/ to add a "DVE" tab/route alongside the existing Wallet / NFT / History tabs.                                  
  
  5.4 Settings Integration                                                                                                                                               
                                                                  
  Update packages/KNIRVBRIDGE/src/pages/popup/certify/settings.tsx to add:                                                                                               
  - DVE mode toggle (enable/disable extension acting as a DVE node)
  - Auto-accept tasks below NRN threshold toggle                                                                                                                         
  - Min stake warning if wallet NRN < badge.stakeRequirement      
                                                                                                                                                                         
  ---                                                                                                                                                                    
  Phase 6 — On-Chain DVE Identity NFT                                                                                                                                    
                                                                                                                                                                         
  6.1 DVE Identity Minting                                                                                                                                               
                                                                                                                                                                         
  When DVE mode is first enabled, KNIRVBRIDGE prompts the user to mint a DVE Identity NFT on KNIRVCHAIN. This is the on-chain anchor for the browser DVE.                
                                                                                                                                                                         
  Token metadata:                                                                                                                                                        
  {                                                               
    "type": "dve_identity",
    "dve_id": "{nodeID}",  
    "dve_uri": "knirv://dve/{walletAddress}/browser",
    "tee_type": "browser-extension",                                                                                                                                     
    "initial_capabilities": ["policy-check", "signature-verify"],
    "badge_refs": [],                                                                                                                                                    
    "created_at": "<unix_ts>"                                                                                                                                            
  }                          
                                                                                                                                                                         
  The mint uses the existing knirv_sendTransaction window provider API already built into KNIRVBRIDGE — no new signing infrastructure required.
                                                                                                                                                                         
  6.2 KNIRVSHELL Command                                                                                                                                                 
                                                                                                                                                                         
  New file: packages/KNIRVSHELL/cmd/browser_dve.go                                                                                                                       
                                                                  
  knirvshell dve browser-register --wallet g1abc... --server https://server:8084                                                                                         
  knirvshell dve browser-list                                                                                                                                            
  knirvshell dve browser-status --dve-id {id}                                                                                                                            
                                                                                                                                                                         
  Provides a CLI path for headless/CI registration of browser DVEs (useful for testing and for power users managing multiple wallets).                                   
                                                                  
  6.3 Chain-Native Discovery Integration                                                                                                                                 
                                                                  
  Update packages/KNIRVSERVER/backend/internal/services/dvemanager/chain_native_discovery.go to recognize DVE Identity NFTs of type "dve_identity" and auto-register them
   as "browser-extension" TEE nodes. This means browser DVEs appear in the DVE fleet without manual KNIRVSERVER admin action — they surface automatically when the chain
  is synced.                                                                                                                                                             
                                                                  
  ---
  Phase 7 — Security Model
                                                                                                                                                                         
  7.1 Sandboxed Task Execution
                                                                                                                                                                         
  Validation tasks run in a Worker spawned by the service worker. The worker is terminated after task completion (no persistent state). Task payload size limit: 100 KB. 
  Execution timeout: 10 seconds. No eval() or Function() constructor — tasks must be declarative (JSON policy rules, not arbitrary code).
                                                                                                                                                                         
  7.2 Stake Gating                                                                                                                                                       
  
  On registration, KNIRVSERVER checks wallet NRN balance via existing GetAccountBalance on the chain client. Nodes with balance below the aggregate stakeRequirement of  
  their badges are registered as "offline" and receive no tasks until stake is added. This prevents sybil abuse of browser DVE slots.
                                                                                                                                                                         
  7.3 Trust Tier Separation                                       

  LoadBalancer.SelectNode (in dve_manager.go) already prefers nodes by ReputationScore. Add a trustTier weight multiplier:                                               
  
  ┌─────────────────────┬────────────────────────────┐                                                                                                                   
  │      Node type      │        Trust weight        │            
  ├─────────────────────┼────────────────────────────┤
  │ sgx / sev-snp / tdx │ 1.0 (hardware attestation) │
  ├─────────────────────┼────────────────────────────┤
  │ software            │ 0.7                        │                                                                                                                   
  ├─────────────────────┼────────────────────────────┤                                                                                                                   
  │ browser-extension   │ 0.4                        │                                                                                                                   
  └─────────────────────┴────────────────────────────┘                                                                                                                   
                                                                  
  Browser DVEs receive only tasks that accept trust_tier: "standard" or below. Tasks requiring hardware attestation are never dispatched to browser nodes.               
  
  7.4 Rate Limiting                                                                                                                                                      
                                                                  
  BrowserDVEHub enforces per-wallet rate limits: max 10 tasks/minute, max 1 concurrent task. This prevents the extension from degrading the user's browser experience and
   limits economic attack surfaces.
                                                                                                                                                                         
  ---                                                             
  Critical File Change Summary
                                                                                                                                                                         
  ┌────────────────────────────────────────────────────────────────────────────┬─────────────┬────────────────────────────────────────────────┐
  │                                    File                                    │ Change type │                    Purpose                     │                          
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤
  │ KNIRVBRIDGE/src/background.ts                                              │ Modify      │ Add WS client, heartbeat alarm, task dispatch  │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤
  │ KNIRVBRIDGE/src/services/dve/dve-identity.ts                               │ New         │ Deterministic node ID derivation               │                          
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/services/dve/dve-registry.ts                               │ New         │ Register/heartbeat/deregister with KNIRVSERVER │                          
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/services/dve/dve-badge-manager.ts                          │ New         │ NFT badge → capability mapping                 │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/services/dve/dve-ws-client.ts                              │ New         │ WS client for task push/result                 │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/services/dve/validation-runtime.ts                         │ New         │ Sandboxed task execution                       │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/hooks/dve/use-dve-status.ts                                │ New         │ React hook for DVE state                       │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/hooks/dve/use-dve-badges.ts                                │ New         │ React hook for badge inventory                 │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/pages/popup/dve/dve-status.tsx                             │ New         │ DVE status popup page                          │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/pages/popup/dve/badge-inventory.tsx                        │ New         │ Badge inventory popup page                     │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVBRIDGE/src/common/constants/dve.constant.ts                           │ New         │ DVE-related constants                          │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVSERVER/backend/internal/objects/dve.go                                │ Modify      │ Add browser DVE fields to DVENode              │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVSERVER/backend/internal/web/dve_browser_ws_handler.go                 │ New         │ Browser DVE WebSocket hub                      │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVSERVER/backend/internal/services/dvemanager/dve_manager.go            │ Modify      │ Browser node handling, trust tier              │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVSERVER/backend/internal/services/dvemanager/chain_native_discovery.go │ Modify      │ Discover browser DVEs from chain               │
  ├────────────────────────────────────────────────────────────────────────────┼─────────────┼────────────────────────────────────────────────┤                          
  │ KNIRVSHELL/cmd/browser_dve.go                                              │ New         │ CLI browser DVE management                     │
  └────────────────────────────────────────────────────────────────────────────┴─────────────┴────────────────────────────────────────────────┘                          
                                                                  
  ---                                                                                                                                                                    
  Recommended Implementation Order
                                                                                                                                                                         
  1. Phase 1 (Identity + Registration) — zero risk; purely additive on both sides
  2. Phase 2 (WebSocket) — enables live connectivity; test with a local KNIRVSERVER instance first                                                                       
  3. Phase 3 (Badge mapping) — requires badge NFT metadata standard to be agreed on first                                                                                
  4. Phase 6.1 (DVE Identity NFT) — chain work; can run in parallel with Phase 4                                                                                         
  5. Phase 4 (Validation runtime) — depends on Phase 2 WS being stable                                                                                                   
  6. Phase 5 (UI) — depends on Phases 1–3 for real data                                                                                                                  
  7. Phase 7 (Security hardening) — implement as each phase ships, not at the end                                                                                        
                                                                                                                                                                         
  The main tradeoff to flag: browser DVEs can't provide hardware attestation (no SGX/TDX in a browser). The trust-tier system above handles this by routing only         
  lightweight, policy-verifiable tasks their way. If the team wants stronger guarantees, a future iteration could integrate WebAuthn platform authenticators as a        
  weaker-but-auditable form of device attestation.