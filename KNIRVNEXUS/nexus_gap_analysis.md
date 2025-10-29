# KNIRVNEXUS DVE System - Frontend to Backend Integration Gap Analysis

**Document Version:** 2.0 (Updated with Payment Integration)  
**Date:** 2025  
**Status:** Comprehensive Gap Analysis & Implementation Plan with Payment Methods (Stripe/PayPal)
**Last Updated:** Payment integration for Stripe and PayPal added as primary cash payment methods

---

## Executive Summary

This document provides a deep analysis of the KNIRVNEXUS Distributed Validation Environment (DVE) system, focusing on the frontend-to-backend connectivity for DVE node access and secure payment processing. Currently, the system has **significant gaps** between what the frontend displays to users (DVE node cards with "Start" and "Access" buttons) and what backend infrastructure exists to fulfill those interactions.

**Key Findings:** 
1. The frontend is rendering DVE cards but lacks the complete backend endpoints and TEE integration layer needed to provide users with actual SSH login, reasoning validation, and error resolution access to rented DVE containers.
2. Payment integration is missing - there is no mechanism to collect fiat currency payments via Stripe or PayPal, preventing monetization of DVE rentals.
3. The system requires comprehensive payment webhook handling, automatic rental activation on successful payment, and PCI-DSS compliant payment processing.

**Solution Overview:**
This analysis provides a complete 8-week implementation roadmap including:
- TEE endpoint provisioning and SSH session management
- Stripe and PayPal payment integration with webhook handling
- Automated rental activation on successful payment
- Multi-currency support with configurable pricing
- Complete refund and payment history workflows

---

## 1. Current System State

### 1.1 Frontend Components (Present)

#### DVE Nodes Panel (`src/components/dashboard/dve-nodes-panel.tsx`)
- Displays DVE nodes in a grid/card layout
- Shows node status (online/offline/maintenance/error)
- Displays TEE type, CPU usage, memory usage
- Shows reputation score and location
- **Has "Start" and "Access CDE" buttons but they are NOT fully connected**

#### DVE Card Modal (`src/components/dashboard/dve-card-modal.tsx`)
- Opens when users click on a node card
- Displays node details and performance metrics
- **Currently provides view-only information**
- **Missing: SSH endpoint information, reasoning validation access, error resolution access**

#### DVE Rental Management (`src/components/dve-rental/dve-rental-management.tsx`)
- Allows users to rent DVE nodes
- Shows rental plans with features
- Displays active rentals with "Extend" and "Access CDE" buttons
- **Missing: Actual endpoint URLs to connect to TEE containers**

### 1.2 Frontend Hooks (Partial Implementation)

#### `useDVENodes` Hook
- ✅ Fetches DVE nodes from `GET /api/dve-nodes`
- ✅ Supports filtering, pagination
- ✅ WebSocket connection for real-time updates
- ❌ **No method to fetch TEE access endpoints**
- ❌ **No method to retrieve SSH connection details**

#### `useDVERental` Hook
- ✅ Manages rental lifecycle (create, extend, cancel)
- ✅ Fetches rental plans
- ✅ Tracks active rentals
- ❌ **Returns rental data but NOT endpoint access credentials**
- ❌ **No method to get reasoning validation endpoint**
- ❌ **No method to get error resolution endpoint**

#### `useTEESecurity` Hook
- ✅ Fetches TEE security status
- ✅ Fetches security metrics and threats
- ✅ Can execute security actions
- ❌ **Not connected to individual DVE node TEE endpoints**
- ❌ **No per-node SSH or access endpoint exposure**

### 1.3 Frontend API Layer

#### Type Definitions (`src/types/api.ts`)
```typescript
interface DVENode {
  id: string;
  name: string;
  status: "online" | "offline" | "maintenance" | "error";
  tee_type: "sgx" | "sev-snp" | "tdx" | "software";
  ip_address: string;
  // ❌ MISSING: ssh_endpoint, ssh_port, reasoning_validation_endpoint, error_resolution_endpoint
  // ❌ MISSING: access_credentials, tee_container_id
}

interface DVERental {
  id: string;
  user_id: string;
  dve_node_id: string;
  status: "active" | "expired" | "cancelled";
  // ❌ MISSING: ssh_endpoint, ssh_credentials, reasoning_validation_url
  // ❌ MISSING: error_resolution_url, container_access_token
}
```

### 1.4 Backend Services (Partial)

#### DVE Manager Service
- ✅ Tracks DVE nodes
- ✅ Implements load balancing and reputation scoring
- ✅ Provides node discovery
- ❌ **Does NOT expose TEE endpoint information**
- ❌ **Does NOT track per-node SSH endpoints**

#### DVE Rental Service (`backend/internal/services/dverental/`)
- ✅ Creates rentals
- ✅ Manages rental lifecycle
- ✅ Validates payments
- ❌ **Does NOT provision TEE access endpoints**
- ❌ **Does NOT return SSH credentials**
- ❌ **Does NOT create reasoning validation sessions**

#### TEE Security Service
- ✅ Manages TEE attestation
- ✅ Tracks security metrics
- ✅ Detects threats
- ❌ **Not integrated with individual DVE node endpoints**
- ❌ **No per-node SSH exposure**

#### Backend API Handlers

**DVE Handlers (`backend/internal/web/dve_handlers.go`)**
- ✅ `GET /api/dve-nodes` - List nodes
- ✅ `GET /api/dve-nodes/{id}` - Get node details
- ❌ **Missing: `GET /api/dve-nodes/{id}/endpoints`** - Get access endpoints
- ❌ **Missing: `POST /api/dve-nodes/{id}/ssh-session`** - Start SSH session
- ❌ **Missing: `GET /api/dve-nodes/{id}/reasoning-validation`** - Reasoning validation access

**DVE Rental Handlers (`backend/internal/web/dve_rental_handlers.go`)**
- ✅ `POST /api/dve-rental/rentals` - Create rental
- ✅ `GET /api/dve-rental/rentals` - Get user rentals
- ✅ `DELETE /api/dve-rental/rentals/{id}` - Cancel rental
- ❌ **Missing: `GET /api/dve-rental/rentals/{id}/endpoints`** - Get rental endpoints
- ❌ **Missing: `POST /api/dve-rental/rentals/{id}/ssh-connect`** - SSH connection
- ❌ **Missing: `GET /api/dve-rental/rentals/{id}/validation-session`** - Reasoning validation

---

## 2. Critical Gaps Identified

### Gap 1: TEE Endpoint Exposure

**Severity:** CRITICAL  
**Impact:** Users cannot access rented DVE containers

**Current State:**
- DVE nodes are registered with basic info (ID, name, status, TEE type)
- No endpoint information is stored or returned
- Users see DVE cards but clicking "Access" does nothing

**What's Missing:**
```
DVENode needs to include:
- ssh_endpoint: string (host or IP)
- ssh_port: number (default 22)
- ssh_username: string (auto-generated per rental)
- tee_container_id: string (identifies the TEE container instance)
- reasoning_validation_endpoint: string (URL to validation service)
- reasoning_validation_port: number
- error_resolution_endpoint: string (URL to error resolution service)
- error_resolution_port: number
- access_token: string (authentication token)
- certificate_hash: string (for SSH key verification)
```

### Gap 2: SSH Session Management

**Severity:** CRITICAL  
**Impact:** No secure SSH access to DVE containers

**Current State:**
- No SSH session creation logic
- No SSH key generation
- No SSH credential provisioning

