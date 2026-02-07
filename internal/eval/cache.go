package eval

import (
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Cache caches compiled programs for the lifetime of a single Trace call.
type Cache struct {
	mu   sync.Mutex
	prog map[string]*vm.Program
}

func NewCache() *Cache {
	return &Cache{prog: make(map[string]*vm.Program, 128)}
}

func (c *Cache) getOrCompile(src string, opts ...expr.Option) (*vm.Program, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p, ok := c.prog[src]; ok {
		return p, nil
	}
	p, err := expr.Compile(src, opts...)
	if err != nil {
		return nil, err
	}
	c.prog[src] = p
	return p, nil
}
