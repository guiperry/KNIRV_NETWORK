package evidence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeysAreDomainSeparatedAndLengthPrefixed(t *testing.T) {
	if HashKey(KindEntity, "paris") == HashKey(KindTerm, "paris") {
		t.Fatal("a term and an entity of the same spelling must not collide")
	}
	if HashKey(KindEntity, "ab", "c") == HashKey(KindEntity, "a", "bc") {
		t.Fatal("length prefixes must separate parts")
	}
	if HashKey(KindEntity, "paris") != HashKey(KindEntity, "paris") {
		t.Fatal("keys must be deterministic")
	}
	snap := SnapshotID([]byte("x"))
	if SpanID(snap, 0, 5) == SpanID(snap, 0, 6) || SpanID(snap, 0, 5) != SpanID(snap, 0, 5) {
		t.Fatal("span ids")
	}
}

func TestNormalizationIsStable(t *testing.T) {
	a := NormalizeText("Hello,\r\n\r\n\r\n\r\nWorld \t  ﬁne\u0007")
	if a != "Hello,\n\nWorld fine" {
		t.Fatalf("%q", a)
	}
	if NormalizeText(a) != a {
		t.Fatal("normalization must be idempotent")
	}
	if Canon("The  Eiffel-Tower!") != "eiffel tower" {
		t.Fatal(Canon("The  Eiffel-Tower!"))
	}
}

func TestDatesAndNumbers(t *testing.T) {
	d := Dates("Born 17 October 2026, died October 3, 2030; also 2031-01-05 and 40-02-30.")
	want := map[string]bool{"2026-10-17": true, "2030-10-03": true, "2031-01-05": true}
	if len(d) != 3 {
		t.Fatalf("%v", d)
	}
	for _, x := range d {
		if !want[x] {
			t.Fatalf("%v", d)
		}
	}
	n := Numbers("It cost 1,234,500.50 dollars, up 7% from 12 and 12.0.")
	if strings.Join(n, ",") != "1234500.5,7,12,12" {
		t.Fatalf("%v", n)
	}
}