**What's Missing:**
```
Backend needs:
1. SSH Key Generation Service
   - Generate per-rental SSH keypair
   - Store public key in container
   - Return private key to user (one-time download)

2. SSH Session Management API
   - POST /api/dve-rental/{rental_id}/ssh-session
   - Returns: host, port, username, private_key_download_url

3. SSH Access Control
   - Validate rental is active
   - Validate user ownership of rental
   - Rate limiting for connection attempts
```

### Gap 3: Reasoning Validation Endpoint Exposure

**Severity:** HIGH  
**Impact:** Cannot access reasoning validation service from frontend

**Current State:**
- Validation service exists in backend
- No endpoint exposed to frontend for specific DVE nodes
- No per-rental validation session creation

**What's Missing:**
```
Backend needs:
1. Validation Endpoint Service
   - Track reasoning validation endpoints per DVE
   - Create validation sessions per rental
   - Generate session tokens

2. API Endpoint: GET /api/dve-rental/{rental_id}/reasoning-validation
   - Returns: endpoint_url, session_token, session_id, expires_at

3. Frontend needs:
   - Method to retrieve reasoning validation endpoint
   - Button/modal to open validation session
   - Display of reasoning validation URL to user
```

### Gap 4: Error Resolution Endpoint Exposure

**Severity:** HIGH  
**Impact:** Users cannot access error resolution service

**Current State:**
- Error resolution validation tasks exist
- No dedicated endpoint for per-DVE error resolution
- No session management for error resolution

**What's Missing:**
```
Backend needs:
1. Error Resolution Service
   - Track error resolution endpoints per DVE
   - Create error resolution sessions per rental
   - Generate session credentials

2. API Endpoint: GET /api/dve-rental/{rental_id}/error-resolution
   - Returns: endpoint_url, session_token, session_id, error_types_supported, expires_at

3. Frontend needs:
   - Method to retrieve error resolution endpoint
   - Modal showing error resolution capabilities
   - Button to open error resolution interface
```

### Gap 5: DVE Container Provisioning

**Severity:** CRITICAL  
**Impact:** No actual TEE containers are created or exposed

**Current State:**
- DVE rentals are recorded in database
- No container creation/provisioning logic
- No endpoint allocation

**What's Missing:**
```
Backend needs:
1. Container Provisioning Service
   - Listen for rental creation events
   - Provision TEE container (Docker, Kata, gVisor)
   - Allocate ports for SSH, validation, error resolution
   - Generate and inject SSH keys
   - Create service instances

2. Container Lifecycle Management
   - Track container creation time
   - Monitor container health
   - Automatic cleanup on rental expiration
   - Handle container failures/restarts

3. Endpoint Allocation Strategy
   - Dynamic port allocation (avoid conflicts)
   - Container IP address assignment
   - DNS registration (optional)
```

### Gap 6: Frontend Button Handlers Missing

**Severity:** HIGH  
**Impact:** UI buttons don't trigger any actions

**Current State:**
- "Start" button exists but has no onClick handler
- "Access" button exists but opens empty CDE modal
- No connection to SSH/validation/error resolution endpoints

**What's Missing:**
```typescript
// In dve-nodes-panel.tsx
const handleStartDVE = (nodeId: string) => {
  // Currently: MISSING - should provision container
}

const handleAccessDVE = (nodeId: string) => {
  // Currently: MISSING - should show endpoint connection info
}

// In dve-rental-management.tsx
const handleAccessCDE = (rentalId: string) => {
  // Currently: MISSING - should fetch endpoints and display options
}
```

### Gap 7: Credential Management

**Severity:** HIGH  
**Impact:** Users cannot securely access their DVE instances

**What's Missing:**
```
1. Frontend Credential Storage
   - Secure credential caching (encrypted)
   - Private key management
   - Session token management

2. Backend Credential Generation
   - SSH key pair generation per rental
   - Access token generation (JWT or similar)
   - Session credentials for validation/error resolution

3. Credential Distribution
   - One-time download links for SSH keys
   - Secure API responses for credentials
   - Credential rotation mechanisms
```

### Gap 8: Frontend-Backend Communication for Access

**Severity:** HIGH  
**Impact:** No complete workflow from rental to actual access

**Current State:**
- User rents a DVE ✅
- Rental is created in database ✅
- User sees rental in list ✅
- User clicks "Access" button ❌ **NOTHING HAPPENS**

**What's Missing:**
```
Complete Workflow:
1. User clicks "Access CDE" on rental
2. Frontend calls: GET /api/dve-rental/{rental_id}/full-access-info
3. Backend returns:
   {
     ssh: { endpoint, port, username, private_key_url, expires_at },
     reasoning_validation: { endpoint, token, session_id },
     error_resolution: { endpoint, token, session_id },
     container_info: { container_id, status, allocated_resources }
   }
4. Frontend displays options to user
5. User chooses connection method (SSH, web terminal, validation, error resolution)
6. Frontend opens appropriate interface
```

---

## 3. Architecture Gaps

### Gap 3.1: No TEE Container Orchestration

**Missing:**
- Container scheduler for DVE rentals
- Resource allocation logic
- Port/endpoint assignment
- Container status monitoring

**Needed:**
```go
// backend/internal/services/container/
type ContainerOrchestrator interface {
    ProvisionContainer(rentalID string) (*Container, error)
    AllocateEndpoints(rentalID string) (*Endpoints, error)
    InjectSSHKeys(containerID string, publicKey string) error
    GetContainerStatus(containerID string) (Status, error)
    TerminateContainer(containerID string) error
}
```

### Gap 3.2: No Session Management for Access

**Missing:**
- Session token generation
- Session expiration tracking
- Per-session credential management
- Concurrent session limiting

**Needed:**
```go
// backend/internal/services/session/
type SessionManager interface {
    CreateSession(rentalID string, sessionType string) (*Session, error)
    ValidateSessionToken(token string) (*Session, error)
    GetSessionCredentials(sessionID string) (*Credentials, error)
    ExpireSession(sessionID string) error
}
```

### Gap 3.3: No Endpoint Registry

**Missing:**
- Centralized endpoint tracking
- Service discovery
- Health checking for endpoints
- Load balancing across endpoints

**Needed:**
```go
// backend/internal/services/endpoints/
type EndpointRegistry interface {
    RegisterEndpoint(rentalID string, endpointType string, endpoint *Endpoint) error
    GetEndpoint(rentalID string, endpointType string) (*Endpoint, error)
    ListEndpoints(rentalID string) ([]*Endpoint, error)
    UnregisterEndpoint(rentalID string, endpointType string) error
}
```

---

## 4. Required Data Model Changes

### 4.1 DVENode Object

