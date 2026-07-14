package knirvproof

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPrivateRoutesFailClosedWithoutAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newTestStore(t)
	service, err := NewService(store, nil, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	RegisterRoutes(router, service, nil)
	body, _ := json.Marshal(ObjectBatchRequest{ProjectID: "project-one", Objects: []ObjectRef{}})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/knirv/cas/objects/batch", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
}

func TestBatchAndObjectUploadRequireProjectAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newTestStore(t)
	service, err := NewService(store, nil, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	authorizer := AuthorizerFunc(func(_ context.Context, token, projectID, action string) (*Principal, error) {
		if token != "valid-token" {
			return nil, ErrUnauthorized
		}
		if projectID != "project-one" || action != ActionCASWrite {
			return nil, ErrForbidden
		}
		return &Principal{ID: "user-1"}, nil
	})
	router := gin.New()
	RegisterRoutes(router, service, authorizer)
	ciphertext := []byte("encrypted object")
	object := ObjectRef{CID: HashBytes(ciphertext), Size: int64(len(ciphertext))}
	body, _ := json.Marshal(ObjectBatchRequest{ProjectID: "project-one", Objects: []ObjectRef{object}})
	batchRequest := httptest.NewRequest(http.MethodPost, "/api/v1/knirv/cas/objects/batch", bytes.NewReader(body))
	batchRequest.Header.Set("Authorization", "Bearer valid-token")
	batchResponse := httptest.NewRecorder()
	router.ServeHTTP(batchResponse, batchRequest)
	if batchResponse.Code != http.StatusOK {
		t.Fatalf("batch status = %d; body=%s", batchResponse.Code, batchResponse.Body.String())
	}

	digest, _ := digestHex(object.CID)
	uploadRequest := httptest.NewRequest(http.MethodPut, "/api/v1/knirv/cas/objects/"+digest+"?project_id=project-one", bytes.NewReader(ciphertext))
	uploadRequest.Header.Set("Authorization", "Bearer valid-token")
	uploadResponse := httptest.NewRecorder()
	router.ServeHTTP(uploadResponse, uploadRequest)
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("upload status = %d; body=%s", uploadResponse.Code, uploadResponse.Body.String())
	}
	if exists, size, err := store.HasObject(object.CID); err != nil || !exists || size != object.Size {
		t.Fatalf("uploaded object exists=%v size=%d err=%v", exists, size, err)
	}
}

func TestProofSignerMustMatchAuthenticatedPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newTestStore(t)
	service, err := NewService(store, nil, nil, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	submission := putTestSubmission(t, store, "project-one", "signer-binding")
	submission.SignerID = "impersonated-user"
	authorizer := AuthorizerFunc(func(_ context.Context, _, _, _ string) (*Principal, error) {
		return &Principal{ID: "authenticated-user"}, nil
	})
	router := gin.New()
	RegisterRoutes(router, service, authorizer)
	body, _ := json.Marshal(submission)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/knirv/projects/project-one/proofs", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
}
