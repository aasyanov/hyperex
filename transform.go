package hyperex

import (
	"fmt"
	"strings"
)

// Simplify optimizes an expression string and returns the simplified version.
// It applies constant folding, strength reduction, and algebraic simplifications.
//
// Example:
//
//	simplified, _ := hyperex.Simplify("x * 2 + 0")
//	// simplified = "x * 2"
//
//	simplified, _ := hyperex.Simplify("2 + 3 * 4")
//	// simplified = "14"
func Simplify(expr string) (string, error) {
	parser := newParser(expr)
	ast, err := parser.parse()
	if err != nil {
		return "", err
	}

	optimizer := &optimizer{
		constantFolding:   true,
		strengthReduction: true,
		algebraicSimplify: true,
	}

	optimized := optimizer.optimize(ast)
	return ASTToString(optimized), nil
}

// Expand replaces macros/aliases in an expression with their definitions.
//
// Example:
//
//	macros := map[string]string{
//	    "ADULT": "age >= 18",
//	    "USA":   `country == "US"`,
//	}
//	expanded, _ := hyperex.Expand("ADULT && USA", macros)
//	// expanded = `(age >= 18) && (country == "US")`
func Expand(expr string, macros map[string]string) (string, error) {
	parser := newParser(expr)
	ast, err := parser.parse()
	if err != nil {
		return "", err
	}

	// Parse all macro definitions
	parsedMacros := make(map[string]Node)
	for name, def := range macros {
		macroParser := newParser(def)
		macroAST, err := macroParser.parse()
		if err != nil {
			return "", fmt.Errorf("invalid macro %s: %w", name, err)
		}
		parsedMacros[name] = macroAST
	}

	// Transform AST, replacing identifiers with macro definitions
	// Use copyNode to avoid sharing AST nodes between multiple macro expansions
	expanded := Transform(ast, func(n Node) Node {
		if ident, ok := n.(*ident); ok {
			if replacement, exists := parsedMacros[ident.Name]; exists {
				return copyNode(replacement)
			}
		}
		return n
	})

	return ASTToString(expanded), nil
}

// Inline expands let expressions by substituting variable references.
// This can make expressions longer but removes variable bindings.
//
// Example:
//
//	inlined, _ := hyperex.Inline("let x = 5; x + x")
//	// inlined = "5 + 5"
func Inline(expr string) (string, error) {
	parser := newParser(expr)
	ast, err := parser.parse()
	if err != nil {
		return "", err
	}

	inlined := inlineLetExprs(ast, make(map[string]Node))
	return ASTToString(inlined), nil
}

