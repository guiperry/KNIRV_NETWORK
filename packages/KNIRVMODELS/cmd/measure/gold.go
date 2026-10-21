package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Gold labels are the owner's (D4). The set is split by group so that no evidence
// passage straddles dev and test, and it is evaluation data only: nothing in this
// repo trains on it (three_layer_inference.md, gold set).
const (
	LabelSupported = "SUPPORTED"
	LabelRefuted   = "REFUTED"
	LabelNEI       = "NEI"

	// goldTarget is the plan's minimum gold-set size.
	goldTarget = 300
	// secondLabelTarget is O7's recommended double-labelling rate.
	secondLabelTarget = 0.10
)

var goldLabels = []string{LabelSupported, LabelRefuted, LabelNEI}

// GoldCase is one claim x evidence pair. DataClass gates which embedder may see it:
// "public" text may go to Cloudflare, "private" never does (E3).
type GoldCase struct {
	ID        string `json:"id"`
	Claim     string `json:"claim"`
	Evidence  string `json:"evidence"`
	Label     string `json:"label"`
	Group     string `json:"group"`
	DataClass string `json:"data_class"`
	Source    string `json:"source,omitempty"`
	Labeler   string `json:"labeler"`
	Label2    string `json:"label2,omitempty"`
	Labeler2  string `json:"labeler2,omitempty"`
	CreatedAt string `json:"created_at"`
}

func goldID(claim, evidence string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(claim) + "\x00" + strings.TrimSpace(evidence)))
	return hex.EncodeToString(h[:8])
}

// splitOf is deterministic from the group id: 40% dev (thresholds are fitted here),
// 60% test (reported, never tuned on).
func splitOf(group string) string {
	h := sha256.Sum256([]byte("measure-split\x00" + group))
	if int(h[0])%10 < 4 {
		return "dev"
	}
	return "test"
}

func validLabel(l string) bool {
	for _, g := range goldLabels {
		if l == g {
			return true
		}
	}
	return false
}

func goldPath(dir string) string { return filepath.Join(dir, "gold.jsonl") }

