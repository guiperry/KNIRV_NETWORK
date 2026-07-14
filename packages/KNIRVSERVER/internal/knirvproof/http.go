package knirvproof

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxJSONBodyBytes = 2 << 20

type HTTPHandler struct {
	service    *Service
	authorizer Authorizer
}

func RegisterRoutes(engine *gin.Engine, service *Service, authorizer Authorizer) {
	handler := &HTTPHandler{service: service, authorizer: authorizer}
	engine.POST("/api/v1/knirv/cas/objects/batch", handler.batch)
	engine.PUT("/api/v1/knirv/cas/objects/:ciphertext_cid", handler.putObject)
	engine.HEAD("/api/v1/knirv/cas/objects/:ciphertext_cid", handler.headObject)
	engine.GET("/api/v1/knirv/cas/objects/:ciphertext_cid", handler.getObject)
	engine.POST("/api/v1/knirv/projects/:project_id/proofs", handler.submitProof)
	engine.GET("/api/v1/knirv/projects/:project_id/proofs/:proof_root", handler.getProof)
	engine.GET("/api/v1/knirv/projects/:project_id/operations/:operation_id", handler.getOperation)
	engine.POST("/api/v1/knirv/projects/:project_id/proofs/:proof_root/access-envelope", handler.addEnvelope)
	engine.GET("/api/v1/knirv/public/projects/:project_id/commits/:commit_digest/proof", handler.publicProof)
}

// RegisterValidatorKeyRoute exposes only the validator's public wrapping key.
// The private key and signing-key registry remain on the backend Unix socket.
func RegisterValidatorKeyRoute(engine *gin.Engine, trust *BackendTrustClient) {
	engine.GET("/api/v1/knirv/validator-key", func(c *gin.Context) {
		if trust == nil {
			writeError(c, ErrAuthUnavailable)
			return
		}
		key, err := trust.ValidatorKey(c.Request.Context())
		if err != nil {
			writeError(c, fmt.Errorf("%w: %v", ErrDependencyUnavailable, err))
			return
		}
		c.JSON(http.StatusOK, key)
	})
}

func (h *HTTPHandler) authorize(c *gin.Context, projectID, action string) (*Principal, bool) {
	if h.authorizer == nil {
		writeError(c, ErrAuthUnavailable)
		return nil, false
	}
	token, err := bearerToken(c.GetHeader("Authorization"))
	if err != nil {
		writeError(c, err)
		return nil, false
	}
	principal, err := h.authorizer.Authorize(c.Request.Context(), token, projectID, action)
	if err != nil {
		writeError(c, err)
		return nil, false
	}
	if principal == nil || principal.ID == "" {
		writeError(c, ErrAuthUnavailable)
		return nil, false
	}
	return principal, true
}

func (h *HTTPHandler) batch(c *gin.Context) {
	var request ObjectBatchRequest
	if err := decodeJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	if _, ok := h.authorize(c, request.ProjectID, ActionCASWrite); !ok {
		return
	}
	response, err := h.service.Batch(request)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *HTTPHandler) objectProject(c *gin.Context, action string) (string, bool) {
	projectID := c.Query("project_id")
	if _, ok := h.authorize(c, projectID, action); !ok {
		return "", false
	}
	reserved, err := h.service.store.ProjectHasObject(projectID, c.Param("ciphertext_cid"))
	if err != nil {
		writeError(c, err)
		return "", false
	}
	if !reserved {
		writeError(c, ErrNotFound)
		return "", false
	}
	return projectID, true
}

func (h *HTTPHandler) putObject(c *gin.Context) {
	if _, ok := h.objectProject(c, ActionCASWrite); !ok {
		return
	}
	size, existed, err := h.service.store.PutObject(c.Request.Context(), c.Param("ciphertext_cid"), c.Request.Body)
	if err != nil {
		writeError(c, err)
		return
	}
	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"cid": normalizedOrInput(c.Param("ciphertext_cid")), "size": size, "existed": existed})
}

func (h *HTTPHandler) headObject(c *gin.Context) {
	if _, ok := h.objectProject(c, ActionCASRead); !ok {
		return
	}
	exists, size, err := h.service.store.HasObject(c.Param("ciphertext_cid"))
	if err != nil {
		writeError(c, err)
		return
	}
	if !exists {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("ETag", `"`+normalizedOrInput(c.Param("ciphertext_cid"))+`"`)
	c.Status(http.StatusOK)
}

func (h *HTTPHandler) getObject(c *gin.Context) {
	if _, ok := h.objectProject(c, ActionCASRead); !ok {
		return
	}
	file, size, err := h.service.store.OpenObject(c.Param("ciphertext_cid"))
	if err != nil {
		writeError(c, err)
		return
	}
	defer file.Close()
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Cache-Control", "private, immutable")
	c.Header("ETag", `"`+normalizedOrInput(c.Param("ciphertext_cid"))+`"`)
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, file)
}

