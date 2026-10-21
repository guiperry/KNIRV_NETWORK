package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const ontologyVersionsFile = "ontology_versions.json"

func (ix *Index) loadOntologyVersions() {
	if b, err := os.ReadFile(filepath.Join(ix.dir, ontologyVersionsFile)); err == nil {
		_ = json.Unmarshal(b, &ix.ontVer)
	}
	if ix.ontVer == nil {
		ix.ontVer = map[string]string{}
	}
}

// SetOntologyVersion makes version the current ontology version for tenant. Ontology
// spans of any other revision stop being returned, so replacing an ontology retires
// its old facts at once without rewriting immutable segments. The mapping is
// persisted next to the index.
func (ix *Index) SetOntologyVersion(tenant, version string) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	next := map[string]string{}
	for k, v := range ix.ontVer {
		next[k] = v
	}
	if version == "" {
		delete(next, tenant)
	} else {
		next[tenant] = version
	}
	b, _ := json.MarshalIndent(next, "", " ")
	if err := writeFileSync(filepath.Join(ix.dir, ontologyVersionsFile), b); err != nil {
		return err
	}
	ix.ontVer = next
	return nil
}

// OntologyVersion is the tenant's current ontology version ("" when none is loaded).
func (ix *Index) OntologyVersion(tenant string) string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.ontVer[tenant]
}