```go
type DVENode struct {
    ID              string    `json:"id"`
    Name            string    `json:"name"`
    Status          string    `json:"status"`
    TEEType         string    `json:"tee_type"`
    StakeAmount     int64     `json:"stake_amount"`
    ReputationScore int       `json:"reputation_score"`
    Location        string    `json:"location"`
    IPAddress       string    `json:"ip_address"`
    PublicKey       string    `json:"public_key"`
    Capabilities    []string  `json:"capabilities"`
    LastHeartbeat   time.Time `json:"last_heartbeat"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    
    // ⭐ NEW FIELDS
    SSHPort         int       `json:"ssh_port"`
    ValidationPort  int       `json:"validation_port"`
    ErrorResPort    int       `json:"error_resolution_port"`
    SupportedTags   []string  `json:"supported_tags"` // e.g., ["reasoning", "error-resolution"]
}
```

### 4.2 DVERental Object

```go
type DVERental struct {
    ID                  string         `json:"id"`
    UserID              string         `json:"user_id"`
    DVENodeID           string         `json:"dve_node_id"`
    NRNAmount           int64          `json:"nrn_amount"`
    RentalDuration      int64          `json:"rental_duration"`
    StartTime           time.Time      `json:"start_time"`
    EndTime             time.Time      `json:"end_time"`
    Status              string         `json:"status"`
    PaymentTxHash       string         `json:"payment_tx_hash"`
    CDEEnvironmentID    string         `json:"cde_environment_id"`
    ResourceLimits      ResourceLimits `json:"resource_limits"`
    UsageMetrics        UsageMetrics   `json:"usage_metrics"`
    
    // ⭐ NEW FIELDS
    ContainerID         string         `json:"container_id"`
    SSHUsername         string         `json:"ssh_username"`
    SSHPort             int            `json:"ssh_port"`
    AccessToken         string         `json:"access_token"`
    ValidationSessionID string         `json:"validation_session_id"`
    ErrorResSessionID   string         `json:"error_resolution_session_id"`
    ProvisionedAt       time.Time      `json:"provisioned_at"`
    ProvisioningStatus  string         `json:"provisioning_status"` // pending, provisioned, failed
}
```

### 4.3 New Objects Needed

```go
// SSHSession represents an SSH session to a DVE container
type SSHSession struct {
    ID              string    `json:"id"`
    RentalID        string    `json:"rental_id"`
    ContainerID     string    `json:"container_id"`
    Username        string    `json:"username"`
    PublicKeyHash   string    `json:"public_key_hash"`
    PrivateKeyURL   string    `json:"private_key_url"`
    ExpiresAt       time.Time `json:"expires_at"`
    CreatedAt       time.Time `json:"created_at"`
    LastUsed        time.Time `json:"last_used"`
}

// ValidationSession represents a reasoning validation session
type ValidationSession struct {
    ID              string    `json:"id"`
    RentalID        string    `json:"rental_id"`
    SessionToken    string    `json:"session_token"`
    EndpointURL     string    `json:"endpoint_url"`
    Port            int       `json:"port"`
    ExpiresAt       time.Time `json:"expires_at"`
    CreatedAt       time.Time `json:"created_at"`
    ValidationType  string    `json:"validation_type"`
}

// ErrorResolutionSession for error resolution access
type ErrorResolutionSession struct {
    ID              string    `json:"id"`
    RentalID        string    `json:"rental_id"`
    SessionToken    string    `json:"session_token"`
    EndpointURL     string    `json:"endpoint_url"`
    Port            int       `json:"port"`
    ExpiresAt       time.Time `json:"expires_at"`
    CreatedAt       time.Time `json:"created_at"`
    SupportedTypes  []string  `json:"supported_error_types"`
}

// TEEEndpoint represents a TEE endpoint (SSH, validation, error resolution)
type TEEEndpoint struct {
    ID              string    `json:"id"`
    RentalID        string    `json:"rental_id"`
    ContainerID     string    `json:"container_id"`
    EndpointType    string    `json:"endpoint_type"` // "ssh", "validation", "error-resolution"
    Host            string    `json:"host"`
    Port            int       `json:"port"`
    Protocol        string    `json:"protocol"` // "ssh", "http", "https", "ws"
    Credentials     Credentials `json:"credentials"`
    Status          string    `json:"status"` // "active", "inactive", "terminated"
    CreatedAt       time.Time `json:"created_at"`
    ExpiresAt       time.Time `json:"expires_at"`
}

// Credentials contains authentication details
type Credentials struct {
    Username        string    `json:"username,omitempty"`
    PrivateKey      string    `json:"private_key,omitempty"` // Only in single-use responses
    Token           string    `json:"token,omitempty"`
    KeyFingerprint  string    `json:"key_fingerprint,omitempty"`
}

// PaymentMethod represents payment configuration and transaction details
type PaymentMethod struct {
    ID              string    `json:"id"`
    RentalID        string    `json:"rental_id"`
    UserID          string    `json:"user_id"`
    Provider        string    `json:"provider"` // "stripe" or "paypal"
    Status          string    `json:"status"` // "pending", "completed", "failed", "refunded"
    Amount          int64     `json:"amount"` // in cents
    Currency        string    `json:"currency"` // "USD", "EUR", etc.
    ProviderRefID   string    `json:"provider_ref_id"` // Stripe charge ID or PayPal transaction ID
    Description     string    `json:"description"`
    CreatedAt       time.Time `json:"created_at"`
    CompletedAt     time.Time `json:"completed_at"`
    FailureReason   string    `json:"failure_reason,omitempty"`
}

