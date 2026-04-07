# Caliber Learnings

Accumulated patterns and anti-patterns from development sessions.
Auto-managed by [caliber](https://github.com/caliber-ai-org/ai-setup) — do not edit manually.

- **[pattern]** When auditing KNIRVBASE cross-language consistency: analyze source code directly (not docs), use Go as source of truth, produce a consistency_status.md with a module coverage matrix (Go/Rust/TS) and issues organized by severity (Critical/High/Medium/Low) with file:line references and a prioritized resolution order
- **[context]** packages/KNIRVBASE/go/ is the source-of-truth implementation for KNIRVBASE; Rust and TS must conform to it. When discrepancies exist between language implementations, Go's behavior is authoritative
