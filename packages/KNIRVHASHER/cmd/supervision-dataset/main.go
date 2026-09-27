// supervision-dataset is the supported Bronze-to-Gold materializer.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"knirvhasher/pkg/hashing/schema"
	"knirvhasher/pkg/hashing/supervision/pipeline"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: supervision-dataset <validate|materialize> --bronze <file> [--out <dir>]")
	}
	switch os.Args[1] {
	case "validate":
		validate(os.Args[2:])
	case "materialize":
		materialize(os.Args[2:])
	default:
		fail("unknown operation")
	}
}
func validate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	bronze := fs.String("bronze", "", "Bronze JSONL file")
	fs.Parse(args)
	events, report, err := load(*bronze)
	if err != nil {
		fail(err.Error())
	}
	fmt.Printf("validated=%d quarantined=%d duplicate=%d\n", len(events), report.Quarantined, report.Deduplicated)
}
func materialize(args []string) {
	fs := flag.NewFlagSet("materialize", flag.ExitOnError)
	bronze := fs.String("bronze", "", "Bronze JSONL file")
	out := fs.String("out", "", "output directory")
	embedder := fs.String("embedder", "deterministic-v1", "embedding model")
	version := fs.String("embedding-version", "v1", "embedding version")
	dims := fs.Int("dimensions", 768, "embedding dimensions")
	fs.Parse(args)
	if *out == "" {
		fail("--out is required")
	}
	events, report, err := load(*bronze)
	if err != nil {
		fail(err.Error())
	}
	g := &pipeline.SplitGenerator{Splits: []pipeline.SplitSpec{{Name: "train"}}}
	eps, m, err := pipeline.Materialize(events, g, *embedder, *version, "supervision-v1", *dims)
	if err != nil {
		fail(err.Error())
	}
	if err := pipeline.WriteJSONL(filepath.Join(*out, "gold.jsonl"), eps); err != nil {
		fail(err.Error())
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(*out, "manifest.json"), b, 0640); err != nil {
		fail(err.Error())
	}
	card := fmt.Sprintf("# Supervision data card\n\nManifest: %s\nEpisodes: %d\nQuarantined: %d\n", m.ManifestID, len(eps), report.Quarantined)
	if err := os.WriteFile(filepath.Join(*out, "DATA_CARD.md"), []byte(card), 0640); err != nil {
		fail(err.Error())
	}
	fmt.Printf("gold=%s manifest=%s\n", filepath.Join(*out, "gold.jsonl"), m.ManifestID)
}
func load(path string) ([]pipeline.SilverEvent, pipeline.QuarantineReport, error) {
	if path == "" {
		return nil, pipeline.QuarantineReport{}, fmt.Errorf("--bronze is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, pipeline.QuarantineReport{}, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64<<10), 64<<20)
	cfg := pipeline.DefaultSilverConfig()
	var events []pipeline.SilverEvent
	for s.Scan() {
		ev, ok := pipeline.NormalizeBronze(s.Bytes(), cfg)
		if ok {
			events = append(events, ev)
		}
	}
	if err := s.Err(); err != nil {
		return nil, pipeline.QuarantineReport{}, err
	}
	clean, r := pipeline.DedupSilver(events)
	if len(clean) == 0 {
		return nil, r, fmt.Errorf("no valid Bronze records")
	}
	return clean, r, nil
}
func fail(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(2) }

var _ schema.SplitName
