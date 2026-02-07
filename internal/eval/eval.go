package eval

import "github.com/expr-lang/expr"

// EvalString compiles (cached) and runs src against env.
// Returns (value, errorString).
func EvalString(src string, env map[string]interface{}, cache *Cache, opts ...expr.Option) (interface{}, string) {
	p, err := cache.getOrCompile(src, opts...)
	if err != nil {
		return nil, err.Error()
	}
	v, err := expr.Run(p, env)
	if err != nil {
		return nil, err.Error()
	}
	return v, ""
}
