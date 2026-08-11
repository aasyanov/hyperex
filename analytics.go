package hyperex

import (
	"context"
	"time"
)

// Complexity represents the estimated computational complexity of an expression.
type Complexity int

// Complexity classes ordered by estimated expression cost.
const (
	ComplexityConstant     Complexity = iota // O(1)
	ComplexityLinear                         // O(n)
	ComplexityLinearithmic                   // O(n log n)
	ComplexityQuadratic                      // O(n²)
	ComplexityUnknown                        // Cannot determine
)

func (c Complexity) String() string {
	switch c {
	case ComplexityConstant:
		return "O(1)"
	case ComplexityLinear:
		return "O(n)"
	case ComplexityLinearithmic:
		return "O(n log n)"
	case ComplexityQuadratic:
		return "O(n²)"
	default:
		return complexityUnknownStr
	}
}

// CheckExpressionType checks if an expression evaluates to the expected type.
// It performs both syntax and type checking.
//
// Parameters:
//   - expr: The expression to check
//   - expectedType: The expected return type
//   - env: Optional environment struct for type inference (can be nil)
//
// Returns an error if the expression doesn't match the expected type or has type errors.
func CheckExpressionType(expr string, expectedType Type, env any) error {
	parser := newParser(expr)
	ast, err := parser.parse()
	if err != nil {
		return err
	}

	tc := NewTypeChecker()
	if env != nil {
		tc.WithStructEnv(env)
	}

	// Check for type errors
	if errs := tc.Check(ast); errs != nil && len(errs.Errors) > 0 {
		return errs
	}

	// Infer the type and compare with expected
	inferredType := tc.Infer(ast)
	if inferredType != expectedType && inferredType != TypeAny && inferredType != TypeUnknown {
		return &TypeError{
			Message:  "expression type mismatch",
			Expected: expectedType.String(),
			Got:      inferredType.String(),
		}
	}

	return nil
}

// Analysis holds static-analysis results for an expression AST.
// Fields describe what the expression uses and how expensive it is likely to be;
// they are filled by [Analyze] / [AnalyzeAST] and do not require execution.
type Analysis struct {
	// Variables lists distinct identifier names referenced by the expression.
	Variables []string
	// Functions lists distinct function names invoked by the expression.
	Functions []string
	// Operators lists distinct operator tokens used (e.g. "+", "in", "??").
	Operators []string
	// Complexity is the estimated big-O class of the expression.
	Complexity Complexity
	// Depth is the maximum AST nesting depth observed.
	Depth int
	// NodeCount is the total number of AST nodes visited.
	NodeCount int
	// HasSideEffects is true when the expression may call side-effecting builtins
	// (for example print).
	HasSideEffects bool
	// IsPure is true when the expression is treated as deterministic with no
	// side effects. It is the inverse of HasSideEffects for known builtins.
	IsPure bool
	// HasAssignment is true when the expression contains let or destructuring
	// bindings.
	HasAssignment bool
	// UsesOptionalChaining is true when the expression uses ?. or ?.[.
	UsesOptionalChaining bool
	// UsesNilCoalescing is true when the expression uses ??.
	UsesNilCoalescing bool
	// EstimatedCost is a coarse unit cost used to size sandbox limits.
	EstimatedCost int
	// Warnings lists stable English diagnostic messages for ambiguous or
	// heuristic-driven constructs (for example pipe vs bitwise |).
	Warnings []string
}

// Stable Analysis.Warnings messages for | pipe vs bitwise OR heuristics.
const (
	warnAmbiguousBitwiseOr = "ambiguous | between two identifiers treated as bitwise OR; use an explicit call (x | f()) for pipe"
	warnBuiltinNamePipe    = "pipe chosen because right-hand identifier matches a default builtin name"
)

// Profile holds runtime profiling results from compiling and/or executing an
// expression. Timing fields are wall-clock durations; OpsCount and MaxStackDepth
// are measured by the vm during Run.
type Profile struct {
	// Expression is the source text that was profiled.
	Expression string
	// CompileTime is the wall time spent compiling Expression into bytecode.
	// Zero when profiling an already-compiled [Program].
	CompileTime time.Duration
	// ExecuteTime is the wall time spent running the compiled program.
	ExecuteTime time.Duration
	// TotalTime is CompileTime + ExecuteTime for [ProfileExpr], or ExecuteTime
	// alone when profiling via [Program.Profile].
	TotalTime time.Duration
	// OpsCount is the number of vm operations executed during the run.
	OpsCount int64
	// MaxStackDepth is the peak vm stack pointer observed during the run.
	MaxStackDepth int
	// Result is the evaluated value on success; the zero Value when Error != nil.
	Result Value
	// Error is the compile or runtime error, if any.
	Error error
}

// Analyze performs static analysis on an expression string.
func Analyze(expr string) (*Analysis, error) {
	parser := newParser(expr)
	ast, err := parser.parse()
	if err != nil {
		return nil, err
	}
	return AnalyzeAST(ast), nil
}