// inlineLetExprs recursively inlines let expressions.
func inlineLetExprs(node Node, bindings map[string]Node) Node {
	switch n := node.(type) {
	case *letExpr:
		// Inline the value first
		value := inlineLetExprs(n.Value, bindings)
		// Add to bindings
		newBindings := copyBindings(bindings)
		newBindings[n.Name] = value
		// Inline the body with new bindings
		return inlineLetExprs(n.Body, newBindings)

	case *ident:
		if replacement, ok := bindings[n.Name]; ok {
			return replacement
		}
		return n

	case *binaryExpr:
		return &binaryExpr{
			Op:    n.Op,
			Left:  inlineLetExprs(n.Left, bindings),
			Right: inlineLetExprs(n.Right, bindings),
		}

	case *unaryExpr:
		return &unaryExpr{
			Op:      n.Op,
			Operand: inlineLetExprs(n.Operand, bindings),
		}

	case *ternaryExpr:
		return &ternaryExpr{
			Cond: inlineLetExprs(n.Cond, bindings),
			Then: inlineLetExprs(n.Then, bindings),
			Else: inlineLetExprs(n.Else, bindings),
		}

	case *coalesceExpr:
		return &coalesceExpr{
			Left:  inlineLetExprs(n.Left, bindings),
			Right: inlineLetExprs(n.Right, bindings),
		}

	case *callExpr:
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = inlineLetExprs(arg, bindings)
		}
		return &callExpr{
			Func: inlineLetExprs(n.Func, bindings),
			Args: args,
		}

	case *memberExpr:
		return &memberExpr{
			Object:   inlineLetExprs(n.Object, bindings),
			Property: n.Property,
			Computed: n.Computed,
			Optional: n.Optional,
		}

	case *indexExpr:
		return &indexExpr{
			Object:   inlineLetExprs(n.Object, bindings),
			Index:    inlineLetExprs(n.Index, bindings),
			Optional: n.Optional,
		}

	case *arrayLit:
		elements := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = inlineLetExprs(elem, bindings)
		}
		return &arrayLit{Elements: elements}

	case *mapLit:
		keys := make([]Node, len(n.Keys))
		values := make([]Node, len(n.Values))
		for i := range n.Keys {
			keys[i] = inlineLetExprs(n.Keys[i], bindings)
			values[i] = inlineLetExprs(n.Values[i], bindings)
		}
		return &mapLit{Keys: keys, Values: values}

	case *destructureExpr:
		// Inline the source value first
		value := inlineLetExprs(n.Value, bindings)

		// For destructuring, we can't truly inline since it creates multiple bindings
		// But we can inline references within the body and default values
		newBindings := copyBindings(bindings)

		// Process default values in pattern
		newPattern := make([]destructureField, len(n.Pattern))
		for i, field := range n.Pattern {
			newPattern[i] = destructureField{
				Key:  field.Key,
				Name: field.Name,
			}
			if field.Default != nil {
				newPattern[i].Default = inlineLetExprs(field.Default, bindings)
			}
		}

		// Inline the body with current bindings (destructure vars shadow any existing)
		body := inlineLetExprs(n.Body, newBindings)

		return &destructureExpr{
			Pattern: newPattern,
			IsArray: n.IsArray,
			Value:   value,
			Body:    body,
		}

	case *lambdaExpr:
		// Lambda body should be inlined with current bindings
		// (but lambda params shadow any existing bindings)
		return &lambdaExpr{
			Params: n.Params,
			Body:   inlineLetExprs(n.Body, bindings),
		}

	case *pipeExpr:
		return &pipeExpr{
			Left:  inlineLetExprs(n.Left, bindings),
			Right: inlineLetExprs(n.Right, bindings),
		}

	case *rangeExpr:
		return &rangeExpr{
			Start: inlineLetExprs(n.Start, bindings),
			End:   inlineLetExprs(n.End, bindings),
		}

	case *sliceExpr:
		var low, high, step Node
		if n.Low != nil {
			low = inlineLetExprs(n.Low, bindings)
		}
		if n.High != nil {
			high = inlineLetExprs(n.High, bindings)
		}
		if n.Step != nil {
			step = inlineLetExprs(n.Step, bindings)
		}
		return &sliceExpr{
			Object: inlineLetExprs(n.Object, bindings),
			Low:    low,
			High:   high,
			Step:   step,
		}

	case *spreadExpr:
		return &spreadExpr{
			Expr: inlineLetExprs(n.Expr, bindings),
		}

	default:
		return node
	}
}

func copyBindings(m map[string]Node) map[string]Node {
	cp := make(map[string]Node, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// copyNode creates a deep copy of an AST node.
// This is used to avoid sharing AST nodes when expanding macros.
func copyNode(node Node) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *intLit:
		return &intLit{Value: n.Value}
	case *floatLit:
		return &floatLit{Value: n.Value}
	case *stringLit:
		return &stringLit{Value: n.Value}
	case *boolLit:
		return &boolLit{Value: n.Value}
	case *nilLit:
		return &nilLit{}
	case *ident:
		return &ident{Name: n.Name}
	case *binaryExpr:
		return &binaryExpr{
			Op:    n.Op,
			Left:  copyNode(n.Left),
			Right: copyNode(n.Right),
		}
	case *unaryExpr:
		return &unaryExpr{
			Op:      n.Op,
			Operand: copyNode(n.Operand),
		}
	case *ternaryExpr:
		return &ternaryExpr{
			Cond: copyNode(n.Cond),
			Then: copyNode(n.Then),
			Else: copyNode(n.Else),
		}
	case *coalesceExpr:
		return &coalesceExpr{
			Left:  copyNode(n.Left),
			Right: copyNode(n.Right),
		}
	case *callExpr:
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = copyNode(arg)
		}
		return &callExpr{
			Func: copyNode(n.Func),
			Args: args,
		}
	case *memberExpr:
		return &memberExpr{
			Object:   copyNode(n.Object),
			Property: copyNode(n.Property),
			Computed: n.Computed,
			Optional: n.Optional,
		}
	case *indexExpr:
		return &indexExpr{
			Object:   copyNode(n.Object),
			Index:    copyNode(n.Index),
			Optional: n.Optional,
		}
	case *sliceExpr:
		return &sliceExpr{
			Object: copyNode(n.Object),
			Low:    copyNode(n.Low),
			High:   copyNode(n.High),
			Step:   copyNode(n.Step),
		}
	case *arrayLit:
		elements := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = copyNode(elem)
		}
		return &arrayLit{Elements: elements}
	case *mapLit:
		keys := make([]Node, len(n.Keys))
		values := make([]Node, len(n.Values))
		for i := range n.Keys {
			keys[i] = copyNode(n.Keys[i])
			values[i] = copyNode(n.Values[i])
		}
		return &mapLit{Keys: keys, Values: values}
	case *rangeExpr:
		return &rangeExpr{
			Start: copyNode(n.Start),
			End:   copyNode(n.End),
		}
	case *pipeExpr:
		return &pipeExpr{
			Left:  copyNode(n.Left),
			Right: copyNode(n.Right),
		}
	case *lambdaExpr:
		params := make([]string, len(n.Params))
		copy(params, n.Params)
		return &lambdaExpr{
			Params: params,
			Body:   copyNode(n.Body),
		}
	case *letExpr:
		return &letExpr{
			Name:  n.Name,
			Value: copyNode(n.Value),
			Body:  copyNode(n.Body),
		}
	case *spreadExpr:
		return &spreadExpr{
			Expr: copyNode(n.Expr),
		}
	case *destructureExpr:
		pattern := make([]destructureField, len(n.Pattern))
		for i, f := range n.Pattern {
			pattern[i] = destructureField{
				Key:     f.Key,
				Name:    f.Name,
				Default: copyNode(f.Default),
			}
		}
		return &destructureExpr{
			Pattern: pattern,
			IsArray: n.IsArray,
			Value:   copyNode(n.Value),
			Body:    copyNode(n.Body),
		}
	default:
		return node
	}
}

