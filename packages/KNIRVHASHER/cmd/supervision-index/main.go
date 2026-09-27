// supervision-index builds and verifies indexes from pinned Gold manifests.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"knirvhasher/pkg/embeddings"
	"knirvhasher/pkg/hashing/schema"
	"knirvhasher/pkg/hashing/supervision"
	"knirvhasher/pkg/hashing/supervision/pipeline"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: supervision-index <build|verify>")
	}
	switch os.Args[1] {
	case "build":
		build(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	default:
		fail("unknown operation")
	}
}
func build(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	episodes := fs.String("episodes", "", "Gold JSONL")
	manifest := fs.String("manifest", "", "manifest JSON")
	out := fs.String("out", "", "index JSON")
	fs.Parse(args)
	m := readManifest(*manifest)
	f, err := os.Open(*episodes)
	if err != nil {
		fail(err.Error())
	}
	defer f.Close()
	eps, err := pipeline.ReadJSONL(f)
	if err != nil {
		fail(err.Error())
	}
	idx, err := supervision.BuildIndexForManifest(eps, m, func(s string) []float32 { return embeddings.NewDeterministicService().GetEmbedding(s) })
	if err != nil {
		fail(err.Error())
	}
	if err := idx.Save(*out); err != nil {
		fail(err.Error())
	}
	fmt.Printf("index=%s manifest=%s\n", *out, m.ManifestID)
}
func verify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	index := fs.String("index", "", "index JSON")
	manifest := fs.String("manifest", "", "manifest JSON")
	fs.Parse(args)
	m := readManifest(*manifest)
	idx, err := supervision.LoadIndex(*index)
	if err != nil {
		fail(err.Error())
	}
	if idx.DatasetManifestID != m.ManifestID || idx.PolicyBundleHash != m.PolicyBundleHash {
		fail("index manifest binding mismatch")
	}
	fmt.Printf("verified index manifest=%s entries=%d\n", m.ManifestID, len(idx.Entries))
}
func readManifest(path string) *schema.DatasetManifest {
	if path == "" {
		fail("--manifest is required")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		fail(err.Error())
	}
	var m schema.DatasetManifest
	if err := json.Unmarshal(b, &m); err != nil {
		fail(err.Error())
	}
	if err := m.Validate(); err != nil {
		fail(err.Error())
	}
	if err := m.VerifyManifestID(); err != nil {
		fail(err.Error())
	}
	return &m
}
func fail(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(2) }
