package schema

import (
	"strings"
	"testing"
	"time"
)

func TestSupervisionEpisode_Validate(t *testing.T) {
	validEpisode := SupervisionEpisode{
		SchemaVersion: SupervisionSchemaVersion,
		EpisodeID:     "sha256:abc123",
		TenantScope:   TenantScopeLocal,
		Workspace: WorkspaceInfo{
			Fingerprint: "fp123",
			RepoFamily:  "knirv",
			Branch:      "main",
		},
		Task: TaskInfo{
			Request: "test request",
			Intent:  IntentDevelop,
			Risk:    RiskMedium,
		},
		StateBefore: StateBefore{
			Phase:         PhaseDevelop,
			Budget:        map[string]int{"iterations": 3},
			AgentChain:    []string{"claude", "codex"},
			LoopIteration: 2,
		},
		PolicyContext: []PolicyRef{
			{ID: "G8", ContentHash: "hash123", Applicability: "required"},
		},
		Evidence: []EvidenceRef{
			{Class: EvidenceClassDiff, Hash: "diffhash", Summary: "redacted diff"},
		},
		Trace: TraceInfo{
			Agent:       "codex",
			Proposal:    "redacted proposal",
			ToolClasses: []ActionClass{ActionClassRead, ActionClassTest},
		},
		Decision: DecisionLabel{
			RecommendedAction: "request_review",
			NextPhase:         PhaseDevelopReview,
			RequiredEvidence:  []EvidenceClass{EvidenceClassReport},
			Approval:          "not_required",
			Abstain:           false,
			Confidence:        0.93,
		},
		Outcome: OutcomeLabel{
			Verified:      true,
			TestsPassed:   true,
			HumanOverride: false,
		},
		Provenance: ProvenanceInfo{
			Source:           "cli-supervisor",
			PolicyBundleHash: "policyhash123",
			RedactionVersion: "v1",
			License:          "operator-generated",
			CreatedAt:        time.Now().UTC(),
		},
	}

	if err := validEpisode.Validate(); err != nil {
		t.Fatalf("valid episode should pass validation: %v", err)
	}

	invalidEpisode := validEpisode
	invalidEpisode.SchemaVersion = 999
	if err := invalidEpisode.Validate(); err == nil {
		t.Fatal("invalid schema version should fail validation")
	}

	invalidEpisode = validEpisode
	invalidEpisode.EpisodeID = ""
	if err := invalidEpisode.Validate(); err == nil {
		t.Fatal("empty episode_id should fail validation")
	}

	invalidEpisode = validEpisode
	invalidEpisode.TenantScope = "invalid"
	if err := invalidEpisode.Validate(); err == nil {
		t.Fatal("invalid tenant_scope should fail validation")
	}

	invalidEpisode = validEpisode
	invalidEpisode.Workspace.Fingerprint = ""
	if err := invalidEpisode.Validate(); err == nil {
		t.Fatal("empty workspace fingerprint should fail validation")
	}

	invalidEpisode = validEpisode
	invalidEpisode.Task.Request = ""
	if err := invalidEpisode.Validate(); err == nil {
		t.Fatal("empty task request should fail validation")
	}

	invalidEpisode = validEpisode
	invalidEpisode.Decision.RecommendedAction = ""
	if err := invalidEpisode.Validate(); err == nil {
		t.Fatal("empty recommended_action should fail validation")
	}

	invalidEpisode = validEpisode
	invalidEpisode.Provenance.PolicyBundleHash = ""
	if err := invalidEpisode.Validate(); err == nil {
		t.Fatal("empty policy_bundle_hash should fail validation")
	}
}

