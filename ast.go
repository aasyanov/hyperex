package hyperex

import "fmt"

// Node is the interface implemented by all AST nodes.
type Node interface {
	node()
	String() string
}

// Expression nodes
type (
	// intLit represents an integer literal.
	intLit struct {
		Value int64
	}

	// floatLit represents a floating-point literal.
	floatLit struct {
		Value float64
	}

	// stringLit represents a string literal.
	stringLit struct {
		Value string
	}

	// boolLit represents a boolean literal.
	boolLit struct {
		Value bool
	}

	// nilLit represents a nil/null literal.
	nilLit struct{}

	// ident represents an identifier (variable reference).
	ident struct {
		Name string
	}

	// binaryExpr represents a binary operation (e.g., a + b).
	binaryExpr struct {
		Op    string
		Left  Node
		Right Node
	}

	// unaryExpr represents a unary operation (e.g., -x, !x).
	unaryExpr struct {
		Op      string
		Operand Node
	}

	// ternaryExpr represents a conditional expression (cond ? then : else).
	ternaryExpr struct {
		Cond Node
		Then Node
		Else Node
	}

	// callExpr represents a function call (e.g., func(arg1, arg2)).
	callExpr struct {
		Func Node
		Args []Node
	}

	// memberExpr represents member access (e.g., obj.field or obj["key"]).
	memberExpr struct {
		Object   Node
		Property Node
		Computed bool // true for obj[key], false for obj.key
		Optional bool // true for obj?.field or obj?.[key]
	}

	// indexExpr represents array/map index access (e.g., arr[0]).
	indexExpr struct {
		Object   Node
		Index    Node
		Optional bool // true for arr?.[0]
	}

	// sliceExpr represents slice operation (e.g., arr[1:3] or arr[::2]).
	sliceExpr struct {
		Object Node
		Low    Node // nil means 0
		High   Node // nil means len
		Step   Node // nil means 1
	}

	// arrayLit represents an array literal [a, b, c].
	arrayLit struct {
		Elements []Node
	}

	// mapLit represents a map/object literal {key: value, ...}.
	mapLit struct {
		Keys   []Node
		Values []Node
	}

	// rangeExpr represents a range expression (start..end).
	rangeExpr struct {
		Start Node
		End   Node
	}

	// pipeExpr represents a pipe operation (arr | filter(x > 0)).
	pipeExpr struct {
		Left  Node
		Right Node
	}

	// lambdaExpr represents an anonymous function (x => x * 2).
	lambdaExpr struct {
		Params []string
		Body   Node
	}

	// coalesceExpr represents nil-coalescing (a ?? b).
	coalesceExpr struct {
		Left  Node
		Right Node
	}

	// letExpr represents a let binding (let x = 1; expr).
	letExpr struct {
		Name  string
		Value Node
		Body  Node
	}

	// spreadExpr represents a spread expression (...arr).
	spreadExpr struct {
		Expr Node
	}

	// destructureExpr represents destructuring assignment.
	// let {a, b} = obj or let [a, b] = arr
	destructureExpr struct {
		Pattern []destructureField // Fields to extract
		IsArray bool               // true for array destructuring
		Value   Node               // Source expression
		Body    Node               // Body expression
	}

	// destructureField represents a single field in destructuring.
	destructureField struct {
		Key     string // Original key name (or empty for array)
		Name    string // Variable name to bind
		Default Node   // Default value if nil/missing
	}
)

// Node interface implementations (marker methods)
func (*intLit) node()          { _ = struct{}{} }
func (*floatLit) node()        { _ = struct{}{} }
func (*stringLit) node()       { _ = struct{}{} }
func (*boolLit) node()         { _ = struct{}{} }
func (*nilLit) node()          { _ = struct{}{} }
func (*ident) node()           { _ = struct{}{} }
func (*binaryExpr) node()      { _ = struct{}{} }
func (*unaryExpr) node()       { _ = struct{}{} }
func (*ternaryExpr) node()     { _ = struct{}{} }
func (*callExpr) node()        { _ = struct{}{} }
func (*memberExpr) node()      { _ = struct{}{} }
func (*indexExpr) node()       { _ = struct{}{} }
func (*sliceExpr) node()       { _ = struct{}{} }
func (*arrayLit) node()        { _ = struct{}{} }
func (*mapLit) node()          { _ = struct{}{} }
func (*rangeExpr) node()       { _ = struct{}{} }
func (*pipeExpr) node()        { _ = struct{}{} }
func (*lambdaExpr) node()      { _ = struct{}{} }
func (*coalesceExpr) node()    { _ = struct{}{} }
func (*letExpr) node()         { _ = struct{}{} }
func (*spreadExpr) node()      { _ = struct{}{} }
func (*destructureExpr) node() { _ = struct{}{} }

