package proofasset

// Verification status constants. Only a successful formal-checker receipt may
// produce FORMALLY_VERIFIED. HARDWARE_ATTESTED is additive metadata and must
// not replace or imply FORMALLY_VERIFIED.
const (
	StatusStructurallyValid    = "STRUCTURALLY_VALID"
	StatusStructurallyRejected = "STRUCTURALLY_REJECTED"
	StatusProofPending         = "PROOF_PENDING"
	StatusFormallyVerified     = "FORMALLY_VERIFIED"
	StatusFormallyRejected     = "FORMALLY_REJECTED"
	StatusCheckerUnavailable   = "CHECKER_UNAVAILABLE"
	StatusAttestationPending   = "ATTESTATION_PENDING"
	StatusHardwareAttested     = "HARDWARE_ATTESTED"
)

// Diagnostic taxonomy labels for checker feedback. These are non-authoritative
// search labels only; the raw checker result remains the audit record.
type DiagnosticTaxonomy string

const (
	DiagnosticParseError          DiagnosticTaxonomy = "PARSE_ERROR"
	DiagnosticUnknownIdentifier   DiagnosticTaxonomy = "UNKNOWN_IDENTIFIER"
	DiagnosticTypeMismatch        DiagnosticTaxonomy = "TYPE_MISMATCH"
	DiagnosticUnsolvedGoal        DiagnosticTaxonomy = "UNSOLVED_GOAL"
	DiagnosticImportPolicyDenied  DiagnosticTaxonomy = "IMPORT_POLICY_DENIED"
	DiagnosticResourceLimit       DiagnosticTaxonomy = "RESOURCE_LIMIT"
	DiagnosticCheckerFailure      DiagnosticTaxonomy = "CHECKER_FAILURE"
)
