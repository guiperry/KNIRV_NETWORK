# The Clarion Programming Language
## Specification, Version 0.1 (Draft)

**Status:** Design draft
**Audience:** Language implementers, agent-framework authors, and LLM agents themselves
**One-line summary:** Clarion is a statically typed, capability-secure, effect-explicit language with exactly one canonical form, designed so that an LLM agent can generate, verify, and surgically edit code with maximal reliability and minimal context.

---

## 1. Motivation and Design Thesis

Every mainstream programming language was designed for a human holding a mental model in working memory, assisted by an editor. LLM agents are a different kind of programmer. They have enormous breadth but a bounded context window; they generate code token-by-token without backtracking; they edit code through textual or structural patches rather than cursors; and they consume compiler feedback as text to condition the next generation, not as a red squiggle.

Optimizing a language for this programmer changes the design priorities in specific, sometimes counterintuitive ways:

**Predictability beats terseness.** Token count matters, but a language that is 20% more verbose and 5x more predictable is a massive net win, because an LLM's per-token error rate compounds. Clarion deliberately spends tokens on explicit terminators, mandatory signatures, and keyword-led statements because each one collapses the space of valid continuations and makes both generation and constrained decoding dramatically more reliable.

**One canonical form.** Stylistic variance is pure noise for a statistical model. If the same program can be written ten ways, the model's probability mass is split ten ways, diffs are polluted by formatting churn, and exact-match patch operations fail. Clarion has no formatting choices: the grammar plus the canonicalizer define a single byte-exact representation for every program. `clarion fmt` is not a linter; it is part of the language definition.

**Local reasoning is the prime directive.** An agent should be able to fully understand what a function can do — what it reads, writes, calls, and is allowed to touch — from its signature alone, without loading the transitive closure of the codebase into context. This drives the three central semantic features: explicit effect annotations, capability-based authority, and the prohibition of all hidden control flow (no exceptions, no operator overloading, no implicit conversions, no inheritance, no ambient globals).

**The compiler is a conversation partner.** Clarion's compiler is specified as a long-running service with a JSON-RPC protocol. Diagnostics are structured data with machine-applicable fix candidates, not prose. Every AST node can be queried for its type, effects, and provenance. Edits are submitted as AST transactions that either apply atomically or return a structured rejection. The "stringly-typed patch that silently corrupts a file" failure mode is designed out.

**Verification lives in the signature.** Functions carry executable examples, contracts, and property declarations as part of their declaration. The compiler runs them. An agent can therefore make a change and get a tight verify loop — typecheck, contract check, example check — in milliseconds, without locating an external test suite.

---

## 2. Design Principles (Normative)

These principles resolve all design disputes in this specification. Lower numbers win.

1. **P1 — Determinism of form:** Every abstract syntax tree has exactly one concrete textual rendering. Any tool emitting Clarion source MUST emit canonical form.
2. **P2 — Signature completeness:** Everything a caller needs to know about a function's behavior boundary (types, effects, capabilities, failure modes, contracts) appears in its signature.
3. **P3 — No hidden semantics:** No exceptions, no implicit conversions, no operator/function overloading, no reflection that alters behavior, no inheritance, no global mutable state, no ambient authority.
4. **P4 — Keyword anchoring:** Every statement begins with a keyword; every block ends with a keyword naming its construct. A truncated program is syntactically detectable as truncated.
5. **P5 — Errors are data:** All compiler and runtime diagnostics are structured, stable-schema JSON with source spans, causal chains, and ranked fix candidates.
6. **P6 — Small orthogonal core:** One way to do each thing. Convenience features are rejected if they introduce a second way.
7. **P7 — Pay-for-what-you-prove:** Contracts are checked dynamically by default, discharged statically when the prover can, and the boundary between the two is always visible.

---

## 3. Lexical Structure

### 3.1 Encoding and tokens

Source is UTF-8. Identifiers are ASCII `snake_case` for values and functions, `PascalCase` for types, `SCREAMING_SNAKE` for constants — enforced by the compiler, not convention (P1). There is exactly one comment form, `--` to end of line; block comments are forbidden because they create non-local lexical state that interacts badly with line-oriented patching.

Statements terminate at end of line. There are no semicolons and no line-continuation characters; an expression that must span lines does so only inside unclosed brackets, which is unambiguous. Indentation is canonically two spaces per nesting level but is **not semantically significant** — semantics come from the `end` keywords. This is deliberate: whitespace-significant syntax is the single largest cause of corrupted LLM-generated patches, while pure brace languages make truncation invisible. Clarion takes the third path: redundant encoding. The canonicalizer enforces indentation; the parser ignores it; disagreement between the two is reported as diagnostic `W0001 indentation_drift` so agents can detect mangled edits even when they remain parseable.

