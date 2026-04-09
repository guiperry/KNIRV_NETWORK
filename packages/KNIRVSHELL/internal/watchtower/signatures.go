package watchtower

import "regexp"

type Signature struct {
	Name    string
	Pattern *regexp.Regexp
}

type SignatureSet struct {
	sigs []Signature
}

// Match returns (true, signatureName) on the first pattern match.
func (s *SignatureSet) Match(line string) (bool, string) {
	for _, sig := range s.sigs {
		if sig.Pattern.MatchString(line) {
			return true, sig.Name
		}
	}
	return false, ""
}

// DefaultSignatures covers Python, Go, Node.js, and major AI API errors.
func DefaultSignatures() *SignatureSet {
	patterns := []struct {
		name    string
		pattern string
	}{
		// Python
		{"PY_TRACEBACK", `Traceback \(most recent call last\)`},
		{"PY_EXCEPTION", `^[A-Za-z]+Error: .+`},
		{"PY_UNHANDLED", `unhandled exception in thread`},
		// Go
		{"GO_PANIC", `^goroutine \d+ \[`},
		{"GO_FATAL", `fatal error:`},
		{"GO_SEGFAULT", `SIGSEGV`},
		// Node.js
		{"NODE_UNHANDLED", `UnhandledPromiseRejectionWarning`},
		{"NODE_EXCEPTION", `(?:^\s*at .+|^\s*(?:\w+)?Error: .+)`},
		{"NODE_CRASH", `node: internal/process`},
		// OpenAI / Anthropic API
		{"API_RATE_LIMIT", `rate.limit|RateLimitError|429`},
		{"API_CONTEXT", `context.length.exceeded|max_tokens|context window`},
		{"API_AUTH", `AuthenticationError|401 Unauthorized`},
		{"API_OVERLOAD", `overloaded_error|529|ServiceUnavailable`},
		// Agentic / LLM loop patterns
		{"AGENT_LOOP", `infinite loop detected|reasoning loop|stuck in loop`},
		{"AGENT_TOOL_FAIL", `Tool execution failed|tool_use.*error`},
		{"AGENT_HALLUCINATION", `hallucination detected|confidence below threshold`},
		// Generic
		{"OOM", `out of memory|OOMKilled|cannot allocate`},
		{"TIMEOUT", `context deadline exceeded|operation timed out`},
		{"DISK_FULL", `no space left on device`},
	}

	set := &SignatureSet{}
	for _, p := range patterns {
		set.sigs = append(set.sigs, Signature{
			Name:    p.name,
			Pattern: regexp.MustCompile(`(?i)` + p.pattern),
		})
	}
	return set
}
