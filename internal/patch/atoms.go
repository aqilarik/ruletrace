package patch

import "github.com/expr-lang/expr/ast"

// IsShortCircuitOp are operators where right subtree may be skipped.
func IsShortCircuitOp(op string) bool {
	switch op {
	case "||", "or", "&&", "and", "??":
		return true
	default:
		return false
	}
}

// IsAtomNode defines what we treat as an “atomic predicate” for explainability.
// Adjust this list as your product needs evolve.
func IsAtomNode(n ast.Node) bool {
	bn, ok := n.(*ast.BinaryNode)
	if !ok {
		return false
	}
	switch bn.Operator {
	case "==", "!=", "<", "<=", ">", ">=",
		"in",
		"matches",
		"contains", "startsWith", "endsWith":
		return true
	default:
		return false
	}
}

func IsCondCall(n ast.Node) bool {
	call, ok := n.(*ast.CallNode)
	if !ok {
		return false
	}
	id, ok := call.Callee.(*ast.IdentifierNode)
	return ok && id.Value == "Cond"
}

// CollectAtoms returns leaf nodes used for atomic tracing.
func CollectAtoms(n ast.Node) []ast.Node {
	out := make([]ast.Node, 0, 8)
	collectAtoms(n, &out)
	return out
}

func collectAtoms(n ast.Node, out *[]ast.Node) {
	if n == nil {
		return
	}
	if ch, ok := n.(*ast.ChainNode); ok {
		collectAtoms(ch.Node, out)
		return
	}
	if IsAtomNode(n) || IsCondCall(n) {
		*out = append(*out, n)
		return
	}
	switch x := n.(type) {
	case *ast.BinaryNode:
		collectAtoms(x.Left, out)
		collectAtoms(x.Right, out)
	case *ast.UnaryNode:
		collectAtoms(x.Node, out)
	case *ast.ConditionalNode:
		collectAtoms(x.Cond, out)
		collectAtoms(x.Exp1, out)
		collectAtoms(x.Exp2, out)
	case *ast.CallNode:
		collectAtoms(x.Callee, out)
		for _, a := range x.Arguments {
			collectAtoms(a, out)
		}
	case *ast.BuiltinNode:
		if x.Map != nil {
			collectAtoms(x.Map, out)
		}
		for _, a := range x.Arguments {
			collectAtoms(a, out)
		}
	case *ast.MemberNode:
		collectAtoms(x.Node, out)
		collectAtoms(x.Property, out)
	case *ast.SliceNode:
		collectAtoms(x.Node, out)
		collectAtoms(x.From, out)
		collectAtoms(x.To, out)
	case *ast.ArrayNode:
		for _, el := range x.Nodes {
			collectAtoms(el, out)
		}
	case *ast.MapNode:
		for _, p := range x.Pairs {
			collectAtoms(p, out)
		}
	case *ast.PairNode:
		collectAtoms(x.Key, out)
		collectAtoms(x.Value, out)
	case *ast.SequenceNode:
		for _, sn := range x.Nodes {
			collectAtoms(sn, out)
		}
	case *ast.VariableDeclaratorNode:
		collectAtoms(x.Value, out)
		collectAtoms(x.Expr, out)
	case *ast.PredicateNode:
		collectAtoms(x.Node, out)
	default:
	}
}