### 3.2 Anchored blocks

Every block-opening construct is closed by `end` plus the construct keyword:

```clarion
fn example() -> u32
do
  if x > 0
  then
    return 1
  end if
  return 0
end fn
```

This costs tokens and pays for them three ways: a truncated generation never parses (P4); `end fn` vs `end if` mismatches localize errors to the exact block rather than "unexpected EOF"; and unique `end <kw>` lines give exact-string patch operations reliable anchors.

### 3.3 Numeric and string literals

Numeric types are explicit-width only: `i8..i64`, `u8..u64`, `f32`, `f64`, and `dec` (decimal, for money). Literals require a suffix unless the type is pinned by the immediate context: `42_u32`, `3.5_f64`, `19.99_dec`. There is no implicit widening or narrowing; conversions are explicit calls (`u32.from_i64(x)` returning `Result`). String literals use double quotes with `\` escapes; multi-line strings use `text` blocks:

```clarion
let prompt = text
  You are a helpful assistant.
  Respond only in JSON.
end text
```

`text` blocks strip the common leading indentation, defined canonically, so the same logical string always has the same encoding.

---

## 4. Declarations and Program Structure

### 4.1 Modules

A module is a file. The first line declares its fully qualified name and semantic version:

```clarion
module acme.billing.invoices v2.1.0
```

All imports are explicit, item-level, and listed once at the top — never mid-file, never glob:

```clarion
use std.result (Result, ok, err)
use std.list (List)
use acme.billing.tax v2 (TaxRate, compute_tax)
```

There is no re-export, no `prelude` beyond a fixed, specified core (`bool`, numeric types, `Result`, `Option`, `List`, `Map`, `String`), and no name shadowing anywhere in the language — a name binds once per scope chain. An agent reading any identifier can resolve it by scanning exactly two places: the local scope chain and the `use` block.

### 4.2 Declaration forms

The complete set: `fn`, `record`, `union` (sum type), `alias`, `const`, `trait`, `impl`, `test`. Examples:

```clarion
record Invoice
  id: InvoiceId
  customer: CustomerId
  lines: List<LineItem>
  issued_at: Timestamp
end record

union PaymentState
  case Pending
  case Settled(at: Timestamp, ref: TxRef)
  case Failed(reason: FailureReason)
end union

alias InvoiceId = Uuid tagged "invoice"
```

`tagged` creates a zero-cost nominal wrapper: `InvoiceId` and `CustomerId` are both `Uuid` at runtime but unmixable at compile time. Agents confuse positionally similar IDs constantly; this makes the confusion a type error with a one-token fix.

### 4.3 Functions: the full signature

A function signature is a contract surface containing, in fixed canonical order: name, parameters, return type, purity/effect clause, capability requirements, contracts, and executable examples.

```clarion
fn apply_payment(inv: Invoice, amount: dec, clock: cap Clock) -> Result<Invoice, PaymentError>
  effects clock.read
  requires amount > 0.0_dec
  ensures result.is_ok() implies result.unwrap().balance() >= 0.0_dec
  example
    let inv = test_invoice(total = 100.00_dec)
    assert apply_payment(inv, 100.00_dec, test_clock()).is_ok()
  end example
do
  ...
