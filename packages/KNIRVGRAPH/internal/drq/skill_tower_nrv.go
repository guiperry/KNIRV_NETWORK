package drq

import (
	"context"
	"fmt"
	"strings"

	"KNIRVGRAPH/internal/nrv"
)

// NRVSkillTowerStore adapts KNIRVGRAPH's NRVSystem skill registry to the
// SkillTowerStore port, so DRQ's minting writes into the graph's real skill
// record store instead of a no-op.
//
// "Idempotent per skill id" is implemented as "already present is success":
// the chain-side mint this pairs with is itself idempotent per skill ID, so a
// re-mint of an existing skill must not be reported as a failure.
type NRVSkillTowerStore struct {
	system *nrv.NRVSystem
}

// NewNRVSkillTowerStore wraps an NRVSystem.
func NewNRVSkillTowerStore(system *nrv.NRVSystem) *NRVSkillTowerStore {
	return &NRVSkillTowerStore{system: system}
}

// PutSkill implements SkillTowerStore.
func (s *NRVSkillTowerStore) PutSkill(_ context.Context, skillID, skillType string, capabilities []string, attributes map[string]interface{}) error {
	if s == nil || s.system == nil {
		return fmt.Errorf("NRV skill registry is not configured")
	}
	if s.system.HasSkillNode(skillID) {
		// Already minted — the idempotent no-op case.
		return nil
	}
	if _, err := s.system.CreateSkillNodeWithID(skillID, skillType, capabilities, attributes); err != nil {
		return fmt.Errorf("create NRV skill node %s: %w", skillID, err)
	}
	return nil
}

// DeleteSkill implements SkillTowerStore.
func (s *NRVSkillTowerStore) DeleteSkill(_ context.Context, skillID string) error {
	if s == nil || s.system == nil {
		return fmt.Errorf("NRV skill registry is not configured")
	}
	if !s.system.HasSkillNode(skillID) {
		// Nothing to roll back.
		return nil
	}
	if err := s.system.DeleteSkillNode(skillID); err != nil {
		return fmt.Errorf("delete NRV skill node %s: %w", skillID, err)
	}
	return nil
}

// SkillResolvingError implements SkillTowerStore by scanning the registry for a
// skill whose attributes list errorID among the errors it resolves.
//
// A scan is acceptable here because the registry is in-process and only walked
// when a cluster converges; an index would be premature. The link is the
// resolves_errors attribute MintSkillTower already writes, so no new state is
// introduced to make this work.
func (s *NRVSkillTowerStore) SkillResolvingError(_ context.Context, errorID string) (*SkillRecord, error) {
	if s == nil || s.system == nil {
		return nil, fmt.Errorf("NRV skill registry is not configured")
	}
	errorID = strings.TrimSpace(errorID)
	if errorID == "" {
		return nil, fmt.Errorf("error id is required to look up a resolving skill")
	}

	for _, node := range s.system.GetAllSkillNodes() {
		if node == nil {
			continue
		}
		resolved := resolvedErrorIDs(node.Requirements)
		for _, id := range resolved {
			if id != errorID {
				continue
			}
			return &SkillRecord{
				SkillID:     node.ID,
				Creator:     stringAttr(node.Requirements, "creator"),
				Description: stringAttr(node.Requirements, "description"),
				// The fix document a uLoRA compile trains on: the skill's
				// description, which the discovery engine builds from the
				// cluster's real contents rather than a placeholder.
				ResolvesErrors: resolved,
				CodePackageURI: stringAttr(node.Requirements, "code_package_uri"),
			}, nil
		}
	}
	// No skill resolves this error yet — the caller skips the member.
	return nil, nil
}

// resolvedErrorIDs reads the resolves_errors attribute, tolerating both the
// []string written in-process and the []interface{} produced by a JSON
// round trip.
func resolvedErrorIDs(attributes map[string]interface{}) []string {
	raw, ok := attributes["resolves_errors"]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return typed
	case []interface{}:
		ids := make([]string, 0, len(typed))
		for _, entry := range typed {
			if id, ok := entry.(string); ok {
				ids = append(ids, id)
			}
		}
		return ids
	default:
		return nil
	}
}

func stringAttr(attributes map[string]interface{}, key string) string {
	if value, ok := attributes[key].(string); ok {
		return value
	}
	return ""
}

// TowerSkillDocSource implements SkillDocSource over a SkillTowerStore, giving
// the uLoRA corpus each member error's validated fix.
type TowerSkillDocSource struct {
	store SkillTowerStore
}

// NewTowerSkillDocSource builds the doc source.
func NewTowerSkillDocSource(store SkillTowerStore) *TowerSkillDocSource {
	return &TowerSkillDocSource{store: store}
}

// SkillDocForError implements SkillDocSource. A member error with no minted
// skill yields an empty document and no error, which the corpus builder treats
// as "skip this member"; only a failed lookup is an error.
func (s *TowerSkillDocSource) SkillDocForError(errorID string) (string, error) {
	if s == nil || s.store == nil {
		return "", fmt.Errorf("no skill tower store configured")
	}
	record, err := s.store.SkillResolvingError(context.Background(), errorID)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", nil
	}
	return strings.TrimSpace(record.Description), nil
}