// String implementations for debugging
func (n *intLit) String() string       { return fmt.Sprintf("%d", n.Value) }
func (n *floatLit) String() string     { return fmt.Sprintf("%g", n.Value) }
func (n *stringLit) String() string    { return fmt.Sprintf("%q", n.Value) }
func (n *boolLit) String() string      { return fmt.Sprintf("%t", n.Value) }
func (n *nilLit) String() string       { return litNil }
func (n *ident) String() string        { return n.Name }
func (n *binaryExpr) String() string   { return fmt.Sprintf("(%s %s %s)", n.Left, n.Op, n.Right) }
func (n *unaryExpr) String() string    { return fmt.Sprintf("(%s%s)", n.Op, n.Operand) }
func (n *ternaryExpr) String() string  { return fmt.Sprintf("(%s ? %s : %s)", n.Cond, n.Then, n.Else) }
func (n *callExpr) String() string     { return fmt.Sprintf("%s(%v)", n.Func, n.Args) }
func (n *memberExpr) String() string   { return fmt.Sprintf("%s.%s", n.Object, n.Property) }
func (n *indexExpr) String() string    { return fmt.Sprintf("%s[%s]", n.Object, n.Index) }
func (n *sliceExpr) String() string    { return fmt.Sprintf("%s[%s:%s]", n.Object, n.Low, n.High) }
func (n *arrayLit) String() string     { return fmt.Sprintf("%v", n.Elements) }
func (n *mapLit) String() string       { return fmt.Sprintf("{map: %d keys}", len(n.Keys)) }
func (n *rangeExpr) String() string    { return fmt.Sprintf("%s..%s", n.Start, n.End) }
func (n *pipeExpr) String() string     { return fmt.Sprintf("%s | %s", n.Left, n.Right) }
func (n *lambdaExpr) String() string   { return fmt.Sprintf("(%v => %s)", n.Params, n.Body) }
func (n *coalesceExpr) String() string { return fmt.Sprintf("(%s ?? %s)", n.Left, n.Right) }
func (n *letExpr) String() string      { return fmt.Sprintf("(let %s = %s; %s)", n.Name, n.Value, n.Body) }
func (n *spreadExpr) String() string   { return fmt.Sprintf("...%s", n.Expr) }
func (n *destructureExpr) String() string {
	if n.IsArray {
		return fmt.Sprintf("(let [...] = %s; %s)", n.Value, n.Body)
	}
	return fmt.Sprintf("(let {...} = %s; %s)", n.Value, n.Body)
}

// Walk traverses an AST in depth-first order.
// The visitor function is called for each node.
// If the visitor returns false, the subtree is not traversed.
func Walk(node Node, visitor func(Node) bool) {
	if !visitor(node) {
		return
	}

	switch n := node.(type) {
	case *binaryExpr:
		Walk(n.Left, visitor)
		Walk(n.Right, visitor)
	case *unaryExpr:
		Walk(n.Operand, visitor)
	case *ternaryExpr:
		Walk(n.Cond, visitor)
		Walk(n.Then, visitor)
		Walk(n.Else, visitor)
	case *callExpr:
		Walk(n.Func, visitor)
		for _, arg := range n.Args {
			Walk(arg, visitor)
		}
	case *memberExpr:
		Walk(n.Object, visitor)
		Walk(n.Property, visitor)
	case *indexExpr:
		Walk(n.Object, visitor)
		Walk(n.Index, visitor)
	case *sliceExpr:
		Walk(n.Object, visitor)
		if n.Low != nil {
			Walk(n.Low, visitor)
		}
		if n.High != nil {
			Walk(n.High, visitor)
		}
		if n.Step != nil {
			Walk(n.Step, visitor)
		}
	case *arrayLit:
		for _, elem := range n.Elements {
			Walk(elem, visitor)
		}
	case *mapLit:
		for i := range n.Keys {
			Walk(n.Keys[i], visitor)
			Walk(n.Values[i], visitor)
		}
	case *rangeExpr:
		Walk(n.Start, visitor)
		Walk(n.End, visitor)
	case *pipeExpr:
		Walk(n.Left, visitor)
		Walk(n.Right, visitor)
	case *lambdaExpr:
		Walk(n.Body, visitor)
	case *coalesceExpr:
		Walk(n.Left, visitor)
		Walk(n.Right, visitor)
	case *letExpr:
		Walk(n.Value, visitor)
		Walk(n.Body, visitor)
	case *spreadExpr:
		Walk(n.Expr, visitor)
	case *destructureExpr:
		Walk(n.Value, visitor)
		for _, f := range n.Pattern {
			if f.Default != nil {
				Walk(f.Default, visitor)
			}
		}
		Walk(n.Body, visitor)
	}
}

