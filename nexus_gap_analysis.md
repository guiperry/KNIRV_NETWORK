# KNIRV-NEXUS DVE Gap Analysis Report

## Executive Summary

This report analyzes the KNIRV-NEXUS DVE (Distributed Validation Environment) system to identify inconsistencies, missing connections between backend and frontend, and non-functional operations. The analysis focuses on ensuring each DVE provides frontend users with access to a TEE that offers SSH login, reasoning validation, and error resolution endpoints, with DVE containers accessible through frontend "Start" and "Access" buttons.

**Analysis Date**: November 21, 2025
**Status**: CRITICAL GAPS IDENTIFIED - Implementation Required

---

## Current System Architecture

### Backend Structure (Go)
```
KNIRVNEXUS/backend/
├── cmd/backend_server/main.go          # Entry point (590+ lines)
├── internal/
│   ├── web/
│   │   ├── dve_handlers.go             # DVE node management (603 lines)
│   │   ├── dve_rental_handlers.go      # DVE rental operations (1401 lines)
│   │   └── routes.go                   # Route registration
│   ├── objects/
│   │   ├── dve.go                      # DVE node model
│   │   └── dve_rental.go               # DVE rental model (246 lines)
│   ├── services/
│   │   ├── dverental/dve_rental_service.go  # Rental logic (712 lines)
│   │   ├── session/session_manager.go       # Session management
│   │   ├── container/orchestrator.go        # Container management
│   │   └── endpoints/registry.go            # Endpoint registry
│   └── database/buntdb.go              # Database layer
└── config/                             # Configuration
```

### Frontend Structure (React/Next.js)
```
KNIRVNEXUS/frontend/src/
├── hooks/
│   ├── use-dve-rental.ts               # Rental API calls (392 lines)
│   ├── use-dve-nodes.ts                # Node API calls (478 lines)
│   ├── use-ssh-session.ts              # SSH session hooks (139 lines)
│   ├── use-validation-session.ts       # Validation hooks (125 lines)
│   └── use-error-resolution-session.ts # Error resolution hooks
├── components/dve-rental/
│   ├── dve-rental-management.tsx       # Main rental UI
│   └── dve-access-flow.tsx             # Access flow component
└── app/                                # Next.js pages
```

---

## Critical Gaps Identified

### CRITICAL GAP #1: SSH Private Key Download Endpoint - MISSING
**Severity**: CRITICAL
**Impact**: SSH access completely blocked

**Frontend Expects** (`use-ssh-session.ts` line 90):
```typescript
GET /api/sessions/ssh/{sessionId}/private-key
```

**Backend Status**: ENDPOINT DOES NOT EXIST
- `PrivateKeyURL` is set in session creation (`dve_rental_handlers.go` line 546)
- URL format: `/api/sessions/ssh/{sessionId}/private-key`
- But NO HANDLER exists to serve this endpoint

**Evidence**:
- Searched all handler files - no route matches this pattern
- Session manager creates session but doesn't store/serve private key

---

### CRITICAL GAP #2: Frontend Modal Components - MISSING
**Severity**: CRITICAL
**Impact**: Users cannot interact with any DVE service

**Missing Components** (imported in `dve-access-flow.tsx` lines 13-15):
```
❌ /frontend/src/components/dve-rental/ssh-access-modal.tsx
❌ /frontend/src/components/dve-rental/validation-access-modal.tsx
❌ /frontend/src/components/dve-rental/error-resolution-modal.tsx
```

**Evidence**: Files do not exist in the filesystem despite being imported

---

### CRITICAL GAP #3: Session Manager - Missing Methods
**Severity**: HIGH
**Impact**: Validation and error resolution sessions cannot be created

**File**: `backend/internal/services/session/session_manager.go`

**Missing Methods**:
| Method | Status | Called By |
|--------|--------|-----------|
| `CreateValidationSession()` | MISSING | dve_rental_handlers.go |
| `CreateErrorResolutionSession()` | MISSING | dve_rental_handlers.go |
| `GetValidationSession()` | MISSING | dve_rental_handlers.go line 350 |
| `GetErrorResolutionSession()` | MISSING | dve_rental_handlers.go line 370 |
| `TerminateValidationSession()` | MISSING | No handler exists |
| `TerminateErrorResolutionSession()` | MISSING | No handler exists |
| SSH key storage/retrieval | MISSING | Private key endpoint |