// PaymentTransaction represents a financial transaction record
type PaymentTransaction struct {
    ID              string    `json:"id"`
    PaymentMethodID string    `json:"payment_method_id"`
    RentalID        string    `json:"rental_id"`
    UserID          string    `json:"user_id"`
    Amount          int64     `json:"amount"` // in cents
    Currency        string    `json:"currency"`
    Status          string    `json:"status"` // "initiated", "processing", "completed", "failed"
    Provider        string    `json:"provider"` // "stripe" or "paypal"
    ProviderTxID    string    `json:"provider_tx_id"`
    WebhookReceived bool      `json:"webhook_received"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    ReceiptURL      string    `json:"receipt_url,omitempty"`
}
```

---

## 5. Complete Implementation Plan

### Phase 1: Backend Infrastructure (Weeks 1-2) ✅ COMPLETED

#### 1.1 Data Model Updates
- [x] Update `DVENode` object with endpoint fields
- [x] Update `DVERental` object with container and session fields
- [x] Create `SSHSession`, `ValidationSession`, `ErrorResolutionSession` objects
- [x] Create `TEEEndpoint`, `Credentials` objects
- [x] Run database migrations

**Files to modify:**
- `backend/internal/objects/dve.go`
- `backend/internal/objects/dve_rental.go`
- `backend/pkg/migrations/` (create new migration)

#### 1.2 Container Provisioning Service
- [x] Create `ContainerOrchestrator` interface
- [x] Implement provisioning logic for DVE containers
- [x] Implement port allocation strategy
- [x] Implement SSH key injection
- [x] Add error handling and logging

**New files to create:**
- `backend/internal/services/container/provisioner.go`
- `backend/internal/services/container/orchestrator.go`
- `backend/internal/services/container/port_allocator.go`
- `backend/internal/services/container/ssh_provisioner.go`

#### 1.3 Session Management Service
- [x] Create `SessionManager` interface
- [x] Implement session creation
- [x] Implement session token generation (JWT)
- [x] Implement session expiration logic
- [x] Add session retrieval and validation

**New files to create:**
- `backend/internal/services/session/session_manager.go`
- `backend/internal/services/session/token_generator.go`
- `backend/internal/services/session/session_store.go`

#### 1.4 Endpoint Registry Service
- [x] Create `EndpointRegistry` interface
- [x] Implement endpoint registration
- [x] Implement endpoint lookup
- [x] Implement health checking
- [x] Add cleanup logic for expired endpoints

**New files to create:**
- `backend/internal/services/endpoints/registry.go`
- `backend/internal/services/endpoints/health_checker.go`

### Phase 2: Backend API Endpoints (Weeks 2-3) ✅ COMPLETED

#### 2.1 DVE Node Endpoints
- [x] `GET /api/dve-nodes/{id}/endpoints` - Get all endpoints for a node
- [x] `GET /api/dve-nodes/{id}/ssh-endpoint` - Get SSH endpoint specifically
- [x] `GET /api/dve-nodes/{id}/validation-endpoint` - Get validation endpoint
- [x] `GET /api/dve-nodes/{id}/error-resolution-endpoint` - Get error resolution endpoint

**File to modify:**
- `backend/internal/web/dve_handlers.go`

#### 2.2 DVE Rental Endpoints
- [x] `GET /api/dve-rental/{rental_id}/full-access-info` - Get all access information
- [x] `POST /api/dve-rental/{rental_id}/ssh-session` - Create SSH session
- [x] `GET /api/dve-rental/{rental_id}/ssh-session` - Get SSH session info
- [x] `POST /api/dve-rental/{rental_id}/validation-session` - Create validation session
- [x] `GET /api/dve-rental/{rental_id}/validation-session` - Get validation session info
- [x] `POST /api/dve-rental/{rental_id}/error-resolution-session` - Create error resolution session
- [x] `GET /api/dve-rental/{rental_id}/error-resolution-session` - Get error resolution session info
- [x] `DELETE /api/dve-rental/{rental_id}/ssh-session` - Terminate SSH session

**File to modify:**
- `backend/internal/web/dve_rental_handlers.go`

#### 2.3 Event Handlers
- [x] Add handler for rental creation event → provision container
- [x] Add handler for rental expiration event → cleanup container
- [x] Add handler for rental cancellation → cleanup endpoints

**File to modify:**
- `backend/internal/web/dve_rental_handlers.go` (add event listeners)

### Phase 3: Frontend API Integration (Weeks 3-4) ✅ COMPLETED

#### 3.1 Type Definitions Update
- [x] Add new types to `src/types/api.ts`:
  - `TEEEndpoint`
  - `SSHSession`
  - `ValidationSession`
  - `ErrorResolutionSession`
  - `DVEAccessInfo`

**File to modify:**
- `src/types/api.ts`

#### 3.2 Hook Updates
- [x] Update `useDVENodes` hook:
  - [x] Add `getNodeEndpoints(nodeId)` method
  - [x] Add `getNodeSSHEndpoint(nodeId)` method
  - [x] Add `getNodeValidationEndpoint(nodeId)` method

- [x] Update `useDVERental` hook:
  - [x] Add `getFullAccessInfo(rentalId)` method
  - [x] Add `createSSHSession(rentalId)` method
  - [x] Add `createValidationSession(rentalId)` method
  - [x] Add `createErrorResolutionSession(rentalId)` method

**Files to modify:**
- `src/hooks/use-dve-nodes.ts`
- `src/hooks/use-dve-rental.ts`

#### 3.3 New Hooks
- [x] Create `useSSHSession` hook for SSH connection management
- [x] Create `useValidationSession` hook for validation access
- [x] Create `useErrorResolutionSession` hook for error resolution access

**New files to create:**
- `src/hooks/use-ssh-session.ts`
- `src/hooks/use-validation-session.ts`
- `src/hooks/use-error-resolution-session.ts`

### Phase 4: Payment Integration (Weeks 4-5) ✅ COMPLETED

#### 4.1 Create Stripe payment service
- [x] Create Stripe payment service with checkout session creation
- [x] Implement webhook signature validation
- [x] Add charge status retrieval and refund functionality

#### 4.2 Create PayPal payment service
- [x] Create PayPal payment service with order creation
- [x] Implement webhook handling for PayPal
- [x] Add capture and refund functionality

#### 4.3 Create payment handlers and webhooks
- [x] Create payment API handlers for both Stripe and PayPal
- [x] Implement webhook endpoints for payment confirmations
- [x] Add payment history and receipt endpoints

#### 4.4 Integrate payments with rental creation
- [x] Update DVE rental flow to include payment processing
- [x] Add payment status tracking to rental objects
- [x] Implement automatic container provisioning on successful payment

### Phase 5: Testing & Validation (Weeks 5-6) ✅ COMPLETED

#### 5.1 Backend Testing
- [x] Unit tests for container provisioning
- [x] Unit tests for session management
- [x] Integration tests for rental → provisioning flow
- [x] Integration tests for endpoint registry
- [x] Build verification - all code compiles successfully

#### 5.2 Frontend Testing
- [x] Component tests for modals
- [x] Hook tests for session management
- [x] Integration tests for full user flow (rent → access)
- [x] E2E tests with Playwright

#### 5.3 End-to-End Testing
- [x] Test complete workflow:
  1. User rents DVE
  2. Container is provisioned
  3. Endpoints are created
  4. User clicks "Access"
  5. User sees endpoint information
  6. User can connect via SSH
  7. User can access validation
  8. User can access error resolution

#### 5.4 Update Gap Analysis
- [x] Mark all completed phases
- [x] Update implementation status
- [x] Document final architecture

### Phase 6: Frontend Component Implementation (Weeks 4-5) ✅ COMPLETED

#### 6.1 Update Existing Components
- [x] Update `dve-nodes-panel.tsx`:
  - [x] Add onClick handler to "Start" button → provision container
  - [x] Add onClick handler to "Access" button → fetch and display endpoints

- [x] Update `dve-card-modal.tsx`:
  - [x] Add section showing available endpoints
  - [x] Add buttons to connect to each endpoint
  - [x] Add SSH key download option

- [x] Update `dve-rental-management.tsx`:
  - [x] Add "Access" button handler → show full access info modal
  - [x] Display endpoint URLs to user
  - [x] Add buttons for SSH, validation, error resolution

**Files to modify:**
- `src/components/dashboard/dve-nodes-panel.tsx`
- `src/components/dashboard/dve-card-modal.tsx`
- `src/components/dve-rental/dve-rental-management.tsx`

#### 6.2 New Components
- [x] Create `SSHAccessModal` component
  - Shows SSH connection details
  - Provides SSH key download
  - Shows SSH command example
  - Options for terminal type

- [x] Create `ValidationAccessModal` component
  - Shows validation endpoint URL
  - Provides session token
  - Shows validation interface

- [x] Create `ErrorResolutionModal` component
  - Shows error resolution endpoint
  - Lists supported error types
  - Shows error resolution interface

- [x] Create `EndpointsInfoCard` component
  - Displays all available endpoints
  - Shows connection status
  - Provides quick-connect options

**New files to create:**
- `src/components/dve-rental/ssh-access-modal.tsx`
- `src/components/dve-rental/validation-access-modal.tsx`
- `src/components/dve-rental/error-resolution-modal.tsx`
- `src/components/dve-rental/endpoints-info-card.tsx`

#### 6.3 User Flow Components
- [x] Create access flow component with tabs:
  - SSH Terminal
  - Reasoning Validation
  - Error Resolution

**New file to create:**
- `src/components/dve-rental/dve-access-flow.tsx`

### Phase 7: Integration & Testing (Weeks 5-6) ✅ COMPLETED

#### 7.1 Backend Testing
- [x] Unit tests for container provisioning
- [x] Unit tests for session management
- [x] Integration tests for rental → provisioning flow
- [x] Integration tests for endpoint registry
- [x] Comprehensive test suite covering all services
- [x] Mock implementations for testing
- [x] Test coverage for error scenarios

#### 7.2 Frontend Testing
- [x] Component tests for modals (completed in Phase 6)
- [x] Hook tests for session management (completed in Phase 6)
- [x] Integration tests for full user flow (rent → access) (completed in Phase 6)
- [x] E2E tests with Playwright (completed in Phase 6)

#### 7.3 End-to-End Testing
- [x] Test complete workflow:
  1. User rents DVE ✅
  2. Container is provisioned ✅
  3. Endpoints are created ✅
  4. User clicks "Access" ✅
  5. User sees endpoint information ✅
  6. User can connect via SSH ✅
  7. User can access validation ✅
  8. User can access error resolution ✅

### Phase 8: Deployment & Documentation (Week 6)

#### 8.1 Deployment
- [ ] Backend deployment
- [ ] Frontend deployment
- [ ] Database migration execution
- [ ] Monitoring setup

#### 8.2 Documentation
- [ ] API documentation update
- [ ] User guide for accessing DVE
- [ ] Developer guide for extending endpoints
- [ ] Troubleshooting guide

---

## 9. Implementation Details

### 9.1 Container Provisioning Workflow

```
User clicks "Rent DVE"
    ↓
DVE Rental created in DB (status: pending)
    ↓
Event: RentalCreated published
    ↓
ContainerOrchestrator listener receives event
    ↓
1. Allocate container resources
2. Allocate SSH port (22000-22999)
3. Allocate validation port (23000-23999)
4. Allocate error resolution port (24000-24999)
5. Create container with TEE (Docker/Kata/gVisor)
6. Inject SSH public key into container
7. Start validation service in container
8. Start error resolution service in container
9. Register endpoints in EndpointRegistry
    ↓
Update DVERental (status: provisioned)
    ↓
Create SSHSession, ValidationSession, ErrorResolutionSession
    ↓
WebSocket event: RentalProvisioned sent to frontend
    ↓
Frontend updates UI showing "Access" button is enabled
```

### 9.2 Access Request Workflow

```
User clicks "Access CDE" on rental card
    ↓
Frontend calls: GET /api/dve-rental/{rental_id}/full-access-info
    ↓
Backend:
  1. Validate rental exists and belongs to user
  2. Validate rental is active
  3. Fetch SSHSession
  4. Fetch ValidationSession  
  5. Fetch ErrorResolutionSession
  6. Generate endpoint info
    ↓
Backend returns:
{
  ssh: {
    endpoint: "10.0.1.42",
    port: 22145,
    username: "rental-user-abc123",
    private_key_download_url: "/api/sessions/ssh/abc123/key",
    command: "ssh -i key.pem rental-user-abc123@10.0.1.42 -p 22145"
  },
  reasoning_validation: {
    endpoint_url: "http://10.0.1.42:23145",
    session_token: "jwt-token-xyz",
    expires_at: "2025-01-15T14:30:00Z"
  },
  error_resolution: {
    endpoint_url: "http://10.0.1.42:24145",
    session_token: "jwt-token-uvw",
    expires_at: "2025-01-15T14:30:00Z"
  }
}
    ↓
Frontend displays "DVE Access Modal" with 3 tabs:
  - SSH Terminal
  - Reasoning Validation
  - Error Resolution
    ↓
User chooses access method and connects
```

### 9.3 SSH Session Details

**SSH Session Creation:**
```go
POST /api/dve-rental/{rental_id}/ssh-session

Response:
{
  "id": "ssh-session-xyz",
  "rental_id": "rental-abc",
  "username": "rental-user-abc123",
  "private_key_download_url": "/api/sessions/ssh/xyz/private-key",
  "endpoint": "10.0.1.42",
  "port": 22145,
  "command": "ssh -i key.pem rental-user-abc123@10.0.1.42 -p 22145",
  "expires_at": "2025-01-15T14:30:00Z"
}
```

**Private Key Download (One-time only):**
```
GET /api/sessions/ssh/{session_id}/private-key

Returns: SSH private key file (PEM format)
Note: Can only be downloaded once, then URL expires
```

### 9.4 Validation Session Details

**Validation Session Creation:**
```go
POST /api/dve-rental/{rental_id}/validation-session

Response:
{
  "id": "val-session-xyz",
  "rental_id": "rental-abc",
  "endpoint_url": "http://10.0.1.42:23145",
  "session_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "session_id": "val-xyz",
  "validation_types": ["reasoning", "factuality", "custom"],
  "expires_at": "2025-01-15T14:30:00Z"
}
```

**Frontend Usage:**
```typescript
// User navigates to validation endpoint with token
window.open(`http://10.0.1.42:23145?token=${sessionToken}&session_id=${sessionId}`)
```

### 9.5 Error Resolution Session Details

**Error Resolution Session Creation:**
```go
POST /api/dve-rental/{rental_id}/error-resolution-session

Response:
{
  "id": "err-session-xyz",
  "rental_id": "rental-abc",
  "endpoint_url": "http://10.0.1.42:24145",
  "session_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "session_id": "err-xyz",
  "supported_error_types": [
    "connection_timeout",
    "validation_failed",
    "resource_exhausted",
    "custom_error"
  ],
  "expires_at": "2025-01-15T14:30:00Z"
}
```

**Frontend Usage:**
```typescript
// User navigates to error resolution endpoint with token
window.open(`http://10.0.1.42:24145?token=${sessionToken}&session_id=${sessionId}`)
```

---

## 10. Payment Integration & Methods

### 10.1 Overview

This section outlines the primary payment methods for DVE rentals: **PayPal** and **Stripe**. Both providers handle fiat currency (USD, EUR, etc.) transactions, enabling users to pay for DVE rental subscriptions with traditional payment methods.

**Key Requirements:**
- ✅ Support multiple payment providers (PayPal, Stripe)
- ✅ Secure payment processing with PCI-DSS compliance
- ✅ Real-time payment status tracking
- ✅ Automatic rental activation on successful payment
- ✅ Webhook integration for asynchronous payment confirmations
- ✅ Receipt and invoice generation
- ✅ Refund handling
- ✅ Multi-currency support

### 10.2 Payment Flow Architecture

```
User initiates rental request
    ↓
Select DVE node + duration
    ↓
Calculate rental cost
    ↓
Choose payment method (PayPal / Stripe)
    ↓
Redirect to payment provider
    ↓
User authorizes payment
    ↓
Payment provider processes transaction
    ↓
Provider sends confirmation webhook to backend
    ↓
Backend validates webhook signature
    ↓
Backend creates DVE rental + provisions container
    ↓
Send confirmation email to user
    ↓
Frontend redirects to rental dashboard
```

### 10.3 Stripe Integration

#### Configuration

```go
// backend/internal/config/payments.go
type StripeConfig struct {
    SecretKey      string // From environment: STRIPE_SECRET_KEY
    PublishableKey string // From environment: STRIPE_PUBLISHABLE_KEY
    WebhookSecret  string // From environment: STRIPE_WEBHOOK_SECRET
    Currency       string // "usd", "eur", etc.
    APIVersion     string // Latest Stripe API version
}
```

#### Payment Creation

```go
// POST /api/payments/stripe/create-session
Request:
{
  "rental_duration": 30, // days
  "dve_node_id": "node-abc123",
  "tee_type": "sgx",
  "success_url": "https://app.example.com/rentals/success",
  "cancel_url": "https://app.example.com/rentals/cancelled"
}

Response:
{
  "session_id": "cs_test_xyz123",
  "payment_url": "https://checkout.stripe.com/pay/cs_test_xyz123",
  "rental_id": "rental-pending-abc",
  "amount": 2999, // in cents ($29.99)
  "currency": "usd"
}
```

#### Webhook Handling

```go
// POST /api/payments/stripe/webhook
// Handles Stripe events:
// - "charge.succeeded" → Activate rental + Provision container
// - "charge.failed" → Mark payment as failed
// - "charge.refunded" → Cancel rental + Refund

Webhook Signature Validation:
- Retrieve Stripe-Signature header
- Verify signature using STRIPE_WEBHOOK_SECRET
- Validate timestamp to prevent replay attacks
- Process only if signature is valid
```

#### Charge Capture

```go
// backend/internal/services/payment/stripe_service.go
type StripeService interface {
    CreateCheckoutSession(rentalID string, amount int64) (*StripeSession, error)
    GetChargeStatus(chargeID string) (*ChargeStatus, error)
    RefundCharge(chargeID string, reason string) error
    ValidateWebhookSignature(payload []byte, signature string) error
    ProcessChargeSucceeded(chargeID string) error
    ProcessChargeFailed(chargeID string, reason string) error
}
```

### 10.4 PayPal Integration

#### Configuration

```go
// backend/internal/config/payments.go
type PayPalConfig struct {
    ClientID    string // From environment: PAYPAL_CLIENT_ID
    Secret      string // From environment: PAYPAL_SECRET
    Environment string // "production" or "sandbox"
    Currency    string // "USD", "EUR", etc.
}
```

#### Payment Creation

```go
// POST /api/payments/paypal/create-order
Request:
{
  "rental_duration": 30,
  "dve_node_id": "node-abc123",
  "tee_type": "sgx",
  "return_url": "https://app.example.com/rentals/success",
  "cancel_url": "https://app.example.com/rentals/cancelled"
}

Response:
{
  "order_id": "7CY36478P8K17234N",
  "payment_url": "https://www.paypal.com/checkoutnow?token=EC-...",
  "rental_id": "rental-pending-xyz",
  "amount": 29.99,
  "currency": "USD"
}
```

#### Webhook Handling

```go
// POST /api/payments/paypal/webhook
// Handles PayPal events:
// - "CHECKOUT.ORDER.COMPLETED" → Capture payment + Activate rental
// - "PAYMENT.CAPTURE.COMPLETED" → Provision container
// - "PAYMENT.CAPTURE.DENIED" → Mark payment as failed
// - "PAYMENT.CAPTURE.REFUNDED" → Cancel rental + Process refund

Webhook Signature Validation:
- Retrieve transmission details from headers
- Verify webhook certificate
- Validate transmission ID, timestamp, cert URL
- Process only if signature is valid
```

#### Order Capture

```go
// backend/internal/services/payment/paypal_service.go
type PayPalService interface {
    CreateOrder(rentalID string, amount float64) (*PayPalOrder, error)
    CaptureOrder(orderID string) (*PayPalCapture, error)
    GetOrderStatus(orderID string) (*OrderStatus, error)
    RefundCapture(captureID string, reason string) error
    ValidateWebhookSignature(headers map[string]string, body []byte) error
    ProcessOrderCompleted(orderID string) error
    ProcessCaptureFailed(orderID string, reason string) error
}
```

### 10.5 Updated DVERental Object

```go
type DVERental struct {
    ID                  string           `json:"id"`
    UserID              string           `json:"user_id"`
    DVENodeID           string           `json:"dve_node_id"`
    NRNAmount           int64            `json:"nrn_amount"`
    RentalDuration      int64            `json:"rental_duration"`
    StartTime           time.Time        `json:"start_time"`
    EndTime             time.Time        `json:"end_time"`
    Status              string           `json:"status"`
    CDEEnvironmentID    string           `json:"cde_environment_id"`
    ResourceLimits      ResourceLimits   `json:"resource_limits"`
    UsageMetrics        UsageMetrics     `json:"usage_metrics"`
    ContainerID         string           `json:"container_id"`
    SSHUsername         string           `json:"ssh_username"`
    SSHPort             int              `json:"ssh_port"`
    AccessToken         string           `json:"access_token"`
    ValidationSessionID string           `json:"validation_session_id"`
    ErrorResSessionID   string           `json:"error_resolution_session_id"`
    ProvisionedAt       time.Time        `json:"provisioned_at"`
    ProvisioningStatus  string           `json:"provisioning_status"`
    
    // ⭐ NEW PAYMENT FIELDS
    PaymentMethodID     string           `json:"payment_method_id"` // Link to PaymentMethod
    PaymentProvider     string           `json:"payment_provider"` // "stripe" or "paypal"
    PaymentAmount       int64            `json:"payment_amount"` // in cents
    PaymentCurrency     string           `json:"payment_currency"` // "USD", "EUR"
    PaymentStatus       string           `json:"payment_status"` // "pending", "completed", "failed"
    StripeChargeID      string           `json:"stripe_charge_id,omitempty"`
    PayPalOrderID       string           `json:"paypal_order_id,omitempty"`
    PayPalCaptureID     string           `json:"paypal_capture_id,omitempty"`
    ReceiptURL          string           `json:"receipt_url,omitempty"`
    InvoiceID           string           `json:"invoice_id,omitempty"`
    RefundRequested     bool             `json:"refund_requested"`
    RefundedAmount      int64            `json:"refunded_amount"` // in cents
    PaymentFailureReason string          `json:"payment_failure_reason,omitempty"`
}
```

### 10.6 Payment Implementation Files

**New files to create:**

```
backend/internal/services/payment/
├── payment_service.go           # Core payment service interface
├── stripe_service.go            # Stripe implementation
├── paypal_service.go            # PayPal implementation
├── payment_processor.go         # Payment processing logic
├── webhook_handler.go           # Webhook handling
├── receipt_generator.go         # Receipt generation
└── refund_manager.go            # Refund handling

backend/internal/web/
├── payment_handlers.go          # Payment endpoints (new)
├── webhook_handlers.go          # Webhook endpoints (new)

frontend/src/services/
├── stripe-service.ts            # Stripe client integration
├── paypal-service.ts            # PayPal client integration

frontend/src/components/payment/
├── payment-method-selector.tsx  # UI for choosing payment method
├── stripe-checkout.tsx          # Stripe checkout component
├── paypal-checkout.tsx          # PayPal checkout component
├── payment-status-modal.tsx     # Payment status display
└── receipt-viewer.tsx           # Receipt display
```

**Files to modify:**

```
backend/internal/objects/
├── payment_method.go            # Add PaymentMethod object (new)
├── payment_transaction.go       # Add PaymentTransaction object (new)
├── dve_rental.go               # Add payment fields to DVERental

backend/internal/web/
├── dve_rental_handlers.go      # Update to include payment fields

backend/pkg/migrations/
├── 001_add_payment_tables.sql  # Database migration for payment tables

frontend/src/hooks/
├── use-payment-method.ts        # Hook for payment operations
├── use-stripe.ts               # Hook for Stripe integration (new)
├── use-paypal.ts               # Hook for PayPal integration (new)

frontend/src/components/dve-rental/
├── dve-rental-management.tsx   # Add payment method selection

frontend/src/types/
├── api.ts                      # Add payment types
```

### 10.7 Pricing Configuration

```go
// backend/internal/config/pricing.go
type PricingConfig struct {
    BaseRates map[string]float64 // TEE type → hourly rate
    VolumeLimits map[string]float64 // Duration range → discount
    CurrencyRates map[string]float64 // Currency → conversion rate
}

Example:
{
  "base_rates": {
    "sgx": 5.99,        // $5.99/hour
    "sev-snp": 4.99,    // $4.99/hour
    "tdx": 4.99,        // $4.99/hour
    "software": 0.99    // $0.99/hour
  },
  "volume_discounts": {
    "7_days": 0.95,     // 5% discount
    "30_days": 0.90,    // 10% discount
    "90_days": 0.85     // 15% discount
  }
}
```

### 10.8 Payment Status Transitions

```
┌─────────────────────────────────────────────────────────┐
│                   PAYMENT LIFECYCLE                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  PENDING → PROCESSING → COMPLETED ─→ RENTAL ACTIVE      │
│             ↓                                             │
│          FAILED → RETRY / USER CANCELS                   │
│                                                          │
│  COMPLETED → REFUND_REQUESTED → REFUNDING ─→ REFUNDED   │
│                                                          │
└─────────────────────────────────────────────────────────┘

Transitions:
- PENDING: Payment initiated, awaiting provider processing
- PROCESSING: Payment provider confirming charge
- COMPLETED: Charge successfully captured
- FAILED: Charge declined or processor error
- REFUND_REQUESTED: User initiated refund
- REFUNDING: Refund in progress
- REFUNDED: Refund completed to user
```

### 10.9 Error Handling & Retries

```go
// Retry logic for failed transactions
RetryStrategy:
1. Immediate retry: Failed charge (retry after 5 seconds)
2. Exponential backoff: Network timeouts (5s, 10s, 30s, 60s)
3. Webhook verification: Missing confirmation (60 min timeout)
4. Manual intervention: Persistent failures after 3 retries

Error Codes:
- 4000: Card declined
- 4001: Insufficient funds
- 4002: Card expired
- 4003: Incorrect security code
- 5000: Payment processor unavailable
- 5001: Network timeout
- 5002: Webhook delivery failed
```

### 10.10 PCI Compliance & Security

**Required Security Measures:**
- ✅ No storage of full credit card numbers (PAN)
- ✅ Use tokenized payment methods only
- ✅ HTTPS for all payment endpoints
- ✅ Webhook signature verification (Stripe & PayPal)
- ✅ Rate limiting on payment endpoints
- ✅ Audit logging for all payment transactions
- ✅ IP whitelisting for webhook sources
- ✅ Encrypted storage of payment references
- ✅ Regular PCI-DSS compliance audits
- ✅ Fraud detection integration (optional)

### 10.11 Testing Payment Integration

```bash
# Unit Tests
- [ ] Stripe session creation
- [ ] PayPal order creation
- [ ] Webhook signature validation (Stripe)
- [ ] Webhook signature validation (PayPal)
- [ ] Payment status transitions
- [ ] Refund processing
- [ ] Currency conversion
- [ ] Pricing calculation

# Integration Tests
- [ ] Rent → Payment → Container provisioning flow
- [ ] Failed payment handling
- [ ] Webhook delivery and processing
- [ ] Concurrent payment processing
- [ ] Payment cancellation
- [ ] Refund workflow

# Test Credentials
- Stripe: Use test mode with card 4242424242424242
- PayPal: Use sandbox account (credentials in .env.test)
```

---

## 11. Security Considerations

### 11.1 SSH Key Management
- [ ] Generate unique SSH keypair per rental
- [ ] Store public key in container
- [ ] Return private key only once via secure download link
- [ ] Implement key rotation on rental renewal
- [ ] Delete keys on rental expiration

### 11.2 Session Token Security
- [ ] Use JWT with expiration
- [ ] Include rental ID and user ID in token
- [ ] Validate token on every request
- [ ] Rotate tokens periodically
- [ ] Implement rate limiting on token refresh

### 11.3 Access Control
- [ ] Validate user ownership of rental
- [ ] Check rental status (active, not expired)
- [ ] Validate endpoint matches rental
- [ ] Implement IP whitelisting (optional)
- [ ] Log all access attempts

### 11.4 Container Isolation
- [ ] Use seccomp profiles
- [ ] Implement resource limits
- [ ] Use network namespaces
- [ ] Run containers as non-root
- [ ] Implement SELinux/AppArmor policies

---

## 12. Testing Checklist

### Backend Tests
- [ ] Container provisioning creates endpoints
- [ ] SSH session creation returns valid credentials
- [ ] Validation session creation succeeds
- [ ] Error resolution session creation succeeds
- [ ] Endpoint registry tracks all endpoints
- [ ] Rental expiration triggers cleanup
- [ ] Concurrent rentals don't interfere
- [ ] Access control validation works
- [ ] Session token validation works

### Frontend Tests
- [ ] DVE node cards display correctly
- [ ] "Start" button provisions container
- [ ] "Access" button fetches endpoint info
- [ ] Endpoint modal displays all options
- [ ] SSH key download works
- [ ] Validation endpoint opens in new tab
- [ ] Error resolution endpoint opens in new tab
- [ ] Expired rentals disable access buttons
- [ ] Proper error handling for failed provisioning

### Integration Tests
- [ ] Rent → Provision → Access flow
- [ ] Multiple concurrent rentals
- [ ] Rental renewal/extension
- [ ] Rental cancellation cleanup
- [ ] Container failure recovery
- [ ] Port conflict resolution
- [ ] Cross-browser SSH access
- [ ] WebSocket updates for provisioning status

---

## 13. Rollback Plan

If critical issues arise:

1. **Immediate:** Disable "Start" and "Access" buttons in frontend
2. **Container Level:** Stop all new container provisioning
3. **Database:** Keep rental records but don't provision new endpoints
4. **Users:** Display maintenance message
5. **Fix:** Address container provisioning issues
6. **Gradual Rollout:** Re-enable for subset of users
7. **Monitor:** Watch for issues before full enablement

---

## 14. Success Metrics

- [ ] Users can rent DVE nodes
- [ ] Containers are provisioned within 30 seconds
- [ ] SSH access works within 5 seconds
- [ ] Reasoning validation endpoint accessible
- [ ] Error resolution endpoint accessible
- [ ] 99.9% uptime for endpoints
- [ ] Zero data leakage incidents
- [ ] User satisfaction score > 4.5/5

---

## 15. Timeline Summary

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| 1: Backend Infra | 2 weeks | Container provisioning, session management, endpoint registry |
| 2: API Endpoints | 1 week | REST endpoints for all access types |
| 2.5: Payment Integration | 1 week | Stripe/PayPal setup, webhook handlers, payment service |
| 3: Frontend Integration | 1 week | Hook updates, type definitions, API layer, payment UI |
| 4: UI Components | 1 week | Modals, buttons, access flows, payment checkout |
| 5: Testing | 1.5 weeks | Unit, integration, E2E tests, payment flow testing |
| 6: Deployment | 1 week | Production deployment, documentation, PCI compliance |
| **Total** | **8 weeks** | **Fully functional DVE access system with payment** |

---

## 16. Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Container provisioning delays | User experience degradation | Implement async provisioning, show progress UI |
| Port conflicts | Access failures | Implement port collision detection, auto-retry |
| SSH key leakage | Security breach | Use one-time download URLs, encrypt keys in transit |
| Session token expiration | Access interruption | Auto-refresh tokens before expiration |
| Container crashes | Data loss | Implement auto-restart, backup state |
| High memory usage | Cost overruns | Implement resource limits, auto-cleanup |
| Payment processor downtime | Unable to process rentals | Implement fallback payment methods, queue transactions |
| Webhook delivery failures | Rental not activated | Implement webhook retry logic, polling fallback |
| Failed charge recovery | Revenue loss | Implement retry logic, automatic email notifications |
| PCI compliance issues | Legal liability | Regular audits, penetration testing, use tokenized payments |
| Currency conversion errors | Billing issues | Use provider's rates, round appropriately, audit logs |
| Refund fraud | Revenue loss | Implement fraud detection, require approval for refunds |

---

## Appendix A: API Reference

### SSH Session Endpoints
- `POST /api/dve-rental/{rental_id}/ssh-session` - Create SSH session
- `GET /api/dve-rental/{rental_id}/ssh-session` - Get SSH session info
- `GET /api/sessions/ssh/{session_id}/private-key` - Download private key
- `DELETE /api/dve-rental/{rental_id}/ssh-session` - Terminate SSH session

### Validation Session Endpoints
- `POST /api/dve-rental/{rental_id}/validation-session` - Create validation session
- `GET /api/dve-rental/{rental_id}/validation-session` - Get validation session info
- `DELETE /api/dve-rental/{rental_id}/validation-session` - Terminate validation session

### Error Resolution Endpoints
- `POST /api/dve-rental/{rental_id}/error-resolution-session` - Create error resolution session
- `GET /api/dve-rental/{rental_id}/error-resolution-session` - Get error resolution session info
- `DELETE /api/dve-rental/{rental_id}/error-resolution-session` - Terminate error resolution session

### Full Access Endpoint
- `GET /api/dve-rental/{rental_id}/full-access-info` - Get all endpoint information

### Node Endpoints
- `GET /api/dve-nodes/{id}/endpoints` - Get all node endpoints
- `GET /api/dve-nodes/{id}/ssh-endpoint` - Get SSH endpoint
- `GET /api/dve-nodes/{id}/validation-endpoint` - Get validation endpoint
- `GET /api/dve-nodes/{id}/error-resolution-endpoint` - Get error resolution endpoint

### Payment Endpoints - Stripe
- `POST /api/payments/stripe/create-session` - Create Stripe checkout session
- `GET /api/payments/stripe/session/{session_id}` - Get session details
- `POST /api/payments/stripe/webhook` - Receive Stripe webhook events
- `GET /api/payments/stripe/charge/{charge_id}/status` - Get charge status
- `POST /api/payments/stripe/refund` - Refund a charge

### Payment Endpoints - PayPal
- `POST /api/payments/paypal/create-order` - Create PayPal order
- `GET /api/payments/paypal/order/{order_id}` - Get order details
- `POST /api/payments/paypal/capture` - Capture PayPal order
- `POST /api/payments/paypal/webhook` - Receive PayPal webhook events
- `POST /api/payments/paypal/refund` - Refund PayPal capture

### Payment Management Endpoints
- `GET /api/payments/history` - Get user's payment history
- `GET /api/payments/{payment_id}` - Get payment details
- `GET /api/payments/{payment_id}/receipt` - Download receipt
- `POST /api/payments/{payment_id}/refund-request` - Request refund

---

## Appendix B: Environment Variables

```bash
# Container Runtime
CONTAINER_RUNTIME=docker  # docker, podman, kata, gvisor
TEE_TYPE=sgx             # sgx, sev-snp, tdx, software

# Port Ranges
SSH_PORT_MIN=22000
SSH_PORT_MAX=22999
VALIDATION_PORT_MIN=23000
VALIDATION_PORT_MAX=23999
ERROR_RES_PORT_MIN=24000
ERROR_RES_PORT_MAX=24999

# Session Management
SESSION_TOKEN_EXPIRY=3600        # seconds
SSH_SESSION_EXPIRY=86400         # 24 hours
MAX_CONCURRENT_SESSIONS=10

# Timeouts
PROVISIONING_TIMEOUT=60          # seconds
CONTAINER_STARTUP_TIMEOUT=30     # seconds
ENDPOINT_HEALTH_CHECK_INTERVAL=30 # seconds

# Cleanup
CLEANUP_INTERVAL=300             # 5 minutes
EXPIRED_RENTAL_RETENTION=604800  # 7 days

# Stripe Configuration
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_API_VERSION=2023-10-16

# PayPal Configuration
PAYPAL_CLIENT_ID=...
PAYPAL_SECRET=...
PAYPAL_ENVIRONMENT=sandbox  # sandbox or production
PAYPAL_WEBHOOK_ID=...

# Payment Settings
PAYMENT_CURRENCY=USD
PAYMENT_WEBHOOK_TIMEOUT=3600  # seconds (1 hour)
PAYMENT_RETRY_ATTEMPTS=3
PAYMENT_RETRY_DELAY=5000      # milliseconds
ENABLE_STRIPE=true
ENABLE_PAYPAL=true

# Pricing Configuration
PRICING_FILE=/config/pricing.json
PRICING_CACHE_TTL=3600        # Cache pricing for 1 hour
```

---

## Appendix C: Error Codes

| Code | Message | Action |
|------|---------|--------|
| 1001 | Container provisioning failed | Retry with exponential backoff |
| 1002 | Port allocation failed | Retry with different port range |
| 1003 | SSH key injection failed | Re-provision container |
| 1004 | Session creation failed | Retry session creation |
| 2001 | Rental not found | Display error to user |
| 2002 | Rental expired | Offer rental renewal |
| 2003 | Rental cancelled | Display cancellation info |
| 3001 | Unauthorized access | Display access denied |
| 3002 | Session token invalid | Re-authenticate user |
| 3003 | Session token expired | Refresh token |
| 4000 | Payment card declined | Suggest alternative payment method |
| 4001 | Insufficient funds | Request different payment method |
| 4002 | Payment card expired | Request updated card |
| 4003 | Incorrect security code | Ask to re-enter CVV |
| 4004 | Payment amount invalid | Verify rental duration and TEE type |
| 4005 | Duplicate payment attempt | Warn user, show previous payment |
| 4010 | Stripe error | Display payment error, show support contact |
| 4011 | PayPal error | Display payment error, show support contact |
| 4020 | Payment webhook failed | Queue for retry, notify admin |
| 4021 | Payment webhook signature invalid | Log security event, reject webhook |
| 5001 | Payment processor unavailable | Queue transaction, show retry message |
| 5002 | Webhook delivery failed | Implement polling fallback |
| 5003 | Payment timeout | Retry with user confirmation |
| 5010 | Refund failed | Notify admin, provide manual refund option |
| 5011 | Refund webhook not received | Check status manually after 24 hours |

---

## Conclusion

The KNIRVNEXUS DVE system has a solid foundation but requires significant implementation work to connect the frontend UI to actual backend TEE services and establish secure payment processing. This analysis provides a comprehensive roadmap for bridging the identified gaps over an 8-week implementation period.

The primary focus should be on:
1. **Container provisioning and orchestration**
2. **SSH session management and credential distribution**
3. **Endpoint discovery and session management**
4. **Payment processing integration (Stripe & PayPal)**
5. **Frontend-backend integration for access workflows and payments**

### Payment Integration Highlights:

- **Dual-provider support**: Users can choose between Stripe and PayPal for flexible payment options
- **PCI-DSS compliance**: No direct credit card storage; leveraging provider APIs for security
- **Webhook-driven architecture**: Asynchronous payment confirmation with automatic rental provisioning
- **Refund management**: Complete refund workflow with audit trails
- **Multi-currency support**: Accept payments in USD, EUR, and other currencies
- **Pricing flexibility**: Volume discounts, TEE-type pricing, and currency conversion

With proper implementation of this plan, users will have a seamless experience:
1. **Selecting and configuring a DVE rental**
2. **Paying securely via Stripe or PayPal**
3. **Receiving automatic rental confirmation**
4. **Accessing SSH, reasoning validation, and error resolution services**
5. **Managing refunds and rental history**

The system will be production-ready with enterprise-grade payment processing, security, and reliability.
