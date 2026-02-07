package ruletrace

type Option interface{ apply(*Tracer) }

type optFunc func(*Tracer)

func (f optFunc) apply(t *Tracer) { f(t) }

// WithMode sets trace verbosity.
func WithMode(m TraceMode) Option { return optFunc(func(t *Tracer) { t.mode = m }) }

// WithShortCircuit enables/disables short-circuit simulation.
func WithShortCircuit(enabled bool) Option {
	return optFunc(func(t *Tracer) { t.shortCircuit = enabled })
}

// WithCond enables/disables semantic IDs + reasons support.
func WithCond(enabled bool) Option { return optFunc(func(t *Tracer) { t.enableCond = enabled }) }
