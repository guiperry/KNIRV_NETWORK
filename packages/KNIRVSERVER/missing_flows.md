# Missing Flows — Codebase vs Specification Gap Analysis

_Generated: 2026-04-20_
_Document: Cross-reference of `USER_WORKFLOWS_AND_PRODUCTION_PLAN.md` vs current codebase_

---

## Executive Summary

This document identifies all UI components and backend wiring that are specified in `USER_WORKFLOWS_AND_PRODUCTION_PLAN.md` but are either missing entirely or incomplete in the current codebase. The gaps are organized by severity and workflow phase.

---

## 1. Critical Blockers (Must-Fix Before Production)

### 1.1 Security — Testnet Hardcoded Tokens

**Files containing hardcoded test tokens:**
- `frontend/src/app/login/page.tsx` (lines 25-27)
- `frontend/src/lib/auth-context.tsx` (lines 100-116, 240-256)
- `desktop/renderer.js` (lines 385-387)
- `frontend/renderer.js` (lines 378-380)
- `backend_server/internal/web/middleware/auth.go` (lines 355-367)

**Example of hardcoded tokens in backend_server/internal/web/middleware/auth.go:**
```go
// testnetTokens maps development testnet tokens to their auth contexts.
// These tokens are only for local development and match the frontend's hardcoded testnet tokens.
var testnetTokens = map[string]*AuthContext{
	"TESTNET_ADMIN_TOKEN": {
		UserID:   "testnet-admin",
		Username: "testnet-admin",
		Role:     "admin",
	},
	"TESTNET_VALIDATOR_TOKEN": {
		UserID:   "testnet-validator",
		Username: "testnet-validator",
		Role:     "validator",
	},
	"TESTNET_OBSERVER_TOKEN": {
		UserID:   "testnet-observer",
		Username: "testnet-observer",
		Role:     "observer",
	},
}
```

**Missing:**
- Production build flag to gate these tokens
- Environment-based token configuration
- Removal from production builds

**Implementation needed:**
```go
// Instead of hardcoded tokens, use environment-based configuration
var testnetTokens map[string]*AuthContext

func init() {
	// Only load testnet tokens in development/test environments
	if os.Getenv("ENVIRONMENT") == "development" || os.Getenv("ENVIRONMENT") == "testnet" {
		testnetTokens = map[string]*AuthContext{
			"TESTNET_ADMIN_TOKEN": {
				UserID:   "testnet-admin",
				Username: "testnet-admin",
				Role:     "admin",
			},
			// ... other tokens
		}
	}
}

// In getTestnetAuthContext function:
func getTestnetAuthContext(token string) *AuthContext {
	// Return nil in production to disable testnet tokens
	if os.Getenv("ENVIRONMENT") == "production" {
		return nil
	}
	
	if ctx, ok := testnetTokens[token]; ok {
		return &AuthContext{
			UserID:   ctx.UserID,
			Username: ctx.Username,
			Role:     ctx.Role,
		}
	}
	return nil
}
```

### 1.2 Security — AuthRequired Enforcement

**Specified:** `config/production.yaml` must set `security.auth_required: true`

**Missing:**
- Verify `config.Security.AuthRequired` cannot be overridden at runtime without admin auth
- Confirm all HTTP routes check this flag before allowing unauthenticated access

**Example implementation:**
```go
// config/config.go
package config

type Config struct {
	Security SecurityConfig `json:"security"`
	// ... other config sections
}

type SecurityConfig struct {
	AuthRequired bool `json:"auth_required"`
	// ... other security settings
}

// Load configuration from file and environment variables
func LoadConfig() (*Config, error) {
	// Implementation would load from config file and env vars
	// This is a simplified example
	return &Config{
		Security: SecurityConfig{
			AuthRequired: true, // Would be loaded from config/production.yaml
		},
	}, nil
}

// Middleware to enforce auth requirement
func AuthRequiredMiddleware(cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In production, if auth is required, check for valid token
			if cfg.Security.AuthRequired && os.Getenv("ENVIRONMENT") == "production" {
				// Extract token from Authorization header
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					http.Error(w, "Authorization required", http.StatusUnauthorized)
					return
				}

				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) != 2 || parts[0] != "Bearer" {
					http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
					return
				}

				tokenString := parts[1]
				
				// Validate token (implementation depends on your auth system)
				// if !authService.ValidateToken(tokenString) {
				//     http.Error(w, "Invalid token", http.StatusUnauthorized)
				//     return
				// }
				
				// For this example, we'll just check if token is not empty
				if tokenString == "" {
					http.Error(w, "Invalid token", http.StatusUnauthorized)
					return
				}
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// Usage in main.go or route setup:
// cfg, err := config.LoadConfig()
// if err != nil {
//     log.Fatal(err)
// }
// r.Use(middleware.AuthRequiredMiddleware(cfg))
```

### 1.3 Authentication — Email Verification

**Specified:** `POST /api/auth/verify-email` flow

**Status:** Route exists but verify:
- SMTP credentials configured in production config
- Frontend verification UX wired end-to-end
- Email sender properly configured

**Example implementation in handlers:**
```go
// POST /api/auth/verify-email
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate token and mark email as verified
	if err := h.authService.VerifyEmail(req.Token); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Email verified successfully"})
}

// Email service implementation example
func (s *EmailService) SendVerificationEmail(to string, token string) error {
	// Configure SMTP from production config
	smtpHost := config.GetString("email.smtp_host")
	smtpPort := config.GetInt("email.smtp_port")
	smtpUser := config.GetString("email.smtp_user")
	smtpPass := config.GetString("email.smtp_pass")
	
	// Send email with verification link containing token
	// Implementation would use net/smtp or a library like go-gomail
}
```

### 1.4 Authentication — Password Reset Flow

**Specified:** `POST /api/auth/forgot-password` / `POST /api/auth/reset-password`

**Missing:** Verify these routes exist or implement them

**Example implementation:**
```go
// POST /api/auth/forgot-password
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate reset token and send email
	if err := h.authService.InitiatePasswordReset(req.Email); err != nil {
		// Don't reveal whether email exists for security
		http.Error(w, "If the email exists, a reset link has been sent", http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "If the email exists, a reset link has been sent"})
}

// POST /api/auth/reset-password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate token and reset password
	if err := h.authService.CompletePasswordReset(req.Token, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password reset successfully"})
}

// Password reset service example
func (s *AuthService) InitiatePasswordReset(email string) error {
	// Find user by email
	user, err := s.db.GetUserByEmail(email)
	if err != nil {
		// Log internally but don't expose to user
		return nil
	}

	// Generate secure token
	token := generateSecureToken()
	
	// Store token with expiration (e.g., 1 hour)
	err = s.db.SavePasswordResetToken(user.ID, token, time.Now().Add(time.Hour))
	if err != nil {
		return err
	}

	// Send email with reset link
	return s.emailService.SendPasswordResetEmail(user.Email, token)
}

func (s *AuthService) CompletePasswordReset(token, password string) error {
	// Validate token
	userID, err := s.db.ValidatePasswordResetToken(token)
	if err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password and delete token
	return s.db.UpdatePassword(userID, hashedPassword, token)
}
```

### 1.5 Authentication — Token Revocation on Logout

**Specified:** `POST /api/auth/revoke`

**Missing:**
- Verify frontend calls this on logout
- Confirm in-memory revocation list is populated

