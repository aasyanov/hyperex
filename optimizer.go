package hyperex

import (
	"math"
	"sync/atomic"
)

// optimizer applies various optimizations to an AST.
type optimizer struct {
	// Configuration flags
	constantFolding   bool
	strengthReduction bool
	deadCodeElim      bool
	algebraicSimplify bool

	// Numeric arithmetic policy (must match Engine / VM for fold results)
	numericMode NumericMode

	// Skip operators that are overloaded and shouldn't be folded
	skipOperators map[string]bool

	// Statistics (updated atomically so concurrent Optimize is safe)
	foldedCount  int64
	reducedCount int64
}

// newOptimizer creates a new optimizer with default settings.
func newOptimizer() *optimizer {
	return &optimizer{
		constantFolding:   true,
		strengthReduction: true,
		deadCodeElim:      true,
		algebraicSimplify: true,
		numericMode:       NumericDefault,
		skipOperators:     make(map[string]bool),
	}
}

// defaultOptimizer returns the default optimizer configuration.
// Alias for newOptimizer.
func defaultOptimizer() *optimizer {
	return newOptimizer()
}

// withConstantFolding enables or disables constant folding.
func (o *optimizer) withConstantFolding(enabled bool) *optimizer {
	o.constantFolding = enabled
	return o
}

// withStrengthReduction enables or disables strength reduction.
func (o *optimizer) withStrengthReduction(enabled bool) *optimizer {
	o.strengthReduction = enabled
	return o
}

// withNumericMode sets the numeric arithmetic policy for constant folding.
func (o *optimizer) withNumericMode(mode NumericMode) *optimizer {
	o.numericMode = mode
	return o
}

// withSkipOperators marks multiple operators to skip during optimization.
func (o *optimizer) withSkipOperators(ops map[string]string) *optimizer {
	if o.skipOperators == nil {
		o.skipOperators = make(map[string]bool)
	}
	for op := range ops {
		o.skipOperators[op] = true
	}
	return o
}

// Optimize applies all enabled optimizations to the AST.
// Concurrent calls on a shared optimizer are safe: each call mutates a
// value copy's counters, then atomically accumulates into o.
func (o *optimizer) optimize(node Node) Node {
	if node == nil {
		return nil
	}

	// Per-call working copy so foldedCount/reducedCount mutations do not race.
	// skipOperators is a shared map (read-only after Engine.New) — OK.
	local := *o
	local.foldedCount = 0
	local.reducedCount = 0

	// Apply optimizations in multiple passes until no more changes
	maxPasses := 10 // Limit passes to prevent infinite loops
	for pass := 0; pass < maxPasses; pass++ {
		prevFolded := local.foldedCount
		prevReduced := local.reducedCount

		if local.constantFolding {
			node = local.foldConstants(node)
		}
		if local.strengthReduction {
			node = local.reduceStrength(node)
		}
		if local.algebraicSimplify {
			node = local.simplifyAlgebra(node)
		}
		if local.deadCodeElim {
			node = local.eliminateDeadCode(node)
		}

		// No changes this pass, we're done
		if local.foldedCount == prevFolded && local.reducedCount == prevReduced {
			break
		}
	}

	atomic.AddInt64(&o.foldedCount, local.foldedCount)
	atomic.AddInt64(&o.reducedCount, local.reducedCount)

	return node
}

