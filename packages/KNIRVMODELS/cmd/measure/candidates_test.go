package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeWiki makes n articles with two numbered paragraphs each.
func writeWiki(t *testing.T, n int) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wiki.jsonl")
	var b strings.Builder
	for i := 0; i < n; i++ {
		para := func(k int) string {
			return fmt.Sprintf("Town%d was founded in %d by settlers from the coast. It had a population of %d people at the last census and %s. "+
				"The town lies on a river and is known for its market, its old church, its mill and its annual festival held each summer for visitors from nearby villages.",
				i, 1700+i, 2000+i*7, strings.Repeat("it grew slowly over many years ", k))
		}
		text := para(1) + "\n\n" + para(2) + "\n\n" + para(3)
		fmt.Fprintf(&b, `{"id":"%d","url":"https://x/wiki/Town%d","title":"Town%d","text":%q}`+"\n", i, i, i, text)
	}
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCandidatesAreUnlabelledBalancedAndReproducible(t *testing.T) {
	wiki := writeWiki(t, 60)
	run := func() string {
		dir := t.TempDir()
		if err := cmdCandidates([]string{"-dir", dir, "-wiki", wiki, "-n", "20", "-seed", "7"}); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(candidatesPath(dir))
		t.Cleanup(func() {})
		os.Setenv("CAND_DIR", dir)
		return string(b)
	}
	a := run()
	dir := os.Getenv("CAND_DIR")
	if b := run(); a != b {
		t.Fatal("same seed and input must give the same candidates")
	}
	cs, _ := readJSONL[GoldCase](candidatesPath(dir))
	if len(cs) != 20 {
		t.Fatalf("got %d", len(cs))
	}
	for _, c := range cs {
		if c.Label != "" || c.DataClass != "public" || c.Group == "" || !strings.HasPrefix(c.Source, "wikipedia:") {
			t.Fatalf("%+v", c)
		}
	}
	kinds, _ := readJSONL[CandidateKind](candidateKindsPath(dir))
	n := map[string]int{}
	for _, k := range kinds {
		n[k.Kind]++
	}
	for _, k := range candidateKinds {
		if n[k] < 3 {
			t.Fatalf("kind %s under-represented: %v", k, n)
		}
	}
	// The sheet must not reveal how a pair was built.
	sheet, _ := os.ReadFile(sheetPath(dir))
	for _, k := range candidateKinds {
		if strings.Contains(string(sheet), k) && k != "unrelated" { // 'unrelated' could appear in prose; others cannot
			t.Fatalf("sheet leaks kind %q", k)
		}
	}
	if _, err := os.Stat(goldPath(dir)); err == nil {
		t.Fatal("candidates must never write gold.jsonl")
	}
}

func TestAlteredClaimDiffersFromTheOriginalByANumber(t *testing.T) {
	rng := newRng(3)
	for _, s := range []string{"The town was founded in 1850 by settlers.", "It had 12,500 people in 2001.", "There are 7 bridges here."} {
		alt, ok := alterNumber(s, rng)
		if !ok || alt == s {
			t.Fatalf("%q -> %q", s, alt)
		}
	}
}

func TestImportSheetTakesOnlyLabelledRowsAndRejectsEditedText(t *testing.T) {
	dir := t.TempDir()
	if err := cmdCandidates([]string{"-dir", dir, "-wiki", writeWiki(t, 30), "-n", "8", "-seed", "1"}); err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(sheetPath(dir))
	rows, _ := csv.NewReader(f).ReadAll()
	f.Close()
	rows[1][1], rows[2][1], rows[3][1] = "supported", "REFUTED", "nei"
	rows[3][4] = "borderline"
	write := func(rows [][]string) string {
		p := filepath.Join(dir, "filled.csv")
		o, _ := os.Create(p)
		w := csv.NewWriter(o)
		w.WriteAll(rows)
		o.Close()
		return p
	}
	imp := func(file string) error {
		return cmdGoldImportSheet(newFlagSet(), &dir, []string{"-file", file, "-labeler", "owner"})
	}
	if err := imp(write(rows)); err != nil {
		t.Fatal(err)
	}
	gold, _ := loadGold(dir)
	if len(gold) != 3 || gold[0].Labeler != "owner" || gold[0].DataClass != "public" || gold[0].Group == "" || !strings.Contains(gold[2].Source, "note: borderline") {
		t.Fatalf("%+v", gold)
	}
	// Re-importing adds nothing.
	if err := imp(write(rows)); err != nil {
		t.Fatal(err)
	}
	if g, _ := loadGold(dir); len(g) != 3 {
		t.Fatalf("duplicates added: %d", len(g))
	}
	// A bad label or edited text is refused and writes nothing.
	rows[4][1] = "MAYBE"
	if err := imp(write(rows)); err == nil {
		t.Fatal("bad label accepted")
	}
	rows[4][1] = "SUPPORTED"
	rows[4][2] += " (edited)"
	if err := imp(write(rows)); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("edited text accepted: %v", err)
	}
	if g, _ := loadGold(dir); len(g) != 3 {
		t.Fatalf("a refused sheet wrote rows: %d", len(g))
	}
}
