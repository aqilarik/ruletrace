package patch

import (
	"github.com/expr-lang/expr/ast"

	"github.com/aqilarik/ruletrace/internal/format"
)

// ConditionSpec is metadata attached to an atomic predicate.
type ConditionSpec struct {
	ID          string
	ReasonTrue  string
	ReasonFalse string
}

// WrapAtomsWithCond patches the AST by wrapping known atoms with Cond(id, rT, rF, atom).
//
// - fmter is used to canonicalize atoms to strings.
// - fingerprint must match how the caller builds spec keys.
// - It preserves node location via ast.Patch.
func WrapAtomsWithCond(
	root ast.Node,
	fmter *format.Formatter,
	fingerprint func(string) string,
	specs map[string]ConditionSpec,
) ast.Node {
	if len(specs) == 0 {
		return root
	}

	var walk func(n *ast.Node)

	walk = func(n *ast.Node) {
		if n == nil || *n == nil {
			return
		}

		// recurse
		switch x := (*n).(type) {
		case *ast.BinaryNode:
			walk(&x.Left)
			walk(&x.Right)
		case *ast.UnaryNode:
			walk(&x.Node)
		case *ast.ConditionalNode:
			walk(&x.Cond)
			walk(&x.Exp1)
			walk(&x.Exp2)
		case *ast.CallNode:
			walk(&x.Callee)
			for i := range x.Arguments {
				walk(&x.Arguments[i])
			}
		case *ast.BuiltinNode:
			if x.Map != nil {
				walk(&x.Map)
			}
			for i := range x.Arguments {
				walk(&x.Arguments[i])
			}
		case *ast.MemberNode:
			walk(&x.Node)
			walk(&x.Property)
		case *ast.SliceNode:
			walk(&x.Node)
			if x.From != nil {
				walk(&x.From)
			}
			if x.To != nil {
				walk(&x.To)
			}
		case *ast.ArrayNode:
			for i := range x.Nodes {
				walk(&x.Nodes[i])
			}
		case *ast.MapNode:
			for i := range x.Pairs {
				walk(&x.Pairs[i])
			}
		case *ast.PairNode:
			walk(&x.Key)
			walk(&x.Value)
		case *ast.SequenceNode:
			for i := range x.Nodes {
				walk(&x.Nodes[i])
			}
		case *ast.VariableDeclaratorNode:
			walk(&x.Value)
			walk(&x.Expr)
		case *ast.PredicateNode:
			walk(&x.Node)
		case *ast.ChainNode:
			walk(&x.Node)
		}

		// wrap atoms post-order
		if IsCondCall(*n) || !IsAtomNode(*n) {
			return
		}

		atomExpr := fmter.Format(*n)
		fp := fingerprint(atomExpr)

		spec, ok := specs[fp]
		if !ok || spec.ID == "" {
			return
		}

		wrapped := &ast.CallNode{
			Callee: &ast.IdentifierNode{Value: "Cond"},
			Arguments: []ast.Node{
				&ast.StringNode{Value: spec.ID},
				&ast.StringNode{Value: spec.ReasonTrue},
				&ast.StringNode{Value: spec.ReasonFalse},
				*n,
			},
		}
		ast.Patch(n, wrapped)
	}

	walk(&root)
	return root
}
