package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"knirv-server/internal/knirvproof"
)

func TestDVEIngestAuthBindsPrincipalAndProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotProject, gotAction string
	authorizer := knirvproof.AuthorizerFunc(func(_ context.Context, token, projectID, action string) (*knirvproof.Principal, error) {
		if token != "valid" {
			return nil, knirvproof.ErrUnauthorized
		}
		gotProject, gotAction = projectID, action
		return &knirvproof.Principal{ID: "user-1"}, nil
	})
	router := gin.New()
	router.Use(dveIngestAuthMiddleware(authorizer))
	router.POST("/api/dve/:dve/sessions/ingest", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	body := `{"bundle":{"project_id":"project-1","user_id":"user-1"}}`
	request := httptest.NewRequest(http.MethodPost, "/api/dve/dve-1/sessions/ingest", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if gotProject != "project-1" || gotAction != knirvproof.ActionProofSubmit {
		t.Fatalf("authorization = (%q, %q)", gotProject, gotAction)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/dve/dve-1/sessions/ingest", strings.NewReader(`{"bundle":{"project_id":"project-1","user_id":"other"}}`))
	request.Header.Set("Authorization", "Bearer valid")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("principal mismatch status = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/dve/dve-1/sessions/ingest", strings.NewReader(`{"bundle":{"project_id":"project-1"}}`))
	request.Header.Set("Authorization", "Bearer valid")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing principal binding status = %d", response.Code)
	}
}