**Current Implementation** (only SSH):
- `CreateSSHSession()` - Lines 33-56 ✓
- `GetSSHSession()` - Lines 58-81 ✓

---

### CRITICAL GAP #4: Container Provisioning Falls Back to Mock
**Severity**: HIGH
**Impact**: SSH and endpoint access may not work in production

**File**: `backend/internal/services/dverental/dve_rental_service.go`

**Issue** (lines 662-709):
```go
func (drs *DVERentalService) provisionTEEContainer(...) (*ContainerInfo, error) {
    if drs.containerOrchestrator == nil {
        // Returns MOCK container - no actual SSH access
        return &ContainerInfo{
            ID: "mock-container-" + uuid.New().String(),
            // ... mock data
        }, nil
    }
    // ... actual provisioning
}
```

**Problems**:
1. Falls back to mock if orchestrator not initialized
2. Mock container has no real SSH server
3. SSH port hardcoded to `localhost` (line 518)

---

### CRITICAL GAP #5: Validation Session Handler - Incomplete
**Severity**: HIGH
**Impact**: Reasoning validation feature non-functional

**File**: `dve_rental_handlers.go` lines 700-915

**Issues**:
1. Handler may create session but doesn't register endpoint properly
2. Session manager method `CreateValidationSession()` doesn't exist
3. Frontend expects `session.session_token` but generation not implemented
4. Endpoint registry expects `GetEndpointByRentalAndType()` which may fail

---

### CRITICAL GAP #6: Error Resolution Session Handler - Incomplete
**Severity**: HIGH
**Impact**: Error resolution feature non-functional

**File**: `dve_rental_handlers.go` lines 917+

**Same issues as validation session**

---

### CRITICAL GAP #7: Session Persistence - In-Memory Only
**Severity**: MEDIUM-HIGH
**Impact**: Sessions lost on server restart

**File**: `session_manager.go` line 16
```go
sessions map[string]interface{} // In-memory only!
```

**Problems**:
1. No database persistence
2. Sessions lost on restart
3. No session recovery mechanism
4. Expired session cleanup may not work properly

---

### CRITICAL GAP #8: Payment Fields Not Used
**Severity**: MEDIUM
**Impact**: Payment tracking incomplete

**File**: `dve_rental.go` lines 37-50

**Defined but unused fields**:
- `PaymentMethodID`
- `PaymentProvider`
- `PaymentAmount`
- `PaymentStatus`
- `PaymentTimestamp`

`CreateRental()` only sets: `NRNAmount`, `PaymentTxHash`

---

### CRITICAL GAP #9: Endpoint Registry Integration
**Severity**: MEDIUM
**Impact**: Access info retrieval partially broken

**Issues**:
1. `GetEndpointByRentalAndType()` - may not be fully implemented
2. Fallback endpoints use hardcoded ports (23145, 24145)
3. No validation that endpoints are actually registered

---

### CRITICAL GAP #10: Terminate Session Endpoints - Missing Handlers
**Severity**: MEDIUM
**Impact**: Sessions cannot be cleanly terminated

**Missing DELETE handlers**:
- `DELETE /api/dve-rental/rentals/{id}/validation-session` - Not implemented
- `DELETE /api/dve-rental/rentals/{id}/error-resolution-session` - Not implemented

---

## Frontend-Backend Connection Analysis

### API Endpoints Status