// AnalyzeAST performs static analysis on an AST.
func AnalyzeAST(node Node) *Analysis {
	a := &Analysis{
		Complexity: ComplexityConstant,
		IsPure:     true,
	}

	a.analyze(node, 0)
	a.Variables = uniqueStrings(a.Variables)
	a.Functions = uniqueStrings(a.Functions)
	a.Operators = uniqueStrings(a.Operators)
	a.Warnings = uniqueStrings(a.Warnings)
	a.EstimatedCost = a.estimateCost()

	return a
}

func (a *Analysis) analyze(node Node, depth int) {
	if node == nil {
		return
	}

	a.NodeCount++
	if depth > a.Depth {
		a.Depth = depth
	}

	switch n := node.(type) {
	case *intLit, *floatLit, *stringLit, *boolLit, *nilLit:
		// Literals are constant

	case *ident:
		a.Variables = append(a.Variables, n.Name)

	case *binaryExpr:
		a.Operators = append(a.Operators, n.Op)
		a.analyze(n.Left, depth+1)
		a.analyze(n.Right, depth+1)

		// Ambiguous | : bitwise OR between two idents that are not known builtins.
		if n.Op == "|" {
			if _, lok := n.Left.(*ident); lok {
				if ri, rok := n.Right.(*ident); rok && !isDefaultBuiltinName(ri.Name) {
					a.Warnings = append(a.Warnings, warnAmbiguousBitwiseOr)
				}
			}
		}

		// Complexity estimation
		if n.Op == "in" {
			// "in" operator on arrays is O(n)
			a.upgradeComplexity(ComplexityLinear)
		}

	case *unaryExpr:
		a.Operators = append(a.Operators, n.Op)
		a.analyze(n.Operand, depth+1)

	case *ternaryExpr:
		a.analyze(n.Cond, depth+1)
		a.analyze(n.Then, depth+1)
		a.analyze(n.Else, depth+1)

	case *callExpr:
		// Get function name if it's an identifier
		if ident, ok := n.Func.(*ident); ok {
			a.Functions = append(a.Functions, ident.Name)

			// Check for side-effect functions
			if isSideEffectFunc(ident.Name) {
				a.HasSideEffects = true
				a.IsPure = false
			}

			// Complexity estimation for built-in functions
			switch ident.Name {
			case nameMap, nameFilter, nameReduce, nameFind, nameFindIndex, nameUnique, nameSum, nameAvg, nameAll, nameAny, nameNone, nameOne, nameCount:
				a.upgradeComplexity(ComplexityLinear)
			case nameSort:
				// Sorting is O(n log n) in typical implementations
				a.upgradeComplexity(ComplexityLinearithmic)
			}
		}

		a.analyze(n.Func, depth+1)
		for _, arg := range n.Args {
			a.analyze(arg, depth+1)
		}

	case *memberExpr:
		if n.Optional {
			a.UsesOptionalChaining = true
		}
		a.analyze(n.Object, depth+1)
		a.analyze(n.Property, depth+1)

	case *indexExpr:
		if n.Optional {
			a.UsesOptionalChaining = true
		}
		a.analyze(n.Object, depth+1)
		a.analyze(n.Index, depth+1)

	case *sliceExpr:
		a.analyze(n.Object, depth+1)
		if n.Low != nil {
			a.analyze(n.Low, depth+1)
		}
		if n.High != nil {
			a.analyze(n.High, depth+1)
		}
		if n.Step != nil {
			a.analyze(n.Step, depth+1)
		}
		// Slicing is O(n) in worst case
		a.upgradeComplexity(ComplexityLinear)

	case *arrayLit:
		for _, elem := range n.Elements {
			a.analyze(elem, depth+1)
		}

	case *mapLit:
		for i := range n.Keys {
			a.analyze(n.Keys[i], depth+1)
			a.analyze(n.Values[i], depth+1)
		}

	case *rangeExpr:
		a.analyze(n.Start, depth+1)
		a.analyze(n.End, depth+1)
		// Range creates O(n) elements
		a.upgradeComplexity(ComplexityLinear)

	case *pipeExpr:
		a.analyze(n.Left, depth+1)
		a.analyze(n.Right, depth+1)
		// Info: ident | builtinIdent was chosen as pipe via the builtin-name rule.
		if _, lok := n.Left.(*ident); lok {
			if ri, rok := n.Right.(*ident); rok && isDefaultBuiltinName(ri.Name) {
				a.Warnings = append(a.Warnings, warnBuiltinNamePipe)
			}
		}
		// Pipe usually involves iteration
		a.upgradeComplexity(ComplexityLinear)

	case *lambdaExpr:
		a.analyze(n.Body, depth+1)

	case *letExpr:
		a.HasAssignment = true
		a.Variables = append(a.Variables, n.Name)
		a.analyze(n.Value, depth+1)
		a.analyze(n.Body, depth+1)

	case *coalesceExpr:
		a.UsesNilCoalescing = true
		a.analyze(n.Left, depth+1)
		a.analyze(n.Right, depth+1)

	case *spreadExpr:
		a.analyze(n.Expr, depth+1)
		// Spread operation is O(n) where n is the size of the spread array
		a.upgradeComplexity(ComplexityLinear)

	case *destructureExpr:
		a.HasAssignment = true
		// Register all destructured variable names
		for _, field := range n.Pattern {
			a.Variables = append(a.Variables, field.Name)
			if field.Default != nil {
				a.analyze(field.Default, depth+1)
			}
		}
		a.analyze(n.Value, depth+1)
		a.analyze(n.Body, depth+1)
	}
}

