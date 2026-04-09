# Caliber Learnings

Accumulated learnings from development sessions.
Auto-managed by [caliber](https://github.com/caliber-ai-org/ai-setup) — do not edit manually.

<!-- LEARNING FOCUS: Record ONLY build, testing, and deployment context — e.g. module/dependency issues that break compilation, test commands and their quirks, deployment steps and environment requirements. Do NOT record code-level bugs, security findings, code patterns, or developer gotchas about API usage. Those belong in code_audit.md or the issue tracker. -->

- **[context]** packages/KNIRVBASE/go/ is the source-of-truth implementation for KNIRVBASE; Rust and TS must conform to it. When discrepancies exist between language implementations, Go's behavior is authoritative
- **[gotcha]** `packages/KNIRVSHELL/go.mod` declares module `github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI`, but the spec and `internal/` imports reference `github.com/knirv/knirvshell`. The package is uncompilable until one name is chosen and applied consistently across all files.
- **[gotcha]** `packages/KNIRVSHELL/go.mod` previously had duplicate `require` entries for OpenTelemetry packages and `golang.org/x/sys` with conflicting versions. Always run `go mod tidy` inside `packages/KNIRVSHELL/` after any dependency change — do not trust the lockfile as-is.
**[pattern]** When auditing KNIRVBASE cross-language consistency: analyze source code directly (not docs), use Go as source of truth, produce a consistency_status.md with a module coverage matrix (Go/Rust/TS) and issues organized by severity (Critical/High/Medium/Low) with file:line references and a prioritized resolution order

- **[context]** packages/KNIRVBASE/go/ is the source-of-truth implementation for KNIRVBASE; Rust and TS must conform to it. When discrepancies exist between language implementations, Go's behavior is authoritative