| Endpoint | Frontend Hook | Backend Handler | Status |
|----------|--------------|-----------------|--------|
| `GET /api/dve-nodes` | use-dve-nodes.ts | dve_handlers.go | ✅ Working |
| `POST /api/dve-nodes` | use-dve-nodes.ts | dve_handlers.go | ✅ Working |
| `GET /api/dve-rental/plans` | use-dve-rental.ts | dve_rental_handlers.go | ✅ Working |
| `POST /api/dve-rental/rentals` | use-dve-rental.ts | dve_rental_handlers.go | ✅ Working |
| `GET /api/dve-rental/rentals` | use-dve-rental.ts | dve_rental_handlers.go | ✅ Working |
| `GET /api/dve-rental/rentals/{id}/full-access-info` | use-dve-rental.ts | dve_rental_handlers.go | ✅ Working |
| `POST /api/dve-rental/rentals/{id}/ssh-session` | use-ssh-session.ts | dve_rental_handlers.go | ⚠️ Partial |
| `GET /api/dve-rental/rentals/{id}/ssh-session` | use-ssh-session.ts | dve_rental_handlers.go | ⚠️ Partial |
| `DELETE /api/dve-rental/rentals/{id}/ssh-session` | use-ssh-session.ts | dve_rental_handlers.go | ✅ Working |
| `GET /api/sessions/ssh/{id}/private-key` | use-ssh-session.ts | **MISSING** | ❌ Missing |
| `POST /api/dve-rental/rentals/{id}/validation-session` | use-validation-session.ts | dve_rental_handlers.go | ❌ Incomplete |
| `GET /api/dve-rental/rentals/{id}/validation-session` | use-validation-session.ts | dve_rental_handlers.go | ❌ Incomplete |
| `DELETE /api/dve-rental/rentals/{id}/validation-session` | use-validation-session.ts | **MISSING** | ❌ Missing |
| `POST /api/dve-rental/rentals/{id}/error-resolution-session` | use-error-resolution-session.ts | dve_rental_handlers.go | ❌ Incomplete |
| `GET /api/dve-rental/rentals/{id}/error-resolution-session` | use-error-resolution-session.ts | dve_rental_handlers.go | ❌ Incomplete |
| `DELETE /api/dve-rental/rentals/{id}/error-resolution-session` | use-error-resolution-session.ts | **MISSING** | ❌ Missing |

### Component Status

| Component | File Exists | Functional | Notes |
|-----------|-------------|------------|-------|
| DVE Rental Management | ✅ | ⚠️ Partial | Missing modal integration |
| DVE Access Flow | ✅ | ❌ | Missing modal components |
| SSH Access Modal | ❌ | ❌ | File does not exist |
| Validation Access Modal | ❌ | ❌ | File does not exist |
| Error Resolution Modal | ❌ | ❌ | File does not exist |

---

## Implementation Plan

### Phase 1: Critical Endpoint Fixes (Priority: IMMEDIATE)

#### 1.1 Implement SSH Private Key Endpoint
**File**: `backend/internal/web/dve_rental_handlers.go`

```go
// Add handler for SSH private key download
func (h *DVERentalHandlers) GetSSHPrivateKey(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    sessionID := vars["sessionId"]

    // Get session from session manager
    session, err := h.sessionManager.GetSSHSession(sessionID)
    if err != nil {
        http.Error(w, "Session not found", http.StatusNotFound)
        return
    }

    // Get private key from container orchestrator
    privateKey, err := h.containerOrchestrator.GetSSHPrivateKey(session.ContainerID)
    if err != nil {
        http.Error(w, "Failed to retrieve private key", http.StatusInternalServerError)
        return
    }

    // Return as downloadable PEM file
    w.Header().Set("Content-Type", "application/x-pem-file")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pem", sessionID))
    w.Write(privateKey)
}

// Register route in RegisterDVERentalRoutes()
r.HandleFunc("/api/sessions/ssh/{sessionId}/private-key", h.GetSSHPrivateKey).Methods("GET")
```

#### 1.2 Complete Session Manager Methods
**File**: `backend/internal/services/session/session_manager.go`

