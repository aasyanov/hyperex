package hyperex

// Stats is a point-in-time snapshot of Engine counters.
// Counters are cumulative since Engine construction.
type Stats struct {
	// Compiles is the number of Compile calls.
	Compiles int64 `json:"compiles"`
	// Evals is the number of Program.Eval (and typed Eval*) invocations.
	Evals int64 `json:"evals"`
	// EvalErrors is the number of Eval invocations that returned a non-nil error.
	EvalErrors int64 `json:"eval_errors"`
}

// Stats returns a snapshot of lock-free counters.
// It is safe to call concurrently with Compile/Eval; fields are read independently
// and may reflect an operation still in progress.
func (e *Engine) Stats() Stats {
	return Stats{
		Compiles:   e.compiles.Load(),
		Evals:      e.evals.Load(),
		EvalErrors: e.evalErrors.Load(),
	}
}