// Transform applies a transformation function to each node in the AST.
// It returns a new AST with transformed nodes.
func Transform(node Node, fn func(Node) Node) Node {
	switch n := node.(type) {
	case *binaryExpr:
		return fn(&binaryExpr{
			Op:    n.Op,
			Left:  Transform(n.Left, fn),
			Right: Transform(n.Right, fn),
		})
	case *unaryExpr:
		return fn(&unaryExpr{
			Op:      n.Op,
			Operand: Transform(n.Operand, fn),
		})
	case *ternaryExpr:
		return fn(&ternaryExpr{
			Cond: Transform(n.Cond, fn),
			Then: Transform(n.Then, fn),
			Else: Transform(n.Else, fn),
		})
	case *callExpr:
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = Transform(arg, fn)
		}
		return fn(&callExpr{
			Func: Transform(n.Func, fn),
			Args: args,
		})
	case *memberExpr:
		return fn(&memberExpr{
			Object:   Transform(n.Object, fn),
			Property: Transform(n.Property, fn),
			Computed: n.Computed,
			Optional: n.Optional,
		})
	case *indexExpr:
		return fn(&indexExpr{
			Object:   Transform(n.Object, fn),
			Index:    Transform(n.Index, fn),
			Optional: n.Optional,
		})
	case *sliceExpr:
		var low, high, step Node
		if n.Low != nil {
			low = Transform(n.Low, fn)
		}
		if n.High != nil {
			high = Transform(n.High, fn)
		}
		if n.Step != nil {
			step = Transform(n.Step, fn)
		}
		return fn(&sliceExpr{
			Object: Transform(n.Object, fn),
			Low:    low,
			High:   high,
			Step:   step,
		})
	case *arrayLit:
		elements := make([]Node, len(n.Elements))
		for i, elem := range n.Elements {
			elements[i] = Transform(elem, fn)
		}
		return fn(&arrayLit{Elements: elements})
	case *mapLit:
		keys := make([]Node, len(n.Keys))
		values := make([]Node, len(n.Values))
		for i := range n.Keys {
			keys[i] = Transform(n.Keys[i], fn)
			values[i] = Transform(n.Values[i], fn)
		}
		return fn(&mapLit{Keys: keys, Values: values})
	case *rangeExpr:
		return fn(&rangeExpr{
			Start: Transform(n.Start, fn),
			End:   Transform(n.End, fn),
		})
	case *pipeExpr:
		return fn(&pipeExpr{
			Left:  Transform(n.Left, fn),
			Right: Transform(n.Right, fn),
		})
	case *lambdaExpr:
		return fn(&lambdaExpr{
			Params: n.Params,
			Body:   Transform(n.Body, fn),
		})
	case *coalesceExpr:
		return fn(&coalesceExpr{
			Left:  Transform(n.Left, fn),
			Right: Transform(n.Right, fn),
		})
	case *letExpr:
		return fn(&letExpr{
			Name:  n.Name,
			Value: Transform(n.Value, fn),
			Body:  Transform(n.Body, fn),
		})
	case *spreadExpr:
		return fn(&spreadExpr{
			Expr: Transform(n.Expr, fn),
		})
	case *destructureExpr:
		pattern := make([]destructureField, len(n.Pattern))
		for i, f := range n.Pattern {
			pattern[i] = destructureField{
				Key:  f.Key,
				Name: f.Name,
			}
			if f.Default != nil {
				pattern[i].Default = Transform(f.Default, fn)
			}
		}
		return fn(&destructureExpr{
			Pattern: pattern,
			IsArray: n.IsArray,
			Value:   Transform(n.Value, fn),
			Body:    Transform(n.Body, fn),
		})
	default:
		return fn(node)
	}
}