**Example implementation:**
```go
// POST /api/auth/revoke
func (h *AuthHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
		return
	}

	tokenString := parts[1]

	// Revoke the token
	if err := h.authMiddleware.RevokeToken(tokenString); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Token revoked successfully"})
}

// Frontend example (TypeScript)
async function handleLogout() {
  try {
    const response = await fetch('/api/auth/revoke', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json',
      },
    });

    if (response.ok) {
      // Clear local storage
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      
      // Redirect to login
      window.location.href = '/login';
    } else {
      console.error('Logout failed');
    }
  } catch (error) {
    console.error('Logout error:', error);
  }
}
```

### 1.6 Stripe/PayPal Webhook Verification

**Specified:**
- Verify `Stripe-Signature` header on Stripe callbacks
- Verify PayPal IPN/webhook callbacks

**Missing:** Confirm webhook signature verification is implemented

**Example implementation for Stripe:**
```go
// POST /api/v1/payments/stripe/webhook
func (h *PaymentHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	stripeSignature := r.Header.Get("Stripe-Signature")
	if stripeSignature == "" {
		http.Error(w, "Missing Stripe signature header", http.StatusBadRequest)
		return
	}

	// Verify webhook signature
	event, err := webhook.ConstructEvent(payload, stripeSignature, config.GetString("stripe.webhook_secret"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error verifying webhook signature: %v", err), http.StatusBadRequest)
		return
	}

	// Handle the event
	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			http.Error(w, "Error parsing webhook JSON", http.StatusBadRequest)
			return
		}
		// Handle successful payment
	case "payment_intent.payment_failed":
		// Handle failed payment
		// ...
	}

	w.WriteHeader(http.StatusOK)
}

// Example implementation for PayPal
func (h *PaymentHandler) PayPalWebhook(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify PayPal webhook signature
	authAlgo := r.Header.Get("Paypal-Auth-Algo")
	transmissionID := r.Header.Get("Paypal-Transmission-ID")
	certURL := r.Header.Get("Paypal-Cert-Url")
	authSignature := r.Header.Get("Paypal-Auth-Signature")
	transmissionSig := r.Header.Get("Paypal-Transmission-Sig")
	transmissionTime := r.Header.Get("Paypal-Transmission-Time")
	webhookID := config.GetString("paypal.webhook_id")

	// In a real implementation, you would verify these headers against PayPal's certificates
	// This is a simplified example
	if authAlgo != "SHA256withRSA" || transmissionID == "" || certURL == "" || 
	   authSignature == "" || transmissionSig == "" || transmissionTime == "" {
		http.Error(w, "Missing required PayPal webhook headers", http.StatusBadRequest)
		return
	}

	// Verify the webhook ID matches
	if payload["webhook_id"] != webhookID {
		http.Error(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	// Process the webhook event
	eventType := payload["event_type"].(string)
	switch eventType {
	case "PAYMENT.SALE.COMPLETED":
		// Handle completed payment
	case "PAYMENT.SALE.DENIED":
		// Handle denied payment
		// ...
	}

	w.WriteHeader(http.StatusOK)
}
```

### 1.7 NRN Transaction Idempotency

**Specified:** Duplicate NRN transfer submissions must be detected

**Missing:** Verify idempotency logic exists in `NRNPaymentHandlers`

**Example implementation:**
```go
// In NRNPaymentHandlers or TransactionService
func (s *TransactionService) TransferNRN(ctx context.Context, req *TransferNRNRequest) (*TransferNRNResponse, error) {
	// Generate idempotency key from request parameters if not provided
	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		// Create a deterministic key based on sender, recipient, amount, and timestamp
		keyData := fmt.Sprintf("%s:%s:%f:%d", req.SenderID, req.RecipientID, req.Amount, time.Now().Unix())
		hash := sha256.Sum256([]byte(keyData))
		idempotencyKey = hex.EncodeToString(hash[:])
	}

	// Check if we've already processed this idempotency key
	existingTx, err := s.db.GetTransactionByIdempotencyKey(idempotencyKey)
	if err != nil && err != buntdb.ErrNotFound {
		return nil, fmt.Errorf("error checking idempotency: %w", err)
	}
	if existingTx != nil {
		// Return existing transaction instead of creating a duplicate
		return &TransferNRNResponse{
			TxHash:   existingTx.TxHash,
			Status:   existingTx.Status,
			Amount:   existingTx.Amount,
			CreatedAt: existingTx.CreatedAt,
		}, nil
	}

	// Process the transfer (create transaction, validate, etc.)
	tx, err := s.processNRNTransfer(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store the transaction with idempotency key for future reference
	if err := s.db.SaveTransactionWithIdempotencyKey(tx, idempotencyKey); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	return &TransferNRNResponse{
		TxHash:   tx.TxHash,
		Status:   tx.Status,
		Amount:   tx.Amount,
		CreatedAt: tx.CreatedAt,
	}, nil
}

// Database helper functions
func (db *BuntDBManager) GetTransactionByIdempotencyKey(key string) (*Transaction, error) {
	var tx *Transaction
	err := db.ViewTransaction(func(tx *buntdb.Tx) error {
		val, err := tx.Get("idempotency_key:" + key)
		if err != nil {
			return err
		}
		if val == "" {
			return buntdb.ErrNotFound
		}
		var storedTx Transaction
		if err := json.Unmarshal([]byte(val), &storedTx); err != nil {
			return err
		}
		tx = &storedTx
		return nil
	})
	return tx, err
}

func (db *BuntDBManager) SaveTransactionWithIdempotencyKey(tx *Transaction, key string) error {
	return db.Transaction(func(tx *buntdb.Tx) error {
		// Store the transaction
		txData, err := json.Marshal(tx)
		if err != nil {
			return err
		}
		
		// Store with expiration (e.g., 24 hours) to prevent infinite growth
		expiration := time.Now().Add(24 * time.Hour)
		_, _, err = tx.Set(
			"idempotency_key:"+key, 
			string(txData), 
			&buntdb.SetOptions{Expires: true, TTL: uint32(expiration.Unix())},
		)
		return err
	})
}
```

---

## 2. High-Priority Items (Pre-Launch Polish)

### 2.1 Frontend — Demo Mode Gating

**Specified:** `demo-mode-toggle.tsx` disables real API calls

**Missing:**
- Confirm toggle is not accessible to non-admin roles in production
- Verify all demo mode code checks role before allowing access

