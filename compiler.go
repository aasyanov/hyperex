package hyperex

import (
	"fmt"
	"reflect"
)

// opcode represents a bytecode instruction.
type opcode uint8

const (
	// Stack operations
	opNop  opcode = iota
	opPush        // Push constant onto stack
	opPop         // Pop top of stack
	opDup         // Duplicate top of stack

	// Load/Store
	opLoadVar    // Load variable by name
	opLoadEnv    // Load from environment by index
	opLoadConst  // Load constant by index
	opStoreVar   // Store variable by name
	opDeleteVar  // Delete variable by name (for cleanup)
	opSaveVar    // Save variable binding for lexical scope restoration
	opRestoreVar // Restore the most recently saved binding

	// Arithmetic
	opAdd // +
	opSub // -
	opMul // *
	opDiv // /
	opMod // %
	opPow // **
	opNeg // unary -

	// Comparison
	opEq   // ==
	opNe   // !=
	opLt   // <
	opLe   // <=
	opGt   // >
	opGe   // >=
	opIn   // in
	opLike // ~=

	// Logical
	opAnd // &&
	opOr  // ||
	opNot // !

	// Bitwise
	opBitAnd // &
	opBitOr  // |
	opBitXor // ^
	opBitNot // ~
	opShl    // <<
	opShr    // >>
	opShrU   // >>> (unsigned)

	// Control flow
	opJump     // Unconditional jump
	opJumpIf   // Jump if true
	opJumpNot  // Jump if false
	opCoalesce // ?? (nil coalescing)

	// Object operations
	opMember    // obj.field
	opIndex     // arr[idx]
	opSlice     // arr[low:high]
	opOptMember // obj?.field
	opOptIndex  // arr?.[idx]

	// Collection literals
	opArray // Create array from N elements
	opMap   // Create map from N key-value pairs

	// Function calls
	opCall   // Call function with N arguments
	opPipe   // Pipe operator |
	opLambda // Create lambda/closure

	// Special
	opRange  // start..end
	opTern   // ternary ?:
	opReturn // Return value
	opHalt   // End of program
	opSpread // Spread array elements
)

// instruction represents a single bytecode instruction.
// Op is a small integer; amd64 inserts pad before Arg (Sizeof 40 = budget ceiling).
type instruction struct {
	Op   opcode
	Arg  int    // Primary argument (index, count, offset)
	Arg2 int    // Secondary argument (for slice, etc.)
	Name string // For opLoadVar, opStoreVar, opMember
}

// bytecode represents compiled bytecode for an expression.
type bytecode struct {
	Instructions []instruction
	Constants    []Value
	Names        []string // Variable/function names
	Params       []string // Lambda parameter names (for closures)
}

// compiler compiles an AST to bytecode.
type compiler struct {
	code      []instruction
	constants []Value
	names     map[string]int
	nameList  []string

	// Constant deduplication using hash map for O(1) lookup
	constIndex map[constKey]int

	// Lambda handling
	lambdaParams []string

	// Operator overloading: maps operator to function name
	operators map[string]string

	// Counter for unique temp variable names (for destructuring)
	tempVarCounter int
}

// constKey is a hashable key for constant deduplication.
// Using separate fields for each type avoids collisions.
type constKey struct {
	kind   valueKind
	intVal int64
	strVal string
	ptrVal uintptr // For complex types that use ptr field
}

// makeConstKey creates a hashable key from a Value.
func makeConstKey(v Value) constKey {
	switch v.kind {
	case kindNil:
		return constKey{kind: kindNil}
	case kindBool, kindInt:
		return constKey{kind: v.kind, intVal: int64(v.bits)}
	case kindFloat:
		// Use int representation of float bits for hashing
		return constKey{kind: kindFloat, intVal: int64(v.bits)}
	case kindString:
		return constKey{kind: kindString, strVal: v.ptr.(string)}
	default:
		// For complex types (func, array, map, any), use pointer address for uniqueness.
		// This ensures each lambda/closure gets a unique constant slot.
		// We use reflect to get the actual pointer value from the interface.
		if v.ptr == nil {
			return constKey{kind: v.kind, ptrVal: 0}
		}
		return constKey{kind: v.kind, ptrVal: reflect.ValueOf(v.ptr).Pointer()}
	}
}

// newCompiler creates a new bytecode compiler.
func newCompiler() *compiler {
	return &compiler{
		code:       make([]instruction, 0, 64),
		constants:  make([]Value, 0, 16),
		names:      make(map[string]int),
		constIndex: make(map[constKey]int),
		operators:  make(map[string]string),
	}
}

