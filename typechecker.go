package hyperex

import (
	"fmt"
	"reflect"
	"strings"
)

// Type represents a type in the HyperEx type system.
type Type int

// Type enumerates HyperEx static types used by the type checker.
const (
	TypeUnknown Type = iota
	TypeNil
	TypeBool
	TypeInt
	TypeFloat
	TypeString
	TypeArray
	TypeMap
	TypeFunc
	TypeAny // For dynamic/unknown types
)

func (t Type) String() string {
	switch t {
	case TypeNil:
		return litNil
	case TypeBool:
		return nameBool
	case TypeInt:
		return nameInt
	case TypeFloat:
		return nameFloat
	case TypeString:
		return nameString
	case TypeArray:
		return typeNameArray
	case TypeMap:
		return nameMap
	case TypeFunc:
		return nameFunc
	case TypeAny:
		return nameAny
	default:
		return typeNameUnknown
	}
}

// FuncSig represents a function signature for type checking.
type FuncSig struct {
	Params   []Type
	Returns  Type // Return type
	Variadic bool
	// MinArgs/MaxArgs allow expressing functions with optional args (e.g. 2..3 args).
	// If both are 0, they are derived from Params/Variadic:
	// - Variadic: MinArgs=len(Params)-1, MaxArgs=-1 (unbounded)
	// - Non-variadic: MinArgs=len(Params), MaxArgs=len(Params)
	MinArgs int
	MaxArgs int // -1 = unbounded
}

// TypeChecker performs static type analysis on expressions.
type TypeChecker struct {
	env    map[string]Type
	funcs  map[string]FuncSig
	errors []error
}

// NewTypeChecker creates a new type checker.
func NewTypeChecker() *TypeChecker {
	return &TypeChecker{
		env:   make(map[string]Type),
		funcs: defaultFuncSigs(),
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

	if errs := tc.Check(ast); errs != nil && len(errs.Errors) > 0 {
		return errs
	}

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

// WithVar registers a variable with a known type.
func (tc *TypeChecker) WithVar(name string, t Type) *TypeChecker {
	tc.env[name] = t
	return tc
}

// WithFunc registers a function signature for type checking.
func (tc *TypeChecker) WithFunc(name string, sig FuncSig) *TypeChecker {
	tc.funcs[name] = sig
	return tc
}

// WithEnvTypes registers multiple types at once.
func (tc *TypeChecker) WithEnvTypes(types map[string]any) *TypeChecker {
	for name, val := range types {
		tc.env[name] = tc.goTypeToHyperexType(reflect.TypeOf(val))
	}
	return tc
}

// WithStructEnv registers a struct's fields as environment types.
func (tc *TypeChecker) WithStructEnv(val any) *TypeChecker {
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return tc
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue // Skip unexported fields
		}
		name := field.Name
		// Check for expr tag first, then json tag
		if tag := field.Tag.Get("expr"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		} else if tag := field.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}
		tc.env[name] = tc.goTypeToHyperexType(field.Type)
	}
	return tc
}

// cloneForCheck returns a TypeChecker safe for a single Check call.
// env is copied (let-bindings mutate it); funcs is shared (read-only after Engine construction).
func (tc *TypeChecker) cloneForCheck() *TypeChecker {
	if tc == nil {
		return nil
	}
	env := make(map[string]Type, len(tc.env))
	for k, v := range tc.env {
		env[k] = v
	}
	return &TypeChecker{
		env:   env,
		funcs: tc.funcs,
	}
}

// Check performs type checking on an AST.
func (tc *TypeChecker) Check(node Node) *TypeCheckErrors {
	tc.errors = nil
	tc.check(node)
	if len(tc.errors) > 0 {
		return &TypeCheckErrors{Errors: tc.errors}
	}
	return nil
}

// Infer returns the inferred type of an expression.
func (tc *TypeChecker) Infer(node Node) Type {
	return tc.inferType(node)
}