func loadGold(dir string) ([]GoldCase, error) {
	cs, err := readJSONL[GoldCase](goldPath(dir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return cs, err
}

func saveGold(dir string, cs []GoldCase) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := goldPath(dir) + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, c := range cs {
		b, _ := json.Marshal(c)
		w.Write(b)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, goldPath(dir))
}

// normalize validates and completes a case. It returns an error rather than
// guessing, because a mislabelled gold item silently moves every gate.
func (c *GoldCase) normalize() error {
	c.Claim, c.Evidence = strings.TrimSpace(c.Claim), strings.TrimSpace(c.Evidence)
	c.Label = strings.ToUpper(strings.TrimSpace(c.Label))
	if c.Claim == "" || c.Evidence == "" {
		return fmt.Errorf("claim and evidence are both required")
	}
	if !validLabel(c.Label) {
		return fmt.Errorf("label %q must be one of %v", c.Label, goldLabels)
	}
	if c.Label2 != "" {
		c.Label2 = strings.ToUpper(strings.TrimSpace(c.Label2))
		if !validLabel(c.Label2) {
			return fmt.Errorf("label2 %q must be one of %v", c.Label2, goldLabels)
		}
	}
	if c.Labeler == "" {
		return fmt.Errorf("labeler is required (who is accountable for this label)")
	}
	switch c.DataClass {
	case "":
		c.DataClass = "private" // the safe default: never leaves the host
	case "public", "private":
	default:
		return fmt.Errorf("data_class %q must be public or private", c.DataClass)
	}
	if c.Group == "" {
		// Cases sharing a passage must share a split, or dev/test leak.
		h := sha256.Sum256([]byte(c.Evidence))
		c.Group = hex.EncodeToString(h[:6])
	}
	if c.ID == "" {
		c.ID = goldID(c.Claim, c.Evidence)
	}
	if c.CreatedAt == "" {
		c.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return nil
}

func cmdGold(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("gold: expected add|label2|import|import-sheet|export|stats|accept")
	}
	sub, rest := args[0], args[1:]
	fs := flag.NewFlagSet("gold "+sub, flag.ExitOnError)
	dir := fs.String("dir", "measure", "working directory")
	switch sub {
	case "add":
		var c GoldCase
		fs.StringVar(&c.Claim, "claim", "", "the claim to judge")
		fs.StringVar(&c.Evidence, "evidence", "", "the evidence passage")
		fs.StringVar(&c.Label, "label", "", "SUPPORTED | REFUTED | NEI")
		fs.StringVar(&c.Labeler, "labeler", os.Getenv("USER"), "who labelled it")
		fs.StringVar(&c.Group, "group", "", "split group (default: hash of the evidence)")
		fs.StringVar(&c.DataClass, "data-class", "private", "public | private (private never reaches Cloudflare)")
		fs.StringVar(&c.Source, "source", "", "where the answer/claim came from")
		fs.Parse(rest)
		if err := c.normalize(); err != nil {
			return err
		}
		cs, err := loadGold(*dir)
		if err != nil {
			return err
		}
		for _, e := range cs {
			if e.ID == c.ID {
				return fmt.Errorf("duplicate case %s", c.ID)
			}
		}
		if err := appendJSONL(goldPath(*dir), c); err != nil {
			return err
		}
		fmt.Printf("added %s (%s, split %s)\n", c.ID, c.Label, splitOf(c.Group))
		return nil

	case "accept":
		return cmdGoldAccept(fs, dir, rest)

	case "import-sheet":
		return cmdGoldImportSheet(fs, dir, rest)

	case "label2":
		var id, label, labeler string
		fs.StringVar(&id, "id", "", "case id")
		fs.StringVar(&label, "label", "", "second labeler's label")
		fs.StringVar(&labeler, "labeler", "", "second labeler (must differ from the first)")
		fs.Parse(rest)
		cs, err := loadGold(*dir)
		if err != nil {
			return err
		}
		for i := range cs {
			if cs[i].ID != id {
				continue
			}
			if labeler == "" || labeler == cs[i].Labeler {
				return fmt.Errorf("second label needs a different labeler than %q", cs[i].Labeler)
			}
			cs[i].Label2, cs[i].Labeler2 = label, labeler
			if err := cs[i].normalize(); err != nil {
				return err
			}
			if err := saveGold(*dir, cs); err != nil {
				return err
			}
			fmt.Printf("second label recorded for %s (agree=%v)\n", id, cs[i].Label == cs[i].Label2)
			return nil
		}
		return fmt.Errorf("no case %q", id)

	case "import":
		var file, labeler string
		fs.StringVar(&file, "file", "", "JSONL of cases ({claim,evidence,label,...})")
		fs.StringVar(&labeler, "labeler", "", "default labeler for rows without one")
		fs.Parse(rest)
		in, err := readJSONL[GoldCase](file)
		if err != nil {
			return err
		}
		cs, err := loadGold(*dir)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, c := range cs {
			seen[c.ID] = true
		}
		added := 0
		for i, c := range in {
			if c.Labeler == "" {
				c.Labeler = labeler
			}
			c.ID = ""
			if err := c.normalize(); err != nil {
				return fmt.Errorf("row %d: %w", i+1, err)
			}
			if seen[c.ID] {
				continue
			}
			seen[c.ID] = true
			cs = append(cs, c)
			added++
		}
		if err := saveGold(*dir, cs); err != nil {
			return err
		}
		fmt.Printf("imported %d new case(s), %d total\n", added, len(cs))
		return nil

	case "export":
		var split string
		fs.StringVar(&split, "split", "", "dev | test (default: all)")
		fs.Parse(rest)
		cs, err := loadGold(*dir)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		for _, c := range cs {
			if split == "" || splitOf(c.Group) == split {
				enc.Encode(c)
			}
		}
		return nil

	case "stats":
		fs.Parse(rest)
		cs, err := loadGold(*dir)
		if err != nil {
			return err
		}
		r := goldStats(cs)
		fmt.Println(prettyJSON(r.Data))
		return nil
	}
	return fmt.Errorf("gold: unknown subcommand %q", sub)
}

// cohenKappa over cases that have a second label.
func cohenKappa(cs []GoldCase) (kappa float64, n int) {
	count := map[string]map[string]int{}
	rows, cols := map[string]int{}, map[string]int{}
	agree := 0
	for _, c := range cs {
		if c.Label2 == "" {
			continue
		}
		n++
		if c.Label == c.Label2 {
			agree++
		}
		if count[c.Label] == nil {
			count[c.Label] = map[string]int{}
		}
		count[c.Label][c.Label2]++
		rows[c.Label]++
		cols[c.Label2]++
	}
	if n == 0 {
		return 0, 0
	}
	po := float64(agree) / float64(n)
	var pe float64
	for _, l := range goldLabels {
		pe += float64(rows[l]) / float64(n) * float64(cols[l]) / float64(n)
	}
	if pe == 1 {
		return 1, n
	}
	return (po - pe) / (1 - pe), n
}

func goldStats(cs []GoldCase) *Result {
	r := newResult("gold")
	byLabel := map[string]int{}
	bySplit := map[string]int{}
	byClass := map[string]int{}
	groups := map[string]bool{}
	for _, c := range cs {
		byLabel[c.Label]++
		bySplit[splitOf(c.Group)]++
		byClass[c.DataClass]++
		groups[c.Group] = true
	}
	kappa, n2 := cohenKappa(cs)
	r.Data["cases"] = len(cs)
	r.Data["groups"] = len(groups)
	r.Data["by_label"] = byLabel
	r.Data["by_split"] = bySplit
	r.Data["by_data_class"] = byClass
	r.Data["second_labelled"] = n2
	r.Data["second_label_rate"] = r3(safeDiv(float64(n2), float64(len(cs))))
	r.Data["cohen_kappa"] = r3(kappa)
	r.Data["policy"] = "evaluation only: never used for training; split by group (dev 40% / test 60%)"
	r.check(fmt.Sprintf("gold set has >= %d cases", goldTarget), len(cs) >= goldTarget, fmt.Sprintf("%d", len(cs)))
	mixed := true
	for _, l := range goldLabels {
		if byLabel[l] < 30 {
			mixed = false
		}
	}
	r.check("each label has >= 30 cases (mixed set)", mixed, fmt.Sprint(byLabel))
	r.check(fmt.Sprintf("double-labelled >= %.0f%% (O7)", secondLabelTarget*100),
		safeDiv(float64(n2), float64(len(cs))) >= secondLabelTarget, fmt.Sprintf("%d, kappa %.2f", n2, kappa))
	r.check("both splits non-empty", bySplit["dev"] > 0 && bySplit["test"] > 0, fmt.Sprint(bySplit))
	if len(cs) == 0 {
		r.blocked("no gold cases yet: label with `measure gold add` (D4, owner)")
	}
	return r
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func prettyJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// goldReady returns the cases of a split or a blocked Result explaining why a
// gold-dependent probe cannot run yet.
func goldReady(dir, probe, split string) ([]GoldCase, *Result) {
	cs, err := loadGold(dir)
	if err != nil {
		return nil, newResult(probe).failed("read gold: %v", err)
	}
	if split != "" && split != "all" {
		var f []GoldCase
		for _, c := range cs {
			if splitOf(c.Group) == split {
				f = append(f, c)
			}
		}
		cs = f
	}
	if len(cs) == 0 {
		return nil, newResult(probe).blocked("no gold cases in split %q: waiting on the owner's labels (D4)", split)
	}
	return cs, nil
}

func sortedKeys[V any](m map[string]V) []string {
	k := make([]string, 0, len(m))
	for s := range m {
		k = append(k, s)
	}
	sort.Strings(k)
	return k
}
