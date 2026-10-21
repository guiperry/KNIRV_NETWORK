// Package evidence is the Factuality Slice's evidence store and index (plan §4.4,
// Phase 2): a content-addressed snapshot store, an inverted index over
// SHA-256-derived keys with a Bloom prefilter, trust tiers, and an append-only
// ingest ledger. It answers "which passages could speak to this claim?" and nothing
// else; judging, gating and flash lookup are later phases.
package evidence

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// KeyVersion prefixes every key domain. Changing the normalization or the key
// layout means a new version and a rebuilt index; old and new keys never meet.
const KeyVersion = "fs1"

// Kind names a key family (§4.4). The kind is part of the hashed preimage, so a
// term can never collide with an entity of the same spelling.
type Kind string

const (
	KindSubjectProperty Kind = "sp" // exact (subject, property) -> value set
	KindEntity          Kind = "e"  // named entity -> chunks mentioning it
	KindNumber          Kind = "n"  // normalized number or date -> chunks containing it
	KindTerm            Kind = "t"  // distinctive lexical term
	KindMinHash         Kind = "mh" // MinHash band of a chunk's shingles (paraphrase-tolerant)
	KindTitle           Kind = "ti" // the canonical title of the document a passage belongs to
)

// Key is a 16-byte truncation of a domain-separated SHA-256. Sixteen bytes keep the
// postings compact (about 2^-64 collision odds at 400 M postings) while the full
// digest is still what identifies snapshots and spans.
type Key [16]byte

func (k Key) String() string { return hex.EncodeToString(k[:]) }

// Digest is a full SHA-256, used for snapshot and span identity.
type Digest [32]byte

func (d Digest) String() string { return "sha256:" + hex.EncodeToString(d[:]) }

// Hex is the digest without the algorithm prefix (used in file names).
func (d Digest) Hex() string { return hex.EncodeToString(d[:]) }

// preimage is the existing assertionIdentity pattern: each part is length-prefixed
// so ("ab","c") and ("a","bc") hash differently, and the domain tag comes first.
func preimage(domain string, parts ...string) []byte {
	var b []byte
	var n [8]byte
	add := func(s string) {
		binary.BigEndian.PutUint64(n[:], uint64(len(s)))
		b = append(b, n[:]...)
		b = append(b, s...)
	}
	add(domain)
	for _, p := range parts {
		add(p)
	}
	return b
}

// HashKey derives a key of the given kind from already-canonical parts.
func HashKey(kind Kind, parts ...string) Key {
	sum := sha256.Sum256(preimage(KeyVersion+":"+string(kind), parts...))
	var k Key
	copy(k[:], sum[:16])
	return k
}

// SnapshotID is the identity of a normalized document: sha256 of its bytes.
func SnapshotID(normalized []byte) Digest { return sha256.Sum256(normalized) }

// SpanID is sha256(snapshot, byte_start, byte_end): the same passage of the same
// snapshot always has the same id, on any machine.
func SpanID(snapshot Digest, start, end int) Digest {
	var b [48]byte
	copy(b[:32], snapshot[:])
	binary.BigEndian.PutUint64(b[32:40], uint64(start))
	binary.BigEndian.PutUint64(b[40:48], uint64(end))
	return sha256.Sum256(b[:])
}