func (tc *TypeChecker) check(node Node) {
	switch n := node.(type) {
	case *binaryExpr:
		tc.checkBinary(n)
	case *unaryExpr:
		tc.checkUnary(n)
	case *ternaryExpr:
		tc.check(n.Cond)
		condType := tc.inferType(n.Cond)
		if condType != TypeBool && condType != TypeAny && condType != TypeUnknown {
			tc.addError(&TypeError{
				Message:  "ternary condition must be boolean",
				Expected: TypeBool.String(),
				Got:      condType.String(),
				Node:     n.Cond,
			})
		}
		tc.check(n.Then)
		tc.check(n.Else)
	case *callExpr:
		tc.checkCall(n)
	case *memberExpr:
		tc.check(n.Object)
		if n.Computed {
			tc.check(n.Property)
		}
	case *indexExpr:
		tc.check(n.Object)
		tc.check(n.Index)
		objType := tc.inferType(n.Object)
		indexType := tc.inferType(n.Index)
		if objType == TypeString || objType == TypeArray {
			if indexType != TypeInt && indexType != TypeAny && indexType != TypeUnknown {
				tc.addError(&TypeError{
					Message:  "index must be integer for string/array",
					Expected: TypeInt.String(),
					Got:      indexType.String(),
					Node:     n.Index,
				})
			}
		}
	case *sliceExpr:
		tc.check(n.Object)
		if n.Low != nil {
			tc.check(n.Low)
		}
		if n.High != nil {
			tc.check(n.High)
		}
		if n.Step != nil {
			tc.check(n.Step)
		}
	case *arrayLit:
		for _, elem := range n.Elements {
			tc.check(elem)
		}
	case *mapLit:
		for i := range n.Keys {
			tc.check(n.Keys[i])
			tc.check(n.Values[i])
		}
	case *rangeExpr:
		tc.check(n.Start)
		tc.check(n.End)
		startType := tc.inferType(n.Start)
		endType := tc.inferType(n.End)
		if startType != TypeInt && startType != TypeAny && startType != TypeUnknown {
			tc.addError(&TypeError{
				Message:  "range start must be integer",
				Expected: TypeInt.String(),
				Got:      startType.String(),
				Node:     n.Start,
			})
		}
		if endType != TypeInt && endType != TypeAny && endType != TypeUnknown {
			tc.addError(&TypeError{
				Message:  "range end must be integer",
				Expected: TypeInt.String(),
				Got:      endType.String(),
				Node:     n.End,
			})
		}
	case *pipeExpr:
		tc.check(n.Left)
		tc.check(n.Right)
	case *lambdaExpr:
		tc.check(n.Body)
	case *letExpr:
		tc.check(n.Value)
		// Register the variable type in environment temporarily
		oldType, hadType := tc.env[n.Name]
		tc.env[n.Name] = tc.inferType(n.Value)
		tc.check(n.Body)
		// Restore old type if it existed
		if hadType {
			tc.env[n.Name] = oldType
		} else {
			delete(tc.env, n.Name)
		}
	case *coalesceExpr:
		tc.check(n.Left)
		tc.check(n.Right)
	case *spreadExpr:
		tc.check(n.Expr)
		exprType := tc.inferType(n.Expr)
		if exprType != TypeArray && exprType != TypeAny && exprType != TypeUnknown {
			tc.addError(&TypeError{
				Message:  "spread operator requires array",
				Expected: TypeArray.String(),
				Got:      exprType.String(),
				Node:     n.Expr,
			})
		}
	case *destructureExpr:
		tc.check(n.Value)
		valueType := tc.inferType(n.Value)
		if n.IsArray {
			if valueType != TypeArray && valueType != TypeAny && valueType != TypeUnknown {
				tc.addError(&TypeError{
					Message:  "array destructuring requires array value",
					Expected: TypeArray.String(),
					Got:      valueType.String(),
					Node:     n.Value,
				})
			}
		} else {
			if valueType != TypeMap && valueType != TypeAny && valueType != TypeUnknown {
				tc.addError(&TypeError{
					Message:  "object destructuring requires map value",
					Expected: TypeMap.String(),
					Got:      valueType.String(),
					Node:     n.Value,
				})
			}
		}
		// Check default values and register variables
		for _, field := range n.Pattern {
			if field.Default != nil {
				tc.check(field.Default)
			}
			// Register destructured variable with unknown type (could be refined)
			tc.env[field.Name] = TypeAny
		}
		tc.check(n.Body)
	}
}

