package hyperex

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Simplify optimizes an expression string and returns the simplified version.
// It applies constant folding, strength reduction, and algebraic simplifications.
//
// Example:
//
//	simplified, _ := hyperex.Simplify("x * 1")
//	// simplified = "x"
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
		nonFiniteMode:     NonFiniteError,
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

	expanded := expandMacroExprs(ast, parsedMacros, make(map[string]bool))
	return ASTToString(expanded), nil
}

func expandMacroExprs(node Node, macros map[string]Node, bound map[string]bool) Node {
	switch n := node.(type) {
	case *ident:
		if !bound[n.Name] {
			if replacement, ok := macros[n.Name]; ok {
				return copyNode(replacement)
			}
		}
		return n
	case *binaryExpr:
		return &binaryExpr{Op: n.Op, Left: expandMacroExprs(n.Left, macros, bound), Right: expandMacroExprs(n.Right, macros, bound)}
	case *unaryExpr:
		return &unaryExpr{Op: n.Op, Operand: expandMacroExprs(n.Operand, macros, bound)}
	case *ternaryExpr:
		return &ternaryExpr{Cond: expandMacroExprs(n.Cond, macros, bound), Then: expandMacroExprs(n.Then, macros, bound), Else: expandMacroExprs(n.Else, macros, bound)}
	case *coalesceExpr:
		return &coalesceExpr{Left: expandMacroExprs(n.Left, macros, bound), Right: expandMacroExprs(n.Right, macros, bound)}
	case *callExpr:
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = expandMacroExprs(arg, macros, bound)
		}
		return &callExpr{Func: expandMacroExprs(n.Func, macros, bound), Args: args}
	case *memberExpr:
		property := n.Property
		if n.Computed {
			property = expandMacroExprs(property, macros, bound)
		}
		return &memberExpr{Object: expandMacroExprs(n.Object, macros, bound), Property: property, Computed: n.Computed, Optional: n.Optional}
	case *indexExpr:
		return &indexExpr{Object: expandMacroExprs(n.Object, macros, bound), Index: expandMacroExprs(n.Index, macros, bound), Optional: n.Optional}
	case *sliceExpr:
		return &sliceExpr{
			Object: expandMacroExprs(n.Object, macros, bound),
			Low:    expandMacroExprs(n.Low, macros, bound),
			High:   expandMacroExprs(n.High, macros, bound),
			Step:   expandMacroExprs(n.Step, macros, bound),
		}
	case *arrayLit:
		elements := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = expandMacroExprs(elem, macros, bound)
		}
		return &arrayLit{Elements: elements}
	case *mapLit:
		keys := make([]Node, len(n.Keys))
		values := make([]Node, len(n.Values))
		for i := range n.Keys {
			keys[i] = expandMacroExprs(n.Keys[i], macros, bound)
			values[i] = expandMacroExprs(n.Values[i], macros, bound)
		}
		return &mapLit{Keys: keys, Values: values}
	case *rangeExpr:
		return &rangeExpr{Start: expandMacroExprs(n.Start, macros, bound), End: expandMacroExprs(n.End, macros, bound)}
	case *pipeExpr:
		return &pipeExpr{Left: expandMacroExprs(n.Left, macros, bound), Right: expandMacroExprs(n.Right, macros, bound)}
	case *lambdaExpr:
		lambdaBound := copyNameSet(bound)
		for _, param := range n.Params {
			lambdaBound[param] = true
		}
		return &lambdaExpr{Params: n.Params, Body: expandMacroExprs(n.Body, macros, lambdaBound)}
	case *letExpr:
		letBound := copyNameSet(bound)
		letBound[n.Name] = true
		return &letExpr{
			Name:         n.Name,
			Value:        expandMacroExprs(n.Value, macros, bound),
			Body:         expandMacroExprs(n.Body, macros, letBound),
			ImplicitBody: n.ImplicitBody,
		}
	case *spreadExpr:
		return &spreadExpr{Expr: expandMacroExprs(n.Expr, macros, bound)}
	case *destructureExpr:
		pattern := make([]destructureField, len(n.Pattern))
		bodyBound := copyNameSet(bound)
		for i, field := range n.Pattern {
			pattern[i] = field
			if field.Default != nil {
				pattern[i].Default = expandMacroExprs(field.Default, macros, bound)
			}
			bodyBound[field.Name] = true
		}
		return &destructureExpr{
			Pattern:      pattern,
			IsArray:      n.IsArray,
			Value:        expandMacroExprs(n.Value, macros, bound),
			Body:         expandMacroExprs(n.Body, macros, bodyBound),
			ImplicitBody: n.ImplicitBody,
		}
	default:
		return node
	}
}

func copyNameSet(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for name := range src {
		dst[name] = true
	}
	return dst
}

