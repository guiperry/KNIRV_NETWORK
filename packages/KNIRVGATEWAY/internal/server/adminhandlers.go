package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/d1"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/rootkey"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// defaultD1DatabaseID is knirv-onboarding's database UID (see
// KNIRV_CORP/websites/ONBOARDING.KNIRV.COM/wrangler.toml). Override via
// CLOUDFLARE_D1_DATABASE_ID if this ever points somewhere else.
const defaultD1DatabaseID = "b25c0eb5-ef54-42cd-afcd-df074b1da8cd"

// newAdminD1Client decrypts root.key once at startup and builds a D1 client
// from it. Returns nil (not an error) when credentials aren't available —
// every handler below checks for that and responds 503 rather than crashing
// the whole gateway over an optional admin feature. This does real scrypt
// work (deliberately slow) — call it once, never per-request.
func newAdminD1Client(logger *zap.Logger) *d1.Client {
	creds, err := rootkey.LoadCloudflareCredentials()
	if err != nil {
		logger.Warn("Admin D1 endpoints disabled: could not load Cloudflare credentials from root.key", zap.Error(err))
		return nil
	}

	databaseID := strings.TrimSpace(os.Getenv("CLOUDFLARE_D1_DATABASE_ID"))
	if databaseID == "" {
		databaseID = defaultD1DatabaseID
	}

	logger.Info("Admin D1 endpoints enabled", zap.String("account_id", creds.AccountID), zap.String("database_id", databaseID))
	return d1.NewClient(creds.AccountID, databaseID, creds.APIToken)
}

func (s *Server) requireD1(w http.ResponseWriter) bool {
	if s.d1Client == nil {
		writeAdminJSON(w, http.StatusServiceUnavailable, map[string]any{
			"success": false,
			"data":    map[string]any{"error": "D1 credentials not configured (root.key missing cloudflare_account_id/cloudflare_api_token, or ORACLE_KEY_PASSWORD unset)"},
		})
		return false
	}
	return true
}