func TestEntitiesFeaturesDropSentenceStarters(t *testing.T) {
	f := ExtractFeatures("The Eiffel Tower is in Paris. It was built in 1889 by Gustave Eiffel of France.")
	got := strings.Join(f.Entities, "|")
	for _, w := range []string{"eiffel tower", "paris", "gustave eiffel", "france"} {
		if !strings.Contains(got, w) {
			t.Errorf("missing entity %q in %q", w, got)
		}
	}
	for _, e := range f.Entities {
		if e == "the" || e == "it" {
			t.Errorf("sentence starter became an entity: %q", e)
		}
	}
	if !contains(f.Numbers, "1889") {
		t.Fatalf("%v", f.Numbers)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func TestChunkWordsCoversTextWithOverlap(t *testing.T) {
	words := make([]string, 1000)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	text := strings.Join(words, " ")
	cs := ChunkWords(text, 384, 48)
	if len(cs) != 3 {
		t.Fatalf("%d chunks", len(cs))
	}
	if cs[0].Start != 0 || cs[len(cs)-1].End != len(text) {
		t.Fatal("chunks must cover the text")
	}
	for i := 1; i < len(cs); i++ {
		if cs[i].Start >= cs[i-1].End {
			t.Fatalf("no overlap between %d and %d", i-1, i)
		}
	}
	if len(ChunkWords("   ", 10, 2)) != 0 || len(ChunkWords("one two", 10, 2)) != 1 {
		t.Fatal("edges")
	}
}

func TestBloomHasNoFalseNegatives(t *testing.T) {
	b := NewBloom(5000, 0.01)
	for i := 0; i < 5000; i++ {
		b.Add(HashKey(KindTerm, fmt.Sprint(i)))
	}
	for i := 0; i < 5000; i++ {
		if !b.MayContain(HashKey(KindTerm, fmt.Sprint(i))) {
			t.Fatalf("false negative %d", i)
		}
	}
	fp := 0
	for i := 5000; i < 15000; i++ {
		if b.MayContain(HashKey(KindTerm, fmt.Sprint(i))) {
			fp++
		}
	}
	if fp > 300 { // 1% target, 3% allowed
		t.Fatalf("false positive rate %d/10000", fp)
	}
	rb, err := UnmarshalBloom(b.Marshal())
	if err != nil || !rb.MayContain(HashKey(KindTerm, "7")) {
		t.Fatal(err)
	}
	if _, err := UnmarshalBloom([]byte{1, 2}); err == nil {
		t.Fatal("corrupt bloom accepted")
	}
}

func TestMerkleProofs(t *testing.T) {
	var ids []Digest
	for i := 0; i < 7; i++ {
		ids = append(ids, SnapshotID([]byte{byte(i)}))
	}
	root := MerkleRoot(ids)
	for i := range ids {
		if !VerifyProof(ids[i], MerkleProof(ids, i), root) {
			t.Fatalf("proof %d", i)
		}
	}
	if VerifyProof(SnapshotID([]byte("other")), MerkleProof(ids, 0), root) {
		t.Fatal("a stranger verified")
	}
	if MerkleRoot(ids) != root || MerkleRoot(ids[:6]) == root {
		t.Fatal("root must depend on exactly the ids")
	}
	if (MerkleRoot(nil) != Digest{}) {
		t.Fatal("empty root")
	}
}

func TestStoreIsContentAddressedAndVerifiable(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	d1, _ := s.Put([]byte("hello world"))
	d2, _ := s.Put([]byte("hello world"))
	if d1 != d2 || !s.Has(d1) {
		t.Fatal("put must be idempotent")
	}
	if got, _ := s.ReadSpan(d1, 6, 11); got != "world" {
		t.Fatalf("%q", got)
	}
	if err := s.Verify(d1); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path(d1), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if s.Verify(d1) == nil {
		t.Fatal("tampering not detected")
	}
}

// ---- index ----

func corpus(n int) []Doc {
	cities := []string{"Aldermoor", "Brightwater", "Cinderfell", "Dunmarrow", "Eastholm", "Foxgrove", "Greywick", "Hollowmere"}
	var docs []Doc
	for i := 0; i < n; i++ {
		c := cities[i%len(cities)]
		docs = append(docs, Doc{
			ID: fmt.Sprint(i), Title: c + " " + fmt.Sprint(i),
			Text: fmt.Sprintf("%s is a town founded in %d by Margaret Holloway. It has a population of %d residents and a harbour called Port %s%d. The town hall was completed in %d.",
				c, 1500+i, 10000+i*37, c, i, 1600+i),
			URL: fmt.Sprintf("https://example.org/%d", i), License: "CC-BY-SA",
		})
	}
	return docs
}

func TestIngestSearchRoundTrip(t *testing.T) {
	ix, err := Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	res, err := ix.Ingest(Batch{Tier: TierReference, Docs: corpus(40)})
	if err != nil || res.Spans != 40 || res.Docs != 40 {
		t.Fatalf("%+v %v", res, err)
	}
	r, err := ix.Search(Claim{Text: "Cinderfell was founded in 1502."}, Options{K: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Hits) == 0 || !strings.Contains(r.Hits[0].Text, "founded in 1502") || r.Hits[0].Tier != TierReference {
		t.Fatalf("%+v", r.Hits)
	}
	if r.Retrieval != "lexical" || r.IndexVer == "" || r.Hits[0].URL == "" || r.Hits[0].License != "CC-BY-SA" {
		t.Fatalf("%+v", r)
	}
	// The cited span is the bytes it names.
	h := r.Hits[0]
	snap, _ := ParseDigest(h.Snapshot)
	if got, _ := ix.Store().ReadSpan(snap, h.Start, h.End); got != h.Text {
		t.Fatal("hit text differs from the stored span")
	}
	if SpanID(snap, h.Start, h.End).String() != h.SpanID {
		t.Fatal("span id does not match its range")
	}
}

func TestNumbersAndEntitiesFindTheRightPassageAmongSimilarOnes(t *testing.T) {
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: corpus(200)}); err != nil {
		t.Fatal(err)
	}
	correct := 0
	for _, i := range []int{3, 17, 42, 99, 123, 150, 188} {
		cities := []string{"Aldermoor", "Brightwater", "Cinderfell", "Dunmarrow", "Eastholm", "Foxgrove", "Greywick", "Hollowmere"}
		q := fmt.Sprintf("The population of %s is %d.", cities[i%8], 10000+i*37)
		r, _ := ix.Search(Claim{Text: q}, Options{K: 3})
		for _, h := range r.Hits {
			if strings.Contains(h.Text, fmt.Sprintf("population of %d residents", 10000+i*37)) {
				correct++
				break
			}
		}
	}
	if correct < 7 {
		t.Fatalf("exact number + entity recall in top 3: %d/7", correct)
	}
}

func TestStructuredFactsAreFoundByTheExactKey(t *testing.T) {
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	_, err := ix.Ingest(Batch{Tier: TierReference, Docs: []Doc{
		{ID: "f", Facts: []Fact{{Subject: "The Eiffel Tower", Property: "height", Value: "330 metres"}}},
		{ID: "d", Text: "A tall structure in Paris that tourists visit every year to see the iron lattice."},
	}})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := ix.Search(Claim{Text: "It is tall.", Subject: "eiffel tower", Property: "Height"}, Options{K: 2})
	if len(r.Hits) == 0 || !r.Hits[0].Fact || !strings.Contains(r.Hits[0].Text, "330 metres") || !contains(r.Hits[0].Matched, "sp") {
		t.Fatalf("%+v", r.Hits)
	}
}

func TestTiersAndTenantsAreIsolated(t *testing.T) {
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	secret := Doc{ID: "s", Text: "Project Nightjar budget is 4200000 dollars and the lead is Ingrid Falk."}
	if _, err := ix.Ingest(Batch{Tier: TierUser, Tenant: "acme", Docs: []Doc{secret}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ix.Ingest(Batch{Tier: TierQuarantine, Docs: []Doc{{ID: "q", Text: "Project Nightjar was cancelled according to a forum post by Ingrid Falk.", URL: "https://forum.example/p"}}}); err != nil {
		t.Fatal(err)
	}
	claim := Claim{Text: "Project Nightjar lead is Ingrid Falk."}
	own, _ := ix.Search(claim, Options{K: 5, Tenant: "acme"})
	if len(own.Hits) != 1 || own.Hits[0].Tier != TierUser {
		t.Fatalf("the owner sees its own document and not quarantine: %+v", own.Hits)
	}
	other, _ := ix.Search(claim, Options{K: 5, Tenant: "globex"})
	if len(other.Hits) != 0 {
		t.Fatalf("another tenant saw private evidence: %+v", other.Hits)
	}
	none, _ := ix.Search(claim, Options{K: 5})
	if len(none.Hits) != 0 {
		t.Fatal("an anonymous query saw a user document")
	}
	q, _ := ix.Search(claim, Options{K: 5, Tiers: []Tier{TierQuarantine}})
	if len(q.Hits) != 1 || q.Hits[0].Tier != TierQuarantine {
		t.Fatalf("quarantine is reachable only when asked for: %+v", q.Hits)
	}
}

func TestIngestValidation(t *testing.T) {
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	for name, b := range map[string]Batch{
		"bad tier":               {Tier: "gold", Docs: []Doc{{Text: "x"}}},
		"user without tenant":    {Tier: TierUser, Docs: []Doc{{Text: "x"}}},
		"quarantine without url": {Tier: TierQuarantine, Docs: []Doc{{Text: "x"}}},
		"oversize":               {Tier: TierReference, Docs: []Doc{{Text: strings.Repeat("a", MaxDocBytes+1)}}},
	} {
		if _, err := ix.Ingest(b); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	// A reference batch ignores any tenant it was given: reference is shared.
	if _, err := ix.Ingest(Batch{Tier: TierReference, Tenant: "acme", Docs: []Doc{{Text: "Public fact about Mount Kestrel at 3100 metres."}}}); err != nil {
		t.Fatal(err)
	}
	r, _ := ix.Search(Claim{Text: "Mount Kestrel is 3100 metres."}, Options{Tenant: "globex"})
	if len(r.Hits) != 1 {
		t.Fatal("reference evidence must be visible to every tenant")
	}
}

func TestDuplicateDocumentsAreSkipped(t *testing.T) {
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	b := Batch{Tier: TierReference, Docs: corpus(5)}
	if _, err := ix.Ingest(b); err != nil {
		t.Fatal(err)
	}
	r, err := ix.Ingest(b)
	if err != nil || r.Docs != 0 || r.Skipped != 5 || r.Spans != 0 {
		t.Fatalf("%+v %v", r, err)
	}
	if ix.Status().Segments != 1 {
		t.Fatal("a fully duplicate batch must not create a segment")
	}
}

func TestIndexBuildIsReproducible(t *testing.T) {
	build := func() (BatchResult, string) {
		ix, _ := Open(t.TempDir(), "")
		defer ix.Close()
		r, err := ix.Ingest(Batch{Tier: TierReference, Docs: corpus(60)})
		if err != nil {
			t.Fatal(err)
		}
		return r, ix.Version()
	}
	a, va := build()
	b, vb := build()
	if a.Root != b.Root || va != vb || a.BatchID != b.BatchID {
		t.Fatalf("same input, different index: %+v %s vs %+v %s", a, va, b, vb)
	}
}

func TestReopenKeepsEverything(t *testing.T) {
	dir := t.TempDir()
	ix, _ := Open(dir, "")
	if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: corpus(10)}); err != nil {
		t.Fatal(err)
	}
	v := ix.Version()
	ix.Close()
	ix2, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer ix2.Close()
	if ix2.Version() != v || ix2.Status().Spans != 10 {
		t.Fatal("reopen changed the index")
	}
	if r, _ := ix2.Search(Claim{Text: "Aldermoor was founded in 1500"}, Options{}); len(r.Hits) == 0 {
		t.Fatal("search after reopen")
	}
	// Re-ingesting after reopen still dedupes.
	if r, _ := ix2.Ingest(Batch{Tier: TierReference, Docs: corpus(10)}); r.Skipped != 10 {
		t.Fatalf("%+v", r)
	}
}

func TestAHalfWrittenSegmentIsIgnored(t *testing.T) {
	dir := t.TempDir()
	ix, _ := Open(dir, "")
	if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: corpus(3)}); err != nil {
		t.Fatal(err)
	}
	ix.Close()
	os.Remove(filepath.Join(dir, "segments", "seg-000001", "COMPLETE"))
	ix2, _ := Open(dir, "")
	defer ix2.Close()
	if ix2.Status().Segments != 0 {
		t.Fatal("a segment without its completion marker must not be served")
	}
}