// ASTToString converts an AST node back to an expression string.
func ASTToString(node Node) string {
	switch n := node.(type) {
	case *intLit:
		return fmt.Sprintf("%d", n.Value)

	case *floatLit:
		return fmt.Sprintf("%g", n.Value)

	case *stringLit:
		return fmt.Sprintf("%q", n.Value)

	case *boolLit:
		if n.Value {
			return litTrue
		}
		return litFalse

	case *nilLit:
		return litNil

	case *ident:
		return n.Name

	case *binaryExpr:
		left := ASTToString(n.Left)
		right := ASTToString(n.Right)
		// Add parentheses for clarity
		if needsParens(n.Left) {
			left = "(" + left + ")"
		}
		if needsParens(n.Right) {
			right = "(" + right + ")"
		}
		return left + " " + n.Op + " " + right

	case *unaryExpr:
		operand := ASTToString(n.Operand)
		if needsParens(n.Operand) {
			operand = "(" + operand + ")"
		}
		if n.Op == kwNot {
			return "not " + operand
		}
		return n.Op + operand

	case *ternaryExpr:
		return fmt.Sprintf("%s ? %s : %s",
			ASTToString(n.Cond),
			ASTToString(n.Then),
			ASTToString(n.Else))

	case *coalesceExpr:
		return fmt.Sprintf("%s ?? %s",
			ASTToString(n.Left),
			ASTToString(n.Right))

	case *callExpr:
		args := make([]string, len(n.Args))
		for i, arg := range n.Args {
			args[i] = ASTToString(arg)
		}
		return fmt.Sprintf("%s(%s)", ASTToString(n.Func), strings.Join(args, ", "))

	case *memberExpr:
		obj := ASTToString(n.Object)
		if n.Computed {
			prop := ASTToString(n.Property)
			if n.Optional {
				return fmt.Sprintf("%s?.[%s]", obj, prop)
			}
			return fmt.Sprintf("%s[%s]", obj, prop)
		}
		prop := n.Property.(*ident).Name
		if n.Optional {
			return fmt.Sprintf("%s?.%s", obj, prop)
		}
		return fmt.Sprintf("%s.%s", obj, prop)

	case *indexExpr:
		obj := ASTToString(n.Object)
		idx := ASTToString(n.Index)
		if n.Optional {
			return fmt.Sprintf("%s?.[%s]", obj, idx)
		}
		return fmt.Sprintf("%s[%s]", obj, idx)

	case *sliceExpr:
		obj := ASTToString(n.Object)
		low := ""
		high := ""
		step := ""
		if n.Low != nil {
			low = ASTToString(n.Low)
		}
		if n.High != nil {
			high = ASTToString(n.High)
		}
		if n.Step != nil {
			step = ASTToString(n.Step)
			return fmt.Sprintf("%s[%s:%s:%s]", obj, low, high, step)
		}
		return fmt.Sprintf("%s[%s:%s]", obj, low, high)

	case *arrayLit:
		elements := make([]string, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = ASTToString(elem)
		}
		return "[" + strings.Join(elements, ", ") + "]"

	case *mapLit:
		pairs := make([]string, len(n.Keys))
		for i := range n.Keys {
			key := ASTToString(n.Keys[i])
			val := ASTToString(n.Values[i])
			pairs[i] = key + ": " + val
		}
		return "{" + strings.Join(pairs, ", ") + "}"

	case *rangeExpr:
		return fmt.Sprintf("(%s..%s)",
			ASTToString(n.Start),
			ASTToString(n.End))

	case *pipeExpr:
		return fmt.Sprintf("%s | %s",
			ASTToString(n.Left),
			ASTToString(n.Right))

	case *lambdaExpr:
		params := strings.Join(n.Params, ", ")
		body := ASTToString(n.Body)
		if len(n.Params) == 1 {
			return fmt.Sprintf("%s => %s", params, body)
		}
		return fmt.Sprintf("(%s) => %s", params, body)

	case *letExpr:
		return fmt.Sprintf("let %s = %s; %s",
			n.Name,
			ASTToString(n.Value),
			ASTToString(n.Body))

	case *spreadExpr:
		return fmt.Sprintf("...%s", ASTToString(n.Expr))

	case *destructureExpr:
		fields := make([]string, len(n.Pattern))
		for i, f := range n.Pattern {
			str := f.Name
			if f.Key != "" && f.Key != f.Name {
				str = f.Key + ": " + f.Name
			}
			if f.Default != nil {
				str += " = " + ASTToString(f.Default)
			}
			fields[i] = str
		}
		if n.IsArray {
			return fmt.Sprintf("let [%s] = %s; %s",
				strings.Join(fields, ", "),
				ASTToString(n.Value),
				ASTToString(n.Body))
		}
		return fmt.Sprintf("let {%s} = %s; %s",
			strings.Join(fields, ", "),
			ASTToString(n.Value),
			ASTToString(n.Body))

	default:
		return fmt.Sprintf("<%T>", node)
	}
}

