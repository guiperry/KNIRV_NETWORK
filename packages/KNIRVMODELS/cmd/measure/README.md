# measure

Baseline-measurement harness for `three_layer_inference.md`. Stdlib-only; each probe
writes `<dir>/results/<probe>.json`, `measure report` rolls them into `<dir>/REPORT.md`.
A probe that cannot run records `blocked` with the reason instead of being skipped.

    make measure && bin/measure help

Runs now, no inputs: `host`, `ollama`, `cloudflare`.
Needs llama up: `llama`, `lora` (raw upstream address), `judge`, `heartbench`.
Needs a readable GGUF + llama-server binary: `llama-sweep -yes` (spawns its own private server).
Needs the owner's labels: `gold add|label2|import|export|stats`, then `retrieval`, `embed`, `judge`.

When Colibri is serving (it is skipped on hosts without AVX2 or too slow), it can take
some of the work off the owner without becoming the authority:
`suggest -candidates FILE` writes suggested labels to `suggestions.jsonl` (NEI when the
two option orders disagree or the margin is under 0.12); the owner reviews them and runs
`gold accept -id ID -label L` (or `-agree`), which records the owner as labeler and marks
the case `teacher-suggested:<model>:agreed|overridden-from-X`. Suggestions can anchor
the labeler, so keep the 10% double-labelling (`gold label2`) independent of them.
`retrieval-judge` has Colibri grade the top-k passages for each gold query and reports
judged precision@k and relevant passages the gold set lacks; it is a machine judgment,
shown next to, never inside, the gold-based `retrieval` recall.

Gold cases default to `data_class=private`; only `public` ones are ever sent to Cloudflare.
`heartbench` reads `<dir>/heart_cases.jsonl` (`HeartCase` in eval.go): verified decisions as
`expected`, optionally the external generator's recorded output as `external`.
To compare llama builds for `lora`, run it once per build (`-server-bin` records `--version`).