func TestSupervisionEpisode_ComputeEpisodeID(t *testing.T) {
	episode := SupervisionEpisode{
		SchemaVersion: SupervisionSchemaVersion,
		EpisodeID:     "sha256:placeholder",
		TenantScope:   TenantScopeLocal,
		Workspace: WorkspaceInfo{
			Fingerprint: "fp123",
			RepoFamily:  "knirv",
			Branch:      "main",
		},
		Task: TaskInfo{
			Request: "test request",
			Intent:  IntentDevelop,
			Risk:    RiskMedium,
		},
		StateBefore: StateBefore{
			Phase:         PhaseDevelop,
			Budget:        map[string]int{"iterations": 3},
			AgentChain:    []string{"claude", "codex"},
			LoopIteration: 2,
		},
		PolicyContext: []PolicyRef{
			{ID: "G8", ContentHash: "hash123", Applicability: "required"},
		},
		Evidence: []EvidenceRef{
			{Class: EvidenceClassDiff, Hash: "diffhash", Summary: "redacted diff"},
		},
		Trace: TraceInfo{
			Agent:       "codex",
			Proposal:    "redacted proposal",
			ToolClasses: []ActionClass{ActionClassRead, ActionClassTest},
		},
		Decision: DecisionLabel{
			RecommendedAction: "request_review",
			NextPhase:         PhaseDevelopReview,
			RequiredEvidence:  []EvidenceClass{EvidenceClassReport},
			Approval:          "not_required",
			Abstain:           false,
			Confidence:        0.93,
		},
		Outcome: OutcomeLabel{
			Verified:      true,
			TestsPassed:   true,
			HumanOverride: false,
		},
		Provenance: ProvenanceInfo{
			Source:           "cli-supervisor",
			PolicyBundleHash: "policyhash123",
			RedactionVersion: "v1",
			License:          "operator-generated",
			CreatedAt:        time.Now().UTC(),
		},
	}

	computed := episode.ComputeEpisodeID()
	if computed == "" || computed == "sha256:" {
		t.Fatal("computed episode ID should not be empty")
	}
	if !strings.HasPrefix(computed, "sha256:") {
		t.Fatalf("episode ID should have sha256 prefix: %s", computed)
	}

	episode.EpisodeID = computed
	if err := episode.VerifyEpisodeID(); err != nil {
		t.Fatalf("verified episode ID should match: %v", err)
	}

	episode.EpisodeID = "sha256:wrong"
	if err := episode.VerifyEpisodeID(); err == nil {
		t.Fatal("mismatched episode ID should fail verification")
	}
}

func TestContrastSet_Validate(t *testing.T) {
	validContrast := ContrastSet{
		SchemaVersion:    SupervisionSchemaVersion,
		ContrastSetID:    "sha256:contrast123",
		AnchorEpisodeID:  "sha256:anchor123",
		VariantEpisodeID: "sha256:variant123",
		Relation:         ContrastMustChange,
		Intervention: InterventionInfo{
			Field:         "evidence.diff_present",
			Before:        "false",
			After:         "true",
			SemanticDelta: "one required artifact was supplied",
		},
		Expected: ExpectedDiff{
			AnchorAction:    "request_evidence",
			VariantAction:   "request_review",
			InvariantFields: []string{"task.intent", "policy_bundle_hash", "risk"},
		},
		LabelSource: LabelSourceDeterministicGate,
		Review: ReviewInfo{
			Status:        ReviewStatusApproved,
			Reviewer:      "operator-alias",
			RubricVersion: "v1",
		},
	}

	if err := validContrast.Validate(); err != nil {
		t.Fatalf("valid contrast set should pass validation: %v", err)
	}

	invalidContrast := validContrast
	invalidContrast.SchemaVersion = 999
	if err := invalidContrast.Validate(); err == nil {
		t.Fatal("invalid schema version should fail validation")
	}

	invalidContrast = validContrast
	invalidContrast.AnchorEpisodeID = ""
	if err := invalidContrast.Validate(); err == nil {
		t.Fatal("empty anchor_episode_id should fail validation")
	}

	invalidContrast = validContrast
	invalidContrast.AnchorEpisodeID = invalidContrast.VariantEpisodeID
	if err := invalidContrast.Validate(); err == nil {
		t.Fatal("anchor and variant episode IDs must differ")
	}

	invalidContrast = validContrast
	invalidContrast.Relation = "invalid"
	if err := invalidContrast.Validate(); err == nil {
		t.Fatal("invalid contrast relation should fail validation")
	}

	invalidContrast = validContrast
	invalidContrast.Intervention.Field = ""
	if err := invalidContrast.Validate(); err == nil {
		t.Fatal("empty intervention field should fail validation")
	}
}