// foldConstants evaluates constant expressions at compile time.
func (o *optimizer) foldConstants(node Node) Node {
	switch n := node.(type) {
	case *binaryExpr:
		// Recursively fold children first
		left := o.foldConstants(n.Left)
		right := o.foldConstants(n.Right)

		// Skip overloaded operators
		if o.skipOperators[n.Op] {
			return &binaryExpr{Op: n.Op, Left: left, Right: right}
		}

		// Check if both operands are constants
		leftLit, leftIsLit := o.asLiteral(left)
		rightLit, rightIsLit := o.asLiteral(right)

		if leftIsLit && rightIsLit {
			result, ok := o.evalBinaryOp(n.Op, leftLit, rightLit)
			if ok {
				o.foldedCount++
				return result
			}
		}

		return &binaryExpr{Op: n.Op, Left: left, Right: right}

	case *unaryExpr:
		operand := o.foldConstants(n.Operand)

		// Check if operand is a constant
		if lit, isLit := o.asLiteral(operand); isLit {
			result, ok := o.evalUnaryOp(n.Op, lit)
			if ok {
				o.foldedCount++
				return result
			}
		}

		return &unaryExpr{Op: n.Op, Operand: operand}

	case *ternaryExpr:
		cond := o.foldConstants(n.Cond)
		then := o.foldConstants(n.Then)
		elseExpr := o.foldConstants(n.Else)

		// If condition is constant, return appropriate branch
		if boolLit, ok := cond.(*boolLit); ok {
			o.foldedCount++
			if boolLit.Value {
				return then
			}
			return elseExpr
		}

		return &ternaryExpr{Cond: cond, Then: then, Else: elseExpr}

	case *coalesceExpr:
		left := o.foldConstants(n.Left)
		right := o.foldConstants(n.Right)

		// If left is constant and not nil, return left
		if lit, isLit := o.asLiteral(left); isLit {
			if _, isNil := lit.(*nilLit); !isNil {
				o.foldedCount++
				return left
			}
			// Left is nil, return right
			o.foldedCount++
			return right
		}

		return &coalesceExpr{Left: left, Right: right}

	case *callExpr:
		// Fold arguments
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = o.foldConstants(arg)
		}
		return &callExpr{Func: o.foldConstants(n.Func), Args: args}

	case *arrayLit:
		elements := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = o.foldConstants(elem)
		}
		return &arrayLit{Elements: elements}

	case *mapLit:
		keys := make([]Node, len(n.Keys))
		values := make([]Node, len(n.Values))
		for i := range n.Keys {
			keys[i] = o.foldConstants(n.Keys[i])
			values[i] = o.foldConstants(n.Values[i])
		}
		return &mapLit{Keys: keys, Values: values}

	case *memberExpr:
		return &memberExpr{
			Object:   o.foldConstants(n.Object),
			Property: o.foldConstants(n.Property),
			Computed: n.Computed,
			Optional: n.Optional,
		}

	case *indexExpr:
		return &indexExpr{
			Object:   o.foldConstants(n.Object),
			Index:    o.foldConstants(n.Index),
			Optional: n.Optional,
		}

	case *sliceExpr:
		return &sliceExpr{
			Object: o.foldConstants(n.Object),
			Low:    o.foldConstantsOrNil(n.Low),
			High:   o.foldConstantsOrNil(n.High),
			Step:   o.foldConstantsOrNil(n.Step),
		}

	case *rangeExpr:
		return &rangeExpr{
			Start: o.foldConstants(n.Start),
			End:   o.foldConstants(n.End),
		}

	case *pipeExpr:
		return &pipeExpr{
			Left:  o.foldConstants(n.Left),
			Right: o.foldConstants(n.Right),
		}

	case *lambdaExpr:
		// Don't fold inside lambdas - they have their own scope
		return n

	case *letExpr:
		return &letExpr{
			Name:  n.Name,
			Value: o.foldConstants(n.Value),
			Body:  o.foldConstants(n.Body),
		}

	case *spreadExpr:
		return &spreadExpr{Expr: o.foldConstants(n.Expr)}

	case *destructureExpr:
		// Don't fold inside destructure - just optimize value and body
		return &destructureExpr{
			Pattern: n.Pattern,
			IsArray: n.IsArray,
			Value:   o.foldConstants(n.Value),
			Body:    o.foldConstants(n.Body),
		}

	default:
		return node
	}
}

func (o *optimizer) foldConstantsOrNil(node Node) Node {
	if node == nil {
		return nil
	}
	return o.foldConstants(node)
}

// asLiteral checks if a node is a literal value.
func (o *optimizer) asLiteral(node Node) (Node, bool) {
	switch node.(type) {
	case *intLit, *floatLit, *stringLit, *boolLit, *nilLit:
		return node, true
	default:
		return nil, false
	}
}

// evalBinaryOp evaluates a binary operation on two literal values.
func (o *optimizer) evalBinaryOp(op string, left, right Node) (Node, bool) {
	switch op {
	case "+":
		return o.evalAdd(left, right)
	case "-":
		return o.evalSub(left, right)
	case "*":
		return o.evalMul(left, right)
	case "/":
		return o.evalDiv(left, right)
	case "%":
		return o.evalMod(left, right)
	case "**":
		return o.evalPow(left, right)
	case "==":
		return o.evalEq(left, right)
	case "!=":
		return o.evalNe(left, right)
	case "<":
		return o.evalLt(left, right)
	case "<=":
		return o.evalLe(left, right)
	case ">":
		return o.evalGt(left, right)
	case ">=":
		return o.evalGe(left, right)
	case "&&":
		return o.evalAnd(left, right)
	case "||":
		return o.evalOr(left, right)
	case "&":
		return o.evalBitAnd(left, right)
	case "|":
		return o.evalBitOr(left, right)
	case "^":
		return o.evalBitXor(left, right)
	case "<<":
		return o.evalShl(left, right)
	case ">>":
		return o.evalShr(left, right)
	default:
		return nil, false
	}
}

// evalUnaryOp evaluates a unary operation on a literal value.
func (o *optimizer) evalUnaryOp(op string, operand Node) (Node, bool) {
	switch op {
	case "-":
		switch n := operand.(type) {
		case *intLit:
			if o.numericMode == NumericFloat64 {
				return &floatLit{Value: -float64(n.Value)}, true
			}
			// Handle MinInt64: -MinInt64 overflows in two's complement
			if n.Value == math.MinInt64 {
				return &floatLit{Value: -float64(n.Value)}, true
			}
			return &intLit{Value: -n.Value}, true
		case *floatLit:
			return &floatLit{Value: -n.Value}, true
		}
	case "!":
		switch n := operand.(type) {
		case *boolLit:
			return &boolLit{Value: !n.Value}, true
		case *intLit:
			return &boolLit{Value: n.Value == 0}, true
		case *floatLit:
			return &boolLit{Value: n.Value == 0}, true
		case *stringLit:
			return &boolLit{Value: n.Value == ""}, true
		case *nilLit:
			return &boolLit{Value: true}, true
		}
	case "~":
		if n, ok := operand.(*intLit); ok {
			return &intLit{Value: ^n.Value}, true
		}
	}
	return nil, false
}

// numericLitFloat extracts a float64 from an int or float literal node.
func numericLitFloat(n Node) (float64, bool) {
	switch v := n.(type) {
	case *intLit:
		return float64(v.Value), true
	case *floatLit:
		return v.Value, true
	default:
		return 0, false
	}
}