// withOperators sets operator overloading map for the compiler.
func (c *compiler) withOperators(ops map[string]string) *compiler {
	c.operators = ops
	return c
}

// compileAST compiles an AST node to bytecode.
func (c *compiler) compileAST(node Node) (*bytecode, error) {
	if err := c.compile(node); err != nil {
		return nil, err
	}
	c.emit(opHalt, 0)

	return &bytecode{
		Instructions: c.code,
		Constants:    c.constants,
		Names:        c.nameList,
	}, nil
}

func (c *compiler) compile(node Node) error {
	switch n := node.(type) {
	case *intLit:
		idx := c.addConstant(Int(n.Value))
		c.emit(opLoadConst, idx)

	case *floatLit:
		idx := c.addConstant(Float(n.Value))
		c.emit(opLoadConst, idx)

	case *stringLit:
		idx := c.addConstant(String(n.Value))
		c.emit(opLoadConst, idx)

	case *boolLit:
		idx := c.addConstant(Bool(n.Value))
		c.emit(opLoadConst, idx)

	case *nilLit:
		idx := c.addConstant(Nil)
		c.emit(opLoadConst, idx)

	case *ident:
		// Check if it's a lambda parameter
		for i, p := range c.lambdaParams {
			if p == n.Name {
				c.emit(opLoadEnv, i)
				return nil
			}
		}
		c.emitName(opLoadVar, n.Name)

	case *binaryExpr:
		if err := c.compileBinary(n); err != nil {
			return err
		}

	case *unaryExpr:
		if err := c.compile(n.Operand); err != nil {
			return err
		}
		switch n.Op {
		case "-":
			c.emit(opNeg, 0)
		case "!":
			c.emit(opNot, 0)
		case "~":
			c.emit(opBitNot, 0)
		default:
			return &CompileError{Message: "unknown unary operator: " + n.Op, Node: n}
		}

	case *ternaryExpr:
		if err := c.compile(n.Cond); err != nil {
			return err
		}
		jumpElse := c.emitJump(opJumpNot)
		c.emit(opPop, 0) // Pop condition before then branch
		if err := c.compile(n.Then); err != nil {
			return err
		}
		jumpEnd := c.emitJump(opJump)
		c.patchJump(jumpElse)
		c.emit(opPop, 0) // Pop condition before else branch
		if err := c.compile(n.Else); err != nil {
			return err
		}
		c.patchJump(jumpEnd)

	case *coalesceExpr:
		if err := c.compile(n.Left); err != nil {
			return err
		}
		jumpEnd := c.emitJump(opCoalesce)
		c.emit(opPop, 0)
		if err := c.compile(n.Right); err != nil {
			return err
		}
		c.patchJump(jumpEnd)

	case *callExpr:
		// Compile arguments first
		for _, arg := range n.Args {
			if err := c.compile(arg); err != nil {
				return err
			}
		}
		// Compile function
		if err := c.compile(n.Func); err != nil {
			return err
		}
		c.emit(opCall, len(n.Args))

	case *memberExpr:
		if err := c.compile(n.Object); err != nil {
			return err
		}
		if n.Computed {
			if err := c.compile(n.Property); err != nil {
				return err
			}
			if n.Optional {
				c.emit(opOptIndex, 0)
			} else {
				c.emit(opIndex, 0)
			}
		} else {
			name := n.Property.(*ident).Name
			if n.Optional {
				c.emitName(opOptMember, name)
			} else {
				c.emitName(opMember, name)
			}
		}

	case *indexExpr:
		if err := c.compile(n.Object); err != nil {
			return err
		}
		if err := c.compile(n.Index); err != nil {
			return err
		}
		if n.Optional {
			c.emit(opOptIndex, 0)
		} else {
			c.emit(opIndex, 0)
		}

	case *sliceExpr:
		if err := c.compile(n.Object); err != nil {
			return err
		}
		if n.Low != nil {
			if err := c.compile(n.Low); err != nil {
				return err
			}
		} else {
			c.emit(opLoadConst, c.addConstant(Nil))
		}
		if n.High != nil {
			if err := c.compile(n.High); err != nil {
				return err
			}
		} else {
			c.emit(opLoadConst, c.addConstant(Nil))
		}
		if n.Step != nil {
			if err := c.compile(n.Step); err != nil {
				return err
			}
		} else {
			c.emit(opLoadConst, c.addConstant(Nil))
		}
		c.emit(opSlice, 0)

	case *arrayLit:
		// Check if array contains spread elements
		hasSpread := false
		for _, elem := range n.Elements {
			if _, ok := elem.(*spreadExpr); ok {
				hasSpread = true
				break
			}
		}

		if hasSpread {
			// Create empty array first, then append elements
			c.emit(opArray, 0)
			for _, elem := range n.Elements {
				if spread, ok := elem.(*spreadExpr); ok {
					if err := c.compile(spread.Expr); err != nil {
						return err
					}
					c.emit(opSpread, 0) // Spread and concat
				} else {
					if err := c.compile(elem); err != nil {
						return err
					}
					c.emit(opArray, 1)  // Wrap in single-element array
					c.emit(opSpread, 0) // Concat
				}
			}
		} else {
			// Simple case: no spread
			for _, elem := range n.Elements {
				if err := c.compile(elem); err != nil {
					return err
				}
			}
			c.emit(opArray, len(n.Elements))
		}

	case *spreadExpr:
		// Standalone spread (shouldn't happen in normal use)
		if err := c.compile(n.Expr); err != nil {
			return err
		}

	case *mapLit:
		for i := range n.Keys {
			if err := c.compile(n.Keys[i]); err != nil {
				return err
			}
			if err := c.compile(n.Values[i]); err != nil {
				return err
			}
		}
		c.emit(opMap, len(n.Keys))

	case *rangeExpr:
		if err := c.compile(n.Start); err != nil {
			return err
		}
		if err := c.compile(n.End); err != nil {
			return err
		}
		c.emit(opRange, 0)

	case *pipeExpr:
		// If right side is a callExpr, inject piped value as first argument
		if call, ok := n.Right.(*callExpr); ok {
			// Stack order for opCall: arg1, arg2, ..., argN, func
			// So compile: piped_value (arg1), then original args, then function

			// Compile piped value as first argument
			if err := c.compile(n.Left); err != nil {
				return err
			}

			// Compile remaining arguments
			for _, arg := range call.Args {
				if err := c.compile(arg); err != nil {
					return err
				}
			}

			// Compile function reference last
			if err := c.compile(call.Func); err != nil {
				return err
			}

			// Call with piped value + original args (1 + len(call.Args))
			c.emit(opCall, len(call.Args)+1)
		} else if ident, ok := n.Right.(*ident); ok {
			// For simple identifiers (like `"hello" | trim` or `arr | sum`),
			// treat as function call with piped value as argument
			// Stack order: arg, func, opCall(1)

			// Compile piped value as argument
			if err := c.compile(n.Left); err != nil {
				return err
			}

			// Compile function name
			c.emitName(opLoadVar, ident.Name)

			// Call with 1 argument
			c.emit(opCall, 1)
		} else {
			// For other expressions, use opPipe
			if err := c.compile(n.Left); err != nil {
				return err
			}
			if err := c.compile(n.Right); err != nil {
				return err
			}
			c.emit(opPipe, 0)
		}

	case *lambdaExpr:
		// Compile lambda as a separate bytecode segment
		innerCompiler := newCompiler()
		innerCompiler.lambdaParams = n.Params
		if err := innerCompiler.compile(n.Body); err != nil {
			return err
		}
		innerCompiler.emit(opReturn, 0)

		// Store the compiled lambda as a constant
		lambdaCode := &bytecode{
			Instructions: innerCompiler.code,
			Constants:    innerCompiler.constants,
			Names:        innerCompiler.nameList,
			Params:       n.Params, // Store param names for binding
		}
		idx := c.addConstant(Value{kind: kindFunc, ptr: lambdaCode})
		c.emit(opLambda, idx)

	case *letExpr:
		c.emitName(opSaveVar, n.Name)
		if err := c.compile(n.Value); err != nil {
			return err
		}
		c.emitName(opStoreVar, n.Name)
		if !n.ImplicitBody {
			if err := c.compile(n.Body); err != nil {
				return err
			}
		}
		c.emitName(opRestoreVar, n.Name)

	case *destructureExpr:
		for _, field := range n.Pattern {
			c.emitName(opSaveVar, field.Name)
		}
		// Compile the source value
		if err := c.compile(n.Value); err != nil {
			return err
		}

		// Store in temp variable with unique name to avoid conflicts
		tempVar := fmt.Sprintf("__destruct_%d__", c.tempVarCounter)
		c.tempVarCounter++
		c.emitName(opStoreVar, tempVar)

		// Extract each field
		for i, field := range n.Pattern {
			// Load the source object/array
			c.emitName(opLoadVar, tempVar)

			if n.IsArray {
				// Array destructuring: use optional index to handle missing elements
				idx := c.addConstant(Int(int64(i)))
				c.emit(opLoadConst, idx)
				c.emit(opOptIndex, 0) // Use optional index - returns nil if out of bounds
			} else {
				// Object destructuring: use optional member to handle missing keys
				c.emitName(opOptMember, field.Key)
			}

			// Handle default value if present or value is nil
			if field.Default != nil {
				// Check if value is nil, if so use default
				jumpNotNil := c.emitJump(opCoalesce)
				c.emit(opPop, 0)
				if err := c.compile(field.Default); err != nil {
					return err
				}
				c.patchJump(jumpNotNil)
			}

			// Store in the target variable
			c.emitName(opStoreVar, field.Name)
			c.emit(opPop, 0) // Pop the stored value
		}

		// Cleanup: delete the temp variable to avoid polluting the environment
		c.emitName(opDeleteVar, tempVar)

		// Without an explicit body, the stored source remains the result.
		if !n.ImplicitBody {
			if err := c.compile(n.Body); err != nil {
				return err
			}
		}
		for i := len(n.Pattern) - 1; i >= 0; i-- {
			c.emitName(opRestoreVar, n.Pattern[i].Name)
		}

	default:
		return &CompileError{Message: "unknown node type", Node: node}
	}

	return nil
}