```go
// Add validation session methods
func (sm *SessionManager) CreateValidationSession(rentalID string, config ValidationConfig) (*ValidationSession, error) {
    sessionID := uuid.New().String()
    session := &ValidationSession{
        ID:            sessionID,
        RentalID:      rentalID,
        Status:        "active",
        SessionToken:  generateSecureToken(),
        EndpointURL:   config.EndpointURL,
        CreatedAt:     time.Now(),
        ExpiresAt:     time.Now().Add(24 * time.Hour),
    }
    sm.mu.Lock()
    sm.sessions[sessionID] = session
    sm.mu.Unlock()

    // Persist to database
    sm.persistSession(session)

    return session, nil
}

func (sm *SessionManager) GetValidationSession(sessionID string) (*ValidationSession, error) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    if session, ok := sm.sessions[sessionID]; ok {
        if vs, ok := session.(*ValidationSession); ok {
            if vs.ExpiresAt.After(time.Now()) {
                return vs, nil
            }
            return nil, errors.New("session expired")
        }
    }
    return nil, errors.New("session not found")
}

// Similar methods for ErrorResolutionSession
func (sm *SessionManager) CreateErrorResolutionSession(...) (*ErrorResolutionSession, error)
func (sm *SessionManager) GetErrorResolutionSession(...) (*ErrorResolutionSession, error)
func (sm *SessionManager) TerminateValidationSession(...) error
func (sm *SessionManager) TerminateErrorResolutionSession(...) error
```

#### 1.3 Add Session Persistence
**File**: `backend/internal/services/session/session_manager.go`

```go
type SessionManager struct {
    sessions   map[string]interface{}
    mu         sync.RWMutex
    db         *database.BuntDB  // Add database reference
}

func (sm *SessionManager) persistSession(session interface{}) error {
    data, _ := json.Marshal(session)
    return sm.db.Set("session:"+getSessionID(session), string(data))
}

func (sm *SessionManager) loadSessions() error {
    // Load all sessions from database on startup
    sessions, _ := sm.db.GetByPrefix("session:")
    for _, data := range sessions {
        // Deserialize and add to in-memory map
    }
    return nil
}
```

### Phase 2: Frontend Modal Components (Priority: HIGH)

#### 2.1 Create SSH Access Modal
**File**: `frontend/src/components/dve-rental/ssh-access-modal.tsx`

```tsx
'use client';

import React, { useState, useEffect, useRef } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { useSSHSession } from '@/hooks/use-ssh-session';

interface SSHAccessModalProps {
  rentalId: string;
  onClose: () => void;
}

export default function SSHAccessModal({ rentalId, onClose }: SSHAccessModalProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const { createSession, getSession, downloadPrivateKey } = useSSHSession();
  const [session, setSession] = useState<any>(null);
  const [terminal, setTerminal] = useState<Terminal | null>(null);

  useEffect(() => {
    initializeSession();
  }, [rentalId]);

  const initializeSession = async () => {
    try {
      // Create or get existing session
      let sshSession = await getSession(rentalId);
      if (!sshSession) {
        sshSession = await createSession(rentalId);
      }
      setSession(sshSession);

      // Initialize terminal
      if (terminalRef.current) {
        const term = new Terminal({
          cursorBlink: true,
          theme: { background: '#1a1a2e' }
        });
        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        term.open(terminalRef.current);
        fitAddon.fit();
        setTerminal(term);

        // Connect WebSocket to SSH endpoint
        connectToSSH(term, sshSession);
      }
    } catch (error) {
      console.error('Failed to initialize SSH session:', error);
    }
  };

  const connectToSSH = (term: Terminal, session: any) => {
    const ws = new WebSocket(session.websocket_url);
    ws.onmessage = (event) => term.write(event.data);
    term.onData((data) => ws.send(data));
  };

  const handleDownloadKey = async () => {
    if (session) {
      await downloadPrivateKey(session.id);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50">
      <div className="bg-slate-900 rounded-lg w-full max-w-4xl h-[600px] flex flex-col">
        <div className="flex justify-between items-center p-4 border-b border-slate-700">
          <h3 className="text-lg font-semibold text-white">SSH Terminal</h3>
          <div className="flex gap-2">
            <button onClick={handleDownloadKey} className="px-3 py-1 bg-blue-600 rounded">
              Download Key
            </button>
            <button onClick={onClose} className="px-3 py-1 bg-slate-700 rounded">
              Close
            </button>
          </div>
        </div>
        <div ref={terminalRef} className="flex-1 p-2" />
        {session && (
          <div className="p-2 border-t border-slate-700 text-sm text-slate-400">
            Host: {session.host} | Port: {session.port} | User: {session.username}
          </div>
        )}
      </div>
    </div>
  );
}
```