func (tc *TypeChecker) checkBinary(n *binaryExpr) {
	tc.check(n.Left)
	tc.check(n.Right)

	leftType := tc.inferType(n.Left)
	rightType := tc.inferType(n.Right)

	// Skip type checking for any/unknown types
	if leftType == TypeAny || leftType == TypeUnknown ||
		rightType == TypeAny || rightType == TypeUnknown {
		return
	}

	switch n.Op {
	case "+":
		// + works on numbers, strings, and arrays
		if !tc.isNumeric(leftType) && leftType != TypeString && leftType != TypeArray {
			tc.addError(&TypeError{
				Message: "+ requires numeric, string, or array operands",
				Got:     leftType.String(),
				Node:    n.Left,
			})
		}
	case "-", "*", "/", "%", "**":
		if !tc.isNumeric(leftType) {
			tc.addError(&TypeError{
				Message: n.Op + " requires numeric operands",
				Got:     leftType.String(),
				Node:    n.Left,
			})
		}
		if !tc.isNumeric(rightType) {
			tc.addError(&TypeError{
				Message: n.Op + " requires numeric operands",
				Got:     rightType.String(),
				Node:    n.Right,
			})
		}
	case "<", "<=", ">", ">=":
		if !tc.isComparable(leftType, rightType) {
			tc.addError(&TypeError{
				Message: "comparison requires comparable types",
				Got:     leftType.String(),
				Node:    n,
			})
		}
	case "&&", "||", kwAnd, "or":
		// Logical operators require boolean operands
		if leftType != TypeBool {
			tc.addError(&TypeError{
				Message:  "logical operator requires boolean operands",
				Expected: TypeBool.String(),
				Got:      leftType.String(),
				Node:     n.Left,
			})
		}
		if rightType != TypeBool {
			tc.addError(&TypeError{
				Message:  "logical operator requires boolean operands",
				Expected: TypeBool.String(),
				Got:      rightType.String(),
				Node:     n.Right,
			})
		}
	case "&", "|", "^", "<<", ">>", litOpShrU:
		if leftType != TypeInt {
			tc.addError(&TypeError{
				Message:  "bitwise operators require integer operands",
				Expected: TypeInt.String(),
				Got:      leftType.String(),
				Node:     n.Left,
			})
		}
		if rightType != TypeInt {
			tc.addError(&TypeError{
				Message:  "bitwise operators require integer operands",
				Expected: TypeInt.String(),
				Got:      rightType.String(),
				Node:     n.Right,
			})
		}
	case "in":
		if rightType != TypeArray && rightType != TypeMap && rightType != TypeString {
			tc.addError(&TypeError{
				Message: "'in' requires array, map, or string on right side",
				Got:     rightType.String(),
				Node:    n.Right,
			})
		}
	case "~=":
		if leftType != TypeString {
			tc.addError(&TypeError{
				Message:  "regex match requires string operand",
				Expected: TypeString.String(),
				Got:      leftType.String(),
				Node:     n.Left,
			})
		}
		if rightType != TypeString {
			tc.addError(&TypeError{
				Message:  "regex pattern must be string",
				Expected: TypeString.String(),
				Got:      rightType.String(),
				Node:     n.Right,
			})
		}
	}
}

func (tc *TypeChecker) checkUnary(n *unaryExpr) {
	tc.check(n.Operand)
	operandType := tc.inferType(n.Operand)

	if operandType == TypeAny || operandType == TypeUnknown {
		return
	}

	switch n.Op {
	case "-":
		if !tc.isNumeric(operandType) {
			tc.addError(&TypeError{
				Message: "unary - requires numeric operand",
				Got:     operandType.String(),
				Node:    n.Operand,
			})
		}
	case "!", kwNot:
		if operandType != TypeBool {
			tc.addError(&TypeError{
				Message:  "logical not requires boolean operand",
				Expected: TypeBool.String(),
				Got:      operandType.String(),
				Node:     n.Operand,
			})
		}
	case "~":
		if operandType != TypeInt {
			tc.addError(&TypeError{
				Message:  "bitwise not requires integer operand",
				Expected: TypeInt.String(),
				Got:      operandType.String(),
				Node:     n.Operand,
			})
		}
	}
}

