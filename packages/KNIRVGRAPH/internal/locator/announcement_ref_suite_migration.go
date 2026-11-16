package locator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// migrateTestSuites moves ErrorTestSuite rows from legacyErrorTestSuiteKeyPrefix
// to errorTestSuiteKeyPrefix and records the counts in rep.Suite*. Suite rows
// have no announcement field, so the move is key-only: the stored bytes are
// copied unchanged.
//
// The rules mirror the error-node rules in MigrateAnnouncementRef: a row must
// decode strictly and its error_node_id must match the key; a row under the new
// prefix is never overwritten by a different legacy row; an identical pair
// resolves by removing the legacy row, so an interrupted run can be repeated.
// A legacy row whose tests are all already present, unchanged, in the current
// row (a superset, e.g. after ContributeTest on a legacy-only suite) is also
// resolved by removing the legacy row. A legacy row with any test the current
// row lacks stays a conflict. A legacy row that carries seal metadata (status,
// version, suite_hash, sealed_at) is only removed when that metadata matches the
// current row exactly (see supersededSafe); otherwise it stays a conflict.
func migrateTestSuites(kv MigrationKV, rep *AnnouncementRefReport) error {
	current, err := kv.ScanPrefix([]byte(errorTestSuiteKeyPrefix))
	if err != nil {
		return fmt.Errorf("scan %s: %w", errorTestSuiteKeyPrefix, err)
	}
	legacy, err := kv.ScanPrefix([]byte(legacyErrorTestSuiteKeyPrefix))
	if err != nil {
		return fmt.Errorf("scan %s: %w", legacyErrorTestSuiteKeyPrefix, err)
	}

	currentIDs := make(map[string]bool, len(current))
	currentNorm := make(map[string]*ErrorTestSuite, len(current))
	for _, key := range sortedKeys(current) {
		rep.SuiteScanned++
		id := strings.TrimPrefix(key, errorTestSuiteKeyPrefix)
		currentIDs[id] = true
		suite, err := decodeTestSuiteRow(current[key], id)
		if err != nil {
			rep.skipSuite(key, false)
			continue
		}
		rep.SuiteAlreadyNew++
		currentNorm[id] = suite
	}

	for _, key := range sortedKeys(legacy) {
		rep.SuiteScanned++
		id := strings.TrimPrefix(key, legacyErrorTestSuiteKeyPrefix)
		suite, err := decodeTestSuiteRow(legacy[key], id)
		if err != nil {
			rep.skipSuite(key, false)
			continue
		}
		newKey := errorTestSuiteKeyPrefix + id

		if currentIDs[id] {
			existing, ok := currentNorm[id]
			if !ok {
				// The current row does not read cleanly: never overwrite it.
				rep.skipSuite(key, true)
				continue
			}
			switch {
			case sameTestSuite(existing, suite):
				if err := kv.Delete([]byte(key)); err != nil {
					return fmt.Errorf("delete duplicate %s: %w", key, err)
				}
				rep.SuiteDuplicatesRemoved++
			case testsContained(suite.Tests, existing.Tests) && supersededSafe(suite, existing):
				// The current row already holds every legacy test, unchanged, and the
				// legacy row carries no seal metadata the current row does not match.
				// This is the normal result of contributing to a legacy-only suite,
				// which writes the full test list under the new key. Nothing is lost by
				// removing the legacy row, so it is treated as migrated and is not a skip.
				if err := kv.Delete([]byte(key)); err != nil {
					return fmt.Errorf("delete superseded %s: %w", key, err)
				}
				rep.SuiteSupersededRemoved++
			default:
				// The legacy row has tests the current row lacks, or a test with the
				// same id but different content. Never overwrite it: a real divergence.
				rep.skipSuite(key, true)
			}
			continue
		}

		if err := kv.Put([]byte(newKey), legacy[key]); err != nil {
			return fmt.Errorf("write %s: %w", newKey, err)
		}
		if err := kv.Delete([]byte(key)); err != nil {
			return fmt.Errorf("delete %s after move: %w", key, err)
		}
		rep.SuiteRewritten++
	}
	return nil
}

// decodeTestSuiteRow decodes one stored suite strictly: unknown fields, trailing
// data and an error_node_id that does not match the key are all errors.
func decodeTestSuiteRow(raw []byte, id string) (*ErrorTestSuite, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var suite ErrorTestSuite
	if err := dec.Decode(&suite); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after test suite")
	}
	if suite.ErrorNodeID != id {
		return nil, fmt.Errorf("row error_node_id %q does not match key id %q", suite.ErrorNodeID, id)
	}
	return &suite, nil
}

// sameTestSuite compares two decoded suites by their canonical JSON encoding.
func sameTestSuite(a, b *ErrorTestSuite) bool {
	ra, errA := json.Marshal(a)
	rb, errB := json.Marshal(b)
	return errA == nil && errB == nil && bytes.Equal(ra, rb)
}

// testsContained reports whether every test in want appears in have. A test is
// matched by its id and must also be identical in content: a same id with a
// different input, expected value or assertion is not the same test.
func testsContained(want, have []ErrorTestCase) bool {
	for _, w := range want {
		found := false
		for _, h := range have {
			if h.ID == w.ID && sameTestCase(w, h) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// supersededSafe reports whether a legacy row may be dropped as superseded by
// the current row. It must never discard seal metadata. A legacy row with no
// seal metadata (not sealed, no suite_hash, no sealed_at) carries nothing to
// lose. A legacy row that carries seal metadata is only superseded when every
// field of it (status, version, suite_hash, sealed_at) equals the current row's.
// Any other combination is a conflict: the owner decides, and both rows stay.
func supersededSafe(legacy, current *ErrorTestSuite) bool {
	if legacy.Status != SuiteSealed && legacy.SuiteHash == "" && legacy.SealedAt == nil {
		return true
	}
	if legacy.Status != current.Status || legacy.Version != current.Version || legacy.SuiteHash != current.SuiteHash {
		return false
	}
	if (legacy.SealedAt == nil) != (current.SealedAt == nil) {
		return false
	}
	return legacy.SealedAt == nil || legacy.SealedAt.Equal(*current.SealedAt)
}

// sameTestCase compares two tests by their canonical JSON encoding.
func sameTestCase(a, b ErrorTestCase) bool {
	ra, errA := json.Marshal(a)
	rb, errB := json.Marshal(b)
	return errA == nil && errB == nil && bytes.Equal(ra, rb)
}

func (rep *AnnouncementRefReport) skipSuite(key string, conflict bool) {
	if conflict {
		rep.SuiteSkippedConflict++
	} else {
		rep.SuiteSkippedUnreadable++
	}
	rep.Skipped = append(rep.Skipped, key)
}