**Example implementation (TypeScript):**
```tsx
// demo-mode-toggle.tsx
import { useAuth } from '@/lib/auth-context';
import { useState } from 'react';

export function DemoModeToggle() {
  const { user, isLoading } = useAuth();
  const [isDemoMode, setIsDemoMode] = useState(false);

  // Only show toggle to admin users
  if (isLoading || !user || user.role !== 'admin') {
    return null;
  }

  const toggleDemoMode = async () => {
    setIsDemoMode(!isDemoMode);
    
    // In demo mode, we mock API calls instead of making real ones
    // This would be implemented in your API service layer
    if (isDemoMode) {
      enableDemoMode();
    } else {
      disableDemoMode();
    }
  };

  return (
    <div className="demo-mode-toggle">
      <label>
        <input
          type="checkbox"
          checked={isDemoMode}
          onChange={toggleDemoMode}
        />
        Demo Mode (Mock API Calls)
      </label>
      {!isDemoMode && (
        <span className="demo-mode-warning">
          Warning: Demo mode uses mock data and does not make real API calls
        </span>
      )}
    </div>
  );
}

// API service example showing demo mode check
export class ApiService {
  private isDemoMode = false;

  setDemoMode(enabled: boolean) {
    this.isDemoMode = enabled;
  }

  async fetch<T>(endpoint: string, options?: RequestInit): Promise<T> {
    // Check if demo mode is enabled
    if (this.isDemoMode) {
      return this.mockResponse(endpoint, options);
    }

    // Make real API call
    const response = await fetch(`/api${endpoint}`, options);
    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }
    return response.json();
  }

  private mockResponse(endpoint: string, options?: RequestInit): Promise<any> {
    // Return mock data based on endpoint
    return new Promise((resolve) => {
      setTimeout(() => {
        // Mock responses for different endpoints
        if (endpoint.includes('/auth/me')) {
          resolve({ id: 'demo-user', username: 'demo', role: 'admin' });
        } else if (endpoint.includes('/dve/nodes')) {
          resolve({ nodes: [] });
        }
        // Add more mock responses as needed
        resolve({});
      }, 500); // Simulate network delay
    });
  }
}
```

### 2.2 Frontend — Error Boundary Coverage

**Specified:** All major panels have React error boundaries

**Missing:**
- Verify `error.tsx` exists at app level
- Confirm each panel has error boundary to prevent full-page crashes

**Example implementation:**
```tsx
// error.tsx
import { Component, ReactNode } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error?: Error;
  errorInfo?: Component.ErrorInfo;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  public state: ErrorBoundaryState = {
    hasError: false,
  };

  public static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true };
  }

  public componentDidCatch(error: Error, errorInfo: Component.ErrorInfo) {
    console.error('Uncaught error:', error, errorInfo);
    this.setState({
      hasError: true,
      error,
      errorInfo,
    });
  }

  public render() {
    if (this.state.hasError) {
      // You can render any custom fallback UI
      return this.props.fallback ?? (
        <div className="error-boundary">
          <h2>Something went wrong.</h2>
          <details style={{ whiteSpace: 'pre-wrap' }}>
            {this.state.error && this.state.error.toString()}
          </details>
        </div>
      );
    }

    return this.props.children;
  }
}

// Usage in a page or component:
// import { ErrorBoundary } from './error';
//
// export function MyPanel() {
//   return (
//     <ErrorBoundary fallback={<div>Loading panel...</div>}>
//       <PanelContent />
//     </ErrorBoundary>
//   );
// }

// Alternative: using function component with hooks (requires react-error-boundary package)
// import { ErrorBoundary } from 'react-error-boundary';
//
// export function MyPanel() {
//   return (
//     <ErrorBoundary FallbackComponent={ErrorFallback}>
//       <PanelContent />
//     </ErrorBoundary>
//   );
// }
//
// function ErrorFallback({ error, resetErrorBoundary }: { error: Error; resetErrorBoundary: () => void }) {
//   return (
//     <div role="alert">
//       <p>Something went wrong:</p>
//       <pre>{error.message}</pre>
//       <button onClick={resetErrorBoundary}>Try again</button>
//     </div>
//   );
// }
```

### 2.3 Frontend — WebSocket Reconnect Logic

**Specified:** `use-knirv-socket.ts` implements exponential backoff reconnect

**Missing:** Verify exponential backoff reconnect logic is implemented

**Example implementation (TypeScript):**
```tsx
// use-knirv-socket.ts
import { useEffect, useRef, useState, useCallback } from 'react';

export function useKnirvSocket(url: string, onMessage: (data: any) => void) {
  const socketRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const [isConnected, setIsConnected] = useState(false);
  
  // Exponential backoff configuration
  const BASE_DELAY = 1000; // 1 second
  const MAX_DELAY = 30000; // 30 seconds
  const MAX_RETRIES = 10;

  const connect = useCallback(() => {
    // Close existing connection if any
    if (socketRef.current) {
      socketRef.current.close();
    }

    try {
      const socket = new WebSocket(url);
      socketRef.current = socket;

      socket.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        reconnectAttemptsRef.current = 0; // Reset on successful connection
      };

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          onMessage(data);
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      socket.onclose = (event) => {
        console.log('WebSocket disconnected', event);
        setIsConnected(false);
        
        // Attempt reconnection with exponential backoff
        if (reconnectAttemptsRef.current < MAX_RETRIES) {
          const delay = Math.min(
            BASE_DELAY * Math.pow(2, reconnectAttemptsRef.current),
            MAX_DELAY
          );
          
          reconnectAttemptsRef.current++;
          
          setTimeout(() => {
            console.log(`Reconnecting in ${delay}ms... (attempt ${reconnectAttemptsRef.current})`);
            connect();
          }, delay);
        } else {
          console.error('Max reconnection attempts reached');
        }
      };

      socket.onerror = (error) => {
        console.error('WebSocket error:', error);
      };
    } catch (error) {
      console.error('Error creating WebSocket:', error);
      setIsConnected(false);
    }
  }, [url, onMessage]);

  useEffect(() => {
    connect();
    
    // Cleanup on unmount
    return () => {
      if (socketRef.current) {
        socketRef.current.close();
      }
    };
  }, [connect]);

  const send = useCallback((data: any) => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify(data));
    } else {
      console.warn('WebSocket is not connected. Cannot send data.');
    }
  }, []);

  return { isConnected, send };
}
```

### 2.4 Frontend — Loading and Empty States

**Specified:** All data panels have loading spinners and empty-state messaging

**Missing:** Audit all panels for proper loading/empty states

**Example implementation:**
```tsx
// LoadingSpinner.tsx
import { useState } from 'react';

export function LoadingSpinner({ 
  isLoading, 
  message = 'Loading...',
  size = 'medium'
}: { 
  isLoading: boolean; 
  message?: string; 
  size?: 'small' | 'medium' | 'large'
}) {
  if (!isLoading) return null;

  const sizeMap = {
    small: 'w-4 h-4',
    medium: 'w-6 h-6',
    large: 'w-8 h-8'
  };

  return (
    <div className="flex items-center space-x-3">
      <div className={`animate-spin rounded-full border-2 border-t-blue-500 border-b-blue-500 ${sizeMap[size]}`}></div>
      <span className="text-sm">{message}</span>
    </div>
  );
}

// EmptyState.tsx
export function EmptyState({ 
  title, 
  description, 
  icon = '📭',
  actionLabel,
  onAction
}: { 
  title: string; 
  description: string; 
  icon?: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <div className="text-center py-12">
      <div className="text-4xl mb-4">{icon}</div>
      <h3 className="text-lg font-semibold mb-2">{title}</h3>
      <p className="text-gray-600 mb-6">{description}</p>
      {actionLabel && onAction && (
        <button 
          onClick={onAction}
          className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}

// Usage in a component:
// import { LoadingSpinner, EmptyState } from './components';
//
// export function DataPanel({ data, isLoading, fetchData }) {
//   if (isLoading) {
//     return <LoadingSpinner isLoading message="Fetching data..." />;
//   }
//
//   if (!data || data.length === 0) {
//     return (
//       <EmptyState
//         title="No Data Available"
//         description="There is currently no data to display. Try refreshing or adding new data."
//         actionLabel="Refresh Data"
//         onAction={fetchData}
//       />
//     );
//   }
//
//   return (
//     <div>
//       {/* Render data here */
//       <DataList data={data} />
//     </div>
//   );
// }
```