#### 2.2 Create Validation Access Modal
**File**: `frontend/src/components/dve-rental/validation-access-modal.tsx`

```tsx
'use client';

import React, { useState } from 'react';
import { useValidationSession } from '@/hooks/use-validation-session';

interface ValidationAccessModalProps {
  rentalId: string;
  onClose: () => void;
}

export default function ValidationAccessModal({ rentalId, onClose }: ValidationAccessModalProps) {
  const { createSession, submitValidation } = useValidationSession();
  const [content, setContent] = useState('');
  const [validationType, setValidationType] = useState('reasoning');
  const [result, setResult] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const handleValidate = async () => {
    setLoading(true);
    try {
      const session = await createSession(rentalId);
      const validationResult = await submitValidation(session.id, {
        content,
        type: validationType,
      });
      setResult(validationResult);
    } catch (error) {
      console.error('Validation failed:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50">
      <div className="bg-slate-900 rounded-lg w-full max-w-4xl p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Reasoning Validation</h3>

        <div className="space-y-4">
          <div>
            <label className="block text-sm text-slate-400 mb-1">Validation Type</label>
            <select
              value={validationType}
              onChange={(e) => setValidationType(e.target.value)}
              className="w-full bg-slate-800 rounded p-2"
            >
              <option value="reasoning">Reasoning Chain</option>
              <option value="proof">Mathematical Proof</option>
              <option value="logic">Logical Consistency</option>
            </select>
          </div>

          <div>
            <label className="block text-sm text-slate-400 mb-1">Content to Validate</label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              className="w-full h-48 bg-slate-800 rounded p-2"
              placeholder="Enter reasoning or proof to validate..."
            />
          </div>

          <button
            onClick={handleValidate}
            disabled={loading}
            className="w-full py-2 bg-green-600 rounded font-semibold"
          >
            {loading ? 'Validating...' : 'Validate'}
          </button>

          {result && (
            <div className="mt-4 p-4 bg-slate-800 rounded">
              <h4 className="font-semibold mb-2">Validation Result</h4>
              <div className={`text-lg ${result.valid ? 'text-green-400' : 'text-red-400'}`}>
                {result.valid ? '✓ Valid' : '✗ Invalid'}
              </div>
              <p className="text-sm text-slate-400 mt-2">{result.explanation}</p>
            </div>
          )}
        </div>

        <div className="flex justify-end mt-6">
          <button onClick={onClose} className="px-4 py-2 bg-slate-700 rounded">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
```

#### 2.3 Create Error Resolution Modal
**File**: `frontend/src/components/dve-rental/error-resolution-modal.tsx`