func TestEmbedderIsPinnedInTheHeader(t *testing.T) {
	dir := t.TempDir()
	ix, err := Open(dir, "ollama:nomic-embed-text")
	if err != nil {
		t.Fatal(err)
	}
	ix.Close()
	if _, err := Open(dir, "cloudflare:bge-base-en-v1.5"); err == nil {
		t.Fatal("opening an index with a different embedder must fail: vectors are never mixed")
	}
	ix2, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer ix2.Close()
	if st := ix2.Status(); len(st.EmbedderIDs) != 1 || st.EmbedderIDs[0] != "ollama:nomic-embed-text" || st.Retrieval != "lexical" {
		t.Fatalf("%+v", st)
	}
}

func TestLedgerRecordsEveryBatch(t *testing.T) {
	dir := t.TempDir()
	ix, _ := Open(dir, "")
	defer ix.Close()
	r, _ := ix.Ingest(Batch{Tier: TierReference, Docs: corpus(4)})
	b, err := os.ReadFile(filepath.Join(dir, "evidence_writes.jsonl"))
	if err != nil || !strings.Contains(string(b), r.Root) || !strings.Contains(string(b), r.BatchID) || !strings.Contains(string(b), "https://example.org/0") {
		t.Fatalf("%v %s", err, b)
	}
}