### 2.5 Role-Based Access Enforcement

**Specified:** Backend routes enforce role checks (not just RequireAuth)

**Missing:**
- Verify backend inspects role from JWT claims for admin-only operations
- Confirm `role-guard.tsx` is properly wired with backend enforcement

**Example backend implementation:**
```go
// In your API route registration
func RegisterRoutes(router *chi.Mux, authMiddleware *middleware.AuthMiddleware) {
	// Public routes
	router.Post("/api/auth/login", authHandler.Login)
	router.Post("/api/auth/register", authHandler.Register)
	router.Post("/api/auth/verify-email", authHandler.VerifyEmail)
	
	// Protected routes requiring authentication
	router.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Get("/api/auth/me", authHandler.GetCurrentUser)
		r.Post("/api/dve/nodes", dveHandler.CreateDVENode) // Requires auth but any role
		
		// Admin-only routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireRole("admin"))
			r.Post("/api/guardrail/policies", guardrailHandler.CreatePolicy)
			r.Post("/api/guardrail/policies/{id}/commit", guardrailHandler.CommitPolicy)
			r.Get("/api/system-health/*", healthHandler.GetSystemHealth)
			r.Post("/api/v1/onboarding/organizations", onboardingHandler.CreateOrganization)
		})
	})
}

// Role guard frontend component (TypeScript)
import { useAuth } from '@/lib/auth-context';

export function RoleGuard({ children, requiredRole }: { children: React.ReactNode; requiredRole: string }) {
  const { user, isLoading } = useAuth();
  
  if (isLoading) {
    return <div>Loading...</div>;
  }
  
  if (!user) {
    return <div>Please log in to access this feature</div>;
  }
  
  // Check if user has required role or is admin (admins can access everything)
  if (user.role !== requiredRole && user.role !== 'admin') {
    return <div>You don't have permission to access this feature</div>;
  }
  
  return children;
}

// Usage in a component
export function AdminOnlyPage() {
  return (
    <RoleGuard requiredRole="admin">
      <h1>Admin Dashboard</h1>
      {/* Admin-only content */}
    </RoleGuard>
  );
}
```

### 2.6 Operations — Structured Logging

**Specified:** All services use structured (JSON) logging

**Missing:** Verify consistent logging fields (service name, correlation ID, severity)

**Example implementation:**
```go
// logging/logger.go
package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel defines the severity level of a log entry
type LogLevel string

const (
	DEBUG  LogLevel = "DEBUG"
	INFO   LogLevel = "INFO"
	WARN   LogLevel = "WARN"
	ERROR  LogLevel = "ERROR"
	FATAL  LogLevel = "FATAL"
)

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// JSONLogger implements Logger with JSON output
type JSONLogger struct {
	writer io.Writer
	serviceName string
}

// NewJSONLogger creates a new JSON logger
func NewJSONLogger(serviceName string, writer io.Writer) *JSONLogger {
	if writer == nil {
		writer = os.Stdout
	}
	return &JSONLogger{
		writer:     writer,
		serviceName: serviceName,
	}
}

// log creates a JSON log entry
func (l *JSONLogger) log(level LogLevel, msg string, fields []Field) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     string(level),
		"service":   l.serviceName,
		"message":   msg,
	}
	
	// Add custom fields
	for _, field := range fields {
		entry[field.Key] = field.Value
	}
	
	// Add correlation ID if present (would typically come from middleware/context)
	// This is just an example - in practice you'd get this from request context
	if correlationID, ok := l.getCorrelationID(); ok {
		entry["correlation_id"] = correlationID
	}
	
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling log entry: %v\n", err)
		return
	}
	
	fmt.Fprintln(l.writer, string(jsonData))
}

// Helper methods for each log level
func (l *JSONLogger) Debug(msg string, fields ...Field) {
	l.log(DEBUG, msg, fields)
}

func (l *JSONLogger) Info(msg string, fields ...Field) {
	l.log(INFO, msg, fields)
}

func (l *JSONLogger) Warn(msg string, fields ...Field) {
	l.log(WARN, msg, fields)
}

func (l *JSONLogger) Error(msg string, fields ...Field) {
	l.log(ERROR, msg, fields)
}

func (l *JSONLogger) Fatal(msg string, fields ...Field) {
	l.log(FATAL, msg, fields)
	os.Exit(1)
}

// getCorrelationID is a placeholder - in practice this would come from request context
func (l *JSONLogger) getCorrelationID() (string, bool) {
	// Example implementation - in reality this would come from middleware
	// that extracts correlation ID from headers or generates one
	return "", false
}

// Example usage in a service:
//
// func NewDVEService(logger logging.Logger) *DVEService {
//     return &DVEService{
//         logger: logger,
//     }
// }
//
// func (s *DVEService) CreateDVE(node *DVENode) error {
//     s.logger.Info("Creating DVE node",
//         logging.Field{"node_id", node.ID},
//         logging.Field{"node_name", node.Name},
//         logging.Field{"cpu_limit", node.CPULimit},
//         logging.Field{"memory_limit", node.MemoryLimit},
//     )
//     // ... implementation
//     return nil
// }
```

### 2.7 Operations — Health Check Completeness

**Specified:** `GET /health` and `GET /api/health` return structured JSON

**Missing:** Confirm `SystemHealthService` covers all critical services

**Example implementation:**
```go
// health/health_service.go
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthChecker defines the interface for health checks
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Service string    `json:"service"`
	Status  string    `json:"status"` // "passing", "warning", "critical"
	Error   string    `json:"error,omitempty"`
	Checked time.Time `json:"checked_at"`
	Latency int64     `json:"latency_ms,omitempty"` // Latency in milliseconds
}

// SystemHealthService aggregates health checks from multiple services
type SystemHealthService struct {
	checkers []HealthChecker
	mu       sync.RWMutex
}

// NewSystemHealthService creates a new health service
func NewSystemHealthService() *SystemHealthService {
	return &SystemHealthService{
		checkers: []HealthChecker{},
	}
}

// RegisterChecker adds a health checker to the service
func (s *SystemHealthService) RegisterChecker(checker HealthChecker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers = append(s.checkers, checker)
}

// CheckAll runs all health checks and returns aggregated status
func (s *SystemHealthService) CheckAll(ctx context.Context) []HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	results := make([]HealthStatus, 0, len(s.checkers))
	
	for _, checker := range s.checkers {
		start := time.Now()
		err := checker.Check(ctx)
		latency := time.Since(start).Milliseconds()
		
		status := HealthStatus{
			Service: checker.Name(),
			Checked: time.Now().UTC(),
			Latency: latency,
		}
		
		if err != nil {
			status.Status = "critical"
			status.Error = err.Error()
		} else {
			status.Status = "passing"
		}
		
		results = append(results, status)
	}
	
	return results
}

// HTTP handler for health endpoint
func (s *SystemHealthService) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get query parameters for detailed/basic view
	basic := r.URL.Query().Get("basic") == "true"
	
	statuses := s.CheckAll(ctx)
	
	// Determine overall status
	overallStatus := "passing"
	for _, status := range statuses {
		if status.Status == "critical" {
			overallStatus = "critical"
			break
		} else if status.Status == "warning" && overallStatus == "passing" {
			overallStatus = "warning"
		}
	}
	
	// Build response
	response := map[string]interface{}{
		"status": overallStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version": "1.0.0", // Would come from build info
	}
	
	if !basic {
		response["checks"] = statuses
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Example health checkers:

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	db *sql.DB
}

func NewDatabaseHealthChecker(db *sql.DB) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{db: db}
}

func (c *DatabaseHealthChecker) Name() string {
	return "database"
}

func (c *DatabaseHealthChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return c.db.PingContext(ctx)
}

// ExternalServiceHealthChecker checks connectivity to external services
type ExternalServiceHealthChecker struct {
	name      string
	url       string
	timeout   time.Duration
	httpClient *http.Client
}

func NewExternalServiceHealthChecker(name, url string, timeout time.Duration) *ExternalServiceHealthChecker {
	return &ExternalServiceHealthChecker{
		name:      name,
		url:       url,
		timeout:   timeout,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *ExternalServiceHealthChecker) Name() string {
	return c.name
}

func (c *ExternalServiceHealthChecker) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// Usage in main.go or initialization:
// healthService := health.NewSystemHealthService()
// healthService.RegisterChecker(health.NewDatabaseHealthChecker(db))
// healthService.RegisterChecker(health.NewExternalServiceHealthChecker("knirvchain", "http://localhost:8090/health", 2*time.Second))
// healthService.RegisterChecker(health.NewExternalServiceHealthChecker("knirvgraph", "http://localhost:8082/health", 2*time.Second))
// http.HandleFunc("/api/health", healthService.HealthHandler)
```