func writeAdminJSON(w http.ResponseWriter, status int, body map[string]any) {
	body["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeAdminError(w http.ResponseWriter, status int, message string) {
	writeAdminJSON(w, status, map[string]any{"success": false, "data": map[string]any{"error": message}})
}

// --- Operator applications -------------------------------------------------

// operatorApplicationRow's JSON tags match the frontend's OnboardingApplication
// type exactly (packages/KNIRVSERVER/frontend/src/hooks/use-onboarding-admin.ts)
// — KNIRVMONITOR's proxyToOnboarding is a raw byte passthrough, so this
// response reaches the browser unmodified.
type operatorApplicationRow struct {
	ID        string `json:"id"`
	LegalName string `json:"legal_name"`
	KYCStatus string `json:"kyc_status"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// handleAdminOperatorsList lists every operator application, joined against
// node_registrations for the real approval-pipeline status — mirrors what
// review.ts actually reads/writes (node_registrations.status), since
// operator_applications itself only tracks kyc_status, not a review status.
func (s *Server) handleAdminOperatorsList(w http.ResponseWriter, r *http.Request) {
	if !s.requireD1(w) {
		return
	}

	result, err := s.d1Client.Query(r.Context(), `
		SELECT oa.id AS id, oa.legal_name AS legal_name, oa.kyc_status AS kyc_status,
		       oa.created_at AS created_at, COALESCE(nr.status, 'pending') AS status
		FROM operator_applications oa
		LEFT JOIN node_registrations nr ON nr.application_id = oa.id
		ORDER BY oa.created_at DESC
	`)
	if err != nil {
		s.logger.Warn("admin operators list query failed", zap.Error(err))
		writeAdminError(w, http.StatusBadGateway, "failed to query operator applications")
		return
	}

	applications := make([]operatorApplicationRow, 0, len(result.Rows))
	for _, row := range result.Rows {
		applications = append(applications, operatorApplicationRow{
			ID:        stringField(row, "id"),
			LegalName: stringField(row, "legal_name"),
			KYCStatus: stringField(row, "kyc_status"),
			Status:    stringField(row, "status"),
			CreatedAt: stringField(row, "created_at"),
		})
	}

	writeAdminJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]any{"applications": applications},
	})
}

// handleAdminOperatorReview approves or rejects an operator application.
// This is a from-scratch Go reimplementation of
// ONBOARDING.KNIRV.COM/functions/api/marketplace/operators/[id]/review.ts's
// exact logic (same lookup, same "only validated nodes can be approved"
// guard, same status values) — keep the two in sync if either changes.
func (s *Server) handleAdminOperatorReview(w http.ResponseWriter, r *http.Request) {
	if !s.requireD1(w) {
		return
	}

	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		writeAdminError(w, http.StatusBadRequest, "id is required")
		return
	}

	var body struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var decision string
	switch body.Decision {
	case "approved":
		decision = "reviewed"
	case "rejected":
		decision = "rejected"
	default:
		writeAdminError(w, http.StatusBadRequest, "decision must be 'approved' or 'rejected'")
		return
	}

	lookup, err := s.d1Client.Query(r.Context(),
		"SELECT registration_id, status FROM node_registrations WHERE application_id = ? OR registration_id = ? LIMIT 1",
		id, id)
	if err != nil {
		s.logger.Warn("admin operator review lookup failed", zap.Error(err))
		writeAdminError(w, http.StatusBadGateway, "failed to look up application")
		return
	}
	if len(lookup.Rows) == 0 {
		writeAdminError(w, http.StatusNotFound, "provisioned node not found")
		return
	}

	registrationID := stringField(lookup.Rows[0], "registration_id")
	currentStatus := stringField(lookup.Rows[0], "status")
	if currentStatus != "validated" && decision == "reviewed" {
		writeAdminJSON(w, http.StatusConflict, map[string]any{
			"success": false,
			"data":    map[string]any{"error": "only validated nodes can be approved", "status": currentStatus},
		})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.d1Client.Query(r.Context(),
		"UPDATE node_registrations SET status = ?, updated_at = ? WHERE registration_id = ?",
		decision, now, registrationID); err != nil {
		s.logger.Warn("admin operator review update failed", zap.Error(err))
		writeAdminError(w, http.StatusBadGateway, "failed to update application status")
		return
	}

	writeAdminJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"registrationId": registrationID,
			"status":         decision,
			"note":           body.Note,
		},
	})
}

// --- User accounts -----------------------------------------------------

var allowedRoles = map[string]bool{"admin": true, "validator": true, "observer": true}
var allowedAccountStatuses = map[string]bool{"active": true, "suspended": true, "banned": true}

// userAccountRow's JSON tags match the frontend's OnboardingUser type
// exactly (packages/KNIRVSERVER/frontend/src/hooks/use-onboarding-admin.ts).
type userAccountRow struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	AccountStatus string `json:"account_status"`
	Plan          string `json:"plan"`
}

func (s *Server) handleAdminUsersList(w http.ResponseWriter, r *http.Request) {
	if !s.requireD1(w) {
		return
	}

	result, err := s.d1Client.Query(r.Context(),
		"SELECT id, username, email, role, account_status, plan FROM accounts ORDER BY created_at DESC")
	if err != nil {
		s.logger.Warn("admin users list query failed", zap.Error(err))
		writeAdminError(w, http.StatusBadGateway, "failed to query accounts")
		return
	}

	users := make([]userAccountRow, 0, len(result.Rows))
	for _, row := range result.Rows {
		users = append(users, userAccountRow{
			ID:            stringField(row, "id"),
			Username:      stringField(row, "username"),
			Email:         stringField(row, "email"),
			Role:          stringField(row, "role"),
			AccountStatus: stringField(row, "account_status"),
			Plan:          stringField(row, "plan"),
		})
	}

	writeAdminJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]any{"users": users},
	})
}

func (s *Server) handleAdminUserRole(w http.ResponseWriter, r *http.Request) {
	// jsonKey "role" matches the endpoint's own name (POST .../role
	// {"role": "..."}); columnName "role" happens to match here too, but
	// isn't required to — see handleAdminUserStatus below where they differ.
	s.handleAdminUserFieldUpdate(w, r, "role", "role", allowedRoles)
}

func (s *Server) handleAdminUserStatus(w http.ResponseWriter, r *http.Request) {
	// jsonKey "status" matches the endpoint's own name (POST .../status
	// {"status": "..."}); columnName "account_status" is the actual D1
	// column (accounts.account_status) — these are deliberately NOT the
	// same string, unlike role above.
	s.handleAdminUserFieldUpdate(w, r, "status", "account_status", allowedAccountStatuses)
}

// handleAdminUserFieldUpdate backs both role and status changes: same shape
// (id from the path, new value from a JSON body keyed by jsonKey), same
// allowed-value validation, same UPDATE-and-check-Changes pattern written to
// columnName. Both jsonKey and columnName are compile-time constants at
// both call sites above, never request-controlled, so building the SQL
// string with columnName is safe — the actual bound value (newValue) always
// goes through a `?` placeholder.
func (s *Server) handleAdminUserFieldUpdate(w http.ResponseWriter, r *http.Request, jsonKey, columnName string, allowed map[string]bool) {
	if !s.requireD1(w) {
		return
	}

	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		writeAdminError(w, http.StatusBadRequest, "id is required")
		return
	}

	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAdminError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	newValue := strings.TrimSpace(body[jsonKey])
	if !allowed[newValue] {
		writeAdminError(w, http.StatusBadRequest, fmt.Sprintf("%s must be one of the allowed values", jsonKey))
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf("UPDATE accounts SET %s = ?, updated_at = ? WHERE id = ?", columnName)
	result, err := s.d1Client.Query(r.Context(), sql, newValue, now, id)
	if err != nil {
		s.logger.Warn("admin user field update failed", zap.String("column", columnName), zap.Error(err))
		writeAdminError(w, http.StatusBadGateway, "failed to update account")
		return
	}
	if result.Changes == 0 {
		writeAdminError(w, http.StatusNotFound, "user not found")
		return
	}

	writeAdminJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]any{"id": id, jsonKey: newValue},
	})
}

// stringField reads a D1 result column as a string, tolerating nil/absent
// values (D1 returns JSON, so a NULL column simply isn't a string type).
func stringField(row map[string]any, key string) string {
	if v, ok := row[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}