func TestStatusAndHitRate(t *testing.T) {
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	_, _ = ix.Ingest(Batch{Tier: TierReference, Docs: corpus(8)})
	ix.Search(Claim{Text: "Aldermoor was founded in 1500"}, Options{})
	ix.Search(Claim{Text: "Zzyzx quantum parliament of 99999999"}, Options{})
	st := ix.Status()
	if st.Searches != 2 || st.HitRate != 0.5 || st.Tiers[TierReference] != 8 || st.Spans != 8 {
		t.Fatalf("%+v", st)
	}
	empty, _ := Open(t.TempDir(), "")
	defer empty.Close()
	if r, err := empty.Search(Claim{Text: "anything"}, Options{}); err != nil || len(r.Hits) != 0 {
		t.Fatal("an empty index answers with no hits, not an error")
	}
}

func answers(t *testing.T, ix *Index, qs []string) []string {
	t.Helper()
	var out []string
	for _, q := range qs {
		r, err := ix.Search(Claim{Text: q}, Options{K: 3})
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, h := range r.Hits {
			ids = append(ids, h.SpanID)
		}
		out = append(out, strings.Join(ids, ","))
	}
	return out
}

func TestCompactionChangesNothingYouCanSee(t *testing.T) {
	dir := t.TempDir()
	ix, _ := Open(dir, "")
	docs := corpus(120)
	for i := 0; i < 120; i += 10 {
		if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: docs[i : i+10]}); err != nil {
			t.Fatal(err)
		}
	}
	if ix.Status().Segments != 12 {
		t.Fatal(ix.Status().Segments)
	}
	qs := []string{"Cinderfell was founded in 1502.", "The population of Brightwater is 10037.", "Hollowmere town hall completed 1607", "Port Foxgrove5"}
	before := answers(t, ix, qs)
	ver, spans := ix.Version(), ix.Status().Spans
	if err := ix.Compact(12); err != nil {
		t.Fatal(err)
	}
	if ix.Status().Segments != 1 || ix.Version() != ver || ix.Status().Spans != spans {
		t.Fatalf("compaction changed the index: %+v %s vs %s", ix.Status(), ix.Version(), ver)
	}
	after := answers(t, ix, qs)
	for i := range qs {
		if before[i] != after[i] || before[i] == "" {
			t.Errorf("query %d: before %q after %q", i, before[i], after[i])
		}
	}
	ix.Close()
	re, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer re.Close()
	if re.Version() != ver || re.Status().Segments != 1 {
		t.Fatal("reopen after compaction")
	}
	if got := answers(t, re, qs); strings.Join(got, "|") != strings.Join(before, "|") {
		t.Fatal("answers changed after reopening a compacted index")
	}
	// New batches after a compaction must not reuse a live segment name.
	if _, err := re.Ingest(Batch{Tier: TierReference, Docs: []Doc{{Text: "A brand new fact about Mount Orrery at 4400 metres."}}}); err != nil {
		t.Fatal(err)
	}
	if re.Status().Segments != 2 {
		t.Fatal("segment name collision")
	}
}

