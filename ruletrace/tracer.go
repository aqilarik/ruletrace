package ruletrace

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"

	"github.com/aqilarik/ruletrace/internal/cond"
	"github.com/aqilarik/ruletrace/internal/eval"
	"github.com/aqilarik/ruletrace/internal/format"
	"github.com/aqilarik/ruletrace/internal/patch"
	"github.com/aqilarik/ruletrace/internal/util"
)

// TraceMode controls trace verbosity and CPU cost.
// - None: no chunks, only Final .
// - Coarse: evaluates subtrees as single chunks (low cost).
// - Atomic: extracts “atoms” (comparisons, membership, string ops) and evaluates each.
// - AtomicFailuresOnly: emit only errors/false/nil/skipped (lowest noise).
type TraceMode uint8

const (
	TraceNone TraceMode = iota
	TraceCoarse
	TraceAtomic
	TraceAtomicFailuresOnly
)

func (tm TraceMode) String() string {
	return [...]string{"TraceNone", "TraceCoarse", "TraceAtomic", "TraceAtomicFailuresOnly"}[tm]
}

// ConditionSpec is metadata attached to an atomic predicate.
// The map key is the engine Fingerprint (hash of the canonical atom expression).
//
// ID must be stable and authored by humans/systems. Do NOT derive it from Expr.
// ReasonTrue / ReasonFalse are both supported so you can explain “why it passed” and “why it failed”.
type ConditionSpec = patch.ConditionSpec

// EvalResult is a single trace item (one evaluated unit shown to UI/logs).
type EvalResult struct {
	ID          string      `json:"id,omitempty"`      // semantic stable ID (from ConditionSpec)
	Fingerprint string      `json:"fingerprint"`       // derived from canonical Expr
	Expr        string      `json:"expr"`              // canonical expression string of this unit
	Value       interface{} `json:"value,omitempty"`   // evaluated value (typically bool for atoms)
	Skipped     bool        `json:"skipped,omitempty"` // short-circuited
	Error       string      `json:"error,omitempty"`   // evaluation error if any
	Reason      string      `json:"reason,omitempty"`  // chosen based on true/false for Cond-wrapped atoms
}

type TraceResult struct {
	Source string       `json:"source"`           // patched canonical source (may include Cond(...))
	Chunks []EvalResult `json:"chunks,omitempty"` // trace units
	Final  interface{}  `json:"final,omitempty"`  // final result (authoritative, same execution path)
	Mode   TraceMode    `json:"mode"`
}

// Tracer runs expr-lang expressions with explainability features.
// This package is domain-agnostic: it does not interpret conditions; it only preserves metadata.
type Tracer struct {
	env          map[string]interface{}
	mode         TraceMode
	shortCircuit bool
	enableCond   bool
}

// New creates a tracer with options.
func New(env map[string]interface{}, opts ...Option) *Tracer {
	t := &Tracer{
		env:          env,
		mode:         TraceAtomic,
		shortCircuit: true,
		enableCond:   true,
	}
	for _, o := range opts {
		o.apply(t)
	}
	return t
}

func (t *Tracer) Trace(input string, specs map[string]ConditionSpec) TraceResult {
	res, _ := t.trace(input, specs, false)
	return res
}

func (t *Tracer) TraceStrict(input string, specs map[string]ConditionSpec) (TraceResult, error) {
	return t.trace(input, specs, true)
}

// trace evaluates `input` deterministically and returns a structured TraceResult.
//
// Error behavior
//   - By default, errors are *embedded* in the TraceResult (e.g. EvalResult.Error),
//     and the returned error is nil.
//   - If forceFailFast is true (or the tracer is configured for fail-fast),
//     the function returns a non-nil error as soon as it encounters a terminal issue.
//
// In fail-fast mode, the returned TraceResult is still best-effort and may contain
// partial chunks that were already produced before the failure. Callers may choose
// to ignore it or use it for debugging/logging.
//
// Condition instrumentation (optional)
//
// If enableCond is true and `specs` is provided, the tracer instruments the AST by
// wrapping matching atomic predicates as:
//
//	Cond(id, reasonTrue, reasonFalse, atom)
//
// This is done via AST patching so rule authors can keep writing clean expressions
// without explicitly calling Cond(...) in the source.
//
// # Execution-path guarantee
//
// The final result is computed using the *same patched source* that was used to
// generate trace chunks. This ensures “simulation” and “real run” share the same
// execution path (same logic, same short-circuit behavior, same Cond() outcomes).
//
// Parameters
//   - input: original authored expression.
//   - specs: metadata keyed by atom fingerprint used to decide which atoms get wrapped.
//   - forceFailFast: when true, return an error immediately on compile/eval failure.
func (t *Tracer) trace(input string, specs map[string]ConditionSpec, forceFailFast bool) (TraceResult, error) {
	ec := eval.NewCache()

	rec := cond.NewRecorder()
	opts := []expr.Option{expr.Env(t.env)}
	if t.enableCond {
		opts = append(opts, expr.Function("Cond", rec.Func()))
	}

	// 1) Compile original input to get AST
	tree, err := expr.Compile(input, opts...)
	if err != nil {
		res := TraceResult{
			Source: input,
			Chunks: []EvalResult{{
				Fingerprint: Fingerprint(input),
				Expr:        input,
				Error:       err.Error(),
			}},
			Final: nil,
			Mode:  t.mode,
		}
		if forceFailFast {
			return res, err
		}
		return res, nil
	}
	root := tree.Node()

	// 2) Patch atoms into Cond(...) if enabled and specs present
	fmter := format.New()
	if t.enableCond && len(specs) > 0 {
		root = patch.WrapAtomsWithCond(root, fmter, Fingerprint, specs)
	}

	// 3) Trace chunks on patched AST
	chunks := t.evalChunks(root, fmter, ec, opts...)

	// 4) Authoritative final evaluation uses patched canonical source
	patchedSource := fmter.Format(root)
	final, _ := eval.EvalString(patchedSource, t.env, ec, opts...)

	// 5) Enrich Cond chunks with semantic ID + reason from recorder.
	for i := range chunks {
		if !strings.HasPrefix(chunks[i].Expr, "Cond(") {
			continue
		}
		id := util.ExtractFirstStringArg(chunks[i].Expr)
		if id == "" {
			continue
		}
		rr, ok := rec.Seen()[id]
		if !ok {
			continue
		}
		chunks[i].ID = rr.ID
		chunks[i].Reason = rr.Reason
	}

	return TraceResult{
		Source: patchedSource,
		Chunks: chunks,
		Final:  final,
		Mode:   t.mode,
	}, nil
}

