// Command framecheck verifies that the model knows what its role frames tell it:
// the per-stage self-knowledge probe, the registry/manifest drift test and the
// tool-use eval (spec §3.3, "verifying awareness"). It talks to the llama wrapper,
// so on a slow host run it as an overnight batch (each probe is a full generation).
//
//	framecheck -socket /var/lib/knirvserver/sockets/llama.sock -threshold 0.75
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/models"
	"github.com/guiperry/knirv-sdk-go/models/frame"

	"knirvmodels/internal/ops"
	"knirvmodels/internal/policy"
	"knirvmodels/pkg/framecheck"
	"knirvmodels/pkg/heartqwen"
)

func main() {
	socket := flag.String("socket", "/var/lib/knirvserver/sockets/llama.sock", "llama wrapper unix socket")
	model := flag.String("model", "local-llama", "model name to send")
	threshold := flag.Float64("threshold", 0.75, "minimum per-stage probe score (set from the Phase-0 measurements)")
	stages := flag.String("stages", "", "comma-separated stage ids (default: all)")
	skipTools := flag.Bool("skip-tools", false, "run only the probe and the drift test")
	out := flag.String("out", "", "write the JSON report here")
	flag.Parse()

	reg := frame.Default()
	facts := frame.LiveFacts{Model: *model, NodeMode: "local", Capabilities: map[string]bool{}}
	llm := &heartqwen.HTTPLLM{Client: sdk.NewUnixHTTPClient(*socket, 30*time.Minute), Base: "http://llama", Model: *model}
	ctx := context.Background()

	report := map[string]any{"threshold": *threshold}
	fail := false

	drift := framecheck.Drift(reg, reg.Manifest(facts), facts)
	report["drift"] = drift
	if len(drift) > 0 {
		fail = true
	}

	ids := reg.StageIDs()
	if *stages != "" {
		ids = strings.Split(*stages, ",")
	}
	scores, err := framecheck.Probe(ctx, reg, facts, llm, ids)
	if err != nil {
		fmt.Fprintln(os.Stderr, "probe:", err)
		report["probe_error"] = err.Error()
		fail = true
	}
	report["probe"] = scores
	for _, s := range scores {
		if s.Score < *threshold {
			fail = true
		}
	}

	if !*skipTools && err == nil {
		eng := &heartqwen.Engine{Reg: reg, Facts: func() frame.LiveFacts { return facts }, LLM: llm,
			Gate: policy.NewGate(reg, nil), Exec: ops.Standard(ops.Deps{
				Manifest: func() any { return map[string]string{"note": "framecheck stub"} },
				Registry: func() any { return []string{} },
				Health:   func() any { return map[string]string{"status": "ok"} },
			})}
		res, terr := framecheck.ToolEval(ctx, eng, framecheck.ToolCases(reg))
		report["tool_eval"] = res
		if terr != nil || res.Executed > 0 {
			fail = true
		}
	}

	b, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(b))
	if *out != "" {
		_ = os.WriteFile(*out, b, 0o644)
	}
	if fail {
		os.Exit(1)
	}
}
