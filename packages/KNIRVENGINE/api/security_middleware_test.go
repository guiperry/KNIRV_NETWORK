package api

import "testing"

func TestSandboxCollectionRootIsExemptFromCSRF(t *testing.T) {
	middleware := NewSecurityMiddleware()

	if !middleware.isExemptFromCSRF("/api/v1/sandboxes") {
		t.Fatal("sandbox create endpoint must be exempt from CSRF like nested sandbox desktop-operation endpoints")
	}
}