func (tc *TypeChecker) checkCall(n *callExpr) {
	for _, arg := range n.Args {
		tc.check(arg)
	}

	// Check function type if it's an identifier
	if ident, ok := n.Func.(*ident); ok {
		if sig, ok := tc.funcs[ident.Name]; ok {
			minArgs, maxArgs := sig.MinArgs, sig.MaxArgs
			if minArgs == 0 && maxArgs == 0 {
				if sig.Variadic {
					minArgs = len(sig.Params) - 1
					maxArgs = -1
				} else {
					minArgs = len(sig.Params)
					maxArgs = len(sig.Params)
				}
			}
			if len(n.Args) < minArgs {
				tc.addError(&TypeError{
					Message: fmt.Sprintf("%s requires at least %d arguments", ident.Name, minArgs),
					Node:    n,
				})
				return
			}
			if maxArgs >= 0 && len(n.Args) > maxArgs {
				tc.addError(&TypeError{
					Message: fmt.Sprintf("%s requires at most %d arguments", ident.Name, maxArgs),
					Node:    n,
				})
				return
			}
		}
	}
}

func (tc *TypeChecker) inferType(node Node) Type {
	switch n := node.(type) {
	case *intLit:
		return TypeInt
	case *floatLit:
		return TypeFloat
	case *stringLit:
		return TypeString
	case *boolLit:
		return TypeBool
	case *nilLit:
		return TypeNil
	case *arrayLit:
		return TypeArray
	case *mapLit:
		return TypeMap
	case *ident:
		if t, ok := tc.env[n.Name]; ok {
			return t
		}
		if _, ok := tc.funcs[n.Name]; ok {
			return TypeFunc
		}
		return TypeUnknown
	case *binaryExpr:
		return tc.inferBinaryType(n)
	case *unaryExpr:
		return tc.inferUnaryType(n)
	case *ternaryExpr:
		thenType := tc.inferType(n.Then)
		elseType := tc.inferType(n.Else)
		if thenType == elseType {
			return thenType
		}
		// Mixed numeric types promote to float
		if (thenType == TypeInt && elseType == TypeFloat) ||
			(thenType == TypeFloat && elseType == TypeInt) {
			return TypeFloat
		}
		return TypeAny
	case *callExpr:
		if ident, ok := n.Func.(*ident); ok {
			if sig, ok := tc.funcs[ident.Name]; ok {
				return sig.Returns
			}
		}
		return TypeAny
	case *memberExpr:
		return TypeAny // Could be refined with struct type info
	case *indexExpr:
		objType := tc.inferType(n.Object)
		if objType == TypeString {
			return TypeString
		}
		return TypeAny
	case *sliceExpr:
		return tc.inferType(n.Object)
	case *rangeExpr:
		return TypeArray
	case *pipeExpr:
		return TypeAny // Result depends on function
	case *lambdaExpr:
		return TypeFunc
	case *letExpr:
		// Temporarily add the bound variable to the environment for inference
		valueType := tc.inferType(n.Value)
		oldType, hadType := tc.env[n.Name]
		tc.env[n.Name] = valueType
		bodyType := tc.inferType(n.Body)
		// Restore old type if it existed
		if hadType {
			tc.env[n.Name] = oldType
		} else {
			delete(tc.env, n.Name)
		}
		return bodyType
	case *coalesceExpr:
		leftType := tc.inferType(n.Left)
		rightType := tc.inferType(n.Right)
		// For nil coalescing: if left is nil, return right's type
		// If left is known non-nil, return left's type
		// Otherwise if both types match, return that type
		if leftType == TypeNil {
			return rightType
		}
		if leftType == rightType {
			return leftType
		}
		// If types differ but left could be nil at runtime, prefer right's type
		return rightType
	case *spreadExpr:
		return TypeArray
	case *destructureExpr:
		return tc.inferType(n.Body)
	default:
		return TypeUnknown
	}
}