func TestContrastSet_ComputeContrastSetID(t *testing.T) {
	contrast := ContrastSet{
		SchemaVersion:    SupervisionSchemaVersion,
		ContrastSetID:    "sha256:placeholder",
		AnchorEpisodeID:  "sha256:anchor123",
		VariantEpisodeID: "sha256:variant123",
		Relation:         ContrastMustChange,
		Intervention: InterventionInfo{
			Field:         "evidence.diff_present",
			Before:        "false",
			After:         "true",
			SemanticDelta: "one required artifact was supplied",
		},
		Expected: ExpectedDiff{
			AnchorAction:    "request_evidence",
			VariantAction:   "request_review",
			InvariantFields: []string{"task.intent", "policy_bundle_hash", "risk"},
		},
		LabelSource: LabelSourceDeterministicGate,
		Review: ReviewInfo{
			Status:        ReviewStatusApproved,
			Reviewer:      "operator-alias",
			RubricVersion: "v1",
		},
	}

	computed := contrast.ComputeContrastSetID()
	if computed == "" || computed == "sha256:" {
		t.Fatal("computed contrast set ID should not be empty")
	}
	if !strings.HasPrefix(computed, "sha256:") {
		t.Fatalf("contrast set ID should have sha256 prefix: %s", computed)
	}

	contrast.ContrastSetID = computed
	if err := contrast.VerifyContrastSetID(); err != nil {
		t.Fatalf("verified contrast set ID should match: %v", err)
	}
}

func TestDatasetManifest_Validate(t *testing.T) {
	manifest := DatasetManifest{
		SchemaVersion:    SupervisionSchemaVersion,
		ManifestID:       "sha256:manifest123",
		CreatedAt:        time.Now().UTC(),
		PolicyBundleHash: "policyhash123",
		RedactionVersion: "v1",
		EmbeddingModel:   "bge-base",
		EmbeddingVersion: "1.0",
		EmbeddingDims:    768,
		TokenizerVersion: "cl100k_base",
		EpisodeCount:     2,
		ContrastSetCount: 1,
		EpisodeIDs:       []string{"sha256:ep1", "sha256:ep2"},
		ContrastSetIDs:   []string{"sha256:cs1"},
		SplitAssignments: map[string]SplitName{
			"sha256:ep1": SplitTrain,
			"sha256:ep2": SplitValidation,
			"sha256:cs1": SplitTest,
		},
		SourceHashes: []string{"source1", "source2"},
		License:      "operator-generated",
	}

	if err := manifest.Validate(); err != nil {
		t.Fatalf("valid manifest should pass validation: %v", err)
	}

	invalidManifest := manifest
	invalidManifest.SchemaVersion = 999
	if err := invalidManifest.Validate(); err == nil {
		t.Fatal("invalid schema version should fail validation")
	}

	invalidManifest = manifest
	invalidManifest.ManifestID = ""
	if err := invalidManifest.Validate(); err == nil {
		t.Fatal("empty manifest_id should fail validation")
	}

	invalidManifest = manifest
	invalidManifest.EpisodeCount = 3
	if err := invalidManifest.Validate(); err == nil {
		t.Fatal("episode_count mismatch should fail validation")
	}

	invalidManifest = manifest
	invalidManifest.SplitAssignments["sha256:ep1"] = "invalid"
	if err := invalidManifest.Validate(); err == nil {
		t.Fatal("invalid split should fail validation")
	}

	invalidManifest = manifest
	invalidManifest.EpisodeIDs = []string{"sha256:ep1", "sha256:ep1"}
	if err := invalidManifest.Validate(); err == nil {
		t.Fatal("duplicate episode_ids should fail validation")
	}
}

