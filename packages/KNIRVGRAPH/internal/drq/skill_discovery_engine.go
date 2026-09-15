package drq

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// KNIRVCHAINClient talks to KNIRVCHAIN over its Unix socket. Its minting
// methods are implemented in knirvchain_client.go.
type KNIRVCHAINClient struct{}

// KNIRVORACLEClient talks to KNIRVORACLE over its Unix socket.
// VerifySkillNode, BurnNRNForSkill, GrantOwnershipRights, and PayBounty are
// implemented in knirvoracle_client.go.
type KNIRVORACLEClient struct{}

// SkillNode is a minted skill (SDD Section 7.3).
type SkillNode struct {
	ID              string
	Creator         string
	Description     string
	ResolvesErrors  []string
	CodePackageURI  string
	ValidationProof []byte
	Timestamp       time.Time
}

// SkillContentStore persists a skill's content bundle and returns a
// content-addressed reference to it.
//
// This is a seam, not an implementation: KNIRVGRAPH has no object store or IPFS
// client wired in this module. When no store is configured the engine derives
// its own content digest (see contentRef) rather than inventing an
// "ipfs://dummy_cid" — a fabricated CID is a fabricated claim about where the
// content lives.
type SkillContentStore interface {
	Put(ctx context.Context, skillID string, content []byte) (string, error)
}

// SkillDiscoveryEngine builds a candidate SkillNode from a converged cluster.
type SkillDiscoveryEngine struct {
	// contentStore is optional; nil falls back to a local content digest.
	contentStore SkillContentStore
	// now is injectable for deterministic tests.
	now func() time.Time
}

// NewSkillDiscoveryEngine builds a discovery engine. A nil store is permitted
// and yields locally-computed content references.
func NewSkillDiscoveryEngine(store SkillContentStore) *SkillDiscoveryEngine {
	return &SkillDiscoveryEngine{contentStore: store, now: time.Now}
}

// DiscoverSkill builds the candidate SkillNode for a converged cluster.
//
// This function is deliberately side-effect free with respect to the network:
// it performs no KNIRVGRAPH, KNIRVORACLE or KNIRVCHAIN calls and mints nothing.
// All minting is orchestrated by SkillMintingProtocol.MintSkillFromCluster, so
// the mint has exactly one owner and one ordering. Previously this function
// minted via a no-op SkillRegistry stub *and* re-verified with the Oracle
// *and* minted on KNIRVCHAIN, while MintSkillFromCluster did the same three
// things again — the double-mint the chain client had to be made idempotent to
// survive.
//
// Discovery is deterministic for a given cluster: the skill ID and content
// reference are content-addressed, so re-running discovery over the same
// cluster yields the same skill rather than a duplicate.
func (sde *SkillDiscoveryEngine) DiscoverSkill(cluster *ErrorCluster) (*SkillNode, error) {
	if cluster == nil {
		return nil, errors.New("cluster is required")
	}
	if strings.TrimSpace(cluster.ClusterID) == "" {
		return nil, errors.New("cluster id is required")
	}
	if len(cluster.Errors) == 0 {
		return nil, fmt.Errorf("cluster %s has no member errors to derive a skill from", cluster.ClusterID)
	}
	if strings.TrimSpace(cluster.OwnerAgent) == "" {
		// The creator becomes the minter address on KNIRVCHAIN and the
		// recipient of the perpetual invocation-fee entitlement; an empty
		// creator would mint an unowned skill.
		return nil, fmt.Errorf("cluster %s has no owner agent to attribute the skill to", cluster.ClusterID)
	}

	errorIDs := extractErrorIDs(cluster.Errors)
	if len(errorIDs) == 0 {
		return nil, fmt.Errorf("cluster %s has no identifiable member errors", cluster.ClusterID)
	}

	description := describeCluster(cluster)
	category := categorizeCluster(cluster)
	skillID := generateSkillID(cluster.ClusterID, errorIDs)

	content := skillContentBundle(skillID, description, category, errorIDs, cluster)
	contentRef, err := sde.contentRef(context.Background(), skillID, content)
	if err != nil {
		return nil, fmt.Errorf("store content for skill %s: %w", skillID, err)
	}

	return &SkillNode{
		ID:              skillID,
		Creator:         cluster.OwnerAgent,
		Description:     description,
		ResolvesErrors:  errorIDs,
		CodePackageURI:  contentRef,
		ValidationProof: cluster.ValidationProof,
		Timestamp:       sde.clock().UTC(),
	}, nil
}

// contentRef stores the bundle when a store is configured, otherwise returns a
// locally-computed content digest.
func (sde *SkillDiscoveryEngine) contentRef(ctx context.Context, skillID string, content []byte) (string, error) {
	if sde.contentStore != nil {
		ref, err := sde.contentStore.Put(ctx, skillID, content)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(ref) == "" {
			return "", errors.New("content store returned an empty reference")
		}
		return ref, nil
	}
	sum := sha256.Sum256(content)
	return "knirv+sha256:" + hex.EncodeToString(sum[:]), nil
}