// Inline expands closed literal let bindings by substituting variable references.
// Other bindings are preserved to avoid duplicated effects, capture, and removed errors.
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
		value := inlineLetExprs(n.Value, bindings)
		if !isClosedLiteralValue(value) {
			bodyBindings := copyBindings(bindings)
			delete(bodyBindings, n.Name)
			return &letExpr{
				Name:         n.Name,
				Value:        value,
				Body:         inlineLetExprs(n.Body, bodyBindings),
				ImplicitBody: n.ImplicitBody,
			}
		}
		newBindings := copyBindings(bindings)
		newBindings[n.Name] = value
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

		// Destructured names shadow outer bindings in the body.
		for _, field := range n.Pattern {
			delete(newBindings, field.Name)
		}
		body := inlineLetExprs(n.Body, newBindings)

		return &destructureExpr{
			Pattern:      newPattern,
			IsArray:      n.IsArray,
			Value:        value,
			Body:         body,
			ImplicitBody: n.ImplicitBody,
		}

	case *lambdaExpr:
		lambdaBindings := copyBindings(bindings)
		for _, param := range n.Params {
			delete(lambdaBindings, param)
		}
		return &lambdaExpr{
			Params: n.Params,
			Body:   inlineLetExprs(n.Body, lambdaBindings),
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

func isClosedLiteralValue(node Node) bool {
	switch n := node.(type) {
	case *intLit, *floatLit, *stringLit, *boolLit, *nilLit:
		return true
	case *arrayLit:
		for _, elem := range n.Elements {
			if !isClosedLiteralValue(elem) {
				return false
			}
		}
		return true
	case *mapLit:
		for i := range n.Keys {
			if !isClosedLiteralValue(n.Keys[i]) || !isClosedLiteralValue(n.Values[i]) {
				return false
			}
		}
		return true
	default:
		return false
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
			Name:         n.Name,
			Value:        copyNode(n.Value),
			Body:         copyNode(n.Body),
			ImplicitBody: n.ImplicitBody,
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
			Pattern:      pattern,
			IsArray:      n.IsArray,
			Value:        copyNode(n.Value),
			Body:         copyNode(n.Body),
			ImplicitBody: n.ImplicitBody,
		}
	default:
		return node
	}
}

// ASTToString converts an AST node back to an expression string.
func ASTToString(node Node) string {
	switch n := node.(type) {
	case *intLit:
		if n.Value == math.MinInt64 {
			return "-9223372036854775807 - 1"
		}
		return fmt.Sprintf("%d", n.Value)

	case *floatLit:
		switch {
		case math.IsNaN(n.Value):
			return "sqrt(-1.0)"
		case math.IsInf(n.Value, 1):
			return "1e308 * 1e308"
		case math.IsInf(n.Value, -1):
			return "-1e308 * 1e308"
		}
		text := strconv.FormatFloat(n.Value, 'g', -1, 64)
		if !strings.ContainsAny(text, ".eE") {
			text += ".0"
		}
		return text

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
		_, nestedUnary := n.Operand.(*unaryExpr)
		if needsParens(n.Operand) || nestedUnary {
			operand = "(" + operand + ")"
		}
		if n.Op == kwNot {
			return "not " + operand
		}
		return n.Op + operand

	case *ternaryExpr:
		cond := ASTToString(n.Cond)
		if needsParens(n.Cond) {
			cond = "(" + cond + ")"
		}
		return fmt.Sprintf("%s ? %s : %s",
			cond,
			ASTToString(n.Then),
			ASTToString(n.Else))

	case *coalesceExpr:
		left, right := ASTToString(n.Left), ASTToString(n.Right)
		if needsParens(n.Left) {
			left = "(" + left + ")"
		}
		if needsParens(n.Right) {
			right = "(" + right + ")"
		}
		return fmt.Sprintf("%s ?? %s", left, right)

	case *callExpr:
		args := make([]string, len(n.Args))
		for i, arg := range n.Args {
			args[i] = ASTToString(arg)
		}
		return fmt.Sprintf("%s(%s)", postfixOperand(n.Func), strings.Join(args, ", "))

	case *memberExpr:
		obj := postfixOperand(n.Object)
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
		obj := postfixOperand(n.Object)
		idx := ASTToString(n.Index)
		if n.Optional {
			return fmt.Sprintf("%s?.[%s]", obj, idx)
		}
		return fmt.Sprintf("%s[%s]", obj, idx)

	case *sliceExpr:
		obj := postfixOperand(n.Object)
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
			if _, plain := n.Keys[i].(*stringLit); !plain {
				key = "[" + key + "]"
			}
			val := ASTToString(n.Values[i])
			pairs[i] = key + ": " + val
		}
		return "{" + strings.Join(pairs, ", ") + "}"

	case *rangeExpr:
		start, end := ASTToString(n.Start), ASTToString(n.End)
		if needsParens(n.Start) {
			start = "(" + start + ")"
		}
		if needsParens(n.End) {
			end = "(" + end + ")"
		}
		return fmt.Sprintf("(%s..%s)", start, end)

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
		if n.ImplicitBody {
			return fmt.Sprintf("let %s = %s", n.Name, ASTToString(n.Value))
		}
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
		terminator := "; " + ASTToString(n.Body)
		if n.ImplicitBody {
			terminator = ""
		}
		if n.IsArray {
			return fmt.Sprintf("let [%s] = %s%s",
				strings.Join(fields, ", "),
				ASTToString(n.Value), terminator)
		}
		return fmt.Sprintf("let {%s} = %s%s",
			strings.Join(fields, ", "),
			ASTToString(n.Value), terminator)

	default:
		return fmt.Sprintf("<%T>", node)
	}
}

// needsParens determines if an expression needs parentheses when used as operand.
func needsParens(node Node) bool {
	switch n := node.(type) {
	case *intLit:
		return n.Value == math.MinInt64
	case *binaryExpr, *unaryExpr, *ternaryExpr, *coalesceExpr, *letExpr, *destructureExpr, *lambdaExpr, *pipeExpr:
		return true
	}
	return false
}

func postfixOperand(node Node) string {
	text := ASTToString(node)
	if needsParens(node) {
		return "(" + text + ")"
	}
	return text
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