```tsx
'use client';

import React, { useState } from 'react';
import { useErrorResolutionSession } from '@/hooks/use-error-resolution-session';

interface ErrorResolutionModalProps {
  rentalId: string;
  onClose: () => void;
}

export default function ErrorResolutionModal({ rentalId, onClose }: ErrorResolutionModalProps) {
  const { createSession, analyzeError } = useErrorResolutionSession();
  const [errorInput, setErrorInput] = useState('');
  const [errorType, setErrorType] = useState('runtime');
  const [analysis, setAnalysis] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const handleAnalyze = async () => {
    setLoading(true);
    try {
      const session = await createSession(rentalId);
      const result = await analyzeError(session.id, {
        error: errorInput,
        type: errorType,
      });
      setAnalysis(result);
    } catch (error) {
      console.error('Analysis failed:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50">
      <div className="bg-slate-900 rounded-lg w-full max-w-4xl p-6 max-h-[90vh] overflow-y-auto">
        <h3 className="text-lg font-semibold text-white mb-4">Error Resolution</h3>

        <div className="space-y-4">
          <div>
            <label className="block text-sm text-slate-400 mb-1">Error Type</label>
            <select
              value={errorType}
              onChange={(e) => setErrorType(e.target.value)}
              className="w-full bg-slate-800 rounded p-2"
            >
              <option value="runtime">Runtime Error</option>
              <option value="logic">Logic Error</option>
              <option value="syntax">Syntax Error</option>
              <option value="inference">Inference Error</option>
            </select>
          </div>

          <div>
            <label className="block text-sm text-slate-400 mb-1">Error Details</label>
            <textarea
              value={errorInput}
              onChange={(e) => setErrorInput(e.target.value)}
              className="w-full h-32 bg-slate-800 rounded p-2 font-mono text-sm"
              placeholder="Paste error message or describe the issue..."
            />
          </div>

          <button
            onClick={handleAnalyze}
            disabled={loading}
            className="w-full py-2 bg-orange-600 rounded font-semibold"
          >
            {loading ? 'Analyzing...' : 'Analyze Error'}
          </button>

          {analysis && (
            <div className="mt-4 space-y-4">
              <div className="p-4 bg-slate-800 rounded">
                <h4 className="font-semibold text-red-400 mb-2">Root Cause</h4>
                <p className="text-sm">{analysis.root_cause}</p>
              </div>

              <div className="p-4 bg-slate-800 rounded">
                <h4 className="font-semibold text-yellow-400 mb-2">Analysis</h4>
                <p className="text-sm">{analysis.explanation}</p>
              </div>

              <div className="p-4 bg-slate-800 rounded">
                <h4 className="font-semibold text-green-400 mb-2">Resolution Steps</h4>
                <ol className="list-decimal list-inside text-sm space-y-1">
                  {analysis.steps?.map((step: string, i: number) => (
                    <li key={i}>{step}</li>
                  ))}
                </ol>
              </div>

              {analysis.code_fix && (
                <div className="p-4 bg-slate-800 rounded">
                  <h4 className="font-semibold text-blue-400 mb-2">Suggested Fix</h4>
                  <pre className="text-sm bg-slate-900 p-2 rounded overflow-x-auto">
                    {analysis.code_fix}
                  </pre>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex justify-end mt-6">
          <button onClick={onClose} className="px-4 py-2 bg-slate-700 rounded">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
```

### Phase 3: Container and Endpoint Integration (Priority: HIGH)

#### 3.1 Fix Container Orchestrator Initialization
**File**: `backend/cmd/backend_server/main.go`

Ensure container orchestrator is properly initialized:
```go
// In main() after database initialization
containerOrchestrator, err := container.NewOrchestrator(container.Config{
    Runtime:     detectContainerRuntime(), // "native-go" for Kali, "podman" otherwise
    NetworkMode: "bridge",
    TEEService:  teeSecurityService,
})
if err != nil {
    log.Fatalf("Failed to initialize container orchestrator: %v", err)
}

// Pass to rental service
rentalService := dverental.NewDVERentalService(db, containerOrchestrator, sessionManager)
```

#### 3.2 Implement SSH Key Generation in Container
**File**: `backend/internal/services/container/orchestrator.go`

```go
func (co *ContainerOrchestrator) ProvisionContainer(spec ContainerSpec) (*ContainerInfo, error) {
    // ... existing provisioning code ...

    // Generate SSH key pair
    privateKey, publicKey, err := generateSSHKeyPair()
    if err != nil {
        return nil, fmt.Errorf("failed to generate SSH keys: %w", err)
    }

    // Store private key for later retrieval
    co.sshKeys[containerID] = privateKey

    // Inject public key into container
    err = co.injectPublicKey(containerID, publicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to inject SSH key: %w", err)
    }

    return &ContainerInfo{
        ID:         containerID,
        SSHPort:    sshPort,
        SSHHost:    co.getContainerHost(containerID), // Not localhost!
        SSHUser:    "dve",
        // ...
    }, nil
}

func (co *ContainerOrchestrator) GetSSHPrivateKey(containerID string) ([]byte, error) {
    co.mu.RLock()
    defer co.mu.RUnlock()

    if key, ok := co.sshKeys[containerID]; ok {
        return key, nil
    }
    return nil, errors.New("SSH key not found for container")
}
```