func (sde *SkillDiscoveryEngine) clock() time.Time {
	if sde.now == nil {
		return time.Now()
	}
	return sde.now()
}

// generateSkillID derives a deterministic, content-addressed skill ID from the
// cluster and the set of errors it resolves.
//
// Deterministic on purpose: the chain client's mint is idempotent per skill ID,
// so a stable ID is what makes "discover the same cluster twice" produce the
// same skill instead of a second one. The previous implementation returned
// "skill-<clusterID>", which collided across every distinct skill ever mined
// from a given cluster.
func generateSkillID(clusterID string, errorIDs []string) string {
	sorted := append([]string(nil), errorIDs...)
	sort.Strings(sorted)
	sum := sha256.Sum256([]byte(strings.Join(sorted, "\x00")))
	return fmt.Sprintf("skill-%s-%s", sanitizeSegment(clusterID), hex.EncodeToString(sum[:])[:16])
}

// sanitizeSegment keeps an identifier component to safe characters.
func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "cluster"
	}
	if len(out) > 48 {
		out = out[:48]
	}
	return out
}

// extractErrorIDs returns the sorted, de-duplicated IDs of a cluster's member
// errors. Previously this returned an empty slice unconditionally, so every
// skill claimed to resolve no errors at all.
func extractErrorIDs(errors []*ErrorNode) []string {
	seen := make(map[string]struct{}, len(errors))
	for _, node := range errors {
		if node == nil {
			continue
		}
		id := strings.TrimSpace(node.Id)
		if id == "" {
			continue
		}
		seen[id] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// describeCluster builds a skill description from the cluster's actual
// contents, so the description records what the skill was derived from rather
// than the literal string "Dummy Skill Description".
func describeCluster(cluster *ErrorCluster) string {
	domains := clusterDomains(cluster)
	resolved := extractErrorIDs(cluster.Errors)

	var b strings.Builder
	fmt.Fprintf(&b, "Derived from %d clustered failure(s)", len(resolved))
	if len(domains) > 0 {
		fmt.Fprintf(&b, " in domain(s) %s", strings.Join(domains, ", "))
	}
	fmt.Fprintf(&b, "; average complexity %.2f", cluster.AvgComplexity)
	if representative := representativeFailure(cluster); representative != "" {
		fmt.Fprintf(&b, ". Representative failure: %s", representative)
	}
	b.WriteString(".")
	return b.String()
}

// representativeFailure returns the description of the highest-complexity
// member error, used to make the generated skill description concrete.
func representativeFailure(cluster *ErrorCluster) string {
	if cluster == nil {
		return ""
	}
	var best *ErrorNode
	for _, node := range cluster.Errors {
		if node == nil {
			continue
		}
		if strings.TrimSpace(node.Description) == "" {
			continue
		}
		if best == nil || node.Complexity > best.Complexity {
			best = node
		}
	}
	if best == nil {
		return ""
	}
	return strings.TrimSpace(best.Description)
}

// categorizeCluster returns the dominant error domain in the cluster — real
// evidence rather than the previous hardcoded "Dummy Category".
func categorizeCluster(cluster *ErrorCluster) string {
	domains := clusterDomains(cluster)
	if len(domains) == 0 {
		return "general"
	}
	return domains[0]
}

// clusterDomains returns the cluster's error domains ordered by frequency
// (descending), then alphabetically for ties.
func clusterDomains(cluster *ErrorCluster) []string {
	if cluster == nil {
		return nil
	}
	counts := make(map[string]int)
	for _, node := range cluster.Errors {
		if node == nil {
			continue
		}
		domain := strings.TrimSpace(node.Domain)
		if domain == "" {
			continue
		}
		counts[domain]++
	}
	domains := make([]string, 0, len(counts))
	for domain := range counts {
		domains = append(domains, domain)
	}
	sort.Slice(domains, func(i, j int) bool {
		if counts[domains[i]] != counts[domains[j]] {
			return counts[domains[i]] > counts[domains[j]]
		}
		return domains[i] < domains[j]
	})
	return domains
}

// skillContentBundle is the canonical bytes describing a skill, used for its
// content address. It is deliberately a stable, sorted rendering so the digest
// is reproducible.
func skillContentBundle(skillID, description, category string, errorIDs []string, cluster *ErrorCluster) []byte {
	owner := ""
	avgComplexity := 0.0
	if cluster != nil {
		owner = cluster.OwnerAgent
		avgComplexity = cluster.AvgComplexity
	}
	return []byte(fmt.Sprintf(
		"id=%s\nowner=%s\ncategory=%s\ndescription=%s\nresolves=%s\navg_complexity=%.4f\n",
		skillID, owner, category, description, strings.Join(errorIDs, ","), avgComplexity))
}