func (c *compiler) compileBinary(n *binaryExpr) error {
	// Check for operator overloading first (except short-circuit operators)
	if fnName, ok := c.operators[n.Op]; ok && n.Op != "&&" && n.Op != "||" {
		// Compile as function call: fnName(left, right)
		if err := c.compile(n.Left); err != nil {
			return err
		}
		if err := c.compile(n.Right); err != nil {
			return err
		}
		c.emitName(opLoadVar, fnName)
		c.emit(opCall, 2)
		return nil
	}

	// Handle short-circuit operators specially
	if n.Op == "&&" {
		if err := c.compile(n.Left); err != nil {
			return err
		}
		jumpFalse := c.emitJump(opJumpNot)
		c.emit(opPop, 0)
		if err := c.compile(n.Right); err != nil {
			return err
		}
		c.patchJump(jumpFalse)
		return nil
	}

	if n.Op == "||" {
		if err := c.compile(n.Left); err != nil {
			return err
		}
		jumpTrue := c.emitJump(opJumpIf)
		c.emit(opPop, 0)
		if err := c.compile(n.Right); err != nil {
			return err
		}
		c.patchJump(jumpTrue)
		return nil
	}

	// Standard binary operators
	if err := c.compile(n.Left); err != nil {
		return err
	}
	if err := c.compile(n.Right); err != nil {
		return err
	}

	switch n.Op {
	case "+":
		c.emit(opAdd, 0)
	case "-":
		c.emit(opSub, 0)
	case "*":
		c.emit(opMul, 0)
	case "/":
		c.emit(opDiv, 0)
	case "%":
		c.emit(opMod, 0)
	case "**":
		c.emit(opPow, 0)
	case "==":
		c.emit(opEq, 0)
	case "!=":
		c.emit(opNe, 0)
	case "<":
		c.emit(opLt, 0)
	case "<=":
		c.emit(opLe, 0)
	case ">":
		c.emit(opGt, 0)
	case ">=":
		c.emit(opGe, 0)
	case "&":
		c.emit(opBitAnd, 0)
	case "|":
		c.emit(opBitOr, 0)
	case "^":
		c.emit(opBitXor, 0)
	case "<<":
		c.emit(opShl, 0)
	case ">>":
		c.emit(opShr, 0)
	case litOpShrU:
		c.emit(opShrU, 0)
	case "in":
		c.emit(opIn, 0)
	case opNotIn:
		c.emit(opIn, 0)
		c.emit(opNot, 0)
	case "~=":
		c.emit(opLike, 0)
	default:
		return &CompileError{Message: "unknown binary operator: " + n.Op, Node: n}
	}

	return nil
}

func (c *compiler) emit(op opcode, arg int) {
	c.code = append(c.code, instruction{Op: op, Arg: arg})
}

func (c *compiler) emitName(op opcode, name string) {
	c.code = append(c.code, instruction{Op: op, Name: name})
}

func (c *compiler) emitJump(op opcode) int {
	idx := len(c.code)
	c.code = append(c.code, instruction{Op: op, Arg: 0})
	return idx
}

func (c *compiler) patchJump(idx int) {
	c.code[idx].Arg = len(c.code)
}

func (c *compiler) addConstant(v Value) int {
	// O(1) lookup using hash map for deduplication
	key := makeConstKey(v)
	if idx, ok := c.constIndex[key]; ok {
		return idx
	}
	idx := len(c.constants)
	c.constants = append(c.constants, v)
	c.constIndex[key] = idx
	return idx
}