func (tc *TypeChecker) inferBinaryType(n *binaryExpr) Type {
	leftType := tc.inferType(n.Left)
	rightType := tc.inferType(n.Right)

	switch n.Op {
	case "+":
		if leftType == TypeString || rightType == TypeString {
			return TypeString
		}
		if leftType == TypeArray && rightType == TypeArray {
			return TypeArray
		}
		if leftType == TypeInt && rightType == TypeInt {
			return TypeInt
		}
		return TypeFloat
	case "-", "*", "**":
		if leftType == TypeInt && rightType == TypeInt {
			return TypeInt
		}
		return TypeFloat
	case "/":
		// Integer division returns int, mixed/float division returns float
		if leftType == TypeInt && rightType == TypeInt {
			return TypeInt
		}
		return TypeFloat
	case "%":
		if leftType == TypeInt && rightType == TypeInt {
			return TypeInt
		}
		return TypeFloat
	case "==", "!=", "<", "<=", ">", ">=", "&&", "||", "in", opNotIn, "~=":
		return TypeBool
	case "&", "|", "^", "<<", ">>", litOpShrU:
		return TypeInt
	default:
		return TypeAny
	}
}

func (tc *TypeChecker) inferUnaryType(n *unaryExpr) Type {
	switch n.Op {
	case "-":
		operandType := tc.inferType(n.Operand)
		if operandType == TypeInt {
			return TypeInt
		}
		return TypeFloat
	case "!":
		return TypeBool
	case "~":
		return TypeInt
	default:
		return TypeAny
	}
}

func (tc *TypeChecker) isNumeric(t Type) bool {
	return t == TypeInt || t == TypeFloat
}

func (tc *TypeChecker) isComparable(a, b Type) bool {
	if a == TypeAny || b == TypeAny || a == TypeUnknown || b == TypeUnknown {
		return true
	}
	if tc.isNumeric(a) && tc.isNumeric(b) {
		return true
	}
	return a == b
}

func (tc *TypeChecker) addError(err error) {
	tc.errors = append(tc.errors, err)
}

func (tc *TypeChecker) goTypeToHyperexType(t reflect.Type) Type {
	if t == nil {
		return TypeNil
	}

	switch t.Kind() {
	case reflect.Bool:
		return TypeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return TypeInt
	case reflect.Float32, reflect.Float64:
		return TypeFloat
	case reflect.String:
		return TypeString
	case reflect.Slice, reflect.Array:
		return TypeArray
	case reflect.Map:
		return TypeMap
	case reflect.Func:
		return TypeFunc
	case reflect.Pointer:
		return tc.goTypeToHyperexType(t.Elem())
	default:
		return TypeAny
	}
}

