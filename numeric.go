package hyperex

// NumericMode selects numeric arithmetic result policy for an Engine.
// Underlying int (not uint8): stabilize keeps the public kind; narrowing would
// shrink execVM (~144→~128) but change the exported type's Sizeof.
type NumericMode int

const (
	// NumericDefault preserves v1.0 arithmetic: int+int→int until overflow; 1/2→0 int.
	NumericDefault NumericMode = iota
	// NumericFloat64 promotes numeric binary +,-,*,/,%,** and unary - to float64 results.
	// Integer operands from the env are promoted at arithmetic entry. Under this mode, 1/2 → 0.5.
	// String concatenation via + and time/duration operators are unchanged.
	NumericFloat64
)

// NonFiniteMode selects whether NaN/Inf float results are allowed from Eval.
type NonFiniteMode int

const (
	// NonFiniteAllow permits NaN/Inf float results (v1.0 / IEEE-like default).
	NonFiniteAllow NonFiniteMode = iota
	// NonFiniteError rejects successful Eval returning NaN/Inf (and rejects such
	// values produced by arithmetic/builtins before they are returned).
	NonFiniteError
)

// ResultHint reports the compile-time As* expectation stored on a Program.
type ResultHint int

const (
	// ResultHintAny means no AsBool/AsInt/AsFloat/AsString was set at Compile.
	ResultHintAny ResultHint = iota
	// ResultHintBool means [AsBool] was set.
	ResultHintBool
	// ResultHintInt means [AsInt] was set.
	ResultHintInt
	// ResultHintFloat means [AsFloat] was set.
	ResultHintFloat
	// ResultHintString means [AsString] was set.
	ResultHintString
)

// String returns a short name for the hint (for diagnostics).
func (h ResultHint) String() string {
	switch h {
	case ResultHintBool:
		return "bool"
	case ResultHintInt:
		return "int"
	case ResultHintFloat:
		return "float"
	case ResultHintString:
		return "string"
	default:
		return "any"
	}
}

// WithNumericMode sets the numeric arithmetic policy for the Engine.
//
// Default: [NumericDefault] (v1.0: int+int→int until overflow; 1/2→0).
//
// Under [NumericFloat64], numeric +,-,*,/,%,** and unary - yield float64
// (so 1/2 → 0.5). String + concat and time/duration operators are unchanged.
func WithNumericMode(mode NumericMode) Option {
	return func(c *engineConfig) {
		c.numericMode = mode
	}
}

// WithNonFinite sets whether NaN/Inf float results are allowed from successful Eval.
//
// Default: [NonFiniteAllow].
//
// Under [NonFiniteError], Eval / EvalValue return an error matching [ErrNonFinite]
// instead of a NaN or Inf float.
func WithNonFinite(mode NonFiniteMode) Option {
	return func(c *engineConfig) {
		c.nonFiniteMode = mode
	}
}