// needsParens determines if an expression needs parentheses when used as operand.
func needsParens(node Node) bool {
	switch node.(type) {
	case *binaryExpr, *ternaryExpr, *coalesceExpr, *letExpr:
		return true
	}
	return false
}

// ExtractOption configures [ExtractVariablesOpts].
type ExtractOption func(*extractConfig)

type extractConfig struct {
	memberRootsOnly bool
}

// ExtractMemberRoots strips property identifiers under non-computed member
// access when extracting variables. For `Alarm.Fire`, only `Alarm` is listed
// (not `Fire`). Computed access (`obj[key]`) still extracts names from `key`.
// Default [ExtractVariables] keeps the historical behavior (Fire is listed).
func ExtractMemberRoots() ExtractOption {
	return func(c *extractConfig) {
		c.memberRootsOnly = true
	}
}

// ExtractVariables returns all variable names used in an expression.
// This is a convenience wrapper around Analyze (default options).
func ExtractVariables(expr string) ([]string, error) {
	return ExtractVariablesOpts(expr)
}

// ExtractVariablesOpts is [ExtractVariables] with optional extraction policy.
func ExtractVariablesOpts(expr string, opts ...ExtractOption) ([]string, error) {
	cfg := &extractConfig{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(cfg)
	}
	parser := newParser(expr)
	ast, err := parser.parse()
	if err != nil {
		return nil, err
	}
	a := &Analysis{
		Complexity:      ComplexityConstant,
		IsPure:          true,
		memberRootsOnly: cfg.memberRootsOnly,
	}
	a.analyze(ast, 0)
	return uniqueStrings(a.Variables), nil
}

// ExtractFunctions returns all function names called in an expression.
// This is a convenience wrapper around Analyze.
func ExtractFunctions(expr string) ([]string, error) {
	analysis, err := Analyze(expr)
	if err != nil {
		return nil, err
	}
	return analysis.Functions, nil
}

// Validate checks if an expression is syntactically valid.
// Returns nil if valid, or a parse error if invalid.
func Validate(expr string) error {
	parser := newParser(expr)
	_, err := parser.parse()
	return err
}

// ValidateWithEnv validates an expression against a typed environment.
// This performs both syntax and type checking.
func ValidateWithEnv(expr string, env any) error {
	engine := New(WithTypeCheck(env))
	_, err := engine.Compile(expr)
	return err
}
