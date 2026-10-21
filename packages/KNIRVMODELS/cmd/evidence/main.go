// Command evidence manages the Factuality Slice's evidence index: the CLI submit
// path for reference corpora and for user/admin ontology documents (plan D3).
//
//	evidence submit -tier reference -jsonl wiki.jsonl -license CC-BY-SA-4.0
//	evidence submit -tier user -tenant acme -dir ./domain-docs -license internal
//	evidence status
//	evidence search -q "The Eiffel Tower is 330 metres tall." [-tenant acme]
//	evidence verify
//	evidence ontology -file acme-config.json
//
// The index is the one the models daemon serves (KNIRV_EVIDENCE_DIR, else
// <app data>/knirvmodels/data/evidence). Quarantine is never written from here: it
// is the tier for live-web write-through only.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"knirvmodels/internal/cli/embedded"
	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/orgontology"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "submit":
		err = cmdSubmit(os.Args[2:])
	case "status":
		err = cmdStatus(os.Args[2:])
	case "search":
		err = cmdSearch(os.Args[2:])
	case "verify":
		err = cmdVerify(os.Args[2:])
	case "ontology":
		err = cmdOntology(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "evidence:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `evidence: the evidence index
  submit  -tier reference|user [-tenant T] (-dir D | -jsonl F) [-license L] [-batch N]
  status
  search  -q TEXT [-subject S -property P] [-tenant T] [-k N]
  verify  re-hash every stored snapshot
  ontology -file CONFIG.json     load an organization's onboarding ontology (the ontology tier)
  ontology -remove ORG           retire an organization's ontology
common flag: -index DIR (default: the daemon's index)
`)
}

func defaultDir() string {
	if v := strings.TrimSpace(os.Getenv("KNIRV_EVIDENCE_DIR")); v != "" {
		return v
	}
	if d, err := embedded.GetAppDataDir(); err == nil {
		return filepath.Join(d, "knirvmodels", "data", "evidence")
	}
	return filepath.Join(os.TempDir(), "knirvmodels", "data", "evidence")
}

func open(dir string) (*evidence.Index, error) { return evidence.Open(dir, "") }

func cmdSubmit(args []string) error {
	fs := flag.NewFlagSet("submit", flag.ExitOnError)
	dir := fs.String("index", defaultDir(), "index directory")
	tier := fs.String("tier", "", "reference or user")
	tenant := fs.String("tenant", "", "tenant that owns user-tier documents")
	src := fs.String("dir", "", "directory of .txt / .md files (recursive)")
	jsonl := fs.String("jsonl", "", "JSONL file, one document per line")
	license := fs.String("license", "", "license recorded with every document")
	urlPrefix := fs.String("url-prefix", "", "for -dir: prefix joined with the relative path as the document URL")
	textF := fs.String("text-field", "text", "JSONL field holding the text")
	idF := fs.String("id-field", "id", "JSONL field holding the id")
	titleF := fs.String("title-field", "title", "JSONL field holding the title")
	urlF := fs.String("url-field", "url", "JSONL field holding the URL")
	batch := fs.Int("batch", 500, "documents per ingest batch (each batch is one segment before compaction)")
	fs.Parse(args)
	t := evidence.Tier(*tier)
	switch t {
	case evidence.TierReference:
	case evidence.TierUser:
		if strings.TrimSpace(*tenant) == "" {
			return fmt.Errorf("-tenant is required for user-tier documents")
		}
	case evidence.TierQuarantine:
		return fmt.Errorf("quarantine is written only by live-web write-through, not from the CLI")
	default:
		return fmt.Errorf("-tier must be reference or user")
	}
	if (*src == "") == (*jsonl == "") {
		return fmt.Errorf("give exactly one of -dir and -jsonl")
	}
	ix, err := open(*dir)
	if err != nil {
		return err
	}
	defer ix.Close()

	var pending []evidence.Doc
	total, skipped, spans := 0, 0, 0
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		r, err := ix.Ingest(evidence.Batch{Tier: t, Tenant: *tenant, Docs: pending})
		if err != nil {
			return err
		}
		total += r.Docs
		skipped += r.Skipped
		spans += r.Spans
		fmt.Fprintf(os.Stderr, "\r%d documents, %d passages", total, spans)
		pending = pending[:0]
		return nil
	}
	add := func(d evidence.Doc) error {
		pending = append(pending, d)
		if len(pending) >= *batch {
			return flush()
		}
		return nil
	}
	if *src != "" {
		err = walkText(*src, *license, *urlPrefix, add)
	} else {
		err = readJSONL(*jsonl, *license, *idF, *titleF, *urlF, *textF, add)
	}
	if err == nil {
		err = flush()
	}
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	fmt.Printf("ingested %d documents (%d passages); %d already present\nindex version %s\n", total, spans, skipped, ix.Version())
	return nil
}

func walkText(root, license, urlPrefix string, add func(evidence.Doc) error) error {
	return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".txt" && ext != ".md" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		doc := evidence.Doc{ID: rel, Title: strings.TrimSuffix(filepath.Base(p), ext), Text: string(b), License: license}
		if urlPrefix != "" {
			doc.URL = strings.TrimRight(urlPrefix, "/") + "/" + filepath.ToSlash(rel)
		}
		return add(doc)
	})
}

func readJSONL(path, license, idF, titleF, urlF, textF string, add func(evidence.Doc) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return fmt.Errorf("line %d: %w", n, err)
		}
		str := func(k string) string {
			switch v := m[k].(type) {
			case string:
				return v
			case float64:
				return fmt.Sprint(int64(v))
			}
			return ""
		}
		doc := evidence.Doc{ID: str(idF), Title: str(titleF), URL: str(urlF), Text: str(textF), License: license}
		if v := str("license"); v != "" && license == "" {
			doc.License = v
		}
		if fs, ok := m["facts"].([]any); ok { // optional structured facts on the line
			for _, x := range fs {
				if fm, ok := x.(map[string]any); ok {
					doc.Facts = append(doc.Facts, evidence.Fact{Subject: fmt.Sprint(fm["subject"]), Property: fmt.Sprint(fm["property"]), Value: fmt.Sprint(fm["value"])})
				}
			}
		}
		if err := add(doc); err != nil {
			return err
		}
	}
	return sc.Err()
}

func cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	dir := fs.String("index", defaultDir(), "index directory")
	fs.Parse(args)
	ix, err := open(*dir)
	if err != nil {
		return err
	}
	defer ix.Close()
	b, _ := json.MarshalIndent(ix.Status(), "", "  ")
	fmt.Println(string(b))
	return nil
}

func cmdSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	dir := fs.String("index", defaultDir(), "index directory")
	q := fs.String("q", "", "claim text")
	subj := fs.String("subject", "", "structured subject")
	prop := fs.String("property", "", "structured property")
	tenant := fs.String("tenant", "", "tenant whose user documents may be searched")
	k := fs.Int("k", 5, "passages to return")
	fs.Parse(args)
	if *q == "" {
		return fmt.Errorf("-q is required")
	}
	ix, err := open(*dir)
	if err != nil {
		return err
	}
	defer ix.Close()
	r, err := ix.Search(evidence.Claim{Text: *q, Subject: *subj, Property: *prop}, evidence.Options{K: *k, Tenant: *tenant})
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(b))
	return nil
}

func cmdVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	dir := fs.String("index", defaultDir(), "index directory")
	fs.Parse(args)
	n, bad, err := evidence.VerifyStore(filepath.Join(*dir, "snapshots"))
	if err != nil {
		return err
	}
	fmt.Printf("%d snapshots checked, %d corrupt\n", n, len(bad))
	for _, b := range bad {
		fmt.Println("corrupt:", b)
	}
	if len(bad) > 0 {
		return fmt.Errorf("%d corrupt snapshots", len(bad))
	}
	return nil
}

// cmdOntology loads an organization's onboarding configuration (backend_server's
// OrganizationConfig JSON) into the ontology tier. A running daemon keeps its own
// view of the index, so for a live node use POST /models/hasher/ontology/sync; this
// is for building an index offline and for tests.
func cmdOntology(args []string) error {
	fs := flag.NewFlagSet("ontology", flag.ExitOnError)
	dir := fs.String("index", defaultDir(), "index directory")
	file := fs.String("file", "", "organization configuration JSON")
	remove := fs.String("remove", "", "retire this organization's ontology")
	fs.Parse(args)
	ix, err := open(*dir)
	if err != nil {
		return err
	}
	defer ix.Close()
	if *remove != "" {
		return orgontology.Remove(ix, *remove)
	}
	if *file == "" {
		return fmt.Errorf("give -file or -remove")
	}
	b, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var cfg orgontology.Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return fmt.Errorf("%s: %w", *file, err)
	}
	res, err := orgontology.Sync(ix, cfg)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
	return nil
}