end fn
```

Nothing in a signature is inferred. This is the heaviest token tax in the language and the most defensible: signatures are what agents read in lieu of bodies, what retrieval systems index, and what makes editing a 500-file codebase with a 200k-token window possible. Inside bodies, `let` types are inferred — local inference is cheap for a reader whose context already contains the whole function.

---

## 5. Type System

Clarion is statically and strongly typed with no escape hatches in safe code (an `unsafe.ffi` module exists, quarantined behind a capability).

**Algebraic data types** (`record`, `union`) with exhaustive `match` — non-exhaustive matches are errors, and the diagnostic lists the missing cases as a ready-to-paste fix candidate.

**Generics** are explicit and constraint-bounded: `fn dedupe<T where T: Eq + Hash>(items: List<T>) -> List<T>`. There is no specialization and no overloading; monomorphization is an implementation detail.

**Traits** are nominal interfaces. A type implements a trait only via an explicit `impl` block; there are no blanket impls, no orphan impls, and no coherence puzzles. Trait method dispatch is always statically resolvable from the call site.

**Refinements** attach predicate clauses to types in signatures: `fn chunk(data: List<u8>, size: u32 where size > 0_u32) -> ...`. Refinements are sugar for `requires` and follow P7's checking model.

**Typed holes.** The expression `todo "reason" : T` typechecks as `T`, panics if executed, and is tracked by the compiler: `clarion holes` returns every hole with its expected type, in-scope bindings, and required effects. This is the language's native mechanism for plan-then-fill agent workflows — an agent can scaffold an entire module with holes, get it compiling, then fill holes one at a time with the compiler narrating exactly what each hole needs.

---

## 6. Effects and Capabilities

This is the semantic heart of the language, and the two systems interlock.

### 6.1 Capabilities: no ambient authority

There is no global filesystem, network, clock, environment, or randomness. All authority enters a program through `main`:

```clarion
fn main(root: cap Root) -> i32
```

`cap Root` can be split into narrower capabilities — `cap Fs`, `cap Net`, `cap Clock`, `cap Rand`, `cap Env`, `cap Proc` — and attenuated: `root.fs.subtree("/var/app/data")` yields a `cap Fs` that cannot name anything outside that subtree; `root.net.restrict(hosts = ["api.stripe.com"])` yields a `cap Net` that can reach one host. Capabilities are unforgeable values passed as ordinary parameters. A function without a `cap Net` parameter *cannot* touch the network — not by policy, by construction.

The payoff for agentic coding is twofold. First, auditability: to answer "can this dependency exfiltrate data?" an agent greps signatures, not bodies. Second, safe self-execution: an agent can run code it just wrote — including third-party packages — inside a capability sandbox expressed in the language itself, with confidence proportional to a type-soundness proof rather than a container configuration.

### 6.2 Effects: what the signature admits to

Every function is `pure` unless it declares effects. Effects name the capability and operation class: `effects fs.read, clock.read`. Effects compose transitively and the compiler enforces the bound: calling a function whose effects exceed yours is a type error. Purity is therefore checkable and meaningful — a `pure` function is deterministic, total over its contracts, and safe to memoize, replay, or speculatively execute, which agent runtimes exploit heavily (§12).

### 6.3 Failure: one model

There are no exceptions. Recoverable failure is `Result<T, E>`; absence is `Option<T>`. Propagation is the keyword-led `try` statement (P4 — even error plumbing starts with a keyword):

```clarion
let user = try fetch_user(id, net)    -- on err: return the err from this fn
```

Unrecoverable failure is `panic`, which unwinds to the enclosing task boundary and is not catchable in ordinary code. Every error union in `std` and every diagnostic-bearing `E` implements trait `Explain`, producing the same structured-diagnostic schema the compiler uses (§9), so runtime failures and compile-time failures arrive in an agent's context in one format.

---

## 7. Statements, Expressions, Mutability

Every statement begins with a keyword: `let`, `var`, `set`, `if`, `match`, `for`, `while`, `return`, `try`, `defer`, `assert`, `panic`, `spawn`, plus bare `call` for effectful calls whose result is discarded (discarding a `Result` without `call ... |> ignore_err` is an error). Bindings are immutable by default; `var` declares a mutable local and `set x = ...` mutates it — mutation is grep-able. There is no global mutable state of any kind.

Data is value-semantic: assignment and passing are conceptually copies (implemented as copy-on-write). There are no reference parameters, no aliasing, and therefore no aliasing bugs — the entire class of "function mutated my argument" surprises is absent, and with it the hardest category of whole-program reasoning. Functions that "modify" data return new values:

```clarion
fn mark_settled(inv: Invoice, at: Timestamp, ref: TxRef) -> Invoice
  pure
do
  return inv with (state = PaymentState.Settled(at = at, ref = ref))
end fn
```

`with` is the record-update expression. Pipelines use `|>` with explicit placement of the piped value as the first argument — the only operator-like sugar in the language, kept because it linearizes nested calls into the order an agent reasons in.

All function calls with more than one argument use named arguments (`compute_tax(rate = r, base = subtotal)`); single-argument calls may be positional. Argument-order transposition — a top-five LLM bug class — becomes either impossible or a compile error.

---

## 8. Inline Verification

Three constructs make verification part of the declaration rather than a parallel artifact:

**`example` blocks** (shown in §4.3) are executable, run by `clarion check`, and required by lint level `strict` on every public function. They double as in-context documentation: when an agent retrieves a signature, it retrieves working usage.

**`requires` / `ensures` contracts** are boolean expressions over parameters and `result`. Per P7: checked at runtime in `dev` profile, discharged statically where the bundled SMT-backed prover can, and in `release` profile any contract that was *not* statically discharged remains a runtime check unless explicitly waived — soundness never silently degrades.

**`property` blocks** declare randomized properties with generated inputs:

```clarion
test reverse_involutive
  property forall xs: List<u32>
    assert reverse(reverse(xs)) == xs
  end property