### 2.8 Operations — Metrics Endpoint

**Specified:** Prometheus `/metrics` endpoint

**Missing:** Verify or add Prometheus endpoint for network monitor

**Example implementation using prometheus/client_golang:**
```go
// metrics/metrics.go
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	Register(collector prometheus.Collector) error
	Unregister(collector prometheus.Collector)
	Handler() http.Handler
}

// PrometheusMetrics implements MetricsCollector using prometheus/client_golang
type PrometheusMetrics struct {
	registerer prometheus.Registerer
	gatherer   prometheus.Gatherer
}

// NewPrometheusMetrics creates a new Prometheus metrics collector
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		registerer: prometheus.NewRegistry(),
		gatherer:   prometheus.DefaultGatherer,
	}
}

// Register registers a collector with the Prometheus registry
func (m *PrometheusMetrics) Register(collector prometheus.Collector) error {
	return m.registerer.Register(collector)
}

// Unregister removes a collector from the Prometheus registry
func (m *PrometheusMetrics) Unregister(collector prometheus.Collector) {
	m.registerer.Unregister(collector)
}

// Handler returns an HTTP handler that serves Prometheus metrics
func (m *PrometheusMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.gatherer, promhttp.HandlerOpts{
		// Opt into OpenMetrics to support exemplars
		EnableOpenMetrics: true,
	})
}

// Example metric definitions:

// HTTP request duration histogram
var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "route", "status_code"},
)

// Active connections gauge
var activeConnections = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "active_connections",
		Help: "Number of active connections.",
	},
)

// Memory usage gauge
var memoryUsage = prometheus.NewGaugeFunc(
	prometheus.GaugeOpts{
		Name: "memory_usage_bytes",
		Help: "Current memory usage.",
	},
	func() float64 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return float64(m.Alloc)
	},
)

// Example usage in middleware:
// func MetricsMiddleware(next http.Handler) http.Handler {
//     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//         start := time.Now()
//         
//         // Create a response writer that captures status code
//         rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
//         
//         next.ServeHTTP(rw, r)
//         
//         // Record metrics
//         duration := time.Since(start).Seconds()
//         httpRequestDuration.WithLabelValues(
//             r.Method,
//             r.URL.Path,
//             strconv.Itoa(rw.statusCode),
//         ).Observe(duration)
//     })
// }

// func main() {
//     metricsCollector := NewPrometheusMetrics()
//     
//     // Register standard Go metrics
//     metricsCollector.Register(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
//     metricsCollector.Register(prometheus.NewGoCollector())
//     
//     // Register application metrics
//     metricsCollector.Register(httpRequestDuration)
//     metricsCollector.Register(activeConnections)
//     metricsCollector.Register(memoryUsage)
//     
//     http.Handle("/metrics", metricsCollector.Handler())
//     http.ListenAndServe(":8084", nil)
// }
```

---

## 3. Workflow Gaps (By Phase)

### Phase 1: Onboard & Configure

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Account Setup (registration) | Full flow | ⚠️ Partial | Email verification UX wired? Password reset flow? |
| Organizational Onboarding | Six modal steps | ✅ Done | Backend wiring for ValueSystem + Ontology persistence |
| API Keys Modal | Step 1 | ⚠️ Partial | Backend save to BuntDB? |
| MCP Servers Modal | Step 2 | ⚠️ Partial | Endpoint registration backend? |
| Policy Certs Modal | Step 3 | ⚠️ Partial | PQC-signed cert backend? |
| Custom Rules Modal | Step 4 | ❌ Missing | GuardrailRule generation from ontology |
| Preferences Modal | Step 5 | ⚠️ Partial | Backend persistence? |
| Cloud Pricing Modal | Step 6 | ⚠️ Partial | Backend pricing config? |

**Backend Wiring Missing:**
- `POST /api/v1/onboarding/organizations` handler to save ValueSystem + Ontology
- GuardrailRule auto-generation from ontology categories
- ValueSystem persistence to user profile

**Example implementation for Organizational Onboarding backend handler:**
```go
// POST /api/v1/onboarding/organizations
func (h *OnboardingHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string                 `json:"user_id"`
		ValueSystem map[string]interface{} `json:"value_system"`
		Ontology   map[string]interface{} `json:"ontology"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	if req.ValueSystem == nil {
		http.Error(w, "Value system is required", http.StatusBadRequest)
		return
	}
	if req.Ontology == nil {
		http.Error(w, "Ontology is required", http.StatusBadRequest)
		return
	}

	// Save to user profile
	if err := h.onboardingService.SaveOrganizationProfile(req.UserID, req.ValueSystem, req.Ontology); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-generate guardrail rules from ontology categories
	if err := h.guardrailService.GenerateRulesFromOntology(req.UserID, req.Ontology); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Organization profile created successfully",
		"user_id": req.UserID,
	})
}

// Service implementation for saving organization profile
func (s *OnboardingService) SaveOrganizationProfile(userID string, valueSystem, ontology map[string]interface{}) error {
	// Save to BuntDB
	profile := map[string]interface{}{
		"user_id":      userID,
		"value_system": valueSystem,
		"ontology":     ontology,
		"created_at":   time.Now().UTC(),
		"updated_at":   time.Now().UTC(),
	}
	
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	
	return s.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(
			"org_profile:"+userID, 
			string(data), 
			nil,
		)
		return err
	})
}