#### 3.3 Add Terminate Session Handlers
**File**: `backend/internal/web/dve_rental_handlers.go`

```go
// TerminateValidationSession terminates an active validation session
func (h *DVERentalHandlers) TerminateValidationSession(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    rentalID := vars["id"]

    userID, _ := auth.GetUserIDFromContext(r.Context())

    // Validate ownership
    rental, err := h.rentalService.GetRental(rentalID)
    if err != nil || rental.UserID != userID {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Terminate session
    err = h.sessionManager.TerminateValidationSession(rental.ValidationSessionID)
    if err != nil {
        http.Error(w, "Failed to terminate session", http.StatusInternalServerError)
        return
    }

    // Update rental
    rental.ValidationSessionID = ""
    h.rentalService.UpdateRental(rental)

    w.WriteHeader(http.StatusNoContent)
}

// Similar for TerminateErrorResolutionSession
```

### Phase 4: Testing and Validation (Priority: MEDIUM)

#### 4.1 Integration Tests
```go
// backend/tests/dve_access_test.go
func TestFullDVEAccessFlow(t *testing.T) {
    // 1. Create rental
    // 2. Create SSH session
    // 3. Download private key
    // 4. Verify SSH connection works
    // 5. Create validation session
    // 6. Submit validation request
    // 7. Create error resolution session
    // 8. Submit error for analysis
    // 9. Terminate all sessions
    // 10. Cancel rental
}
```

#### 4.2 Frontend E2E Tests
```typescript
// frontend/tests/dve-access.spec.ts
test('complete DVE access flow', async ({ page }) => {
    // Login
    // Navigate to DVE rental
    // Create rental
    // Click "Access" button
    // Verify modal opens
    // Test SSH terminal
    // Test validation interface
    // Test error resolution interface
});
```

---

## Risk Assessment

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| SSH Key Exposure | Critical | Medium | Encrypt keys at rest, short expiry |
| Session Hijacking | High | Low | Secure tokens, IP binding |
| Container Escape | Critical | Low | TEE enforcement, Kali hardening |
| DoS via Sessions | Medium | Medium | Rate limiting, resource quotas |
| Data Loss on Restart | High | High | Database persistence |

---

## Success Criteria

### Functional Requirements
- [ ] Users can click "Access CDE" and see modal with all access options
- [ ] SSH access provides working terminal with key download
- [ ] Validation interface accepts input and returns results
- [ ] Error resolution interface analyzes errors and provides solutions
- [ ] All sessions persist across server restarts
- [ ] Sessions can be cleanly terminated

### Non-Functional Requirements
- [ ] SSH connection established < 3 seconds
- [ ] Validation response < 5 seconds
- [ ] Error analysis response < 10 seconds
- [ ] 99.9% uptime for DVE services
- [ ] All sessions encrypted in transit and at rest

---

## Implementation Timeline

| Phase | Tasks | Duration | Dependencies |
|-------|-------|----------|--------------|
| Phase 1 | Private key endpoint, session manager methods, persistence | 3 days | None |
| Phase 2 | Frontend modal components | 2 days | Phase 1 |
| Phase 3 | Container/endpoint integration | 3 days | Phase 1 |
| Phase 4 | Testing and validation | 2 days | Phase 2, 3 |

**Total Estimated Time**: 10 days

---

## Files to Create/Modify

### New Files
```
frontend/src/components/dve-rental/ssh-access-modal.tsx
frontend/src/components/dve-rental/validation-access-modal.tsx
frontend/src/components/dve-rental/error-resolution-modal.tsx
backend/tests/dve_access_test.go
frontend/tests/dve-access.spec.ts
```

### Modified Files
```
backend/internal/web/dve_rental_handlers.go         # Add private key + terminate handlers
backend/internal/services/session/session_manager.go # Add validation/error session methods
backend/internal/services/container/orchestrator.go  # Add SSH key management
backend/cmd/backend_server/main.go                   # Ensure proper initialization
frontend/src/components/dve-rental/dve-access-flow.tsx # Import new modals
```

---

**Report Generated**: November 21, 2025
**Analysis Depth**: Full codebase examination
**Analyst**: Claude Code
