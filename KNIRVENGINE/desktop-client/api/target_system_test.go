package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestTargetSystemService(t *testing.T) {
	// Create a new target system service
	service := NewTargetSystemService()

	// Test creating a target system
	target := &TargetSystem{
		Name:        "Test Filesystem",
		Type:        TargetTypeFilesystem,
		Description: "Test filesystem connection",
		Config: map[string]interface{}{
			"basePath":    "/tmp",
			"readOnly":    false,
			"maxFileSize": 1024 * 1024, // 1MB
		},
		OwnerID: 1,
	}

	err := service.CreateTarget(target)
	if err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	if target.ID == "" {
		t.Fatal("Target ID should be generated")
	}

	// Test retrieving the target
	retrieved, err := service.GetTarget(target.ID)
	if err != nil {
		t.Fatalf("Failed to get target: %v", err)
	}

	if retrieved.Name != target.Name {
		t.Errorf("Expected name %s, got %s", target.Name, retrieved.Name)
	}

	// Test listing targets
	targets, err := service.ListTargets(1)
	if err != nil {
		t.Fatalf("Failed to list targets: %v", err)
	}

	if len(targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets))
	}

	// Test connecting to the target
	ctx := context.Background()
	err = service.ConnectTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("Failed to connect to target: %v", err)
	}

	// Test getting status
	status, err := service.GetTargetStatus(target.ID)
	if err != nil {
		t.Fatalf("Failed to get target status: %v", err)
	}

	if status["status"] != StatusConnected {
		t.Errorf("Expected status %s, got %s", StatusConnected, status["status"])
	}

	// Test executing an operation
	result, err := service.ExecuteOperation(ctx, target.ID, "list_directory", map[string]interface{}{
		"path": ".",
	})
	if err != nil {
		t.Fatalf("Failed to execute operation: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	if !resultMap["success"].(bool) {
		t.Error("Expected operation to succeed")
	}

	// Test disconnecting
	err = service.DisconnectTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("Failed to disconnect target: %v", err)
	}

	// Test deleting the target
	err = service.DeleteTarget(target.ID)
	if err != nil {
		t.Fatalf("Failed to delete target: %v", err)
	}

	// Verify target is deleted
	_, err = service.GetTarget(target.ID)
	if err == nil {
		t.Error("Expected error when getting deleted target")
	}
}

func TestTargetSystemHTTPHandlers(t *testing.T) {
	// Create service and router
	service := NewTargetSystemService()
	router := mux.NewRouter()
	service.RegisterHandlers(router)

	// Test creating a target via HTTP
	targetJSON := `{
		"name": "Test Database",
		"type": "database",
		"description": "Test database connection",
		"config": {
			"dbType": "sqlite3",
			"path": ":memory:"
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/targets", strings.NewReader(targetJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var createdTarget TargetSystem
	err := json.NewDecoder(w.Body).Decode(&createdTarget)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	targetID := createdTarget.ID

	// Test getting the target via HTTP
	req = httptest.NewRequest("GET", "/api/v1/targets/"+targetID, nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test connecting via HTTP
	req = httptest.NewRequest("POST", "/api/v1/targets/"+targetID+"/connect", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test executing operation via HTTP
	operationJSON := `{
		"operation": "list_tables",
		"params": {}
	}`

	req = httptest.NewRequest("POST", "/api/v1/targets/"+targetID+"/execute", strings.NewReader(operationJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test getting status via HTTP
	req = httptest.NewRequest("GET", "/api/v1/targets/"+targetID+"/status", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test disconnecting via HTTP
	req = httptest.NewRequest("POST", "/api/v1/targets/"+targetID+"/disconnect", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test deleting via HTTP
	req = httptest.NewRequest("DELETE", "/api/v1/targets/"+targetID, nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestSecurityManager(t *testing.T) {
	manager := NewConnectionSecurityManager()

	// Test creating security context
	permissions := []Permission{PermissionRead, PermissionWrite}
	constraints := map[string]interface{}{
		"maxFileSize": 1024,
	}

	ctx, err := manager.CreateSecurityContext(1, permissions, constraints)
	if err != nil {
		t.Fatalf("Failed to create security context: %v", err)
	}

	if ctx.SessionID == "" {
		t.Error("Session ID should be generated")
	}

	if len(ctx.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(ctx.Permissions))
	}

	// Test validating security context
	validated, err := manager.ValidateSecurityContext(ctx.SessionID)
	if err != nil {
		t.Fatalf("Failed to validate security context: %v", err)
	}

	if validated.UserID != ctx.UserID {
		t.Errorf("Expected user ID %d, got %d", ctx.UserID, validated.UserID)
	}

	// Test checking permissions
	err = manager.CheckPermission(ctx.SessionID, PermissionRead)
	if err != nil {
		t.Errorf("Expected read permission to be allowed: %v", err)
	}

	err = manager.CheckPermission(ctx.SessionID, PermissionDelete)
	if err == nil {
		t.Error("Expected delete permission to be denied")
	}

	// Test revoking session
	err = manager.RevokeSession(ctx.SessionID)
	if err != nil {
		t.Fatalf("Failed to revoke session: %v", err)
	}

	// Test that revoked session is invalid
	_, err = manager.ValidateSecurityContext(ctx.SessionID)
	if err == nil {
		t.Error("Expected revoked session to be invalid")
	}
}

func TestSecureConnection(t *testing.T) {
	// Create a filesystem target
	target := &TargetSystem{
		Name: "Test Secure Filesystem",
		Type: TargetTypeFilesystem,
		Config: map[string]interface{}{
			"basePath": "/tmp",
			"readOnly": false,
		},
	}

	// Create base connection
	baseConn, err := NewFilesystemConnection(target)
	if err != nil {
		t.Fatalf("Failed to create filesystem connection: %v", err)
	}

	// Create security manager and context
	securityManager := NewConnectionSecurityManager()
	permissions := []Permission{PermissionRead, PermissionExecute}
	ctx, err := securityManager.CreateSecurityContext(1, permissions, nil)
	if err != nil {
		t.Fatalf("Failed to create security context: %v", err)
	}

	// Create secure connection
	secureConn := NewSecureTargetSystemConnection(
		baseConn,
		securityManager,
		ctx.SessionID,
		SecurityLevelStandard,
	)

	// Test connecting
	connectCtx := context.Background()
	err = secureConn.Connect(connectCtx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test that read operations are allowed
	result, err := secureConn.Execute(connectCtx, "list_directory", map[string]interface{}{
		"path": ".",
	})
	if err != nil {
		t.Fatalf("Read operation should be allowed: %v", err)
	}

	if !result.(map[string]interface{})["success"].(bool) {
		t.Error("Expected read operation to succeed")
	}

	// Test that write operations are denied (user only has read permission)
	_, err = secureConn.Execute(connectCtx, "write_file", map[string]interface{}{
		"path": "test.txt",
		"data": "test data",
	})
	if err == nil {
		t.Error("Write operation should be denied for read-only user")
	}

	// Test capabilities filtering
	capabilities := secureConn.GetCapabilities()
	hasWrite := false
	for _, cap := range capabilities {
		if cap == "write_file" {
			hasWrite = true
			break
		}
	}
	if hasWrite {
		t.Error("Write capability should not be available for read-only user")
	}

	// Test disconnect
	err = secureConn.Disconnect(connectCtx)
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}
}
