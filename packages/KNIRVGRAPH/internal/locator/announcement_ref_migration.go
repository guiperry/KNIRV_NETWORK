package locator

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	gproto "google.golang.org/protobuf/proto"
)

// AnnouncementRefMigrationMarker is written as the last step of a successful
// MigrateAnnouncementRef run. Its presence makes a second run refuse.
const AnnouncementRefMigrationMarker = "migration:announcement_ref"

// ErrAnnouncementRefMigrationDone is returned when the marker is already present.
// Nothing is read or written in that case.
var ErrAnnouncementRefMigrationDone = errors.New("announcement_ref migration already recorded; no changes made")

// MigrationKV is the storage the migration needs. storage.Storage satisfies it.
type MigrationKV interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Delete(key []byte) error
	Has(key []byte) (bool, error)
	ScanPrefix(prefix []byte) (map[string][]byte, error)
}

// AnnouncementRefReport counts what one migration run did.
type AnnouncementRefReport struct {
	Scanned           int      // error-node rows examined under either prefix
	AlreadyNew        int      // rows already under graph:error_nodes: that read cleanly without the old field
	Rewritten         int      // legacy rows moved to graph:error_nodes:
	DuplicatesRemoved int      // legacy rows whose identical graph:error_nodes: row already existed; legacy row deleted
	OldFieldRows      int      // rows (either prefix) that carried nrvSource or nrv_source; the field was renamed
	SkippedUnreadable int      // rows that did not parse or failed the strict protojson decode; left in place
	SkippedConflict   int      // legacy rows whose graph:error_nodes: row differs or is unreadable; both left in place
	Skipped           []string // keys of every skipped row (error nodes and test suites), sorted

	// MarkerWritten is true only when the run finished with no skipped rows.
	// A run that skipped rows leaves the marker unwritten, so the legacy rows
	// stay tracked and the run can be repeated after the rows are fixed.
	MarkerWritten bool

	// Test-suite rows (nrv:error_tests: -> graph:error_tests:). Counted apart
	// from the error-node counts above. Suite rows carry no announcement field,
	// so the move is key-only.
	SuiteScanned           int // suite rows examined under either prefix
	SuiteAlreadyNew        int // rows already under graph:error_tests: that read cleanly
	SuiteRewritten         int // legacy rows moved to graph:error_tests:
	SuiteDuplicatesRemoved int // legacy rows whose identical graph:error_tests: row already existed; legacy row deleted
	SuiteSupersededRemoved int // legacy rows whose tests are all contained, unchanged, in the graph:error_tests: row; legacy row deleted, not a skip
	SuiteSkippedUnreadable int // suite rows that failed the strict decode or whose id does not match the key; left in place
	SuiteSkippedConflict   int // legacy suite rows whose graph:error_tests: row differs or is unreadable; both left in place
}

