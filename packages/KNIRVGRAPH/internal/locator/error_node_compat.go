package locator

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
)

// Persisted rows written before the nrv_source -> announcement_ref rename carry
// the old JSON key. unmarshalPersistedErrorNode reads them through an alias so
// existing stores keep loading.
//
// TEMPORARY ALIAS. Remove one release after the rename ships (owner rename rule).
// Note: DVE_Connection.md D-5 says "no alias period"; this alias is a deliberate
// deviation requested in the DVE-U2 brief and is flagged for owner decision.
func unmarshalPersistedErrorNode(raw []byte, node *ErrorNode) error {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return protojson.Unmarshal(raw, node)
	}
	oldKeys := []string{"nrvSource", "nrv_source"}
	found := false
	var value json.RawMessage
	for _, key := range oldKeys {
		if v, ok := doc[key]; ok {
			found = true
			if value == nil {
				value = v
			}
			delete(doc, key)
		}
	}
	if found {
		if _, ok := doc["announcement_ref"]; !ok {
			if _, ok := doc["announcementRef"]; !ok {
				doc["announcement_ref"] = value
			}
		}
		rewritten, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		raw = rewritten
	}
	return protojson.Unmarshal(raw, node)
}
