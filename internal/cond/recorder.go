package cond

import "fmt"

// Recorded is the minimal info we store per semantic condition.
type Recorded struct {
	ID     string
	Value  bool
	Reason string
}

// Recorder captures Cond(...) outcomes during evaluation.
type Recorder struct {
	seen map[string]Recorded
}

func NewRecorder() *Recorder {
	return &Recorder{seen: map[string]Recorded{}}
}

func (r *Recorder) Seen() map[string]Recorded { return r.seen }

// Func returns a function compatible with expr.Function("Cond", ...).
// Signature: Cond(id, reasonTrue, reasonFalse, predicateBool) bool
func (r *Recorder) Func() func(params ...any) (any, error) {
	return func(params ...any) (any, error) {
		if len(params) != 4 {
			return nil, fmt.Errorf("Cond expects 4 args (id, reasonTrue, reasonFalse, predicate)")
		}
		id, _ := params[0].(string)
		rt, _ := params[1].(string)
		rf, _ := params[2].(string)
		pred, ok := params[3].(bool)
		if !ok {
			return nil, fmt.Errorf("Cond 4th arg must be bool")
		}

		reason := rf
		if pred {
			reason = rt
		}
		r.seen[id] = Recorded{ID: id, Value: pred, Reason: reason}
		return pred, nil
	}
}