// MigrateAnnouncementRef moves error-node rows from nrv:error_nodes: to
// graph:error_nodes:, renames the JSON field nrvSource / nrv_source to
// announcementRef, moves test-suite rows from nrv:error_tests: to
// graph:error_tests: (key-only, counted separately), and writes the migration
// marker last. Both prefixes are handled in one run under one marker.
//
// It is idempotent until the marker lands: a run that stops part-way can be
// repeated, because each row is written under its new key before the old key
// is deleted, and an identical pair is resolved by removing the legacy row.
// Rows are decoded with the strict protojson loader; a row that does not decode
// is skipped and counted, never dropped, and never rewritten.
//
// The marker is written only when the run skipped no rows. If any row was
// skipped, the function returns normally with MarkerWritten false and the
// Skipped keys filled in. Clean rows already moved stay moved, so the run can
// be repeated after the skipped rows are fixed or removed. Readers keep the
// legacy fallback until a clean run has written the marker.
func MigrateAnnouncementRef(kv MigrationKV, now time.Time) (AnnouncementRefReport, error) {
	var rep AnnouncementRefReport

	done, err := kv.Has([]byte(AnnouncementRefMigrationMarker))
	if err != nil {
		return rep, fmt.Errorf("check migration marker: %w", err)
	}
	if done {
		return rep, ErrAnnouncementRefMigrationDone
	}

	current, err := kv.ScanPrefix([]byte(errorNodeKeyPrefix))
	if err != nil {
		return rep, fmt.Errorf("scan %s: %w", errorNodeKeyPrefix, err)
	}
	legacy, err := kv.ScanPrefix([]byte(legacyErrorNodeKeyPrefix))
	if err != nil {
		return rep, fmt.Errorf("scan %s: %w", legacyErrorNodeKeyPrefix, err)
	}

	// Rows already under the new prefix: normalise any stale field in place and
	// remember each readable row's bytes, so legacy rows can be compared with them.
	currentIDs := make(map[string]bool, len(current))
	currentNorm := make(map[string][]byte, len(current))
	for _, key := range sortedKeys(current) {
		rep.Scanned++
		id := strings.TrimPrefix(key, errorNodeKeyPrefix)
		currentIDs[id] = true
		norm, hadOld, err := normalizeAnnouncementRow(current[key], id)
		if err != nil {
			rep.skip(key, false)
			continue
		}
		if hadOld {
			rep.OldFieldRows++
			if err := kv.Put([]byte(key), norm); err != nil {
				return rep, fmt.Errorf("rewrite %s: %w", key, err)
			}
		} else {
			rep.AlreadyNew++
		}
		currentNorm[id] = norm
	}

	for _, key := range sortedKeys(legacy) {
		rep.Scanned++
		id := strings.TrimPrefix(key, legacyErrorNodeKeyPrefix)
		norm, hadOld, err := normalizeAnnouncementRow(legacy[key], id)
		if err != nil {
			rep.skip(key, false)
			continue
		}
		if hadOld {
			rep.OldFieldRows++
		}
		newKey := errorNodeKeyPrefix + id

		if currentIDs[id] {
			existing, ok := currentNorm[id]
			if !ok || !sameErrorNode(existing, norm) {
				// Never overwrite a row under the new prefix that differs from the
				// legacy one, or that does not read cleanly.
				rep.skip(key, true)
				continue
			}
			if err := kv.Delete([]byte(key)); err != nil {
				return rep, fmt.Errorf("delete duplicate %s: %w", key, err)
			}
			rep.DuplicatesRemoved++
			continue
		}

		if err := kv.Put([]byte(newKey), norm); err != nil {
			return rep, fmt.Errorf("write %s: %w", newKey, err)
		}
		if err := kv.Delete([]byte(key)); err != nil {
			return rep, fmt.Errorf("delete %s after move: %w", key, err)
		}
		rep.Rewritten++
	}

	// Test suites are part of the same cutover, under the same marker.
	if err := migrateTestSuites(kv, &rep); err != nil {
		return rep, err
	}

	sort.Strings(rep.Skipped)
	if len(rep.Skipped) > 0 {
		return rep, nil
	}
	marker, err := json.Marshal(map[string]interface{}{
		"migrated_at":              now.UTC().Format(time.RFC3339),
		"scanned":                  rep.Scanned,
		"rewritten":                rep.Rewritten,
		"already_new":              rep.AlreadyNew,
		"old_field_rows":           rep.OldFieldRows,
		"duplicates_removed":       rep.DuplicatesRemoved,
		"skipped_unreadable":       rep.SkippedUnreadable,
		"skipped_conflict":         rep.SkippedConflict,
		"suite_scanned":            rep.SuiteScanned,
		"suite_rewritten":          rep.SuiteRewritten,
		"suite_already_new":        rep.SuiteAlreadyNew,
		"suite_duplicates_removed": rep.SuiteDuplicatesRemoved,
		"suite_superseded_removed": rep.SuiteSupersededRemoved,
		"suite_skipped_unreadable": rep.SuiteSkippedUnreadable,
		"suite_skipped_conflict":   rep.SuiteSkippedConflict,
	})
	if err != nil {
		return rep, fmt.Errorf("encode marker: %w", err)
	}
	if err := kv.Put([]byte(AnnouncementRefMigrationMarker), marker); err != nil {
		return rep, fmt.Errorf("write migration marker: %w", err)
	}
	rep.MarkerWritten = true
	return rep, nil
}

func (rep *AnnouncementRefReport) skip(key string, conflict bool) {
	if conflict {
		rep.SkippedConflict++
	} else {
		rep.SkippedUnreadable++
	}
	rep.Skipped = append(rep.Skipped, key)
}

// normalizeAnnouncementRow renames the legacy field in one stored row and checks
// the result with the strict protojson loader. It returns the bytes to store,
// whether the legacy field was present, and an error if the row is unreadable
// or its id does not match the key it was stored under.
func normalizeAnnouncementRow(raw []byte, id string) ([]byte, bool, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, false, err
	}
	if doc == nil {
		return nil, false, errors.New("row is not a JSON object")
	}

	hadOld := false
	var oldValue json.RawMessage
	for _, legacyKey := range []string{"nrvSource", "nrv_source"} {
		v, ok := doc[legacyKey]
		if !ok {
			continue
		}
		hadOld = true
		if oldValue != nil && !jsonEqual(oldValue, v) {
			return nil, false, fmt.Errorf("conflicting values for nrvSource and nrv_source")
		}
		oldValue = v
		delete(doc, legacyKey)
	}

	out := raw
	if hadOld {
		for _, newKey := range []string{"announcementRef", "announcement_ref"} {
			if cur, ok := doc[newKey]; ok && !jsonEqual(cur, oldValue) {
				return nil, false, fmt.Errorf("row has announcementRef and a different legacy nrvSource")
			}
		}
		doc["announcementRef"] = oldValue
		delete(doc, "announcement_ref")
		b, err := json.Marshal(doc)
		if err != nil {
			return nil, false, err
		}
		out = b
	}

	node := &ErrorNode{}
	if err := protojson.Unmarshal(out, node); err != nil {
		return nil, false, err
	}
	if node.Id != id {
		return nil, false, fmt.Errorf("row id %q does not match key id %q", node.Id, id)
	}
	return out, hadOld, nil
}

// sameErrorNode reports whether two stored rows decode to the same ErrorNode.
func sameErrorNode(a, b []byte) bool {
	na, nb := &ErrorNode{}, &ErrorNode{}
	if protojson.Unmarshal(a, na) != nil || protojson.Unmarshal(b, nb) != nil {
		return false
	}
	return gproto.Equal(na, nb)
}

func jsonEqual(a, b json.RawMessage) bool {
	var va, vb interface{}
	if json.Unmarshal(a, &va) != nil || json.Unmarshal(b, &vb) != nil {
		return false
	}
	return reflect.DeepEqual(va, vb)
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