func TestIngestKeepsTheSegmentCountBounded(t *testing.T) {
	saved := maxSegments
	maxSegments = 6
	defer func() { maxSegments = saved }()
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	for i := 0; i < maxSegments+20; i++ {
		if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: []Doc{{ID: fmt.Sprint(i), Text: fmt.Sprintf("Unique record number %d about the Zebulon %d expedition in %d.", i, i*13, 1700+i)}}}); err != nil {
			t.Fatal(err)
		}
	}
	if n := ix.Status().Segments; n > maxSegments {
		t.Fatalf("%d segments", n)
	}
	if ix.Status().Spans != maxSegments+20 {
		t.Fatal("spans lost in compaction")
	}
	r, _ := ix.Search(Claim{Text: "The Zebulon 195 expedition in 1715."}, Options{K: 2})
	if len(r.Hits) == 0 || !strings.Contains(r.Hits[0].Text, "number 15 ") {
		t.Fatalf("%+v", r.Hits)
	}
}

// An article is found by its title even when its own text spells the name another way,
// and a passage that merely shares common words with the claim does not outrank it.
func TestTitleKeyFindsTheArticleWhoseTextUsesAnotherSpelling(t *testing.T) {
	ix, _ := Open(t.TempDir(), "")
	defer ix.Close()
	docs := []Doc{
		{Title: "Barack Obama", Text: "Barack Hussein Obama II (born August 4, 1961) is an American politician who was the 44th president of the United States."},
		{Title: "Barack Obama Sr.", Text: "Barack Hussein Obama Sr. (1936 - 1982) was a Kenyan economist, the father of Barack Obama, who was born in 1961 to him and Ann Dunham."},
		{Title: "Thomas R. Nides", Text: "Thomas Richard Nides (born 1961) is an American banker and politician."},
		{Title: "Paris FC", Text: "Paris F.C. is a football club in France. The capital is a city of Paris and football is popular there."},
		{Title: "France", Text: "France is a country in Europe. The capital of France is Paris, its largest city."},
	}
	for i := range docs {
		docs[i].ID = fmt.Sprint(i)
		docs[i].URL = "https://x/" + fmt.Sprint(i)
	}
	if _, err := ix.Ingest(Batch{Tier: TierReference, Docs: docs}); err != nil {
		t.Fatal(err)
	}
	for q, want := range map[string]string{
		"Barack Obama was born in 1961.":  "Barack Obama",
		"The capital of France is Paris.": "France",
	} {
		r, _ := ix.Search(Claim{Text: q}, Options{K: 2})
		if len(r.Hits) == 0 || r.Hits[0].Title != want {
			var got []string
			for _, h := range r.Hits {
				got = append(got, h.Title)
			}
			t.Errorf("%q: top hits %v, want %q first", q, got, want)
		}
	}
}
