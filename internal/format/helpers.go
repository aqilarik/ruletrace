package format

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr/ast"
)

func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

func formatBytesLiteral(b []byte) string {
	printable := true
	for _, x := range b {
		if x < 0x20 || x > 0x7e || x == '"' || x == '\\' {
			printable = false
			break
		}
	}
	if printable {
		return fmt.Sprintf(`b"%s"`, string(b))
	}
	var sb strings.Builder
	sb.WriteString(`b"`)
	for _, x := range b {
		sb.WriteString(fmt.Sprintf(`\x%02x`, x))
	}
	sb.WriteString(`"`)
	return sb.String()
}

func formatMember(f *Formatter, n *ast.MemberNode) string {
	base := f.Format(n.Node)

	if sn, ok := n.Property.(*ast.StringNode); ok && !n.Method {
		if n.Optional {
			return base + "?." + sn.Value
		}
		return base + "." + sn.Value
	}

	prop := f.Format(n.Property)
	if n.Optional {
		return fmt.Sprintf("%s?.[%s]", base, prop)
	}
	return fmt.Sprintf("%s[%s]", base, prop)
}

func formatSlice(f *Formatter, n *ast.SliceNode) string {
	base := f.Format(n.Node)
	from := ""
	to := ""
	if n.From != nil {
		from = f.Format(n.From)
	}
	if n.To != nil {
		to = f.Format(n.To)
	}
	return fmt.Sprintf("%s[%s:%s]", base, from, to)
}
