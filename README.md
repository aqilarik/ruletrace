# ruletrace

`ruletrace` is a small, domain-agnostic explainability layer for the
[`expr-lang`](https://github.com/expr-lang/expr) expression engine.

It helps you answer:

- Why did a rule pass/fail?
- Which conditions were evaluated?
- Which were skipped due to short-circuit?

This library intentionally does not embed domain meaning (promo, pricing, policy, entitlement, feature flags).
It only produces structured evaluation traces that higher-level systems can interpret.

---

You can write rules like:

```expr
user.Group in ["admin","moderator"] || user.Id == comment.UserId
```

…or attach:

- semantic IDs (`c_group`, `c_owner`)
- reason codes for pass and fail
```expr
Cond("c_group", "GROUP_ALLOWED", "GROUP_NOT_ALLOWED", user.Group in ["admin","moderator"])
```

### Why this library depends on expr-lang
ruletrace uses `expr-lang/expr` for parsing, type-checking, and evaluation. expr is a proven, fast expression engine that supports common operators (comparisons, membership, string ops), predicates, and functions.

By delegating expression execution to expr, ruletrace can focus on the runtime capabilities that rule systems usually need but expression engines intentionally don’t provide out of the box:

- deterministic tracing and explainability
- per-condition identity and reason codes
- explicit “skipped vs evaluated” visibility for short-circuit operators

> In short: **expr evaluates, ruletrace explains**.

### Why Cond(...) exists
Most real rule systems need more than a boolean. When a rule passes or fails, callers typically want:
- which specific condition decided the outcome
- a stable condition identifier (for logging, UI, analytics)
- a reason code for the true/false path
- visibility into short-circuiting (what wasn’t evaluated)

expr is an expression language, so it doesn’t define a standard way to attach “reason codes” or “explain traces” to sub-expressions. ruletrace introduces a minimal, generic instrumentation hook:
```expr
Cond(id, reasonTrue, reasonFalse, predicate)
```
This is not a business feature, it’s a runtime mechanism for capturing explainability metadata in a domain-agnostic way.

You can use `Cond(...)` in two ways:

- **Explicit instrumentation**: author it directly in the rule string.
- **Implicit instrumentation**: keep the rule clean, and let ruletrace patch the AST to wrap selected atomic predicates with `Cond(...)` based on your ConditionSpec mapping.

Both modes keep evaluation deterministic and allow **simulation** and **real execution** to share the same execution path.

## How it works

At trace time:

1. Compile input into an AST using expr-lang.
2. Extract “atomic predicates” (comparisons, membership, string ops).
3. If you provided metadata for a predicate, rewrite that AST node into:

```expr
Cond("c_group", "GROUP_ALLOWED", "GROUP_NOT_ALLOWED", user.Group in ["admin","moderator"])
```

4. Evaluate the patched AST with a registered `Cond` function that records outcomes.
5. Return structured `TraceResult` with chunks + final value.

This keeps the engine domain-agnostic while enabling explainability.

---

## Installation

```bash
go get github.com/aqilarik/ruletrace
```

---

## Example

```go
env := map[string]interface{}{
  "user": map[string]interface{}{"Group":"admin","Id":1},
  "comment": map[string]interface{}{"UserId":1},
}

input := `user.Group in ["admin","moderator"] || user.Id == comment.UserId`

specs := map[string]ruletrace.ConditionSpec{
  ruletrace.Fingerprint(`user.Group in ["admin","moderator"]`): {
    ID: "c_group", ReasonTrue:"GROUP_ALLOWED", ReasonFalse:"GROUP_NOT_ALLOWED",
  },
}

tracer := ruletrace.New(env, ruletrace.WithMode(ruletrace.TraceAtomic))
res := tracer.Trace(input, specs)
```

---

## Trace modes

- `TraceNone`: only `Final`
- `TraceCoarse`: one chunk per subtree (cheap)
- `TraceAtomic`: evaluate each atom (best explainability)
- `TraceAtomicFailuresOnly`: only errors/false/nil/skipped (low noise)

---

## Make targets

- `make test` – run unit tests
- `make lint` – run golangci-lint
- `make fmt` – gofmt
- `make tidy` – go mod tidy
- `make build` – build playground binary

---

## Design notes

1. **Fingerprint stability**: Fingerprints are derived from a canonical-ish formatter. If formatting changes between versions,
   fingerprints may drift. For stable production setups you typically generate fingerprints from the
   engine itself and store them (future enhancement).

2. **Cond chunk enrichment**: We attach semantic IDs/reasons to chunks by parsing the formatted `Cond("id", ...)` string
   (best-effort). A future version can attach IDs directly using AST metadata rather than string parsing.

3. **“Atom” definition is heuristic**: What counts as an atomic predicate is defined in `internal/patch/atoms.go`.

4. **Thread safety**: `Tracer` is safe for concurrent use if its `env` map is not mutated concurrently. For multi-goroutine usage, treat env as immutable or pass a copy per call.
