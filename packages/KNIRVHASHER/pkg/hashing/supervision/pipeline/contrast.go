package pipeline

import (
	"fmt"
	"knirvhasher/pkg/hashing/schema"
	"reflect"
)

func MinimalityValidator(anchor, variant *schema.SupervisionEpisode, field string) error {
	if anchor == nil || variant == nil {
		return fmt.Errorf("nil contrast episode")
	}
	if anchor.EpisodeID == variant.EpisodeID {
		return fmt.Errorf("contrast requires distinct episodes")
	}
	changed := []string{}
	if !reflect.DeepEqual(anchor.Task, variant.Task) {
		changed = append(changed, "task")
	}
	if !reflect.DeepEqual(anchor.StateBefore, variant.StateBefore) {
		changed = append(changed, "state_before")
	}
	if !reflect.DeepEqual(anchor.PolicyContext, variant.PolicyContext) {
		changed = append(changed, "policy_context")
	}
	if !reflect.DeepEqual(anchor.Evidence, variant.Evidence) {
		changed = append(changed, "evidence")
	}
	if !reflect.DeepEqual(anchor.Trace, variant.Trace) {
		changed = append(changed, "trace")
	}
	if !reflect.DeepEqual(anchor.Decision, variant.Decision) {
		changed = append(changed, "decision")
	}
	if !reflect.DeepEqual(anchor.Outcome, variant.Outcome) {
		changed = append(changed, "outcome")
	}
	if anchor.Provenance.PolicyBundleHash != variant.Provenance.PolicyBundleHash {
		changed = append(changed, "provenance.policy_bundle_hash")
	}
	if len(changed) != 1 || changed[0] != field {
		return fmt.Errorf("minimality violation: changed=%v declared=%q", changed, field)
	}
	return nil
}
func CurateContrastSet(anchor, variant *schema.SupervisionEpisode, field string, relation schema.ContrastRelation, label schema.LabelSource, review schema.ReviewInfo) (*schema.ContrastSet, error) {
	if err := MinimalityValidator(anchor, variant, field); err != nil {
		return nil, err
	}
	c := &schema.ContrastSet{SchemaVersion: schema.SupervisionSchemaVersion, AnchorEpisodeID: anchor.EpisodeID, VariantEpisodeID: variant.EpisodeID, Relation: relation, Intervention: schema.InterventionInfo{Field: field, SemanticDelta: "single declared field changed"}, Expected: schema.ExpectedDiff{AnchorAction: anchor.Decision.RecommendedAction, VariantAction: variant.Decision.RecommendedAction}, LabelSource: label, Review: review}
	c.ContrastSetID = c.ComputeContrastSetID()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}