func TestDatasetManifest_ComputeManifestID(t *testing.T) {
	manifest := DatasetManifest{
		SchemaVersion:    SupervisionSchemaVersion,
		ManifestID:       "sha256:placeholder",
		CreatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PolicyBundleHash: "policyhash123",
		RedactionVersion: "v1",
		EmbeddingModel:   "bge-base",
		EmbeddingVersion: "1.0",
		EmbeddingDims:    768,
		TokenizerVersion: "cl100k_base",
		EpisodeCount:     2,
		ContrastSetCount: 1,
		EpisodeIDs:       []string{"sha256:ep2", "sha256:ep1"},
		ContrastSetIDs:   []string{"sha256:cs1"},
		SplitAssignments: map[string]SplitName{
			"sha256:ep2": SplitTrain,
			"sha256:ep1": SplitValidation,
			"sha256:cs1": SplitTest,
		},
		SourceHashes: []string{"source2", "source1"},
		License:      "operator-generated",
	}

	computed := manifest.ComputeManifestID()
	if computed == "" || computed == "sha256:" {
		t.Fatal("computed manifest ID should not be empty")
	}

	manifest.ManifestID = computed
	if err := manifest.VerifyManifestID(); err != nil {
		t.Fatalf("verified manifest ID should match: %v", err)
	}

	manifest2 := manifest
	manifest2.EpisodeIDs = []string{"sha256:ep1", "sha256:ep2"}
	computed2 := manifest2.ComputeManifestID()
	if computed != computed2 {
		t.Fatal("manifest ID should be deterministic regardless of episode order")
	}
}

func TestSplitLock(t *testing.T) {
	manifest := DatasetManifest{
		SchemaVersion:    SupervisionSchemaVersion,
		ManifestID:       "sha256:manifest123",
		CreatedAt:        time.Now().UTC(),
		PolicyBundleHash: "policyhash123",
		RedactionVersion: "v1",
		EmbeddingModel:   "bge-base",
		EmbeddingVersion: "1.0",
		EmbeddingDims:    768,
		TokenizerVersion: "cl100k_base",
		EpisodeCount:     2,
		ContrastSetCount: 1,
		EpisodeIDs:       []string{"sha256:ep1", "sha256:ep2"},
		ContrastSetIDs:   []string{"sha256:cs1"},
		SplitAssignments: map[string]SplitName{
			"sha256:ep1": SplitTrain,
			"sha256:ep2": SplitValidation,
			"sha256:cs1": SplitTest,
		},
		SourceHashes: []string{"source1"},
		License:      "operator-generated",
	}

	lock := NewSplitLock(&manifest)
	if lock.ManifestID != manifest.ManifestID {
		t.Fatal("split lock manifest ID should match")
	}
	if len(lock.SplitAssignments) != 3 {
		t.Fatal("split lock should have all assignments")
	}

	if err := lock.VerifyAgainstManifest(&manifest); err != nil {
		t.Fatalf("split lock should verify against its manifest: %v", err)
	}

	badManifest := manifest
	badManifest.SplitAssignments["sha256:ep1"] = SplitTest
	if err := lock.VerifyAgainstManifest(&badManifest); err == nil {
		t.Fatal("split lock should detect assignment mismatch")
	}

	badManifest2 := manifest
	badManifest2.ManifestID = "sha256:different"
	if err := lock.VerifyAgainstManifest(&badManifest2); err == nil {
		t.Fatal("split lock should detect manifest ID mismatch")
	}
}