func (t *Tracer) evalChunks(node ast.Node, fmter *format.Formatter, ec *eval.Cache, opts ...expr.Option) []EvalResult {
	if t.mode == TraceNone {
		return nil
	}

	if ch, ok := node.(*ast.ChainNode); ok {
		return t.evalChunks(ch.Node, fmter, ec, opts...)
	}

	// Short-circuit ops: structurally trace left/right and mark skipped subtrees.
	if bn, ok := node.(*ast.BinaryNode); ok && patch.IsShortCircuitOp(bn.Operator) {
		switch bn.Operator {
		case "||", "or":
			left := t.evalChunks(bn.Left, fmter, ec, opts...)
			if t.shortCircuit && lastTrue(left) {
				return append(left, t.markSkipped(bn.Right, fmter)...)
			}
			return append(left, t.evalChunks(bn.Right, fmter, ec, opts...)...)
		case "&&", "and":
			left := t.evalChunks(bn.Left, fmter, ec, opts...)
			if t.shortCircuit && lastFalse(left) {
				return append(left, t.markSkipped(bn.Right, fmter)...)
			}
			return append(left, t.evalChunks(bn.Right, fmter, ec, opts...)...)
		case "??":
			left := t.evalChunks(bn.Left, fmter, ec, opts...)
			if t.shortCircuit && lastNotNil(left) {
				return append(left, t.markSkipped(bn.Right, fmter)...)
			}
			return append(left, t.evalChunks(bn.Right, fmter, ec, opts...)...)
		}
	}

	if t.mode == TraceCoarse {
		return []EvalResult{t.evalNode(node, fmter, ec, opts...)}
	}

	atoms := patch.CollectAtoms(node)
	if len(atoms) == 0 {
		return []EvalResult{t.evalNode(node, fmter, ec, opts...)}
	}

	out := make([]EvalResult, 0, len(atoms))
	for _, a := range atoms {
		r := t.evalNode(a, fmter, ec, opts...)
		if t.mode == TraceAtomicFailuresOnly {
			if r.Skipped || r.Error != "" || r.Value == false || r.Value == nil {
				out = append(out, r)
			}
			continue
		}
		out = append(out, r)
	}
	return out
}

func (t *Tracer) evalNode(node ast.Node, fmter *format.Formatter, ec *eval.Cache, opts ...expr.Option) EvalResult {
	exprStr := fmter.Format(node)
	val, errStr := eval.EvalString(exprStr, t.env, ec, opts...)
	return EvalResult{
		Fingerprint: Fingerprint(exprStr),
		Expr:        exprStr,
		Value:       val,
		Error:       errStr,
	}
}

func (t *Tracer) markSkipped(node ast.Node, fmter *format.Formatter) []EvalResult {
	exprStr := fmter.Format(node)
	return []EvalResult{
		{
			Fingerprint: Fingerprint(exprStr),
			Expr:        exprStr,
			Value:       nil,
			Skipped:     true,
		},
	}
}

func lastTrue(r []EvalResult) bool {
	return len(r) > 0 && r[len(r)-1].Error == "" && r[len(r)-1].Value == true
}
func lastFalse(r []EvalResult) bool {
	return len(r) > 0 && r[len(r)-1].Error == "" && r[len(r)-1].Value == false
}
func lastNotNil(r []EvalResult) bool {
	return len(r) > 0 && r[len(r)-1].Error == "" && r[len(r)-1].Value != nil
}

// ValidateSpecs is a lightweight guard for obvious mistakes.
func ValidateSpecs(specs map[string]ConditionSpec) error {
	for fp, s := range specs {
		if fp == "" {
			return fmt.Errorf("empty fingerprint key in specs")
		}
		if s.ID == "" {
			return fmt.Errorf("spec for fp=%s has empty ID", fp)
		}
	}
	return nil
}