func (a *Analysis) upgradeComplexity(c Complexity) {
	if c > a.Complexity {
		a.Complexity = c
	}
}

func (a *Analysis) estimateCost() int {
	cost := a.NodeCount

	// Add cost for function calls
	cost += len(a.Functions) * 10

	// Add cost based on complexity
	switch a.Complexity {
	case ComplexityLinear:
		cost *= 2
	case ComplexityQuadratic:
		cost *= 4
	}

	return cost
}

func isSideEffectFunc(name string) bool {
	switch name {
	case namePrint, nameSprintf:
		return true
	default:
		return false
	}
}

func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	// Sort for consistent ordering
	sortStrings(result)
	return result
}

// ProfileExpr compiles and executes an expression with profiling.
// Returns the profile, result value, and any error that occurred.
func ProfileExpr(ctx context.Context, expr string, env any) (*Profile, any, error) {
	p := &Profile{Expression: expr}

	// Time compilation - use engine with high ops limit to track execution
	compileStart := time.Now()
	eng := New(WithSandbox(SandboxConfig{MaxOps: 1_000_000_000})) // High limit just for tracking
	prog, err := eng.Compile(expr)
	p.CompileTime = time.Since(compileStart)

	if err != nil {
		p.Error = err
		p.TotalTime = p.CompileTime
		return p, nil, err
	}

	// Time execution with ops tracking
	execStart := time.Now()

	// Build sandbox limits for tracking
	limits := buildSandboxLimits(&SandboxConfig{MaxOps: 1_000_000_000})

	// Read builtins with read lock
	prog.engine.mu.RLock()
	builtins := prog.engine.builtins
	prog.engine.mu.RUnlock()

	vm := newVMWithLimits(prog.bytecode, builtins, limits)
	// Inherit strict mode and maxCallDepth from engine config
	if prog.engine.config.strictMode {
		vm.strictMode = true
	}
	if prog.engine.config.maxCallDepth > 0 {
		vm.maxDepth = prog.engine.config.maxCallDepth
	}
	resultVal, err := vm.run(ctx, toEnvMap(env))
	p.OpsCount = vm.opsCount
	p.MaxStackDepth = vm.maxSP
	releaseVM(vm)

	p.ExecuteTime = time.Since(execStart)

	if err != nil {
		p.Error = err
		return p, nil, err
	}
	p.Result = resultVal

	p.TotalTime = p.CompileTime + p.ExecuteTime

	return p, resultVal.Interface(), nil
}

// Profile runs the program with profiling enabled.
// Returns the profile, result value, and any error.
func (p *Program) Profile(ctx context.Context, env any) (*Profile, any, error) {
	prof := &Profile{
		Expression: p.Source(),
	}

	execStart := time.Now()

	// Build sandbox limits for ops tracking
	limits := buildSandboxLimits(&SandboxConfig{MaxOps: 1_000_000_000})

	// Read builtins with read lock
	p.engine.mu.RLock()
	builtins := p.engine.builtins
	p.engine.mu.RUnlock()

	vm := newVMWithLimits(p.bytecode, builtins, limits)
	// Inherit strict mode and maxCallDepth from engine config
	if p.engine.config.strictMode {
		vm.strictMode = true
	}
	if p.engine.config.maxCallDepth > 0 {
		vm.maxDepth = p.engine.config.maxCallDepth
	}
	resultVal, err := vm.run(ctx, toEnvMap(env))
	prof.OpsCount = vm.opsCount
	prof.MaxStackDepth = vm.maxSP
	releaseVM(vm)

	prof.ExecuteTime = time.Since(execStart)
	prof.TotalTime = prof.ExecuteTime

	if err != nil {
		prof.Error = err
		return prof, nil, err
	}
	prof.Result = resultVal

	return prof, resultVal.Interface(), nil
}

// Analyze performs static analysis on the program's expression.
func (p *Program) Analyze() *Analysis {
	parser := newParser(p.Source())
	ast, err := parser.parse()
	if err != nil {
		return &Analysis{}
	}
	return AnalyzeAST(ast)
}