// Service implementation for generating guardrail rules from ontology
func (s *GuardrailService) GenerateRulesFromOntology(userID string, ontology map[string]interface{}) error {
	// Extract ontology categories (this would depend on your ontology structure)
	categories, ok := ontology["categories"].([]interface{})
	if !ok {
		// Try alternative structure
		categoriesInterface, ok := ontology["elements"].([]interface{})
		if !ok {
			return fmt.Errorf("could not find categories in ontology")
		}
		categories = categoriesInterface
	}
	
	var rules []GuardrailRule
	for _, categoryInterface := range categories {
		categoryMap, ok := categoryInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		categoryName, ok := categoryMap["name"].(string)
		if !ok || categoryName == "" {
			continue
		}
		
		// Create a guardrail rule for this category
		rule := GuardrailRule{
			ID:          generateRuleID(),
			UserID:      userID,
			Category:    categoryName,
			Description: fmt.Sprintf("Access control for %s category", categoryName),
			Action:      "allow", // Default action
			Conditions: map[string]interface{}{
				"category": categoryName,
			},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		
		rules = append(rules, rule)
	}
	
	// Save all rules to database
	for _, rule := range rules {
		if err := s.saveGuardrailRule(rule); err != nil {
			return err
		}
	}
	
	return nil
}
```

### Phase 2: Provision Compute (DVE Nodes)

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| DVE Creation Form | `dve-creation-form.tsx` | ✅ Done | — |
| DVECreationService | Backend provision | ✅ Done | — |
| P2P Discovery | Chain + DHT | ✅ Done | DHT toggle wiring |
| SSH Access Modal | `ssh-access-modal.tsx` | ✅ Done | Port 22000-22999 backend |
| Validation Access | Port 23000-23999 | ✅ Done | — |
| Error Resolution | Port 24000-24999 | ✅ Done | — |

**Backend Wiring Missing:**
- KNIRVGATEWAY (TURN/STUN) NAT traversal integration

### Phase 3: Ingest Data & Build Knowledge

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Knowledge Base Indexing | `POST /api/v1/knowledge-base/objects/{id}/index` | ✅ Done | GraphRAG FFI wired? |
| Index Status | `GET /api/v1/knowledge-base/objects/{id}/index-status` | ⚠️ Check | Polling endpoint? |
| Deploy Index | `POST /api/v1/knowledge-base/objects/{id}/deploy` | ⚠️ Check | Live inference deployment? |
| Plugin Upload | `POST /objects/upload` | ✅ Done | — |
| Plugin Runtime | `POST /objects/runtime/start` | ⚠️ Check | WASM runtime integration |
| ICME Objectives | `POST /api/icme/objectives` | ✅ Done | Alignment loop wired? |

### Phase 4: Delegate Credentials & Run Inference

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| DelegatorService | Provider chain config | ✅ Done | MOA ensemble config UI? |
| Inference Endpoints | `POST /api/inference/*` | ✅ Done | — |
| ContextStrategist | Document chunking | ✅ Done | — |
| ConversationMemory | Session context | ✅ Done | — |

### Phase 5: Enforce Guardrails & Validate

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Policy Editor | `policy-editor.tsx` | ✅ Done | — |
| Policy Commit | `POST /api/guardrail/policies/{id}/commit` | ⚠️ Check | Blockchain anchoring? |
| GuardrailEngine | Real-time enforcement | ✅ Done | KNIRVHASHER gRPC? |
| Violations Panel | `guardrail-violations-panel.tsx` | ✅ Done | Real-time updates? |
| Validation Service | `POST /api/validation/*` | ✅ Done | — |
| ProofGenerator | Cryptographic proof | ⚠️ Check | Implementation? |

### Phase 6: Resolve Errors & Mine Knowledge

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Error Node Creation | `POST /api/knirvgraph/error-node` | ✅ Done | DHT propagation? |
| Error Queue | `GET /api/knirvgraph/error-queue` | ⚠️ Check | UI endpoint? |
| Error Resolution Modal | WebSocket session | ⚠️ Partial | xterm.js wired? |
| Solution Node | Vault encrypted | ⚠️ Check | Encryption backend? |
| Anchoring Service | `POST /api/anchoring/` | ⚠️ Check | PQC evidence pack? |

### Phase 7: Analytics & Scaling

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| System Health Panel | Real-time polling | ✅ Done | — |
| Predictive Analytics | `predictive-analytics-panel.tsx` | ✅ Done | CPU/memory forecasts? |
| ProactiveDetector | Anomaly surfacing | ⚠️ Check | Pre-threshold alerting? |
| Module Log Viewer | SSE streaming | ⚠️ Check | Real-time SSE? |
| Rollup Service | Batch submission | ⚠️ Check | Oracle integration? |
| DistributedScaler | Auto-scaling | ⚠️ Check | K8s/native orchestration? |

---

## 4. Secondary Workflows (Incomplete)

### Workflow A — Financial Compliance

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Fintech Plugin | Enable/disable | ✅ Done | — |
| Fintech Validator | Regulatory checks | ✅ Done | KYC/AML/Basel/SEC ontologies? |
| FidelityScorer | Quality scoring | ⚠️ Check | Implementation? |
| ReplayEngine | Audit trails | ⚠️ Check | Implementation? |
| EvidenceBuilder | Compliance packs | ⚠️ Check | Implementation? |
| NRVTracer | Token tracing | ⚠️ Check | Implementation? |

### Workflow B — SSH Access

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| SSH Modal | Port 22000-22999 | ✅ Done | — |
| Web Terminal | xterm.js | ⚠️ Partial | Backend SSH proxy? |

### Workflow C — Mobile Controller Pairing

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| QR Code Display | `qr-code-display.tsx` | ✅ Done | — |
| WebSocket Pairing | Real-time events | ⚠️ Check | Backend pairing service? |

### Workflow D — DNS Management

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| DNS Management Panel | `dns-management.tsx` | ✅ Done | CloudFlare API integration? |
| Auto-update A Record | 5-minute intervals | ⚠️ Check | Cron/scheduler wiring? |

### Workflow E — Badge Creation Lab & DVE Badge Attachment

| Feature | Specified | Status | Missing |
|---------|-----------|--------|---------|
| Badge Lab Panel | `badge-lab-panel.tsx` | ✅ Done | — |
| Badge purpose input | Text field | ✅ Done | — |
| Values selection | 7 value tags | ✅ Done | — |
| Ontology element selection | 9 ontology tags | ✅ Done | — |
| **Auth credential scoping** | API keys, JWT bindings, provider creds in badge | ❌ Missing | `BadgeCreateRequest.Metadata` does not capture auth credentials; UI has no credential fields |
| Badge generation (AI/SVG) | AI-driven design encoding selected signals | ⚠️ Stub | Client-side static SVG — no actual AI generation or signal-to-visual encoding |
| Mint to Chain | `POST /api/knirvshell/chain/badge/create` | ✅ Done | — |
| `use-badge-lab.ts` hook | createBadge / mintBadge / getBadge | ✅ Done | — |
| Backend badge routes | `/chain/badge/create`, `/mint`, `/{id}` | ✅ Done | — |
| **DVE badge attachment** | `POST /api/knirvshell/chain/badge/mint` with DVE node ID | ⚠️ Partial | Route exists; no enforcement pipeline wired to GuardrailEngine |
| **GuardrailEngine badge injection** | Badge ontology tags → guardrail rules for DVE | ❌ Missing | GuardrailEngine does not read badge metadata to generate scoped rules |
| **ICME value alignment from badge** | Badge value signals → AlignmentLoop scoring | ❌ Missing | AlignmentLoop does not receive badge value signals |
| **Auth credential scope enforcement** | DVE agents restricted to badge-scoped credentials | ❌ Missing | DelegatorService does not check badge-scoped credential boundaries |
| **badge_id stamping on tasks** | Every DVE task/validation result/error node records active badge_id | ❌ Missing | Task/validation/error node schemas have no badge_id field |
| **Badge stacking (AND-evaluation)** | Multiple badges per DVE with merged rules | ❌ Missing | No merge logic defined or implemented |
| **Badge status on DVE card** | Attached badges displayed on DVE node card | ❌ Missing | `dve-nodes-panel.tsx` has no badge attachment display |
| Badge retrieval | `GET /api/knirvshell/chain/badge/{id}` | ✅ Done | — |

**Backend Wiring Missing:**
- `GuardrailEngine` must accept badge metadata at DVE attachment time and inject ontology-scoped rules
- `ICME AlignmentLoop` must receive badge value signals when a badge is attached to a DVE
- `DelegatorService` / `InferenceService` must restrict credential usage to the badge-scoped set for badge-attached DVEs
- `DVEManager` must store the badge-to-DVE binding and surface it on the DVE node status
- Task, validation result, and error node creation paths must stamp the active `badge_id`

---

### Workflow F — Workflow Execution

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Workflow Definition | Step ordering | ⚠️ Check | UI for step definition? |
| Execute API | `POST /api/workflow/execute` | ✅ Done | Dependency graph resolver? |
| Execution Events | WebSocket broadcast | ⚠️ Check | SSE events? |
| Status Endpoint | `GET /api/workflow/executions/{id}` | ⚠️ Check | Execution state tracking? |

### Workflow G — NRN Token Payments

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Transfer API | `POST /api/nrn/transfer` | ✅ Done | On-chain submission? |
| Oracle Balance | oracleBalanceAdapter | ⚠️ Check | Root node integration? |
| Stripe/PayPal | `/api/v1/payments/` | ⚠️ Check | Fiat checkout? |

### Workflow H — KNIRVSHELL Terminal

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Console Panel | `console-panel.tsx` | ✅ Done | — |
| Execute API | `POST /api/v1/shell/execute` | ✅ Done | — |
| Sub-commands | wallet/validation/tee/p2p/chain | ⚠️ Check | Full implementation? |
| Session Management | create/list/stop/input | ⚠️ Check | Full state machine? |

### Workflow I — Active Memory & Reasoning

| Feature | Specified | Status | Missing |
|---------|-----------|-------|---------|
| Active Memory Panel | `active-memory-panel.tsx` | ✅ Done | — |
| Reasoning Explorer | Graph traces | ⚠️ Check | Graph UI implementation? |
| Solution Vault | `plugin-vault-panel.tsx` | ✅ Done | PQC decryption? |

---

## 5. Backend API Routes (Missing / Unverified)

### Missing Endpoints

| Endpoint | Method | Specified In | Status |
|----------|--------|-------------|--------|
| `/api/guardrail/violations/{id}/resolve` | POST | Step 11 | ❌ Missing |
| `/api/system-health/*` | GET | Step 16 | ⚠️ Partial |
| `/api/cognitive/telemetry` | GET | Step 16 | ⚠️ Missing SSE |
| `/api/rollups/*` | GET | Step 17 | ⚠️ Partial |
| `/api/nrn/balance` | GET | UC-2 | ⚠️ Check |
| `/api/v1/knowledge-base/objects/{id}/query` | POST | UC-5 | ⚠️ Check |

### Unverified Implementation

| Endpoint | Method | Specified In | Notes |
|-----------|--------|-------------|-------|
| `/api/icme/alignment/status` | GET | Step 9 | Return alignment score? |
| `/api/active-memory/*` | PUT | Step 15 | Write reasoning traces? |

---

## 6. Frontend Components (Missing)

| Component | Specified In | Status |
|-----------|--------------|--------|
| `dve-creation-form.tsx` | Step 3 | ✅ Done |
| `policy-editor.tsx` | Step 10 | ✅ Done |
| `guardrail-violations-panel.tsx` | Step 11 | ✅ Done |
| `ssh-access-modal.tsx` | Workflow B | ✅ Done |
| `error-resolution-dashboard.tsx` | Step 14 | ⚠️ Partial |
| `error-resolution-modal.tsx` | Step 14 | ✅ Done |
| `predictive-analytics-panel.tsx` | Step 16 | ✅ Done |
| `dns-management.tsx` | Workflow D | ✅ Done |
| `qr-code-display.tsx` | Workflow C | ✅ Done |
| `financial-compliance-dashboard.tsx` | Workflow A | ✅ Done |
| `plugin-vault-panel.tsx` | Step 15 | ✅ Done |
| `knirvshell-console.tsx` | Workflow G | ⚠️ Partial (uses console-panel.tsx) |

---

## 7. Services (Missing / Incomplete)

| Service | Specified In | Status |
|---------|--------------|--------|
| DVECreationService | Step 3 | ✅ Done |
| P2P Manager (libp2p + DHT) | Step 4 | ✅ Done |
| GraphRAG-rs | Step 5 | ⚠️ FFI wired |
| ICME AlignmentLoop | Step 9 | ⚠️ Partial |
| GuardrailEngine | Step 11 | ✅ Done |
| ValidationCore | Step 12 | ✅ Done |
| ProofGenerator | Step 12 | ⚠️ Check |
| ErrorNodeService | Step 13 | ⚠️ Partial |
| AnchoringService | Step 15 | ⚠️ Partial |
| RollupService | Step 17 | ⚠️ Partial |
| DistributedScaler | Step 18 | ⚠️ Partial |
| FintechValidator | Workflow A | ⚠️ Check |
| FidelityScorer | Workflow A | ❌ Missing |
| ReplayEngine | Workflow A | ❌ Missing |
| EvidenceBuilder | Workflow A | ❌ Missing |
| NRVTracer | Workflow A | ❌ Missing |

---

## 8. Testing Gaps

| Test | Specified In | Status |
|------|--------------|--------|
| E2E primary workflow | 4.3 | ❌ Missing |
| Load testing | 4.3 | ❌ Missing |
| Guardrail policy fuzz | 4.3 | ❌ Missing |
| Root key loading | 4.3 | ❌ Missing |

---

## 9. Integration Gaps

| Integration | Specified In | Status |
|-------------|--------------|--------|
| KNIRVARENA (HERO Model) | 4.3 | ❌ Missing |
| KNIRVARENA (Dataset Forge) | 4.3 | ❌ Missing |
| CDE Service UX | 4.3 | ❌ Missing |
| oh-my-pi agent runtime | 4.3 | ❌ Missing |

---

## 10. Observability Gaps

| Feature | Specified In | Status |
|---------|--------------|--------|
| OpenTelemetry tracing | 4.3 | ❌ Missing |
| Alertmanager → PagerDuty/Slack | 4.3 | ⚠️ Config exists |
| eBPF dashboards | 4.3 | ❌ Missing |

---

## 11. DVE Installation & Browser Routing

This section documents the distributed DVE Installation workflow that distributes installer functions across KNIRVGATEWAY, KNIRVORACLE, and KNIRVSERVER.

### Workflow: DVE Installation

| Feature | Phase | Status | Missing |
|---------|-------|--------|---------|
| Registry + STUN Discovery | KNIRVGATEWAY | ⚠️ Partial | New installer methods in registry.go |
| Port Discovery | KNIRVGATEWAY | ⚠️ Partial | New STUN client |
| Wallet Generation | KNIRVORACLE | ⚠️ Check | New wallet route using existing crypto |
| DVE URI Generation | KNIRVORACLE | ⚠️ Check | New DVE URI route |
| Service Setup | KNIRVSERVER | ⚠️ Check | New InstallerService |
| InstallComplete Tracking | KNIRVSERVER | ❌ Missing | New fields in objects/dve.go |
| Validation Chain DVE URI | KNIRVSERVER | ❌ Missing | New methods in validationchain/client.go |
| DVE URI Registry | KNIRVSERVER | ❌ Missing | New dve_uri_registry.go |
| DVE Proxy Handlers | KNIRVSERVER | ❌ Missing | New dve_proxy_handlers.go |
| Public DVE HTML Templates | KNIRVSERVER | ❌ Missing | New templates/dve/ directory |
| DVE WebSocket Client | KNIRVARENA | ❌ Missing | New networking/DVEClient.ts |

### Files to Modify

| Phase | File | Change |
|-------|------|--------|
| 1 | `packages/KNIRVGATEWAY/internal/tunnel/registry.go` | Add RegisterBootnode, GetBootnodes |
| 1 | `packages/KNIRVGATEWAY/internal/server/server.go` | Register routes |
| 3 | `packages/KNIRVSERVER/backend_server/internal/objects/dve.go` | Add InstallComplete fields |
| 3 | `packages/KNIRVSERVER/backend_server/internal/services/blockchain/validationchain/client.go` | Add DVE URI methods |
| 4 | `packages/KNIRVSERVER/backend_server/internal/web/api_router.go` | Register DVE proxy routes |
| 4 | `packages/KNIRVARENA/packages/ts_client_2/src/App.tsx` | Add DVE route |

### New Files to Create

| Phase | File |
|-------|------|
| 1 | `packages/KNIRVGATEWAY/internal/installer/stun_client.go` |
| 1 | `packages/KNIRVGATEWAY/internal/installer/dve_uri.go` |
| 2 | `packages/KNIRVORACLE/internal/oracle/routes/wallet.go` |
| 2 | `packages/KNIRVORACLE/internal/oracle/routes/dve_uri.go` |
| 3 | `packages/KNIRVSERVER/backend_server/internal/services/installer/installer.go` |
| 4 | `packages/KNIRVSERVER/backend_server/internal/services/dve_uri_registry.go` |
| 4 | `packages/KNIRVSERVER/backend_server/internal/web/dve_proxy_handlers.go` |
| 4 | `packages/KNIRVSERVER/backend_server/templates/dve/public_page.gohtml` |
| 4 | `packages/KNIRVSERVER/backend_server/templates/dve/validation_records.gohtml` |
| 4 | `packages/KNIRVSERVER/backend_server/templates/dve/metrics_panel.gohtml` |
| 4 | `packages/KNIRVSERVER/backend_server/templates/dve/search_form.gohtml` |
| 4 | `packages/KNIRVARENA/packages/ts_client_2/src/networking/DVEClient.ts` |

### Integration Points (Reuse Existing Code)

| New Component | Reuse From |
|--------------|-----------|
| STUN Client | `tunnel/stun.go` STUNServer pattern |
| Wallet Generation | `crypto/ecdsa.go` GenerateKeyPair() |
| DVE URI Generation | `crosschain/router.go` generateTransferID pattern |
| InstallerService | `pkg/knirvchain/manager.go` Manager pattern |
| DVE Proxy | `runtime/viewport_proxy.go` ViewportProxyImpl |
| DVEClient.ts | Networking ArenaClient.ts pattern |

---

## Summary Checklist

### Critical (Must Fix)
- [ ] Remove testnet tokens from production code
- [ ] Enforce AuthRequired in production config
- [ ] Implement email verification flow
- [ ] Implement password reset flow
- [ ] Implement token revocation on logout
- [ ] Implement Stripe/PayPal webhook verification
- [ ] Implement NRN idempotency

### High Priority
- [ ] Gate demo mode behind admin role
- [ ] Add error boundaries to all panels
- [ ] Implement WebSocket reconnect with exponential backoff
- [ ] Add loading/empty states to all panels
- [ ] Enforce role-based access on backend routes
- [ ] Add structured logging to all services
- [ ] Complete health check coverage
- [ ] Add Prometheus metrics endpoint

### Badge Lab Wiring
- [ ] Add auth credential fields to badge creation UI and `BadgeCreateRequest.Metadata`
- [ ] Wire badge attachment to `GuardrailEngine` (ontology tags → scoped guardrail rules)
- [ ] Wire badge value signals to `ICME AlignmentLoop` scoring at DVE attachment time
- [ ] Restrict `DelegatorService` to badge-scoped credentials for badge-attached DVEs
- [ ] Stamp `badge_id` on all DVE task results, validation results, and error nodes
- [ ] Implement badge stacking AND-evaluation in `GuardrailEngine`
- [ ] Display attached badges and enforcement status on DVE node card
- [ ] Replace client-side SVG stub with real badge generation (AI or server-side SVG templating)
- [ ] Store badge-to-DVE bindings in `DVEManager` with status endpoint

### DVE Installation & Browser Routing
- [ ] Extend KNIRVGATEWAY registry.go with RegisterBootnode, GetBootnodes
- [ ] Create KNIRVGATEWAY installer/stun_client.go
- [ ] Create KNIRVGATEWAY installer/dve_uri.go (proxy to ORACLE)
- [ ] Create KNIRVORACLE routes/wallet.go (reuse crypto)
- [ ] Create KNIRVORACLE routes/dve_uri.go
- [ ] Add InstallComplete fields to KNIRVSERVER objects/dve.go
- [ ] Create KNIRVSERVER services/installer/installer.go
- [ ] Add DVE URI methods to KNIRVSERVER validationchain/client.go
- [ ] Create KNIRVSERVER services/dve_uri_registry.go
- [ ] Create KNIRVSERVER web/dve_proxy_handlers.go
- [ ] Create KNIRVSERVER templates/dve/ Go HTML templates
- [ ] Create KNIRVARENA networking/DVEClient.ts (WebSocket)
- [ ] Register /dve/{dve_id}/ routes in api_router.go
- [ ] Add DVE route in KNIRVARENA App.tsx

### Workflow Wiring
- [ ] Wire ValueSystem + Ontology to backend persistence
- [ ] Wire GuardrailRule auto-generation from ontology
- [ ] Wire ICME alignment status to UI
- [ ] Wire pre-threshold alerting (ProactiveDetector)
- [ ] Wire SSE for module logs
- [ ] Wire RollupService to Oracle
- [ ] Wire DistributedScaler to container orchestrator

### Secondary Workflows
- [ ] Complete FidelityScorer implementation
- [ ] Complete ReplayEngine implementation
- [ ] Complete EvidenceBuilder implementation
- [ ] Complete NRVTracer implementation
- [ ] Wire xterm.js to SSH backend
- [ ] Wire mobile controller pairing service
- [ ] Wire DNS auto-update cron

### Testing
- [ ] Write E2E test for primary workflow
- [ ] Write load tests for InferenceService
- [ ] Write fuzz tests for GuardrailEngine
- [ ] Write root key loading test

---

_This document reflects gaps identified as of 2026-04-20. Update after each milestone._