end test
```

The agent workflow this enables: generate a function, generate properties it should satisfy, and let `clarion check --shrink` find minimal counterexamples — returned, of course, as structured data containing a ready-to-paste failing `example` block.

---

## 9. The Toolchain Protocol (Compiler as Service)

The reference toolchain is a daemon speaking JSON-RPC; the CLI is a thin client. The protocol is part of the language specification because agents are its primary users.

**Diagnostics schema.** Every diagnostic carries: stable `code` (e.g. `E0312`), `severity`, primary `span` and labeled secondary spans, a `message` written for machine consumption (declarative, no rhetorical questions, no "did you mean...?" prose — suggestions go in fix candidates), a `cause_chain` for derived errors, and `fixes`: ranked, machine-applicable AST transformations with confidence scores. The compiler MUST cap diagnostics per request and MUST topologically order them so root causes precede downstream noise — an agent reading the first diagnostic should be reading the actual problem.

**Core methods.**

```
check(scope)            -> [Diagnostic]            -- parse+type+effect+contract
edit(txn)               -> Applied | Rejected      -- atomic AST transaction
query.node(loc)         -> {type, effects, caps, decl_site, doc}
query.holes(scope)      -> [{span, expected_type, in_scope, required_effects}]
query.callers(symbol)   -> [Span]
query.deps(symbol)      -> signature closure       -- min context to edit symbol
test.run(selector)      -> structured results incl. shrunk counterexamples
trace.run(entry, args)  -> recorded effect trace   -- see §12
fmt.canon(source)       -> canonical text + drift report
```

`edit` transactions operate on stable node IDs (the canonicalizer assigns content-derived IDs to every declaration), so an agent can say "replace the body of `acme.billing.invoices/apply_payment`" without exact-string matching against possibly-stale text. Textual patching still works — canonical form plus `end` anchors make it reliable — but structural editing is the blessed path.

**`query.deps` deserves emphasis:** given a symbol, it returns the minimal signature closure needed to correctly modify it — exactly the right thing to stuff into a context window. The language's signature-completeness principle (P2) is what makes this closure small and sufficient.

---

## 10. Concurrency

Structured concurrency only. Tasks are created inside a `scope` block and cannot outlive it; a scope joins or cancels all children on exit, and panics propagate to the scope.

```clarion
fn fetch_all(ids: List<UserId>, net: cap Net) -> Result<List<User>, FetchError>
  effects net.get
do
  scope s
    var handles: List<Task<Result<User, FetchError>>> = list()
    for id in ids
    do
      spawn in s fetch_user(id, net) into handles
    end for
    return collect_results(join_all(handles))
  end scope