// Binary operation evaluators
func (o *optimizer) evalAdd(left, right Node) (Node, bool) {
	// String concat is unchanged under NumericFloat64.
	if l, ok := left.(*stringLit); ok {
		if r, ok := right.(*stringLit); ok {
			return &stringLit{Value: l.Value + r.Value}, true
		}
		return nil, false
	}
	if o.numericMode == NumericFloat64 {
		lf, lok := numericLitFloat(left)
		rf, rok := numericLitFloat(right)
		if lok && rok {
			return &floatLit{Value: lf + rf}, true
		}
		return nil, false
	}
	switch l := left.(type) {
	case *intLit:
		if r, ok := right.(*intLit); ok {
			// Check for overflow
			result := l.Value + r.Value
			if (r.Value > 0 && result < l.Value) || (r.Value < 0 && result > l.Value) {
				// Overflow - convert to float
				return &floatLit{Value: float64(l.Value) + float64(r.Value)}, true
			}
			return &intLit{Value: result}, true
		}
		if r, ok := right.(*floatLit); ok {
			return &floatLit{Value: float64(l.Value) + r.Value}, true
		}
	case *floatLit:
		if r, ok := right.(*intLit); ok {
			return &floatLit{Value: l.Value + float64(r.Value)}, true
		}
		if r, ok := right.(*floatLit); ok {
			return &floatLit{Value: l.Value + r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalSub(left, right Node) (Node, bool) {
	if o.numericMode == NumericFloat64 {
		lf, lok := numericLitFloat(left)
		rf, rok := numericLitFloat(right)
		if lok && rok {
			return &floatLit{Value: lf - rf}, true
		}
		return nil, false
	}
	switch l := left.(type) {
	case *intLit:
		if r, ok := right.(*intLit); ok {
			// Check for overflow
			result := l.Value - r.Value
			if (r.Value < 0 && result < l.Value) || (r.Value > 0 && result > l.Value) {
				// Overflow - convert to float
				return &floatLit{Value: float64(l.Value) - float64(r.Value)}, true
			}
			return &intLit{Value: result}, true
		}
		if r, ok := right.(*floatLit); ok {
			return &floatLit{Value: float64(l.Value) - r.Value}, true
		}
	case *floatLit:
		if r, ok := right.(*intLit); ok {
			return &floatLit{Value: l.Value - float64(r.Value)}, true
		}
		if r, ok := right.(*floatLit); ok {
			return &floatLit{Value: l.Value - r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalMul(left, right Node) (Node, bool) {
	if o.numericMode == NumericFloat64 {
		lf, lok := numericLitFloat(left)
		rf, rok := numericLitFloat(right)
		if lok && rok {
			return &floatLit{Value: lf * rf}, true
		}
		return nil, false
	}
	switch l := left.(type) {
	case *intLit:
		if r, ok := right.(*intLit); ok {
			// Check for overflow using mulOverflows helper
			if mulOverflows(l.Value, r.Value) {
				return &floatLit{Value: float64(l.Value) * float64(r.Value)}, true
			}
			return &intLit{Value: l.Value * r.Value}, true
		}
		if r, ok := right.(*floatLit); ok {
			return &floatLit{Value: float64(l.Value) * r.Value}, true
		}
	case *floatLit:
		if r, ok := right.(*intLit); ok {
			return &floatLit{Value: l.Value * float64(r.Value)}, true
		}
		if r, ok := right.(*floatLit); ok {
			return &floatLit{Value: l.Value * r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalDiv(left, right Node) (Node, bool) {
	if o.numericMode == NumericFloat64 {
		lf, lok := numericLitFloat(left)
		rf, rok := numericLitFloat(right)
		if !lok || !rok || rf == 0 {
			return nil, false // Don't fold division by zero
		}
		return &floatLit{Value: lf / rf}, true
	}
	switch l := left.(type) {
	case *intLit:
		if r, ok := right.(*intLit); ok {
			if r.Value == 0 {
				return nil, false // Don't fold division by zero
			}
			return &intLit{Value: l.Value / r.Value}, true
		}
		if r, ok := right.(*floatLit); ok {
			if r.Value == 0 {
				return nil, false
			}
			return &floatLit{Value: float64(l.Value) / r.Value}, true
		}
	case *floatLit:
		if r, ok := right.(*intLit); ok {
			if r.Value == 0 {
				return nil, false
			}
			return &floatLit{Value: l.Value / float64(r.Value)}, true
		}
		if r, ok := right.(*floatLit); ok {
			if r.Value == 0 {
				return nil, false
			}
			return &floatLit{Value: l.Value / r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalMod(left, right Node) (Node, bool) {
	if o.numericMode == NumericFloat64 {
		lf, lok := numericLitFloat(left)
		rf, rok := numericLitFloat(right)
		if !lok || !rok || rf == 0 {
			return nil, false
		}
		return &floatLit{Value: math.Mod(lf, rf)}, true
	}
	switch l := left.(type) {
	case *intLit:
		if r, ok := right.(*intLit); ok {
			if r.Value == 0 {
				return nil, false
			}
			return &intLit{Value: l.Value % r.Value}, true
		}
		if r, ok := right.(*floatLit); ok {
			if r.Value == 0 {
				return nil, false
			}
			return &floatLit{Value: math.Mod(float64(l.Value), r.Value)}, true
		}
	case *floatLit:
		if r, ok := right.(*intLit); ok {
			if r.Value == 0 {
				return nil, false
			}
			return &floatLit{Value: math.Mod(l.Value, float64(r.Value))}, true
		}
		if r, ok := right.(*floatLit); ok {
			if r.Value == 0 {
				return nil, false
			}
			return &floatLit{Value: math.Mod(l.Value, r.Value)}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalPow(left, right Node) (Node, bool) {
	if o.numericMode == NumericFloat64 {
		lf, lok := numericLitFloat(left)
		rf, rok := numericLitFloat(right)
		if lok && rok {
			return &floatLit{Value: math.Pow(lf, rf)}, true
		}
		return nil, false
	}
	// For integer operands with non-negative exponent, try to return intLit
	if li, ok := left.(*intLit); ok {
		if ri, ok := right.(*intLit); ok {
			if ri.Value >= 0 {
				base := li.Value
				exp := ri.Value
				// Check if result will fit in int64
				if canIntPowWithoutOverflow(base, exp) {
					result := intPow(base, exp)
					return &intLit{Value: result}, true
				}
			}
			// Integer with integer exponent but would overflow - return float
			return &floatLit{Value: math.Pow(float64(li.Value), float64(ri.Value))}, true
		}
		// Integer base with float exponent
		if rf, ok := right.(*floatLit); ok {
			return &floatLit{Value: math.Pow(float64(li.Value), rf.Value)}, true
		}
	}

	// Float base
	if lf, ok := left.(*floatLit); ok {
		var exp float64
		switch r := right.(type) {
		case *intLit:
			exp = float64(r.Value)
		case *floatLit:
			exp = r.Value
		default:
			return nil, false
		}
		return &floatLit{Value: math.Pow(lf.Value, exp)}, true
	}

	return nil, false
}

// canIntPowWithoutOverflow checks if base^exp can be computed without overflow for int64.
// Defined in optimizer for use by constant folding; also defined in vm.go for runtime use.
func canIntPowWithoutOverflow(base, exp int64) bool {
	if exp < 0 {
		return false
	}
	if exp == 0 || base == 0 || base == 1 {
		return true
	}
	if base == -1 {
		return true
	}
	// Deterministic overflow check using the same repeated-squaring shape as intPow().
	// This avoids false positives/negatives near boundaries due to float64 rounding.
	result := int64(1)
	b := base
	e := exp
	for e > 0 {
		if e&1 == 1 {
			if mulOverflows(result, b) {
				return false
			}
			result *= b
		}
		e >>= 1
		if e == 0 {
			break
		}
		if mulOverflows(b, b) {
			return false
		}
		b *= b
	}
	return true
}

// intPow computes base^exp for integers using repeated squaring.
func intPow(base, exp int64) int64 {
	if exp == 0 {
		return 1
	}
	result := int64(1)
	for exp > 0 {
		if exp&1 == 1 {
			result *= base
		}
		base *= base
		exp >>= 1
	}
	return result
}

func (o *optimizer) evalEq(left, right Node) (Node, bool) {
	switch l := left.(type) {
	case *intLit:
		switch r := right.(type) {
		case *intLit:
			return &boolLit{Value: l.Value == r.Value}, true
		case *floatLit:
			return &boolLit{Value: float64(l.Value) == r.Value}, true
		}
	case *floatLit:
		switch r := right.(type) {
		case *intLit:
			return &boolLit{Value: l.Value == float64(r.Value)}, true
		case *floatLit:
			return &boolLit{Value: l.Value == r.Value}, true
		}
	case *stringLit:
		if r, ok := right.(*stringLit); ok {
			return &boolLit{Value: l.Value == r.Value}, true
		}
	case *boolLit:
		if r, ok := right.(*boolLit); ok {
			return &boolLit{Value: l.Value == r.Value}, true
		}
	case *nilLit:
		_, isNil := right.(*nilLit)
		return &boolLit{Value: isNil}, true
	}
	return nil, false
}

func (o *optimizer) evalNe(left, right Node) (Node, bool) {
	result, ok := o.evalEq(left, right)
	if ok {
		return &boolLit{Value: !result.(*boolLit).Value}, true
	}
	return nil, false
}

func (o *optimizer) evalLt(left, right Node) (Node, bool) {
	switch l := left.(type) {
	case *intLit:
		switch r := right.(type) {
		case *intLit:
			return &boolLit{Value: l.Value < r.Value}, true
		case *floatLit:
			return &boolLit{Value: float64(l.Value) < r.Value}, true
		}
	case *floatLit:
		switch r := right.(type) {
		case *intLit:
			return &boolLit{Value: l.Value < float64(r.Value)}, true
		case *floatLit:
			return &boolLit{Value: l.Value < r.Value}, true
		}
	case *stringLit:
		if r, ok := right.(*stringLit); ok {
			return &boolLit{Value: l.Value < r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalLe(left, right Node) (Node, bool) {
	switch l := left.(type) {
	case *intLit:
		switch r := right.(type) {
		case *intLit:
			return &boolLit{Value: l.Value <= r.Value}, true
		case *floatLit:
			return &boolLit{Value: float64(l.Value) <= r.Value}, true
		}
	case *floatLit:
		switch r := right.(type) {
		case *intLit:
			return &boolLit{Value: l.Value <= float64(r.Value)}, true
		case *floatLit:
			return &boolLit{Value: l.Value <= r.Value}, true
		}
	case *stringLit:
		if r, ok := right.(*stringLit); ok {
			return &boolLit{Value: l.Value <= r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalGt(left, right Node) (Node, bool) {
	// a > b is equivalent to b < a
	return o.evalLt(right, left)
}

func (o *optimizer) evalGe(left, right Node) (Node, bool) {
	// a >= b is equivalent to b <= a
	return o.evalLe(right, left)
}

func (o *optimizer) evalAnd(left, right Node) (Node, bool) {
	// Short-circuit: if left is false, return false
	if l, ok := left.(*boolLit); ok {
		if !l.Value {
			return &boolLit{Value: false}, true
		}
		// Left is true, result depends on right
		if r, ok := right.(*boolLit); ok {
			return &boolLit{Value: r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalOr(left, right Node) (Node, bool) {
	// Short-circuit: if left is true, return true
	if l, ok := left.(*boolLit); ok {
		if l.Value {
			return &boolLit{Value: true}, true
		}
		// Left is false, result depends on right
		if r, ok := right.(*boolLit); ok {
			return &boolLit{Value: r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalBitAnd(left, right Node) (Node, bool) {
	if l, ok := left.(*intLit); ok {
		if r, ok := right.(*intLit); ok {
			return &intLit{Value: l.Value & r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalBitOr(left, right Node) (Node, bool) {
	if l, ok := left.(*intLit); ok {
		if r, ok := right.(*intLit); ok {
			return &intLit{Value: l.Value | r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalBitXor(left, right Node) (Node, bool) {
	if l, ok := left.(*intLit); ok {
		if r, ok := right.(*intLit); ok {
			return &intLit{Value: l.Value ^ r.Value}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalShl(left, right Node) (Node, bool) {
	if l, ok := left.(*intLit); ok {
		if r, ok := right.(*intLit); ok {
			if r.Value < 0 || r.Value > 63 {
				return nil, false // Invalid shift
			}
			return &intLit{Value: l.Value << uint(r.Value)}, true
		}
	}
	return nil, false
}

func (o *optimizer) evalShr(left, right Node) (Node, bool) {
	if l, ok := left.(*intLit); ok {
		if r, ok := right.(*intLit); ok {
			if r.Value < 0 || r.Value > 63 {
				return nil, false // Invalid shift
			}
			return &intLit{Value: l.Value >> uint(r.Value)}, true
		}
	}
	return nil, false
}

// reduceStrength replaces expensive operations with cheaper equivalents.
func (o *optimizer) reduceStrength(node Node) Node {
	switch n := node.(type) {
	case *binaryExpr:
		left := o.reduceStrength(n.Left)
		right := o.reduceStrength(n.Right)

		// Skip overloaded operators
		if o.skipOperators[n.Op] {
			return &binaryExpr{Op: n.Op, Left: left, Right: right}
		}

		switch n.Op {
		case "**":
			// x ** 2 -> x * x only for numeric x
			if r, ok := right.(*intLit); ok && r.Value == 2 && o.isSimple(left) && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return &binaryExpr{Op: "*", Left: left, Right: left}
			}
			// x ** 1 -> x only for numeric x (pow coerces via Float)
			if r, ok := right.(*intLit); ok && r.Value == 1 && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return left
			}
			// x ** 0 -> 1 only for numeric x
			if r, ok := right.(*intLit); ok && r.Value == 0 && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return o.numericOneLit()
			}
		case "*":
			// x * 2 -> x + x only for numeric x (string repeat / array would diverge)
			if r, ok := right.(*intLit); ok && r.Value == 2 && o.isSimple(left) && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return &binaryExpr{Op: "+", Left: left, Right: left}
			}
			// 2 * x -> x + x
			if l, ok := left.(*intLit); ok && l.Value == 2 && o.isSimple(right) && o.isDefinitelyNumeric(right) {
				o.reducedCount++
				return &binaryExpr{Op: "+", Left: right, Right: right}
			}
		case "/":
			// x / 2 -> x * 0.5 (float path, or NumericFloat64 where int/2 is 0.5)
			if r, ok := right.(*intLit); ok && r.Value == 2 {
				if o.numericMode == NumericFloat64 && o.isDefinitelyNumeric(left) {
					o.reducedCount++
					return &binaryExpr{Op: "*", Left: left, Right: &floatLit{Value: 0.5}}
				}
				if l, ok := left.(*floatLit); ok {
					o.reducedCount++
					return &binaryExpr{Op: "*", Left: l, Right: &floatLit{Value: 0.5}}
				}
			}
		}

		return &binaryExpr{Op: n.Op, Left: left, Right: right}

	case *unaryExpr:
		return &unaryExpr{Op: n.Op, Operand: o.reduceStrength(n.Operand)}

	case *ternaryExpr:
		return &ternaryExpr{
			Cond: o.reduceStrength(n.Cond),
			Then: o.reduceStrength(n.Then),
			Else: o.reduceStrength(n.Else),
		}

	case *callExpr:
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = o.reduceStrength(arg)
		}
		return &callExpr{Func: o.reduceStrength(n.Func), Args: args}

	case *arrayLit:
		elements := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = o.reduceStrength(elem)
		}
		return &arrayLit{Elements: elements}

	case *mapLit:
		keys := make([]Node, len(n.Keys))
		values := make([]Node, len(n.Values))
		for i := range n.Keys {
			keys[i] = o.reduceStrength(n.Keys[i])
			values[i] = o.reduceStrength(n.Values[i])
		}
		return &mapLit{Keys: keys, Values: values}

	case *memberExpr:
		return &memberExpr{
			Object:   o.reduceStrength(n.Object),
			Property: o.reduceStrength(n.Property),
			Computed: n.Computed,
			Optional: n.Optional,
		}

	case *indexExpr:
		return &indexExpr{
			Object:   o.reduceStrength(n.Object),
			Index:    o.reduceStrength(n.Index),
			Optional: n.Optional,
		}

	case *coalesceExpr:
		return &coalesceExpr{
			Left:  o.reduceStrength(n.Left),
			Right: o.reduceStrength(n.Right),
		}

	case *letExpr:
		return &letExpr{
			Name:  n.Name,
			Value: o.reduceStrength(n.Value),
			Body:  o.reduceStrength(n.Body),
		}

	case *spreadExpr:
		return &spreadExpr{Expr: o.reduceStrength(n.Expr)}

	case *destructureExpr:
		return &destructureExpr{
			Pattern: n.Pattern,
			IsArray: n.IsArray,
			Value:   o.reduceStrength(n.Value),
			Body:    o.reduceStrength(n.Body),
		}

	default:
		return node
	}
}

// isSimple checks if an expression is simple enough to duplicate.
func (o *optimizer) isSimple(node Node) bool {
	switch node.(type) {
	case *ident, *intLit, *floatLit, *stringLit, *boolLit, *nilLit:
		return true
	default:
		return false
	}
}

// simplifyAlgebra applies algebraic simplifications.
func (o *optimizer) simplifyAlgebra(node Node) Node {
	switch n := node.(type) {
	case *binaryExpr:
		left := o.simplifyAlgebra(n.Left)
		right := o.simplifyAlgebra(n.Right)

		// Skip overloaded operators
		if o.skipOperators[n.Op] {
			return &binaryExpr{Op: n.Op, Left: left, Right: right}
		}

		switch n.Op {
		case "+":
			// + is also string/array concat — only fold additive identity on
			// operands that are definitely numeric.
			// x + 0 -> x
			if o.isZero(right) && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return left
			}
			// 0 + x -> x
			if o.isZero(left) && o.isDefinitelyNumeric(right) {
				o.reducedCount++
				return right
			}
		case "-":
			// x - 0 -> x
			if o.isZero(right) && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return left
			}
			// x - x -> 0 (only for simple expressions)
			if o.nodesEqual(left, right) && o.isSideEffectFree(left) && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return o.numericZeroLit()
			}
		case "*":
			// x * 1 -> x (safe for string repeat and numbers)
			if r, ok := right.(*intLit); ok && r.Value == 1 {
				o.reducedCount++
				return left
			}
			if r, ok := right.(*floatLit); ok && r.Value == 1 {
				o.reducedCount++
				return left
			}
			// 1 * x -> x only when x is numeric (1*"ab" is not string-repeat)
			if o.isOne(left) && o.isDefinitelyNumeric(right) {
				o.reducedCount++
				return right
			}
			// x * 0 -> 0 only for numeric x ("ab"*0 yields "")
			if o.isZero(right) && o.isSideEffectFree(left) && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return o.numericZeroLit()
			}
			// 0 * x -> 0 only for numeric x (0*"ab" is a runtime error)
			if o.isZero(left) && o.isSideEffectFree(right) && o.isDefinitelyNumeric(right) {
				o.reducedCount++
				return o.numericZeroLit()
			}
		case "/":
			// x / 1 -> x
			if r, ok := right.(*intLit); ok && r.Value == 1 && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return left
			}
			if r, ok := right.(*floatLit); ok && r.Value == 1 && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return left
			}
			// x / x -> 1 (only for simple, non-zero expressions)
			if o.nodesEqual(left, right) && o.isSideEffectFree(left) && !o.isZero(left) && o.isDefinitelyNumeric(left) {
				o.reducedCount++
				return o.numericOneLit()
			}
		case "&&":
			// Semicolon-style short-circuit: return an operand (not a coerced bool).
			// true && x -> x
			if l, ok := left.(*boolLit); ok && l.Value {
				o.reducedCount++
				return right
			}
			// false && x -> false (RHS unevaluated)
			if l, ok := left.(*boolLit); ok && !l.Value {
				o.reducedCount++
				return &boolLit{Value: false}
			}
			// NOTE: do NOT fold `x && true -> x` — when x is truthy the result is
			// `true`, not x (e.g. `1 && true` → true).
		case "||":
			// false || x -> x
			if l, ok := left.(*boolLit); ok && !l.Value {
				o.reducedCount++
				return right
			}
			// true || x -> true (RHS unevaluated)
			if l, ok := left.(*boolLit); ok && l.Value {
				o.reducedCount++
				return &boolLit{Value: true}
			}
			// NOTE: do NOT fold `x || false -> x` — when x is falsy the result is
			// `false`, not x (e.g. `0 || false` → false).
		}

		if left != n.Left || right != n.Right {
			return &binaryExpr{Op: n.Op, Left: left, Right: right}
		}
		return n

	case *unaryExpr:
		operand := o.simplifyAlgebra(n.Operand)

		// !!x -> x (when x is bool)
		if n.Op == "!" {
			if inner, ok := operand.(*unaryExpr); ok && inner.Op == "!" {
				// Only simplify if inner operand is known bool
				if _, isBool := inner.Operand.(*boolLit); isBool {
					o.reducedCount++
					return inner.Operand
				}
			}
		}

		// --x -> x only for numeric x
		if n.Op == "-" {
			if inner, ok := operand.(*unaryExpr); ok && inner.Op == "-" && o.isDefinitelyNumeric(inner.Operand) {
				o.reducedCount++
				return inner.Operand
			}
		}

		// ~~x -> x only for integer x. Bitwise ~ always goes through Int(), so
		// ~~0.1 → 0 at runtime — cancelling would wrongly yield 0.1.
		if n.Op == "~" {
			if inner, ok := operand.(*unaryExpr); ok && inner.Op == "~" && o.isDefinitelyInt(inner.Operand) {
				o.reducedCount++
				return inner.Operand
			}
		}

		return &unaryExpr{Op: n.Op, Operand: operand}

	case *ternaryExpr:
		return &ternaryExpr{
			Cond: o.simplifyAlgebra(n.Cond),
			Then: o.simplifyAlgebra(n.Then),
			Else: o.simplifyAlgebra(n.Else),
		}

	case *callExpr:
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = o.simplifyAlgebra(arg)
		}
		return &callExpr{Func: o.simplifyAlgebra(n.Func), Args: args}

	case *arrayLit:
		elements := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = o.simplifyAlgebra(elem)
		}
		return &arrayLit{Elements: elements}

	case *mapLit:
		keys := make([]Node, len(n.Keys))
		values := make([]Node, len(n.Values))
		for i := range n.Keys {
			keys[i] = o.simplifyAlgebra(n.Keys[i])
			values[i] = o.simplifyAlgebra(n.Values[i])
		}
		return &mapLit{Keys: keys, Values: values}

	case *coalesceExpr:
		return &coalesceExpr{
			Left:  o.simplifyAlgebra(n.Left),
			Right: o.simplifyAlgebra(n.Right),
		}

	case *letExpr:
		return &letExpr{
			Name:  n.Name,
			Value: o.simplifyAlgebra(n.Value),
			Body:  o.simplifyAlgebra(n.Body),
		}

	case *spreadExpr:
		return &spreadExpr{Expr: o.simplifyAlgebra(n.Expr)}

	case *destructureExpr:
		return &destructureExpr{
			Pattern: n.Pattern,
			IsArray: n.IsArray,
			Value:   o.simplifyAlgebra(n.Value),
			Body:    o.simplifyAlgebra(n.Body),
		}

	default:
		return node
	}
}

// isZero checks if a node represents zero.
func (o *optimizer) isZero(node Node) bool {
	if i, ok := node.(*intLit); ok && i.Value == 0 {
		return true
	}
	if f, ok := node.(*floatLit); ok && f.Value == 0 {
		return true
	}
	return false
}

// numericZeroLit returns 0 as int or float lit matching NumericMode.
func (o *optimizer) numericZeroLit() Node {
	if o.numericMode == NumericFloat64 {
		return &floatLit{Value: 0}
	}
	return &intLit{Value: 0}
}

// numericOneLit returns 1 as int or float lit matching NumericMode.
func (o *optimizer) numericOneLit() Node {
	if o.numericMode == NumericFloat64 {
		return &floatLit{Value: 1}
	}
	return &intLit{Value: 1}
}

// isOne reports whether node is the numeric literal 1.
func (o *optimizer) isOne(node Node) bool {
	if i, ok := node.(*intLit); ok && i.Value == 1 {
		return true
	}
	if f, ok := node.(*floatLit); ok && f.Value == 1 {
		return true
	}
	return false
}

// isDefinitelyNumeric reports whether node can only evaluate to a number.
// Used to avoid algebraic folds that are wrong for string concat / repeat.
func (o *optimizer) isDefinitelyNumeric(node Node) bool {
	switch n := node.(type) {
	case *intLit, *floatLit:
		return true
	case *unaryExpr:
		switch n.Op {
		case "-", "+", "~":
			return o.isDefinitelyNumeric(n.Operand)
		}
	case *binaryExpr:
		switch n.Op {
		case "-", "*", "/", "%", "**", "&", "|", "^", "<<", ">>":
			return o.isDefinitelyNumeric(n.Left) && o.isDefinitelyNumeric(n.Right)
		case "+":
			// + is numeric only when both sides are numeric (else string/array concat).
			return o.isDefinitelyNumeric(n.Left) && o.isDefinitelyNumeric(n.Right)
		}
	}
	return false
}

// isDefinitelyInt reports whether node can only evaluate to an int64.
// Required for ~~x → x (bitwise ~ always uses Int() coercion).
func (o *optimizer) isDefinitelyInt(node Node) bool {
	switch n := node.(type) {
	case *intLit:
		return true
	case *unaryExpr:
		switch n.Op {
		case "-", "+", "~":
			return o.isDefinitelyInt(n.Operand)
		}
	case *binaryExpr:
		switch n.Op {
		case "-", "*", "%", "&", "|", "^", "<<", ">>":
			return o.isDefinitelyInt(n.Left) && o.isDefinitelyInt(n.Right)
		case "+":
			return o.isDefinitelyInt(n.Left) && o.isDefinitelyInt(n.Right)
		}
	}
	return false
}

// nodesEqual checks if two nodes are structurally equal.
func (o *optimizer) nodesEqual(a, b Node) bool {
	switch na := a.(type) {
	case *ident:
		if nb, ok := b.(*ident); ok {
			return na.Name == nb.Name
		}
	case *intLit:
		if nb, ok := b.(*intLit); ok {
			return na.Value == nb.Value
		}
	case *floatLit:
		if nb, ok := b.(*floatLit); ok {
			return na.Value == nb.Value
		}
	case *stringLit:
		if nb, ok := b.(*stringLit); ok {
			return na.Value == nb.Value
		}
	case *boolLit:
		if nb, ok := b.(*boolLit); ok {
			return na.Value == nb.Value
		}
	case *nilLit:
		_, ok := b.(*nilLit)
		return ok
	}
	return false
}

// isSideEffectFree checks if an expression has no side effects.
func isSideEffectFree(node Node) bool {
	o := &optimizer{}
	return o.isSideEffectFree(node)
}

// isSideEffectFree checks if an expression has no side effects.
func (o *optimizer) isSideEffectFree(node Node) bool {
	switch n := node.(type) {
	case *intLit, *floatLit, *stringLit, *boolLit, *nilLit, *ident:
		return true
	case *binaryExpr:
		return o.isSideEffectFree(n.Left) && o.isSideEffectFree(n.Right)
	case *unaryExpr:
		return o.isSideEffectFree(n.Operand)
	case *memberExpr:
		return o.isSideEffectFree(n.Object) && o.isSideEffectFree(n.Property)
	case *indexExpr:
		return o.isSideEffectFree(n.Object) && o.isSideEffectFree(n.Index)
	case *ternaryExpr:
		return o.isSideEffectFree(n.Cond) && o.isSideEffectFree(n.Then) && o.isSideEffectFree(n.Else)
	case *arrayLit:
		for _, elem := range n.Elements {
			if !o.isSideEffectFree(elem) {
				return false
			}
		}
		return true
	case *mapLit:
		for i := range n.Keys {
			if !o.isSideEffectFree(n.Keys[i]) || !o.isSideEffectFree(n.Values[i]) {
				return false
			}
		}
		return true
	case *callExpr:
		// Function calls may have side effects
		return false
	default:
		// Assume expressions have side effects to be safe
		return false
	}
}

// eliminateDeadCode removes unreachable or useless code.
func (o *optimizer) eliminateDeadCode(node Node) Node {
	switch n := node.(type) {
	case *ternaryExpr:
		cond := o.eliminateDeadCode(n.Cond)
		then := o.eliminateDeadCode(n.Then)
		elseExpr := o.eliminateDeadCode(n.Else)

		// If condition is constant, return appropriate branch
		if boolLit, ok := cond.(*boolLit); ok {
			o.reducedCount++
			if boolLit.Value {
				return then
			}
			return elseExpr
		}

		return &ternaryExpr{Cond: cond, Then: then, Else: elseExpr}

	case *coalesceExpr:
		left := o.eliminateDeadCode(n.Left)
		right := o.eliminateDeadCode(n.Right)

		// If left is known non-nil constant, return left
		if lit, isLit := o.asLiteral(left); isLit {
			if _, isNil := lit.(*nilLit); !isNil {
				o.reducedCount++
				return left
			}
			// Left is nil, return right
			o.reducedCount++
			return right
		}

		return &coalesceExpr{Left: left, Right: right}

	case *binaryExpr:
		left := o.eliminateDeadCode(n.Left)
		right := o.eliminateDeadCode(n.Right)

		// Short-circuit evaluation
		if n.Op == "&&" {
			if l, ok := left.(*boolLit); ok && !l.Value {
				// false && anything -> false
				o.reducedCount++
				return &boolLit{Value: false}
			}
		}
		if n.Op == "||" {
			if l, ok := left.(*boolLit); ok && l.Value {
				// true || anything -> true
				o.reducedCount++
				return &boolLit{Value: true}
			}
		}

		return &binaryExpr{Op: n.Op, Left: left, Right: right}

	case *callExpr:
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = o.eliminateDeadCode(arg)
		}
		return &callExpr{Func: o.eliminateDeadCode(n.Func), Args: args}

	case *unaryExpr:
		return &unaryExpr{Op: n.Op, Operand: o.eliminateDeadCode(n.Operand)}

	case *arrayLit:
		elements := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = o.eliminateDeadCode(elem)
		}
		return &arrayLit{Elements: elements}

	case *mapLit:
		keys := make([]Node, len(n.Keys))
		values := make([]Node, len(n.Values))
		for i := range n.Keys {
			keys[i] = o.eliminateDeadCode(n.Keys[i])
			values[i] = o.eliminateDeadCode(n.Values[i])
		}
		return &mapLit{Keys: keys, Values: values}

	case *memberExpr:
		return &memberExpr{
			Object:   o.eliminateDeadCode(n.Object),
			Property: o.eliminateDeadCode(n.Property),
			Computed: n.Computed,
			Optional: n.Optional,
		}

	case *indexExpr:
		return &indexExpr{
			Object:   o.eliminateDeadCode(n.Object),
			Index:    o.eliminateDeadCode(n.Index),
			Optional: n.Optional,
		}

	case *letExpr:
		return &letExpr{
			Name:  n.Name,
			Value: o.eliminateDeadCode(n.Value),
			Body:  o.eliminateDeadCode(n.Body),
		}

	case *spreadExpr:
		return &spreadExpr{Expr: o.eliminateDeadCode(n.Expr)}

	case *destructureExpr:
		return &destructureExpr{
			Pattern: n.Pattern,
			IsArray: n.IsArray,
			Value:   o.eliminateDeadCode(n.Value),
			Body:    o.eliminateDeadCode(n.Body),
		}

	default:
		return node
	}
}
