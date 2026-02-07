package util

import "strings"

// ExtractFirstStringArg best-effort parses Cond("id", ...) to "id".
// Limitation: assumes formatter prints Cond("...") with double quotes.
func ExtractFirstStringArg(exprStr string) string {
	p := strings.Index(exprStr, `Cond("`)
	if p < 0 {
		return ""
	}
	start := p + len(`Cond("`)
	end := strings.Index(exprStr[start:], `"`)
	if end < 0 {
		return ""
	}
	return exprStr[start : start+end]
}
