package format

import (
	"fmt"
	"sort"
	"strings"

	"github.com/expr-lang/expr/ast"
)

// Formatter produces a canonical-ish expression string for hashing/tracing.
// This is designed for explainability, not perfect round-trip printing.
type Formatter struct{}

func New() *Formatter { return &Formatter{} }

func (f *Formatter) Format(node ast.Node) string {
	switch n := node.(type) {
	case *ast.NilNode:
		return "nil"
	case *ast.IdentifierNode:
		return n.Value
	case *ast.IntegerNode:
		return fmt.Sprintf("%d", n.Value)
	case *ast.FloatNode:
		return fmt.Sprintf("%v", n.Value)
	case *ast.BoolNode:
		return fmt.Sprintf("%v", n.Value)
	case *ast.StringNode:
		return fmt.Sprintf(`"%s"`, escapeString(n.Value))

	case *ast.ConstantNode:
		switch v := n.Value.(type) {
		case string:
			return fmt.Sprintf(`"%s"`, escapeString(v))
		case map[string]struct{}:
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, fmt.Sprintf(`"%s"`, escapeString(k)))
			}
			return "[" + strings.Join(parts, ", ") + "]"
		case []byte:
			return formatBytesLiteral(v)
		default:
			return fmt.Sprintf("%v", n.Value)
		}

	case *ast.UnaryNode:
		if n.Operator == "not" {
			return "not " + f.Format(n.Node)
		}
		return n.Operator + f.Format(n.Node)

	case *ast.BinaryNode:
		return fmt.Sprintf("%s %s %s", f.Format(n.Left), n.Operator, f.Format(n.Right))

	case *ast.ConditionalNode:
		return fmt.Sprintf("%s ? %s : %s", f.Format(n.Cond), f.Format(n.Exp1), f.Format(n.Exp2))

	case *ast.SequenceNode:
		parts := make([]string, 0, len(n.Nodes))
		for _, sn := range n.Nodes {
			parts = append(parts, f.Format(sn))
		}
		return strings.Join(parts, "; ")

	case *ast.VariableDeclaratorNode:
		return fmt.Sprintf("let %s = %s; %s", n.Name, f.Format(n.Value), f.Format(n.Expr))

	case *ast.ArrayNode:
		var parts []string
		for _, el := range n.Nodes {
			parts = append(parts, f.Format(el))
		}
		return "[" + strings.Join(parts, ", ") + "]"

	case *ast.MapNode:
		var parts []string
		for _, p := range n.Pairs {
			pair := p.(*ast.PairNode)
			parts = append(parts, fmt.Sprintf("%s: %s", f.Format(pair.Key), f.Format(pair.Value)))
		}
		return "{" + strings.Join(parts, ", ") + "}"

	case *ast.PairNode:
		return fmt.Sprintf("%s: %s", f.Format(n.Key), f.Format(n.Value))

	case *ast.CallNode:
		var args []string
		for _, a := range n.Arguments {
			args = append(args, f.Format(a))
		}
		return fmt.Sprintf("%s(%s)", f.Format(n.Callee), strings.Join(args, ", "))

	case *ast.BuiltinNode:
		var args []string
		if n.Map != nil {
			args = append(args, f.Format(n.Map))
		}
		for _, a := range n.Arguments {
			args = append(args, f.Format(a))
		}
		return fmt.Sprintf("%s(%s)", n.Name, strings.Join(args, ", "))

	case *ast.PredicateNode:
		return fmt.Sprintf("{%s}", f.Format(n.Node))

	case *ast.PointerNode:
		if n.Name == "" {
			return "#"
		}
		return "#" + n.Name

	case *ast.MemberNode:
		return formatMember(f, n)

	case *ast.ChainNode:
		return f.Format(n.Node)

	case *ast.SliceNode:
		return formatSlice(f, n)

	default:
		return node.String()
	}
}