// defaultFuncSigs returns type signatures for built-in functions.
func defaultFuncSigs() map[string]FuncSig {
	return map[string]FuncSig{
		// Math
		// abs returns int for int inputs (except MinInt64 -> float), float otherwise.
		// Use TypeAny to avoid lying to the checker.
		nameAbs: {Params: []Type{TypeAny}, Returns: TypeAny},
		nameMin: {Params: []Type{TypeAny}, Returns: TypeAny, Variadic: true},
		nameMax: {Params: []Type{TypeAny}, Returns: TypeAny, Variadic: true},
		// Note: runtime math builtins accept any numeric-ish value (they call Value.Float()).
		// We keep parameter types broad to avoid false positives under WithTypeCheck.
		// ceil/floor/round usually return int, but can return float for out-of-range values.
		nameCeil:  {Params: []Type{TypeAny}, Returns: TypeAny},
		nameFloor: {Params: []Type{TypeAny}, Returns: TypeAny},
		nameRound: {Params: []Type{TypeAny}, Returns: TypeAny},
		nameSqrt:  {Params: []Type{TypeAny}, Returns: TypeFloat},
		namePow:   {Params: []Type{TypeAny, TypeAny}, Returns: TypeFloat},
		nameLog:   {Params: []Type{TypeAny}, Returns: TypeFloat},
		nameExp:   {Params: []Type{TypeAny}, Returns: TypeFloat},
		nameSin:   {Params: []Type{TypeAny}, Returns: TypeFloat},
		nameCos:   {Params: []Type{TypeAny}, Returns: TypeFloat},

		// String
		nameLen:        {Params: []Type{TypeAny}, Returns: TypeInt},
		nameUpper:      {Params: []Type{TypeString}, Returns: TypeString},
		nameLower:      {Params: []Type{TypeString}, Returns: TypeString},
		nameTrim:       {Params: []Type{TypeString}, Returns: TypeString},
		nameTrimLeft:   {Params: []Type{TypeString, TypeString}, Returns: TypeString, MinArgs: 1, MaxArgs: 2},
		nameTrimRight:  {Params: []Type{TypeString, TypeString}, Returns: TypeString, MinArgs: 1, MaxArgs: 2},
		nameSplit:      {Params: []Type{TypeString, TypeString}, Returns: TypeArray},
		nameJoin:       {Params: []Type{TypeArray, TypeString}, Returns: TypeString},
		nameContains:   {Params: []Type{TypeString, TypeString}, Returns: TypeBool},
		nameStartsWith: {Params: []Type{TypeString, TypeString}, Returns: TypeBool},
		nameEndsWith:   {Params: []Type{TypeString, TypeString}, Returns: TypeBool},
		nameReplace:    {Params: []Type{TypeString, TypeString, TypeString}, Returns: TypeString},
		nameSubstr:     {Params: []Type{TypeString, TypeInt, TypeInt}, Returns: TypeString, MinArgs: 2, MaxArgs: 3},
		nameIndexOf:    {Params: []Type{TypeString, TypeString}, Returns: TypeInt},

		// Array
		nameFirst:   {Params: []Type{TypeArray}, Returns: TypeAny},
		nameLast:    {Params: []Type{TypeArray}, Returns: TypeAny},
		nameTake:    {Params: []Type{TypeArray, TypeInt}, Returns: TypeArray},
		nameSkip:    {Params: []Type{TypeArray, TypeInt}, Returns: TypeArray},
		nameReverse: {Params: []Type{TypeArray}, Returns: TypeArray},
		nameSort:    {Params: []Type{TypeArray}, Returns: TypeArray},
		nameUnique:  {Params: []Type{TypeArray}, Returns: TypeArray},
		nameConcat:  {Params: []Type{TypeArray}, Returns: TypeArray, Variadic: true},
		nameFlatten: {Params: []Type{TypeArray}, Returns: TypeArray},

		// Aggregation
		// sum can return int (all ints, no overflow) or float.
		nameSum:   {Params: []Type{TypeArray}, Returns: TypeAny},
		nameAvg:   {Params: []Type{TypeArray}, Returns: TypeFloat},
		nameCount: {Params: []Type{TypeAny}, Returns: TypeInt},
		// Optional predicate: all/any/none/one(arr) or (arr, pred) where pred is kindFunc.
		nameAll:  {Params: []Type{TypeArray, TypeFunc}, Returns: TypeBool, MinArgs: 1, MaxArgs: 2},
		nameAny:  {Params: []Type{TypeArray, TypeFunc}, Returns: TypeBool, MinArgs: 1, MaxArgs: 2},
		nameNone: {Params: []Type{TypeArray, TypeFunc}, Returns: TypeBool, MinArgs: 1, MaxArgs: 2},
		nameOne:  {Params: []Type{TypeArray, TypeFunc}, Returns: TypeBool, MinArgs: 1, MaxArgs: 2},

		// Higher-order
		nameMap:       {Params: []Type{TypeArray, TypeFunc}, Returns: TypeArray},
		nameFilter:    {Params: []Type{TypeArray, TypeFunc}, Returns: TypeArray},
		nameReduce:    {Params: []Type{TypeArray, TypeFunc, TypeAny}, Returns: TypeAny, MinArgs: 2, MaxArgs: 3},
		nameFind:      {Params: []Type{TypeArray, TypeFunc}, Returns: TypeAny},
		nameFindIndex: {Params: []Type{TypeArray, TypeFunc}, Returns: TypeInt},

		// Type
		nameInt:    {Params: []Type{TypeAny}, Returns: TypeInt},
		nameFloat:  {Params: []Type{TypeAny}, Returns: TypeFloat},
		nameString: {Params: []Type{TypeAny}, Returns: TypeString},
		nameBool:   {Params: []Type{TypeAny}, Returns: TypeBool},
		nameType:   {Params: []Type{TypeAny}, Returns: TypeString},
		nameIsNil:  {Params: []Type{TypeAny}, Returns: TypeBool},

		// Utility
		nameKeys:     {Params: []Type{TypeMap}, Returns: TypeArray},
		nameValues:   {Params: []Type{TypeMap}, Returns: TypeArray},
		nameClamp:    {Params: []Type{TypeAny, TypeAny, TypeAny}, Returns: TypeAny},
		nameBetween:  {Params: []Type{TypeAny, TypeAny, TypeAny}, Returns: TypeBool},
		nameHasKey:   {Params: []Type{TypeMap, TypeString}, Returns: TypeBool},
		nameEmpty:    {Params: []Type{TypeAny}, Returns: TypeBool},
		nameNotEmpty: {Params: []Type{TypeAny}, Returns: TypeBool},
		nameDefault:  {Params: []Type{TypeAny, TypeAny}, Returns: TypeAny},
		nameCoalesce: {Params: []Type{TypeAny}, Returns: TypeAny, Variadic: true},
		nameRepeat:   {Params: []Type{TypeAny, TypeInt}, Returns: TypeArray},
		nameRange:    {Params: []Type{TypeInt, TypeInt, TypeInt}, Returns: TypeArray, MinArgs: 2, MaxArgs: 3},
		nameZip:      {Params: []Type{TypeArray, TypeArray}, Returns: TypeArray, Variadic: true, MinArgs: 2, MaxArgs: -1},
		nameChunk:    {Params: []Type{TypeArray, TypeInt}, Returns: TypeArray},
		nameCompact:  {Params: []Type{TypeArray}, Returns: TypeArray},
		nameTruthy:   {Params: []Type{TypeArray}, Returns: TypeArray},
		namePluck:    {Params: []Type{TypeArray, TypeString}, Returns: TypeArray},
		nameGroupBy:  {Params: []Type{TypeArray, TypeFunc}, Returns: TypeArray},
		nameSortBy:   {Params: []Type{TypeArray, TypeFunc, TypeBool}, Returns: TypeArray, MinArgs: 2, MaxArgs: 3},
		nameToJSON:   {Params: []Type{TypeAny}, Returns: TypeString},
		nameFromJSON: {Params: []Type{TypeString}, Returns: TypeAny},

		// Time / duration (v1.1) — TypeAny keeps checker honest until dedicated TypeTime exists.
		nameTimeUTC:         {Params: []Type{TypeAny, TypeAny, TypeAny, TypeAny, TypeAny, TypeAny}, Returns: TypeAny},
		nameTimeUnix:        {Params: []Type{TypeAny}, Returns: TypeAny},
		nameTimeUnixNano:    {Params: []Type{TypeAny}, Returns: TypeAny},
		nameUnix:            {Params: []Type{TypeAny}, Returns: TypeInt},
		nameUnixNano:        {Params: []Type{TypeAny}, Returns: TypeInt},
		nameTimeYear:        {Params: []Type{TypeAny}, Returns: TypeInt},
		nameTimeMonth:       {Params: []Type{TypeAny}, Returns: TypeInt},
		nameTimeDay:         {Params: []Type{TypeAny}, Returns: TypeInt},
		nameTimeHour:        {Params: []Type{TypeAny}, Returns: TypeInt},
		nameTimeMinute:      {Params: []Type{TypeAny}, Returns: TypeInt},
		nameTimeSecond:      {Params: []Type{TypeAny}, Returns: TypeInt},
		nameTimeWeekday:     {Params: []Type{TypeAny}, Returns: TypeInt},
		nameTimeAdd:         {Params: []Type{TypeAny, TypeAny}, Returns: TypeAny},
		nameDurationSeconds: {Params: []Type{TypeAny}, Returns: TypeAny},
		nameDurationMs:      {Params: []Type{TypeAny}, Returns: TypeAny},
		nameDurationNs:      {Params: []Type{TypeAny}, Returns: TypeAny},
		nameDurSeconds:      {Params: []Type{TypeAny}, Returns: TypeFloat},
		nameDurMs:           {Params: []Type{TypeAny}, Returns: TypeInt},
		nameDurNs:           {Params: []Type{TypeAny}, Returns: TypeInt},

		// Quality (map-shaped {v,q})
		nameV:           {Params: []Type{TypeAny}, Returns: TypeAny},
		nameQ:           {Params: []Type{TypeAny}, Returns: TypeAny},
		nameIsGood:      {Params: []Type{TypeAny}, Returns: TypeBool},
		nameIsBad:       {Params: []Type{TypeAny}, Returns: TypeBool},
		nameIsUncertain: {Params: []Type{TypeAny}, Returns: TypeBool},
		nameWithQ:       {Params: []Type{TypeAny, TypeAny}, Returns: TypeMap},
	}
}
