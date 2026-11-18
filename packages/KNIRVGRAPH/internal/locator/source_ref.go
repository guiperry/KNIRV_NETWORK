package locator

import (
	"fmt"
	"regexp"
	"strings"

	gproto "google.golang.org/protobuf/proto"
)

// SourceRef is the proto message recording where an ErrorNode was first
// observed (shared-proto graph/v1 SourceRef, plan D5.4a).
//
// It holds pointers and digests only. It never holds code, diffs or file
// contents. ValidateSourceRef enforces that by charset and length rather than
// by trusting the caller, because ErrorNodes are global and public.

// maxSourceLine bounds line numbers. A real source file is far shorter.
const maxSourceLine = 10_000_000

var (
	sourceIdentRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
	sourceRefRe   = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,256}$`)
	sourcePathRe  = regexp.MustCompile(`^[A-Za-z0-9._/@+-]{1,512}$`)
	sourceSHARe   = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	sourceDigest  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// ValidateSourceRef rejects any SourceRef whose fields are not pointers or
// digests. A nil source is valid: older rows and direct creates have none.
func ValidateSourceRef(s *SourceRef) error {
	if s == nil {
		return nil
	}
	if s.ProjectId != "" && !sourceIdentRe.MatchString(s.ProjectId) {
		return fmt.Errorf("source project_id is not a valid repo identity")
	}
	if s.SessionId != "" && !sourceIdentRe.MatchString(s.SessionId) {
		return fmt.Errorf("source session_id is not a valid identifier")
	}
	if s.Ref != "" && (!sourceRefRe.MatchString(s.Ref) || strings.Contains(s.Ref, "..")) {
		return fmt.Errorf("source ref is not a valid branch or tag name")
	}
	if s.Path != "" {
		if !sourcePathRe.MatchString(s.Path) || strings.HasPrefix(s.Path, "/") || hasDotDotSegment(s.Path) {
			return fmt.Errorf("source path must be a repo-relative file path")
		}
	}
	if s.CommitSha != "" && !sourceSHARe.MatchString(s.CommitSha) {
		return fmt.Errorf("source commit_sha must be a hex object id")
	}
	for name, digest := range map[string]string{
		"diff_digest":          s.DiffDigest,
		"workspace_base_hash":  s.WorkspaceBaseHash,
		"workspace_final_hash": s.WorkspaceFinalHash,
	} {
		if digest != "" && !sourceDigest.MatchString(digest) {
			return fmt.Errorf("source %s must be sha256:<64 hex>", name)
		}
	}
	if s.LineStart > maxSourceLine || s.LineEnd > maxSourceLine {
		return fmt.Errorf("source line number out of range")
	}
	if s.Path == "" && (s.LineStart != 0 || s.LineEnd != 0) {
		return fmt.Errorf("source line range requires a path")
	}
	if s.LineStart == 0 && s.LineEnd != 0 {
		return fmt.Errorf("source line_end requires line_start")
	}
	// line_end of 0 means a single line at line_start.
	if s.LineStart != 0 && s.LineEnd != 0 && s.LineEnd < s.LineStart {
		return fmt.Errorf("source line_end is before line_start")
	}
	return nil
}

func hasDotDotSegment(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

// PublicSourceRef returns the view a public reader may see: digests only. The
// project, session, ref, path, lines and commit are private to the owner and
// to validators (plan D-12). The input is not modified.
func PublicSourceRef(s *SourceRef) *SourceRef {
	if s == nil {
		return nil
	}
	return &SourceRef{
		DiffDigest:         s.DiffDigest,
		WorkspaceBaseHash:  s.WorkspaceBaseHash,
		WorkspaceFinalHash: s.WorkspaceFinalHash,
	}
}

// PublicErrorNode returns a copy of node as the public may see it (Q83): the
// source is cut to its digests, and the context and metadata lose code_context,
// project_path and any absolute path (see PublicStruct), and the description,
// failure_context and the top-level text fields (error_type, error_signature,
// domain, model_origin, resolved_by, cluster_id) lose any path (see redactText),
// and each resolution step's action, skill_id and parameters are reduced the
// same way. The input is not modified; the owner and validators keep the full stored
// record.
func PublicErrorNode(node *ErrorNode) *ErrorNode {
	if node == nil {
		return nil
	}
	pub := gproto.Clone(node).(*ErrorNode)
	if node.Source != nil {
		pub.Source = PublicSourceRef(node.Source)
	}
	pub.Description = redactText(node.Description)
	pub.ErrorType = redactText(node.ErrorType)
	pub.ErrorSignature = redactText(node.ErrorSignature)
	pub.Domain = redactText(node.Domain)
	pub.ModelOrigin = redactText(node.ModelOrigin)
	pub.ResolvedBy = redactText(node.ResolvedBy)
	pub.ClusterId = redactText(node.ClusterId)
	if redacted := redactText(string(node.FailureContext)); redacted != string(node.FailureContext) {
		pub.FailureContext = []byte(redacted)
	}
	// Resolution steps carry the error context as parameters (findResolutionPath),
	// so the public copy reduces each step the same way as Context. pub is a
	// deep clone, so the stored steps are not touched.
	if pub.Resolution != nil {
		for _, step := range pub.Resolution.Steps {
			if step == nil {
				continue
			}
			step.Action = redactText(step.Action)
			step.SkillId = redactText(step.SkillId)
			step.Parameters = PublicStruct(step.Parameters)
		}
	}
	pub.Context = PublicStruct(node.Context)
	pub.Metadata = PublicStruct(node.Metadata)
	return pub
}

// PublicErrorNodes applies PublicErrorNode to every node in the slice.
func PublicErrorNodes(nodes []*ErrorNode) []*ErrorNode {
	out := make([]*ErrorNode, len(nodes))
	for i, n := range nodes {
		out[i] = PublicErrorNode(n)
	}
	return out
}
