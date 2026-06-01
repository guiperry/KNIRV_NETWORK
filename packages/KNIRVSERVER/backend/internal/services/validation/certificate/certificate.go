// Package certificate provides canonical types for PQC-signed certificates of correctness.
// These types are promoted from the fintech plugin package to a platform-level
// package so they can be shared across compliance validators.
package certificate

import (
	"backend_server/internal/services/plugins/fintech"
)

// Re-export canonical types from the fintech package.

// CertificateOfCorrectness is an alias for fintech.CertificateOfCorrectness.
type CertificateOfCorrectness = fintech.CertificateOfCorrectness

// SubjectInfo is an alias for fintech.SubjectInfo.
type SubjectInfo = fintech.SubjectInfo

// ValidationResult is an alias for fintech.ValidationResult.
type ValidationResult = fintech.ValidationResult

// CertificateTemplate is an alias for fintech.CertificateTemplate.
type CertificateTemplate = fintech.CertificateTemplate

// Builder is an alias for fintech.CertificateBuilder.
type Builder = fintech.CertificateBuilder

// Manager is an alias for fintech.CertificateManager.
type Manager = fintech.CertificateManager

// DefaultTemplates re-exports the default certificate templates.
var DefaultTemplates = fintech.DefaultTemplates

// NewBuilder creates a new CertificateBuilder.
func NewBuilder() *Builder {
	return fintech.NewCertificateBuilder()
}