end fn
```

Because data is value-semantic and globals don't exist, there is no shared mutable memory and no data race is expressible; cross-task communication is via channels (`cap`-free, since they confer no external authority). Deadlock remains possible but is detectable by the runtime's wait-graph, reported as — naturally — a structured diagnostic with the cycle.

---

## 11. Memory and Runtime Model

Clarion is garbage-collected with copy-on-write value semantics, compiled AOT or JIT; performance ceilings are a non-goal relative to predictability (the FFI exists for hot kernels, behind `cap Unsafe`, with effects declared at the boundary and trusted-not-proven — the one place the language says "audit this"). Evaluation is strict, left-to-right, fully specified; there is no undefined behavior anywhere in safe Clarion, and no implementation-defined behavior in the core language. Two compliant implementations given the same program and inputs produce identical results, including iteration order of `Map` (insertion-ordered by definition). Nondeterminism enters only through capabilities — which is what makes §12 possible.

## 12. Record/Replay and the Agent Loop

Because every effect flows through a capability, the runtime can record every effectful interaction (`trace.run`) into a deterministic trace, and replay a program against a recorded trace with zero live side effects. This gives agents three superpowers specified as standard, not bolted on: **time-travel debugging** (replay to any point, query any value — the agent debugging experience becomes "ask questions about a frozen world"); **regression oracles** (record production-shaped traces once, replay against modified code forever, diff the effect streams); and **counterfactual editing** (change code, replay the same world, observe exactly what diverges — the tightest possible feedback signal for an iterating model).

## 13. Standard Library (Outline)

`std` is small, capability-honest, and complete enough that typical agent tasks need zero third-party dependencies: `std.result`, `std.option`, `std.list/map/set/string/bytes` (persistent structures), `std.json` and `std.schema` (typed codecs; `derive codec` on any record/union — the only `derive` in the language), `std.fs/net.http/clock/rand/env/proc` (all capability-gated), `std.task/channel`, `std.test`, and `std.llm` — a capability-gated, schema-typed interface for model calls (`cap Llm`), because programs written by agents increasingly *contain* agents, and an untyped string-prompt boundary inside an otherwise verified program is a hole the language should close: prompts are typed templates, responses are schema-decoded `Result`s, and `llm.infer` is an effect like any other — recordable, replayable, and visible in every signature above it.

## 14. What Clarion Deliberately Omits

Each omission follows from P3/P6: exceptions and `try/catch` (hidden control flow), inheritance (action at a distance through super-classes), operator and function overloading (call sites must resolve locally), macros and reader extensions (they fork the grammar and break P1), implicit conversions, nullable types, global mutable state, ambient I/O, reflection-driven behavior, and configurable formatting. The consistent theme: any feature whose value is "saves a human some typing" at the cost of "requires non-local knowledge to read" is a bad trade when the typing is free and the reading is metered in context tokens.

---

## Appendix A — Grammar (Abridged EBNF)

```ebnf
module      = "module" qual_name version NL use* decl* ;
use         = "use" qual_name [version] "(" ident_list ")" NL ;
decl        = fn_decl | record_decl | union_decl | alias_decl
            | const_decl | trait_decl | impl_decl | test_decl ;

fn_decl     = "fn" ident [generics] "(" params ")" "->" type NL
              [effect_clause] [contract*] [example*]
              "do" NL stmt* "end" "fn" NL ;
effect_clause = ("pure" | "effects" effect_list) NL ;
contract    = ("requires" | "ensures") expr NL ;

stmt        = let_s | var_s | set_s | if_s | match_s | for_s | while_s
            | return_s | try_s | defer_s | assert_s | panic_s
            | spawn_s | call_s ;
if_s        = "if" expr NL "then" NL stmt* ["else" NL stmt*] "end" "if" NL ;
match_s     = "match" expr NL ("case" pattern NL stmt*)+ "end" "match" NL ;
try_s       = "let" ident "=" "try" expr NL ;
```

(The full grammar is LL(2); every statement is keyword-initial and every block keyword-terminal, satisfying P4 and enabling grammar-constrained decoding.)

## Appendix B — A Complete Small Program

```clarion
module demo.wordcount v0.1.0

use std.fs (read_text)
use std.string (split_words, to_lower)
use std.map (Map)
use std.result (Result)

fn count_words(text: String) -> Map<String, u64>
  pure
  example
    assert count_words("a B a").get("a") == some(2_u64)
  end example
do
  var counts: Map<String, u64> = map()
  for word in split_words(to_lower(text))
  do
    set counts = counts.update(word, fn (n: Option<u64>) -> u64
      do
        return n.unwrap_or(0_u64) + 1_u64
      end fn)
  end for
  return counts
end fn

fn main(root: cap Root) -> i32
do
  let fs = root.fs.subtree("./data")
  let text = match read_text(fs, "input.txt")
    case ok(t)
      t
    case err(e)
      panic e.explain()
  end match
  call print(root.stdio, count_words(text).to_json())
  return 0
end fn
```

## Appendix C — Diagnostic Example

```json
{
  "code": "E0231",
  "severity": "error",
  "message": "effect 'net.get' not permitted: caller 'render_report' is declared pure",
  "span": {"module": "acme.reports", "decl": "render_report", "node": "n_8f2c"},
  "cause_chain": [
    {"decl": "fetch_logo", "declares": ["net.get"]},
    {"decl": "render_header", "inherits": ["net.get"]}
  ],
  "fixes": [
    {"confidence": 0.91, "title": "add 'effects net.get' to render_report and thread 'net: cap Net'",
     "edit": {"txn": "..."}},
    {"confidence": 0.62, "title": "pass logo bytes as a parameter; keep render_report pure",
     "edit": null}
  ]
}
```

Note the second fix candidate: lower confidence, no automatic edit, but it names the *architecturally better* option. The diagnostic format is designed to let the compiler be a design advisor, not just a gatekeeper.

---

*End of specification draft v0.1.*