func (h *HTTPHandler) submitProof(c *gin.Context) {
	projectID := c.Param("project_id")
	principal, ok := h.authorize(c, projectID, ActionProofSubmit)
	if !ok {
		return
	}
	var submission ProofSubmission
	if err := decodeJSON(c, &submission); err != nil {
		writeError(c, err)
		return
	}
	if submission.ProjectID != projectID {
		writeError(c, fmt.Errorf("path and body project ids differ"))
		return
	}
	if submission.SignerID != principal.ID {
		writeError(c, ErrForbidden)
		return
	}
	response, created, err := h.service.Submit(submission)
	if err != nil {
		writeError(c, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusAccepted
	}
	c.JSON(status, response)
	go func() { _ = h.service.Process(contextWithoutCancel(c.Request.Context()), response.OperationID) }()
}

func (h *HTTPHandler) getProof(c *gin.Context) {
	projectID := c.Param("project_id")
	if _, ok := h.authorize(c, projectID, ActionProofRead); !ok {
		return
	}
	record, err := h.service.store.GetProof(projectID, c.Param("proof_root"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (h *HTTPHandler) getOperation(c *gin.Context) {
	projectID := c.Param("project_id")
	if _, ok := h.authorize(c, projectID, ActionProofRead); !ok {
		return
	}
	operation, err := h.service.store.GetOperation(c.Param("operation_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	if operation.ProjectID != projectID {
		writeError(c, ErrNotFound)
		return
	}
	c.JSON(http.StatusOK, operation)
	if !operation.Status.Terminal() && operation.Retryable && time.Since(operation.UpdatedAt) > time.Second {
		go func() { _ = h.service.Process(contextWithoutCancel(c.Request.Context()), operation.ID) }()
	}
}

func (h *HTTPHandler) addEnvelope(c *gin.Context) {
	projectID := c.Param("project_id")
	if _, ok := h.authorize(c, projectID, ActionProofShare); !ok {
		return
	}
	var request AccessEnvelopeRequest
	if err := decodeJSON(c, &request); err != nil {
		writeError(c, err)
		return
	}
	record, err := h.service.AddEnvelope(projectID, c.Param("proof_root"), request.Envelope)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"proof_root": record.Submission.ProofRoot, "recipient_id": request.Envelope.RecipientID})
}

func (h *HTTPHandler) publicProof(c *gin.Context) {
	proof, err := h.service.PublicProof(c.Param("project_id"), c.Param("commit_digest"))
	if err != nil {
		writeError(c, err)
		return
	}
	if proof.Status != StatusCertified || proof.Receipt == nil || !proof.Receipt.Final {
		writeError(c, ErrNotFound)
		return
	}
	c.JSON(http.StatusOK, proof)
}

func decodeJSON(c *gin.Context, destination any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("JSON body must contain one value")
	}
	return nil
}

func writeError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	code := "invalid_request"
	switch {
	case errors.Is(err, ErrUnauthorized):
		status, code = http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, ErrForbidden):
		status, code = http.StatusForbidden, "forbidden"
	case errors.Is(err, ErrAuthUnavailable), errors.Is(err, ErrDependencyUnavailable):
		status, code = http.StatusServiceUnavailable, "dependency_unavailable"
	case errors.Is(err, ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, ErrConflict):
		status, code = http.StatusConflict, "conflict"
	case errors.Is(err, ErrTooLarge):
		status, code = http.StatusRequestEntityTooLarge, "object_too_large"
	}
	c.JSON(status, gin.H{"error": code, "message": err.Error()})
}

func normalizedOrInput(value string) string {
	normalized, err := NormalizeSHA256(value)
	if err != nil {
		return strings.ToLower(value)
	}
	return normalized
}

func contextWithoutCancel(parent interface{ Value(any) any }) *detachedContext {
	return &detachedContext{parent: parent}
}

type detachedContext struct {
	parent interface{ Value(any) any }
}

func (c *detachedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *detachedContext) Done() <-chan struct{}       { return nil }
func (c *detachedContext) Err() error                  { return nil }
func (c *detachedContext) Value(key any) any           { return c.parent.Value(key) }
