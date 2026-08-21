package dveevidence

// This package adds server-side ingestion and persistence around the SDK's
// evidence model. Explicit aliases keep that SDK model available to sibling
// server packages; a dot import in ingest.go is file-scoped and cannot do so.
import sdk "github.com/guiperry/knirv-sdk-go/dveevidence"

const (
	SchemaVersion              = sdk.SchemaVersion
	StatusRejected             = sdk.StatusRejected
	StatusVerified             = sdk.StatusVerified
	StatusVerifiedWithWarnings = sdk.StatusVerifiedWithWarnings
)

type (
	ArtifactRef        = sdk.ArtifactRef
	Bundle             = sdk.Bundle
	CheckResult        = sdk.CheckResult
	Evidence           = sdk.Evidence
	Event              = sdk.Event
	KeyResolver        = sdk.KeyResolver
	MemvidRef          = sdk.MemvidRef
	PermissionDecision = sdk.PermissionDecision
	Policy             = sdk.Policy
	Signer             = sdk.Signer
	ToolRun            = sdk.ToolRun
	ValidationReport   = sdk.ValidationReport
	VerifyError        = sdk.VerifyError
	VerifyOptions      = sdk.VerifyOptions
)

var (
	BuildEventLog             = sdk.BuildEventLog
	MerkleRoot                = sdk.MerkleRoot
	ReplayPolicy              = sdk.ReplayPolicy
	ResolverFromPublicKeys    = sdk.ResolverFromPublicKeys
	ResolveKeyResolverFromEnv = sdk.ResolveKeyResolverFromEnv
	ResolvePolicyFromEnv      = sdk.ResolvePolicyFromEnv
	SignerFromSeed            = sdk.SignerFromSeed
	VerifyEventLog            = sdk.VerifyEventLog
	VerifySignature           = sdk.VerifySignature
	VerifyBundle              = sdk.VerifyBundle
)
