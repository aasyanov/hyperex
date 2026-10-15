package hyperex

import (
	"container/list"
	"context"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	sandboxCheckMask         = 1023 // ops/memory cadence: i&sandboxCheckMask
	sandboxOpsBatch          = 1024 // opsCount increment per cadence check
	defaultVMStackCap        = 256
	envClearRebuildThreshold = 32
	maxBitShift              = 63
	vmValueBaseBytes         = 32 // approximate sizeof(Value) for memory estimates
)

// vm executes compiled bytecode.
// Field order packs ints/pointers first; strictMode bool last to avoid mid-struct pad
// before numericMode/nonFiniteMode (amd64 Sizeof still 144 with trailing align).
type execVM struct {
	code      *bytecode
	stack     []Value
	sp        int // stack pointer
	maxSP     int // maximum stack pointer reached during execution (for profiling/limits)
	ip        int // instruction pointer
	env       map[string]Value
	builtins  map[string]Value
	callDepth int
	maxDepth  int

	// Sandbox support
	limits   *sandboxLimits
	opsCount int64

	// Numeric arithmetic policy (NumericDefault / NumericFloat64)
	numericMode NumericMode

	// Non-finite float policy (NonFiniteAllow / NonFiniteError)
	nonFiniteMode NonFiniteMode

	// ctx is set only for the duration of Run / nested execute; cleared on releaseVM.
	ctx context.Context

	// Strict mode - error on invalid type conversions
	strictMode bool
}

// sandboxLoopCheck makes long-running in-vm or builtin loops interruptible by
// sandbox limits and context cancellation.
// It is intentionally cheap: checks every sandboxOpsBatch iterations and accounts approximate ops cost.
func (vm *execVM) sandboxLoopCheck(i int) error {
	if vm == nil {
		return nil
	}
	if (i & sandboxCheckMask) != 0 {
		return nil
	}
	if vm.limits != nil {
		vm.opsCount += sandboxOpsBatch
	}
	return vm.checkLimits()
}

// sandboxDeny returns a sandbox error with consistent typing.
// access_denied has no matching sentinel; Type alone identifies it.
func sandboxDeny(msg string) error {
	return sandboxErr(nil, "access_denied", msg, 0, 0)
}

// vmPool pools vm instances for reuse.
var vmPool = sync.Pool{
	New: func() any {
		return &execVM{
			stack:    make([]Value, defaultVMStackCap),
			env:      make(map[string]Value),
			maxDepth: defaultMaxCallDepth,
		}
	},
}

// regexCache is an LRU cache for compiled regular expressions.
// Limited to maxRegexCacheSize entries to prevent unbounded memory growth.
// Uses container/list for O(1) LRU operations.
const maxRegexCacheSize = 256

type regexCacheEntry struct {
	re      *regexp.Regexp
	pattern string
	element *list.Element // Pointer to list element for O(1) removal
}

type regexLRUCache struct {
	mu    sync.Mutex
	list  *list.List                  // Doubly-linked list for O(1) LRU
	index map[string]*regexCacheEntry // pattern -> entry (O(1) lookup)
}

var regexCache = &regexLRUCache{
	list:  list.New(),
	index: make(map[string]*regexCacheEntry, maxRegexCacheSize),
}

// getRegexp returns a cached or newly compiled regexp for the given pattern.
// Uses LRU eviction when cache is full. All operations are O(1).
func getRegexp(pattern string) (*regexp.Regexp, error) {
	regexCache.mu.Lock()
	if entry, ok := regexCache.index[pattern]; ok {
		// Move to front (MRU) - O(1)
		regexCache.list.MoveToFront(entry.element)
		re := entry.re
		regexCache.mu.Unlock()
		return re, nil
	}
	regexCache.mu.Unlock()

	// Slow path: compile outside lock
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexCache.mu.Lock()
	// Double-check after acquiring lock
	if entry, ok := regexCache.index[pattern]; ok {
		regexCache.list.MoveToFront(entry.element)
		regexCache.mu.Unlock()
		return entry.re, nil
	}

	// Evict oldest if at capacity - O(1)
	if regexCache.list.Len() >= maxRegexCacheSize {
		oldest := regexCache.list.Back()
		if oldest != nil {
			oldEntry := oldest.Value.(*regexCacheEntry)
			delete(regexCache.index, oldEntry.pattern)
			regexCache.list.Remove(oldest)
		}
	}

	// Add new entry at front (MRU) - O(1)
	entry := &regexCacheEntry{re: re, pattern: pattern}
	entry.element = regexCache.list.PushFront(entry)
	regexCache.index[pattern] = entry
	regexCache.mu.Unlock()

	return re, nil
}

// newVM gets a vm from the pool.
func newVM(code *bytecode, builtins map[string]Value) *execVM {
	vm := vmPool.Get().(*execVM)
	vm.code = code
	vm.sp = 0
	vm.maxSP = 0
	vm.ip = 0
	vm.callDepth = 0
	// Reset pooled configuration to safe defaults. Specific callers may override (engine/sandbox).
	vm.maxDepth = defaultMaxCallDepth
	vm.builtins = builtins
	vm.limits = nil
	vm.opsCount = 0
	vm.ctx = nil
	vm.strictMode = false
	vm.numericMode = NumericDefault
	vm.nonFiniteMode = NonFiniteAllow
	// Clear env - use new map if current is too large to avoid O(n) delete loop
	if len(vm.env) > envClearRebuildThreshold {
		vm.env = make(map[string]Value)
	} else {
		for k := range vm.env {
			delete(vm.env, k)
		}
	}
	return vm
}

// newVMWithLimits gets a vm from the pool with sandbox limits.
func newVMWithLimits(code *bytecode, builtins map[string]Value, limits *sandboxLimits) *execVM {
	vm := newVM(code, builtins)
	vm.limits = limits
	if limits != nil {
		if limits.maxCallDepth > 0 {
			vm.maxDepth = limits.maxCallDepth
		}
	}
	return vm
}

// releaseVM returns a vm to the pool.
func releaseVM(vm *execVM) {
	vm.code = nil
	vm.builtins = nil
	vm.limits = nil
	vm.strictMode = false
	vm.numericMode = NumericDefault
	vm.nonFiniteMode = NonFiniteAllow
	vm.ctx = nil
	// Clear stack to release references to large objects (prevent memory leak)
	// Use maxSP because sp at the end can be 0 even if large temporaries were pushed/popped.
	for i := 0; i < vm.maxSP; i++ {
		vm.stack[i] = Nil
	}
	vm.sp = 0
	vm.maxSP = 0
	// Clear env to release references - use new map if too large
	if len(vm.env) > envClearRebuildThreshold {
		vm.env = make(map[string]Value)
	} else {
		for k := range vm.env {
			delete(vm.env, k)
		}
	}
	vmPool.Put(vm)
}

// Run executes the bytecode and returns the result.
// ctx is observed during execution; cancellation returns [ErrOperationCancelled].
// Sandbox [WithTimeout] deadlines and ctx deadlines both apply — whichever fires first wins.
func (vm *execVM) run(ctx context.Context, env map[string]any) (Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	vm.ctx = ctx
	if err := vm.checkContext(); err != nil {
		return Nil, err
	}

	// Load environment
	if vm.limits != nil && vm.limits.secure {
		for k, v := range env {
			sv, err := vm.sandboxCoerceEnvValue(v, 0)
			if err != nil {
				return Nil, err
			}
			vm.env[k] = sv
		}
	} else {
		for k, v := range env {
			vm.env[k] = NewValue(v)
		}
	}

	return vm.execute()
}

const sandboxMaxEnvDepth = 32

// sandboxCoerceEnvValue converts environment values into a safe subset for Secure sandbox:
// nil/bool/number/string/time.Time/time.Duration/[]any/map[string]any (recursively),
// rejecting other structs, pointers, funcs, etc.
func (vm *execVM) sandboxCoerceEnvValue(v any, depth int) (Value, error) {
	if depth > sandboxMaxEnvDepth {
		return Nil, sandboxDeny("environment value too deeply nested")
	}
	if v == nil {
		return Nil, nil
	}
	switch x := v.(type) {
	case Value:
		// If caller manually passes Value, still enforce that it doesn't contain external Go objects.
		if x.kind == kindAny || x.kind == kindFunc {
			return Nil, sandboxDeny("environment Value contains disallowed type in secure mode")
		}
		return x, nil
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
		return NewValue(x), nil
	case time.Time:
		return Time(x), nil
	case time.Duration:
		return Duration(x), nil
	case *time.Time:
		if x == nil {
			return Nil, nil
		}
		return Time(*x), nil
	case *time.Duration:
		if x == nil {
			return Nil, nil
		}
		return Duration(*x), nil
	case []any:
		out := make([]any, len(x))
		for i, elem := range x {
			ev, err := vm.sandboxCoerceEnvValue(elem, depth+1)
			if err != nil {
				return Nil, err
			}
			out[i] = ev.Interface()
			// Best-effort: periodically honor timeout/ops in long env conversions
			if vm.limits != nil && (i&sandboxCheckMask) == 0 {
				if err := vm.checkLimits(); err != nil {
					return Nil, err
				}
			}
		}
		return Array(out), nil
	case map[string]any:
		out := make(map[string]any, len(x))
		i := 0
		for k, elem := range x {
			ev, err := vm.sandboxCoerceEnvValue(elem, depth+1)
			if err != nil {
				return Nil, err
			}
			out[k] = ev.Interface()
			i++
			if vm.limits != nil && (i&sandboxCheckMask) == 0 {
				if err := vm.checkLimits(); err != nil {
					return Nil, err
				}
			}
		}
		return Map(out), nil
	default:
		// Reflectively accept only slices/arrays and maps with string keys; reject structs, ptrs, funcs, chans, etc.
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			n := rv.Len()
			out := make([]any, n)
			for i := 0; i < n; i++ {
				ev, err := vm.sandboxCoerceEnvValue(rv.Index(i).Interface(), depth+1)
				if err != nil {
					return Nil, err
				}
				out[i] = ev.Interface()
				if vm.limits != nil && (i&sandboxCheckMask) == 0 {
					if err := vm.checkLimits(); err != nil {
						return Nil, err
					}
				}
			}
			return Array(out), nil
		case reflect.Map:
			if rv.Type().Key().Kind() != reflect.String {
				return Nil, sandboxDeny("environment map must have string keys in secure mode")
			}
			out := make(map[string]any, rv.Len())
			iter := rv.MapRange()
			i := 0
			for iter.Next() {
				k := iter.Key().String()
				ev, err := vm.sandboxCoerceEnvValue(iter.Value().Interface(), depth+1)
				if err != nil {
					return Nil, err
				}
				out[k] = ev.Interface()
				i++
				if vm.limits != nil && (i&sandboxCheckMask) == 0 {
					if err := vm.checkLimits(); err != nil {
						return Nil, err
					}
				}
			}
			return Map(out), nil
		default:
			return Nil, sandboxDeny("environment value type not allowed in secure mode")
		}
	}
}

func (vm *execVM) execute() (Value, error) {
	code := vm.code.Instructions
	constants := vm.code.Constants

	for vm.ip < len(code) {
		// Sandbox limits: every op. Context-only: every sandboxOpsBatch ops (same cadence as sandboxLoopCheck).
		if vm.limits != nil {
			if err := vm.checkLimits(); err != nil {
				return Nil, err
			}
		} else if vm.ctx != nil && (vm.opsCount&sandboxCheckMask) == 0 {
			if err := vm.checkContext(); err != nil {
				return Nil, err
			}
		}
		vm.opsCount++

		inst := code[vm.ip]
		vm.ip++

		switch inst.Op {
		case opNop:
			// Do nothing

		case opHalt:
			if vm.sp > 0 {
				return vm.pop(), nil
			}
			return Nil, nil

		case opReturn:
			if vm.sp > 0 {
				return vm.pop(), nil
			}
			return Nil, nil

		case opPush:
			if inst.Arg < 0 || inst.Arg >= len(constants) {
				return Nil, &RuntimeError{Message: errInvalidConstIndex}
			}
			vm.push(constants[inst.Arg])

		case opPop:
			vm.pop()

		case opDup:
			v := vm.peek()
			vm.push(v)

		case opLoadConst:
			if inst.Arg < 0 || inst.Arg >= len(constants) {
				return Nil, &RuntimeError{Message: errInvalidConstIndex}
			}
			vm.push(constants[inst.Arg])

		case opLoadVar:
			name := inst.Name
			if v, ok := vm.env[name]; ok {
				vm.push(v)
			} else if v, ok := vm.builtins[name]; ok {
				// Sandbox: check if function is allowed
				if vm.limits != nil {
					if err := vm.checkFunctionAllowed(name); err != nil {
						return Nil, err
					}
				}
				vm.push(v)
			} else {
				// Determine if this looks like a function call by checking next instruction
				kind := kindVariable
				if vm.ip < len(code) && code[vm.ip].Op == opCall {
					kind = kindFunction
				}
				return Nil, &UndefinedError{Name: name, Kind: kind}
			}

		case opLoadEnv:
			// Load from lambda environment (positional parameter $0, $1, etc.)
			name := fmt.Sprintf("$%d", inst.Arg)
			if v, ok := vm.env[name]; ok {
				vm.push(v)
			} else {
				vm.push(Nil)
			}

		case opStoreVar:
			// Sandbox: check read-only mode
			if vm.limits != nil && vm.limits.readOnly {
				return Nil, sandboxErr(ErrAssignmentNotAllowed, sandboxAssignmentDenied, ErrAssignmentNotAllowed.Error(), 0, 0)
			}
			name := inst.Name
			v := vm.pop()
			vm.env[name] = v
			vm.push(v) // Let expressions return the value

		case opDeleteVar:
			// Delete variable from environment (used for temp variable cleanup)
			name := inst.Name
			delete(vm.env, name)

		case opAdd:
			right := vm.pop()
			left := vm.pop()
			result, err := vm.add(left, right)
			if err != nil {
				return Nil, err
			}
			vm.push(result)

		case opSub:
			right := vm.pop()
			left := vm.pop()
			result, err := vm.sub(left, right)
			if err != nil {
				return Nil, err
			}
			vm.push(result)

		case opMul:
			right := vm.pop()
			left := vm.pop()
			result, err := vm.mul(left, right)
			if err != nil {
				return Nil, err
			}
			vm.push(result)

		case opDiv:
			right := vm.pop()
			left := vm.pop()
			result, err := vm.div(left, right)
			if err != nil {
				return Nil, err
			}
			vm.push(result)

		case opMod:
			right := vm.pop()
			left := vm.pop()
			result, err := vm.mod(left, right)
			if err != nil {
				return Nil, err
			}
			vm.push(result)

		case opPow:
			right := vm.pop()
			left := vm.pop()
			powVal, powErr := vm.pow(left, right)
			if powErr != nil {
				return Nil, powErr
			}
			vm.push(powVal)

		case opNeg:
			v := vm.pop()
			negVal, negErr := vm.neg(v)
			if negErr != nil {
				return Nil, negErr
			}
			vm.push(negVal)

		case opEq:
			right := vm.pop()
			left := vm.pop()
			vm.push(Bool(left.Equal(right)))

		case opNe:
			right := vm.pop()
			left := vm.pop()
			vm.push(Bool(!left.Equal(right)))

		case opLt:
			right := vm.pop()
			left := vm.pop()
			vm.push(Bool(left.Compare(right) < 0))

		case opLe:
			right := vm.pop()
			left := vm.pop()
			vm.push(Bool(left.Compare(right) <= 0))

		case opGt:
			right := vm.pop()
			left := vm.pop()
			vm.push(Bool(left.Compare(right) > 0))

		case opGe:
			right := vm.pop()
			left := vm.pop()
			vm.push(Bool(left.Compare(right) >= 0))

		case opAnd:
			right := vm.pop()
			left := vm.pop()
			vm.push(Bool(left.Bool() && right.Bool()))

		case opOr:
			right := vm.pop()
			left := vm.pop()
			vm.push(Bool(left.Bool() || right.Bool()))

		case opNot:
			v := vm.pop()
			vm.push(Bool(!v.Bool()))

		case opBitAnd:
			right := vm.pop()
			left := vm.pop()
			vm.push(Int(left.Int() & right.Int()))

		case opBitOr:
			right := vm.pop()
			left := vm.pop()
			vm.push(Int(left.Int() | right.Int()))

		case opBitXor:
			right := vm.pop()
			left := vm.pop()
			vm.push(Int(left.Int() ^ right.Int()))

		case opBitNot:
			v := vm.pop()
			vm.push(Int(^v.Int()))

		case opShl:
			right := vm.pop()
			left := vm.pop()
			shift := right.Int()
			if shift < 0 {
				return Nil, &RuntimeError{Message: errNegativeShift}
			}
			if shift > maxBitShift {
				vm.push(Int(0)) // Shifting by 64+ bits results in 0
			} else {
				vm.push(Int(left.Int() << uint(shift)))
			}

		case opShr:
			right := vm.pop()
			left := vm.pop()
			shift := right.Int()
			if shift < 0 {
				return Nil, &RuntimeError{Message: errNegativeShift}
			}
			if shift > maxBitShift {
				// For right shift, result depends on sign
				if left.Int() < 0 {
					vm.push(Int(-1)) // All 1s for negative numbers
				} else {
					vm.push(Int(0))
				}
			} else {
				vm.push(Int(left.Int() >> uint(shift)))
			}

		case opShrU:
			// Unsigned right shift: treats left operand as unsigned
			right := vm.pop()
			left := vm.pop()
			shift := right.Int()
			if shift < 0 {
				return Nil, &RuntimeError{Message: errNegativeShift}
			}
			if shift > maxBitShift {
				vm.push(Int(0)) // Always 0 for unsigned shift >= 64
			} else {
				// Cast to uint64 for unsigned shift, then back to int64
				vm.push(Int(int64(uint64(left.Int()) >> uint(shift))))
			}

		case opIn:
			right := vm.pop()
			left := vm.pop()
			result, err := vm.contains(right, left)
			if err != nil {
				return Nil, err
			}
			vm.push(Bool(result))

		case opLike:
			right := vm.pop()
			left := vm.pop()
			pattern := right.String()
			re, err := getRegexp(pattern)
			if err != nil {
				return Nil, ErrInvalidRegex
			}
			vm.push(Bool(re.MatchString(left.String())))

		case opJump:
			vm.ip = inst.Arg

		case opJumpIf:
			v := vm.peek()
			if v.Bool() {
				vm.ip = inst.Arg
			}

		case opJumpNot:
			v := vm.peek()
			if !v.Bool() {
				vm.ip = inst.Arg
			}

		case opCoalesce:
			v := vm.peek()
			if !v.IsNil() {
				vm.ip = inst.Arg
			}

		case opMember:
			obj := vm.pop()
			name := inst.Name
			if vm.limits != nil && vm.limits.secure && obj.kind == kindAny {
				return Nil, sandboxDeny("property access on external Go objects is not allowed in secure mode")
			}
			val, ok := obj.Get(name)
			if !ok {
				return Nil, &RuntimeError{Message: fmt.Sprintf("property '%s' not found", name)}
			}
			vm.push(val)

		case opOptMember:
			obj := vm.pop()
			if obj.IsNil() {
				vm.push(Nil)
			} else {
				name := inst.Name
				if vm.limits != nil && vm.limits.secure && obj.kind == kindAny {
					return Nil, sandboxDeny("property access on external Go objects is not allowed in secure mode")
				}
				val, _ := obj.Get(name)
				vm.push(val)
			}

		case opIndex:
			index := vm.pop()
			obj := vm.pop()
			val, err := vm.index(obj, index)
			if err != nil {
				return Nil, err
			}
			vm.push(val)

		case opOptIndex:
			index := vm.pop()
			obj := vm.pop()
			if obj.IsNil() {
				vm.push(Nil)
			} else {
				// Optional index returns nil on any error (out of bounds, not indexable, etc.)
				// Optional chaining: missing member yields nil without error.
				val, err := vm.index(obj, index)
				if err != nil {
					vm.push(Nil)
				} else {
					vm.push(val)
				}
			}

		case opSlice:
			step := vm.pop()
			high := vm.pop()
			low := vm.pop()
			obj := vm.pop()
			val, err := vm.slice(obj, low, high, step)
			if err != nil {
				return Nil, err
			}
			vm.push(val)

		case opArray:
			count := inst.Arg
			arr := make([]any, count)
			for i := count - 1; i >= 0; i-- {
				arr[i] = vm.pop().Interface()
			}
			vm.push(Array(arr))

		case opMap:
			count := inst.Arg
			m := make(map[string]any, count)
			for i := 0; i < count; i++ {
				val := vm.pop()
				key := vm.pop()
				m[key.String()] = val.Interface()
			}
			vm.push(Map(m))

		case opRange:
			end := vm.pop()
			start := vm.pop()
			startInt := start.Int()
			endInt := end.Int()

			// Limit range size to prevent DoS (max 1 million elements)
			const maxRangeSize int64 = 1_000_000

			// Safe calculation of range size to prevent overflow
			var rangeSize int64
			if startInt <= endInt {
				// Check for overflow: endInt - startInt + 1
				diff := endInt - startInt
				if diff < 0 {
					// Overflow occurred (endInt - startInt wrapped)
					return Nil, &RuntimeError{Message: errRangeOverflow}
				}
				if diff > maxRangeSize-1 {
					// Adding 1 would exceed max or overflow
					return Nil, &RuntimeError{Message: fmt.Sprintf("range size exceeds maximum %d", maxRangeSize)}
				}
				rangeSize = diff + 1
			} else {
				// start > end: empty range
				rangeSize = 0
			}

			if rangeSize > maxRangeSize {
				return Nil, &RuntimeError{Message: fmt.Sprintf("range size %d exceeds maximum %d", rangeSize, maxRangeSize)}
			}

			arr := make([]any, 0, rangeSize)
			// Use a counted loop to avoid int64 overflow when endInt == MaxInt64 (i++ would wrap).
			for k := int64(0); k < rangeSize; k++ {
				arr = append(arr, startInt+k)
			}
			vm.push(Array(arr))

		case opCall:
			argc := inst.Arg
			fn := vm.pop()
			args := make([]Value, argc)
			for i := argc - 1; i >= 0; i-- {
				args[i] = vm.pop()
			}
			result, err := vm.call(fn, args)
			if err != nil {
				return Nil, err
			}
			vm.push(result)

		case opPipe:
			fn := vm.pop()
			arg := vm.pop()
			result, err := vm.call(fn, []Value{arg})
			if err != nil {
				return Nil, err
			}
			vm.push(result)

		case opLambda:
			// The lambda bytecode is stored in constants
			if inst.Arg < 0 || inst.Arg >= len(constants) {
				return Nil, &RuntimeError{Message: "invalid lambda constant index"}
			}
			lv := constants[inst.Arg]
			lambdaCode, ok := lv.ptr.(*bytecode)
			if !ok || lambdaCode == nil {
				return Nil, &RuntimeError{Message: "invalid lambda bytecode"}
			}
			// Create a closure value with captured environment, builtins and sandbox limits
			vm.push(Value{kind: kindFunc, ptr: &closure{
				code:          lambdaCode,
				env:           vm.copyEnv(),
				params:        lambdaCode.Params,
				builtins:      vm.builtins,      // Capture builtins including custom functions
				limits:        vm.limits,        // Inherit sandbox limits
				maxDepth:      vm.maxDepth,      // Inherit max call depth from parent vm
				strictMode:    vm.strictMode,    // Inherit strict mode from parent vm
				numericMode:   vm.numericMode,   // Inherit numeric policy
				nonFiniteMode: vm.nonFiniteMode, // Inherit non-finite policy
			}})

		case opSpread:
			// Concatenate top two arrays on stack
			right := vm.pop() // Array to spread
			left := vm.pop()  // Existing array
			if !left.IsArray() {
				return Nil, &RuntimeError{Message: "spread: left operand must be an array"}
			}
			if !right.IsArray() {
				return Nil, &RuntimeError{Message: "spread: cannot spread non-array value"}
			}
			result := append(left.Array(), right.Array()...)
			vm.push(Array(result))

		default:
			return Nil, &RuntimeError{Message: fmt.Sprintf("unknown opcode: %d", inst.Op)}
		}
	}

	if vm.sp > 0 {
		return vm.pop(), nil
	}
	return Nil, nil
}

// debugMode enables additional runtime checks and panics on bytecode bugs.
// Toggle from tests via SetDebugModeForTest in export_test.go.
var debugMode = false

// Stack operations
func (vm *execVM) push(v Value) {
	if vm.sp >= len(vm.stack) {
		// Grow stack
		newStack := make([]Value, len(vm.stack)*2)
		copy(newStack, vm.stack)
		vm.stack = newStack
	}
	vm.stack[vm.sp] = v
	vm.sp++
	if vm.sp > vm.maxSP {
		vm.maxSP = vm.sp
	}
}

func (vm *execVM) pop() Value {
	if vm.sp <= 0 {
		if debugMode {
			panic("hyperex: stack underflow - this indicates a bytecode compiler bug")
		}
		// Return Nil for safety in production - this indicates bytecode bug
		return Nil
	}
	vm.sp--
	v := vm.stack[vm.sp]
	// Clear slot to avoid retaining references (VMs are pooled).
	vm.stack[vm.sp] = Nil
	return v
}

func (vm *execVM) peek() Value {
	if vm.sp <= 0 {
		if debugMode {
			panic("hyperex: peek on empty stack - this indicates a bytecode compiler bug")
		}
		return Nil
	}
	return vm.stack[vm.sp-1]
}

// Overflow checking helpers for safe integer arithmetic
func addOverflows(a, b int64) bool {
	if b > 0 {
		return a > math.MaxInt64-b
	}
	return a < math.MinInt64-b
}

func subOverflows(a, b int64) bool {
	if b < 0 {
		return a > math.MaxInt64+b
	}
	return a < math.MinInt64+b
}

func mulOverflows(a, b int64) bool {
	if a == 0 || b == 0 {
		return false
	}
	if a == -1 {
		return b == math.MinInt64
	}
	if b == -1 {
		return a == math.MinInt64
	}
	c := a * b
	return c/b != a
}

// NOTE: intPow and canIntPowWithoutOverflow are defined in optimizer.go
// to be shared between constant folding and runtime evaluation.

// strictCheckNumeric validates that a value can be converted to a number in strict mode.
// Returns an error if the value is a string that cannot be parsed as a number.
func (vm *execVM) strictCheckNumeric(v Value) error {
	if !vm.strictMode {
		return nil
	}
	if v.IsString() {
		s := v.String()
		if s == "" {
			return &RuntimeError{
				Message: "cannot convert empty string to number in strict mode",
				cause:   ErrInvalidConversion,
			}
		}
		// Try to parse as number
		if _, err := parseInt(s); err != nil {
			if _, err := parseFloat(s); err != nil {
				return &RuntimeError{
					Message: fmt.Sprintf("cannot convert %q to number in strict mode", s),
					cause:   ErrInvalidConversion,
				}
			}
		}
	}
	return nil
}

func parseInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	// Base 0: supports optional 0x/0o/0b prefixes, consistent with Value.Int().
	return strconv.ParseInt(s, 0, 64)
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	return strconv.ParseFloat(s, 64)
}

// floatResult wraps a float64 and optionally rejects NaN/Inf under NonFiniteError.
func (vm *execVM) floatResult(f float64) (Value, error) {
	if vm.nonFiniteMode == NonFiniteError && (math.IsNaN(f) || math.IsInf(f, 0)) {
		return Nil, wrapRuntime(ErrNonFinite)
	}
	return Float(f), nil
}

// rejectNonFinite returns ErrNonFinite when mode is NonFiniteError and v is NaN/Inf float.
func (vm *execVM) rejectNonFinite(v Value) error {
	if vm == nil || vm.nonFiniteMode != NonFiniteError || !v.IsFloat() {
		return nil
	}
	f := v.Float()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return wrapRuntime(ErrNonFinite)
	}
	return nil
}

// Arithmetic helpers
func (vm *execVM) add(left, right Value) (Value, error) {
	// Temporal arithmetic before string concat (no silent string+time).
	if left.IsTime() || right.IsTime() || left.IsDuration() || right.IsDuration() {
		return vm.addTemporal(left, right)
	}
	// Strict mode: check for invalid string-to-number conversions
	if vm.strictMode && !left.IsString() && !right.IsString() {
		if err := vm.strictCheckNumeric(left); err != nil {
			return Nil, err
		}
		if err := vm.strictCheckNumeric(right); err != nil {
			return Nil, err
		}
	}
	if left.IsString() || right.IsString() {
		return String(left.String() + right.String()), nil
	}
	if vm.numericMode == NumericFloat64 && left.IsNumber() && right.IsNumber() {
		return vm.floatResult(left.Float() + right.Float())
	}
	if left.IsInt() && right.IsInt() {
		a, b := left.Int(), right.Int()
		if addOverflows(a, b) {
			// Overflow - use float arithmetic instead
			return vm.floatResult(float64(a) + float64(b))
		}
		return Int(a + b), nil
	}
	if left.IsNumber() && right.IsNumber() {
		return vm.floatResult(left.Float() + right.Float())
	}
	if left.IsArray() && right.IsArray() {
		arr := append(left.Array(), right.Array()...)
		return Array(arr), nil
	}
	return Nil, &RuntimeError{Op: "+", Message: fmt.Sprintf("cannot add %s and %s", left.Type(), right.Type())}
}

func (vm *execVM) addTemporal(left, right Value) (Value, error) {
	switch {
	case left.IsTime() && right.IsDuration():
		return addTimeDuration(left, right)
	case left.IsDuration() && right.IsTime():
		return addTimeDuration(right, left)
	case left.IsDuration() && right.IsDuration():
		a, b := int64(left.bits), int64(right.bits)
		if addOverflows(a, b) {
			return Nil, &RuntimeError{Op: "+", Message: errDurationAddOverflow}
		}
		return Duration(time.Duration(a + b)), nil
	case left.IsTime() && right.IsTime():
		return Nil, &RuntimeError{Op: "+", Message: "cannot add time and time"}
	default:
		// string+time / time+number / etc.
		return Nil, &RuntimeError{Op: "+", Message: fmt.Sprintf("cannot add %s and %s", left.Type(), right.Type())}
	}
}

func addTimeDuration(t, d Value) (Value, error) {
	a, b := int64(t.bits), int64(d.bits)
	if addOverflows(a, b) {
		return Nil, &RuntimeError{Op: "+", Message: errTimeDurationOverflow}
	}
	return Time(time.Unix(0, a+b).UTC()), nil
}

func (vm *execVM) sub(left, right Value) (Value, error) {
	if left.IsTime() || right.IsTime() || left.IsDuration() || right.IsDuration() {
		return vm.subTemporal(left, right)
	}
	// Strict mode: check for invalid string-to-number conversions
	if vm.strictMode {
		if err := vm.strictCheckNumeric(left); err != nil {
			return Nil, err
		}
		if err := vm.strictCheckNumeric(right); err != nil {
			return Nil, err
		}
	}
	if vm.numericMode == NumericFloat64 && left.IsNumber() && right.IsNumber() {
		return vm.floatResult(left.Float() - right.Float())
	}
	if left.IsInt() && right.IsInt() {
		a, b := left.Int(), right.Int()
		if subOverflows(a, b) {
			// Overflow - use float arithmetic instead
			return vm.floatResult(float64(a) - float64(b))
		}
		return Int(a - b), nil
	}
	if left.IsNumber() && right.IsNumber() {
		return vm.floatResult(left.Float() - right.Float())
	}
	return Nil, &RuntimeError{Op: "-", Message: fmt.Sprintf("cannot subtract %s from %s", right.Type(), left.Type())}
}

func (vm *execVM) subTemporal(left, right Value) (Value, error) {
	switch {
	case left.IsTime() && right.IsDuration():
		a, b := int64(left.bits), int64(right.bits)
		if subOverflows(a, b) {
			return Nil, &RuntimeError{Op: "-", Message: errTimeSubDurationOverflow}
		}
		return Time(time.Unix(0, a-b).UTC()), nil
	case left.IsTime() && right.IsTime():
		a, b := int64(left.bits), int64(right.bits)
		if subOverflows(a, b) {
			return Nil, &RuntimeError{Op: "-", Message: "time - time overflow"}
		}
		return Duration(time.Duration(a - b)), nil
	case left.IsDuration() && right.IsDuration():
		a, b := int64(left.bits), int64(right.bits)
		if subOverflows(a, b) {
			return Nil, &RuntimeError{Op: "-", Message: errDurationSubOverflow}
		}
		return Duration(time.Duration(a - b)), nil
	default:
		return Nil, &RuntimeError{Op: "-", Message: fmt.Sprintf("cannot subtract %s from %s", right.Type(), left.Type())}
	}
}

func (vm *execVM) mul(left, right Value) (Value, error) {
	if left.IsDuration() || right.IsDuration() {
		return mulDuration(left, right)
	}
	// Strict mode: check for invalid string-to-number conversions
	if vm.strictMode {
		// Only check if not doing string repetition
		if !left.IsString() || !right.IsInt() {
			if err := vm.strictCheckNumeric(left); err != nil {
				return Nil, err
			}
			if err := vm.strictCheckNumeric(right); err != nil {
				return Nil, err
			}
		}
	}
	if vm.numericMode == NumericFloat64 && left.IsNumber() && right.IsNumber() {
		return vm.floatResult(left.Float() * right.Float())
	}
	if left.IsInt() && right.IsInt() {
		a, b := left.Int(), right.Int()
		if mulOverflows(a, b) {
			// Overflow - use float arithmetic instead
			return vm.floatResult(float64(a) * float64(b))
		}
		return Int(a * b), nil
	}
	if left.IsNumber() && right.IsNumber() {
		return vm.floatResult(left.Float() * right.Float())
	}
	// String repetition - O(n) using strings.Repeat
	if left.IsString() && right.IsInt() {
		s := left.String()
		n64 := right.Int()
		if n64 <= 0 {
			return String(""), nil
		}
		// Empty string repeated is still empty; avoid division by zero below.
		if len(s) == 0 {
			return String(""), nil
		}
		// Limit result size to prevent OOM (max 10MB)
		const maxStringSize = 10 * 1024 * 1024
		// Check size using int64 arithmetic and avoid overflow.
		if n64 > int64(maxStringSize)/int64(len(s)) {
			return Nil, &RuntimeError{Message: fmt.Sprintf("string repeat result too large (max %d bytes)", maxStringSize)}
		}
		n, ok := intFromInt64Checked(n64)
		if !ok {
			return Nil, &RuntimeError{Message: "string repeat count out of range"}
		}
		return String(strings.Repeat(s, n)), nil
	}
	return Nil, &RuntimeError{Op: "*", Message: fmt.Sprintf("cannot multiply %s and %s", left.Type(), right.Type())}
}

func mulDuration(left, right Value) (Value, error) {
	var d Value
	var n Value
	switch {
	case left.IsDuration() && right.IsNumber():
		d, n = left, right
	case left.IsNumber() && right.IsDuration():
		d, n = right, left
	default:
		return Nil, &RuntimeError{Op: "*", Message: fmt.Sprintf("cannot multiply %s and %s", left.Type(), right.Type())}
	}
	a := int64(d.bits)
	if n.IsInt() {
		b := n.Int()
		if mulOverflows(a, b) {
			return Nil, &RuntimeError{Op: "*", Message: errDurationMulOverflow}
		}
		return Duration(time.Duration(a * b)), nil
	}
	product := float64(a) * n.Float()
	if math.IsNaN(product) || math.IsInf(product, 0) {
		return Nil, &RuntimeError{Op: "*", Message: errDurationMulOverflow}
	}
	const maxInt64Plus1 = 9223372036854775808.0
	if product < float64(math.MinInt64) || product >= maxInt64Plus1 {
		return Nil, &RuntimeError{Op: "*", Message: errDurationMulOverflow}
	}
	return Duration(time.Duration(int64(product))), nil
}

func (vm *execVM) div(left, right Value) (Value, error) {
	if left.IsDuration() {
		return divDuration(left, right)
	}
	// Strict mode: check for invalid string-to-number conversions
	if vm.strictMode {
		if err := vm.strictCheckNumeric(left); err != nil {
			return Nil, err
		}
		if err := vm.strictCheckNumeric(right); err != nil {
			return Nil, err
		}
	}
	if right.Float() == 0 {
		return Nil, ErrDivisionByZero
	}
	if vm.numericMode == NumericFloat64 && left.IsNumber() && right.IsNumber() {
		return vm.floatResult(left.Float() / right.Float())
	}
	if left.IsInt() && right.IsInt() {
		return Int(left.Int() / right.Int()), nil
	}
	return vm.floatResult(left.Float() / right.Float())
}

func divDuration(left, right Value) (Value, error) {
	if !left.IsDuration() || !right.IsNumber() {
		return Nil, &RuntimeError{Op: "/", Message: fmt.Sprintf("cannot divide %s by %s", left.Type(), right.Type())}
	}
	if right.Float() == 0 {
		return Nil, ErrDivisionByZero
	}
	a := int64(left.bits)
	if right.IsInt() {
		b := right.Int()
		if b == 0 {
			return Nil, ErrDivisionByZero
		}
		return Duration(time.Duration(a / b)), nil
	}
	quot := float64(a) / right.Float()
	if math.IsNaN(quot) || math.IsInf(quot, 0) {
		return Nil, &RuntimeError{Op: "/", Message: errDurationDivOverflow}
	}
	const maxInt64Plus1 = 9223372036854775808.0
	if quot < float64(math.MinInt64) || quot >= maxInt64Plus1 {
		return Nil, &RuntimeError{Op: "/", Message: errDurationDivOverflow}
	}
	return Duration(time.Duration(int64(quot))), nil
}

func (vm *execVM) mod(left, right Value) (Value, error) {
	// Strict mode: check for invalid string-to-number conversions
	if vm.strictMode {
		if err := vm.strictCheckNumeric(left); err != nil {
			return Nil, err
		}
		if err := vm.strictCheckNumeric(right); err != nil {
			return Nil, err
		}
	}
	if vm.numericMode == NumericFloat64 && left.IsNumber() && right.IsNumber() {
		r := right.Float()
		if r == 0 {
			return Nil, ErrDivisionByZero
		}
		return vm.floatResult(math.Mod(left.Float(), r))
	}
	// Support both integer and float modulo
	if left.IsInt() && right.IsInt() {
		r := right.Int()
		if r == 0 {
			return Nil, ErrDivisionByZero
		}
		return Int(left.Int() % r), nil
	}
	// Float modulo using math.Mod (sign follows dividend).
	r := right.Float()
	if r == 0 {
		return Nil, ErrDivisionByZero
	}
	return vm.floatResult(math.Mod(left.Float(), r))
}

func (vm *execVM) pow(left, right Value) (Value, error) {
	if vm.numericMode == NumericFloat64 {
		return vm.floatResult(math.Pow(left.Float(), right.Float()))
	}
	// For integer operands with non-negative exponent, try to compute integer result
	if left.IsInt() && right.IsInt() && right.Int() >= 0 {
		base := left.Int()
		exp := right.Int()
		if canIntPowWithoutOverflow(base, exp) {
			return Int(intPow(base, exp)), nil
		}
		return vm.floatResult(math.Pow(float64(base), float64(exp)))
	}
	return vm.floatResult(math.Pow(left.Float(), right.Float()))
}

func (vm *execVM) neg(v Value) (Value, error) {
	if v.IsDuration() {
		n := int64(v.bits)
		if n == math.MinInt64 {
			return Nil, &RuntimeError{Op: "-", Message: errDurationNegOverflow}
		}
		return Duration(time.Duration(-n)), nil
	}
	if v.IsTime() {
		return Nil, &RuntimeError{Op: "-", Message: "cannot negate time"}
	}
	if vm.numericMode == NumericFloat64 && v.IsNumber() {
		return vm.floatResult(-v.Float())
	}
	if v.IsInt() {
		n := v.Int()
		// Handle MinInt64: -MinInt64 overflows in two's complement
		if n == math.MinInt64 {
			return vm.floatResult(-float64(n))
		}
		return Int(-n), nil
	}
	return vm.floatResult(-v.Float())
}

func (vm *execVM) contains(collection, item Value) (bool, error) {
	switch {
	case collection.IsArray():
		arr := collection.Array()
		for i, elem := range arr {
			// Make 'in' on large arrays interruptible by sandbox limits.
			if err := vm.sandboxLoopCheck(i); err != nil {
				return false, err
			}
			if NewValue(elem).Equal(item) {
				return true, nil
			}
		}
		return false, nil
	case collection.IsMap():
		_, ok := collection.Map()[item.String()]
		return ok, nil
	case collection.IsString():
		// Empty string is contained in any string (matches Go's strings.Contains behavior)
		itemStr := item.String()
		if len(itemStr) == 0 {
			return true, nil
		}
		return len(collection.String()) >= len(itemStr) &&
			strings.Contains(collection.String(), itemStr), nil
	default:
		// Handle kindAny (native Go slices and maps) via reflection
		if vm.limits != nil && vm.limits.secure {
			// In Secure mode we explicitly deny probing external Go objects.
			if collection.kind == kindAny {
				return false, sandboxDeny("membership test on external Go objects is not allowed in secure mode")
			}
		}
		if collection.kind == kindAny && collection.ptr != nil {
			rv := reflect.ValueOf(collection.ptr)
			switch rv.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i < rv.Len(); i++ {
					if err := vm.sandboxLoopCheck(i); err != nil {
						return false, err
					}
					if NewValue(rv.Index(i).Interface()).Equal(item) {
						return true, nil
					}
				}
				return false, nil
			case reflect.Map:
				// Check if key exists in map
				keyVal := reflect.ValueOf(item.Interface())
				keyType := rv.Type().Key()
				if !keyVal.IsValid() {
					// Allow nil key only for interface{}-like keys (e.g. map[any]T)
					if keyType.Kind() == reflect.Interface {
						return rv.MapIndex(reflect.Zero(keyType)).IsValid(), nil
					}
					return false, nil
				}
				if keyVal.Type().AssignableTo(keyType) {
					return rv.MapIndex(keyVal).IsValid(), nil
				}
				if keyVal.Type().ConvertibleTo(keyType) {
					return rv.MapIndex(keyVal.Convert(keyType)).IsValid(), nil
				}
				// Try string key
				if keyType.Kind() == reflect.String {
					return rv.MapIndex(reflect.ValueOf(item.String())).IsValid(), nil
				}
				return false, nil
			}
		}
		return false, nil
	}
}

func (vm *execVM) index(obj, index Value) (Value, error) {
	switch {
	case obj.IsArray():
		i, ok := intFromInt64Checked(index.Int())
		if !ok {
			return Nil, ErrIndexOutOfBounds
		}
		arr := obj.Array()
		if i < 0 {
			i = len(arr) + i
		}
		if i < 0 || i >= len(arr) {
			return Nil, ErrIndexOutOfBounds
		}
		return NewValue(arr[i]), nil
	case obj.IsMap():
		key := index.String()
		if v, ok := obj.Map()[key]; ok {
			return NewValue(v), nil
		}
		return Nil, nil
	case obj.IsString():
		s := obj.String()
		runes := []rune(s)
		i, ok := intFromInt64Checked(index.Int())
		if !ok {
			return Nil, ErrIndexOutOfBounds
		}
		if i < 0 {
			i = len(runes) + i
		}
		if i < 0 || i >= len(runes) {
			return Nil, ErrIndexOutOfBounds
		}
		return String(string(runes[i])), nil
	default:
		// Handle kindAny (native Go slices and maps) via reflection
		if vm.limits != nil && vm.limits.secure {
			return Nil, sandboxDeny("indexing external Go objects is not allowed in secure mode")
		}
		if obj.kind == kindAny && obj.ptr != nil {
			rv := reflect.ValueOf(obj.ptr)
			switch rv.Kind() {
			case reflect.Slice, reflect.Array:
				i, ok := intFromInt64Checked(index.Int())
				if !ok {
					return Nil, ErrIndexOutOfBounds
				}
				length := rv.Len()
				if i < 0 {
					i = length + i
				}
				if i < 0 || i >= length {
					return Nil, ErrIndexOutOfBounds
				}
				return NewValue(rv.Index(i).Interface()), nil
			case reflect.Map:
				// Try to get value from map
				keyVal := reflect.ValueOf(index.Interface())
				keyType := rv.Type().Key()
				if !keyVal.IsValid() {
					// Allow nil key only for interface{}-like keys (e.g. map[any]T)
					if keyType.Kind() == reflect.Interface {
						result := rv.MapIndex(reflect.Zero(keyType))
						if result.IsValid() {
							return NewValue(result.Interface()), nil
						}
						return Nil, nil
					}
					return Nil, nil
				}
				if keyVal.Type().AssignableTo(keyType) {
					result := rv.MapIndex(keyVal)
					if result.IsValid() {
						return NewValue(result.Interface()), nil
					}
					return Nil, nil
				}
				if keyVal.Type().ConvertibleTo(keyType) {
					result := rv.MapIndex(keyVal.Convert(keyType))
					if result.IsValid() {
						return NewValue(result.Interface()), nil
					}
					return Nil, nil
				}
				// Try string key
				if keyType.Kind() == reflect.String {
					result := rv.MapIndex(reflect.ValueOf(index.String()))
					if result.IsValid() {
						return NewValue(result.Interface()), nil
					}
				}
				return Nil, nil
			}
		}
		return Nil, ErrNotIndexable
	}
}

func (vm *execVM) slice(obj, low, high, step Value) (Value, error) {
	switch {
	case obj.IsArray():
		arr := obj.Array()
		length := len(arr)
		l, h, s := 0, length, 1

		// Parse step first as it affects default low/high
		if !step.IsNil() {
			stepInt, ok := intFromInt64Checked(step.Int())
			if !ok {
				return Nil, ErrInvalidSliceIndex
			}
			s = stepInt
			if s == 0 {
				return Nil, ErrInvalidSliceIndex
			}
		}

		// Parse low/high with step-aware defaults
		if !low.IsNil() {
			li, ok := intFromInt64Checked(low.Int())
			if !ok {
				return Nil, ErrInvalidSliceIndex
			}
			l = li
		} else if s < 0 {
			l = length - 1 // Default for negative step
		}
		if !high.IsNil() {
			hi, ok := intFromInt64Checked(high.Int())
			if !ok {
				return Nil, ErrInvalidSliceIndex
			}
			h = hi
		} else if s < 0 {
			h = -length - 1 // Default for negative step (go past beginning)
		}

		// Handle negative indices
		if l < 0 {
			l = length + l
		}
		if h < 0 {
			h = length + h
		}

		// Build result with step
		if s > 0 {
			// Clamp indices for positive step
			if l < 0 {
				l = 0
			}
			if h > length {
				h = length
			}
			if l >= h {
				return Array([]any{}), nil
			}
			result := make([]any, 0, (h-l+s-1)/s)
			iter := 0
			for i := l; i < h; i += s {
				if err := vm.sandboxLoopCheck(iter); err != nil {
					return Nil, err
				}
				iter++
				result = append(result, arr[i])
			}
			return Array(result), nil
		} else {
			// Negative step: iterate backwards
			if l >= length {
				l = length - 1
			}
			if h < -1 {
				h = -1
			}
			if l <= h {
				return Array([]any{}), nil
			}
			result := make([]any, 0, (l-h-s-1)/(-s))
			iter := 0
			for i := l; i > h; i += s {
				if err := vm.sandboxLoopCheck(iter); err != nil {
					return Nil, err
				}
				iter++
				if i >= 0 && i < length {
					result = append(result, arr[i])
				}
			}
			return Array(result), nil
		}

	case obj.IsString():
		s := obj.String()
		runes := []rune(s)
		length := len(runes)
		l, h, st := 0, length, 1

		// Parse step first
		if !step.IsNil() {
			stepInt, ok := intFromInt64Checked(step.Int())
			if !ok {
				return Nil, ErrInvalidSliceIndex
			}
			st = stepInt
			if st == 0 {
				return Nil, ErrInvalidSliceIndex
			}
		}

		// Parse low/high with step-aware defaults
		if !low.IsNil() {
			li, ok := intFromInt64Checked(low.Int())
			if !ok {
				return Nil, ErrInvalidSliceIndex
			}
			l = li
		} else if st < 0 {
			l = length - 1
		}
		if !high.IsNil() {
			hi, ok := intFromInt64Checked(high.Int())
			if !ok {
				return Nil, ErrInvalidSliceIndex
			}
			h = hi
		} else if st < 0 {
			h = -length - 1
		}

		// Handle negative indices
		if l < 0 {
			l = length + l
		}
		if h < 0 {
			h = length + h
		}

		// Build result with step
		if st > 0 {
			if l < 0 {
				l = 0
			}
			if h > length {
				h = length
			}
			if l >= h {
				return String(""), nil
			}
			result := make([]rune, 0, (h-l+st-1)/st)
			iter := 0
			for i := l; i < h; i += st {
				if err := vm.sandboxLoopCheck(iter); err != nil {
					return Nil, err
				}
				iter++
				result = append(result, runes[i])
			}
			return String(string(result)), nil
		} else {
			if l >= length {
				l = length - 1
			}
			if h < -1 {
				h = -1
			}
			if l <= h {
				return String(""), nil
			}
			result := make([]rune, 0, (l-h-st-1)/(-st))
			iter := 0
			for i := l; i > h; i += st {
				if err := vm.sandboxLoopCheck(iter); err != nil {
					return Nil, err
				}
				iter++
				if i >= 0 && i < length {
					result = append(result, runes[i])
				}
			}
			return String(string(result)), nil
		}

	default:
		// Handle kindAny (native Go slices) via reflection
		if obj.kind == kindAny && obj.ptr != nil {
			if vm.limits != nil && vm.limits.secure {
				return Nil, sandboxDeny("slicing external Go objects is not allowed in secure mode")
			}
			rv := reflect.ValueOf(obj.ptr)
			if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
				length := rv.Len()
				l, h, s := 0, length, 1

				// Parse step
				if !step.IsNil() {
					stepInt, ok := intFromInt64Checked(step.Int())
					if !ok {
						return Nil, ErrInvalidSliceIndex
					}
					s = stepInt
					if s == 0 {
						return Nil, ErrInvalidSliceIndex
					}
				}

				// Parse low/high
				if !low.IsNil() {
					li, ok := intFromInt64Checked(low.Int())
					if !ok {
						return Nil, ErrInvalidSliceIndex
					}
					l = li
				} else if s < 0 {
					l = length - 1
				}
				if !high.IsNil() {
					hi, ok := intFromInt64Checked(high.Int())
					if !ok {
						return Nil, ErrInvalidSliceIndex
					}
					h = hi
				} else if s < 0 {
					h = -length - 1
				}

				// Handle negative indices
				if l < 0 {
					l = length + l
				}
				if h < 0 {
					h = length + h
				}

				// Build result
				if s > 0 {
					if l < 0 {
						l = 0
					}
					if h > length {
						h = length
					}
					if l >= h {
						return Array([]any{}), nil
					}
					result := make([]any, 0, (h-l+s-1)/s)
					iter := 0
					for i := l; i < h; i += s {
						if err := vm.sandboxLoopCheck(iter); err != nil {
							return Nil, err
						}
						iter++
						result = append(result, rv.Index(i).Interface())
					}
					return Array(result), nil
				} else {
					if l >= length {
						l = length - 1
					}
					if h < -1 {
						h = -1
					}
					if l <= h {
						return Array([]any{}), nil
					}
					result := make([]any, 0, (l-h-s-1)/(-s))
					iter := 0
					for i := l; i > h; i += s {
						if err := vm.sandboxLoopCheck(iter); err != nil {
							return Nil, err
						}
						iter++
						if i >= 0 && i < length {
							result = append(result, rv.Index(i).Interface())
						}
					}
					return Array(result), nil
				}
			}
		}
		return Nil, ErrNotIndexable
	}
}

func (vm *execVM) call(fn Value, args []Value) (Value, error) {
	vm.callDepth++
	if vm.callDepth > vm.maxDepth {
		return Nil, ErrMaxCallDepth
	}
	defer func() { vm.callDepth-- }()

	switch fn.kind {
	case kindFunc:
		// Check if it's a closure (lambda)
		if closure, ok := fn.ptr.(*closure); ok {
			return vm.callClosure(closure, args)
		}

		// Check if it's a builtinFunc
		if bf, ok := fn.ptr.(*builtinFunc); ok {
			// Sandbox: ensure function is allowed even if this builtinFunc came from env
			// (opLoadVar checks only builtins[name], but env can also carry functions).
			if vm.limits != nil {
				if err := vm.checkFunctionAllowed(bf.name); err != nil {
					return Nil, err
				}
			}
			out, err := bf.fn(vm, args)
			if err != nil {
				return Nil, err
			}
			if err := vm.rejectNonFinite(out); err != nil {
				return Nil, err
			}
			return out, nil
		}

		// Sandbox: disallow calling native Go funcs that are not "named" builtins.
		// - In Secure mode: always deny
		// - Otherwise: deny only when allow/deny lists are active (prevents allowlist bypass via env functions)
		if vm.limits != nil && (vm.limits.secure || vm.limits.allowedFuncs != nil || vm.limits.deniedFuncs != nil) {
			return Nil, sandboxErr(ErrFunctionNotAllowed, sandboxFunctionDenied, ErrFunctionNotAllowed.Error()+": (env function)", 0, 0)
		}

		// Native Go function
		out, err := vm.callNative(fn.ptr, args)
		if err != nil {
			return Nil, err
		}
		if err := vm.rejectNonFinite(out); err != nil {
			return Nil, err
		}
		return out, nil
	default:
		return Nil, ErrNotCallable
	}
}

func (vm *execVM) callClosure(closure *closure, args []Value) (Value, error) {
	// Create a new vm for the closure, inheriting sandbox limits
	var innerVM *execVM
	if vm.limits != nil {
		innerVM = newVMWithLimits(closure.code, vm.builtins, vm.limits)
		// Inherit current ops count so limits are cumulative
		innerVM.opsCount = vm.opsCount
		innerVM.callDepth = vm.callDepth
		// Inherit maxDepth even with sandbox enabled when sandbox doesn't override it.
		// This ensures Engine.WithMaxCallDepth applies consistently to nested closures.
		if vm.limits.maxCallDepth <= 0 {
			innerVM.maxDepth = vm.maxDepth
		}
	} else {
		innerVM = newVM(closure.code, vm.builtins)
		// Inherit maxDepth even without sandbox (for WithMaxCallDepth without WithSandbox)
		innerVM.maxDepth = vm.maxDepth
		innerVM.callDepth = vm.callDepth
	}
	// Inherit strict mode, numeric/non-finite policy, and context from parent vm
	innerVM.strictMode = vm.strictMode
	innerVM.numericMode = vm.numericMode
	innerVM.nonFiniteMode = vm.nonFiniteMode
	innerVM.ctx = vm.ctx
	defer func() {
		// Propagate ops count back to parent vm
		if vm.limits != nil {
			vm.opsCount = innerVM.opsCount
		}
		releaseVM(innerVM)
	}()

	// Copy closure environment (captured variables)
	for k, v := range closure.env {
		innerVM.env[k] = v
	}

	// Bind arguments by position ($0, $1, etc.) and by name
	for i, arg := range args {
		// Positional binding for opLoadEnv
		innerVM.env[fmt.Sprintf("$%d", i)] = arg
		// Named binding for regular variable access (needed for nested lambdas)
		if i < len(closure.params) {
			innerVM.env[closure.params[i]] = arg
		}
	}

	// Special: first argument is also available as "#" in pipe context
	if len(args) > 0 {
		innerVM.env["#"] = args[0]
	}

	return innerVM.execute()
}

func (vm *execVM) callNative(fn any, args []Value) (result Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = Nil
			err = &RuntimeError{Message: fmt.Sprintf("panic in native function call: %v", r)}
		}
	}()

	rv := reflect.ValueOf(fn)
	if rv.Kind() != reflect.Func {
		return Nil, ErrNotCallable
	}

	rt := rv.Type()
	numIn := rt.NumIn()
	isVariadic := rt.IsVariadic()

	// Check argument count
	if isVariadic {
		if len(args) < numIn-1 {
			return Nil, ErrArgumentCount
		}
	} else {
		if len(args) != numIn {
			return Nil, ErrArgumentCount
		}
	}

	// Convert arguments
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		var targetType reflect.Type
		switch {
		case isVariadic && i >= numIn-1:
			targetType = rt.In(numIn - 1).Elem()
		case i < numIn:
			targetType = rt.In(i)
		default:
			in[i] = reflect.ValueOf(arg.Interface())
			continue
		}

		in[i] = vm.convertArg(arg, targetType)
	}

	// Call function
	out := rv.Call(in)

	// Handle return values
	if len(out) == 0 {
		return Nil, nil
	}

	// Check for error return
	if len(out) == 2 {
		// Check if second return value is an error (only call IsNil on nillable types)
		errVal := out[1]
		errKind := errVal.Kind()
		isNil := false
		switch errKind {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
			isNil = errVal.IsNil()
		}
		if !isNil {
			if err, ok := errVal.Interface().(error); ok && err != nil {
				return Nil, err
			}
		}
		return NewValue(out[0].Interface()), nil
	}

	return NewValue(out[0].Interface()), nil
}

func (vm *execVM) convertArg(arg Value, targetType reflect.Type) reflect.Value {
	// Handle nil safely: reflect.ValueOf(nil) is invalid and can panic in Call().
	if arg.IsNil() {
		// For any reference/interface type, use typed nil/zero.
		switch targetType.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
			return reflect.Zero(targetType)
		default:
			return reflect.Zero(targetType)
		}
	}
	switch targetType.Kind() {
	case reflect.Int:
		return reflect.ValueOf(int(arg.Int()))
	case reflect.Int8:
		return reflect.ValueOf(int8(arg.Int()))
	case reflect.Int16:
		return reflect.ValueOf(int16(arg.Int()))
	case reflect.Int32:
		return reflect.ValueOf(int32(arg.Int()))
	case reflect.Int64:
		return reflect.ValueOf(arg.Int())
	case reflect.Uint:
		return reflect.ValueOf(uint(arg.Int()))
	case reflect.Uint8:
		return reflect.ValueOf(uint8(arg.Int()))
	case reflect.Uint16:
		return reflect.ValueOf(uint16(arg.Int()))
	case reflect.Uint32:
		return reflect.ValueOf(uint32(arg.Int()))
	case reflect.Uint64:
		return reflect.ValueOf(uint64(arg.Int()))
	case reflect.Float32:
		return reflect.ValueOf(float32(arg.Float()))
	case reflect.Float64:
		return reflect.ValueOf(arg.Float())
	case reflect.String:
		return reflect.ValueOf(arg.String())
	case reflect.Bool:
		return reflect.ValueOf(arg.Bool())
	case reflect.Slice:
		if arg.IsArray() {
			// Convert []any to []T when native function expects typed slice.
			src := arg.Array()
			// Fast path: []any
			if targetType.Elem().Kind() == reflect.Interface && targetType.Elem().NumMethod() == 0 {
				return reflect.ValueOf(src)
			}
			dst := reflect.MakeSlice(targetType, len(src), len(src))
			elemType := targetType.Elem()
			for i := range src {
				ev := vm.convertArg(NewValue(src[i]), elemType)
				// Ensure assignable; best-effort Convert when possible.
				if !ev.IsValid() {
					ev = reflect.Zero(elemType)
				}
				switch {
				case ev.Type().AssignableTo(elemType):
					dst.Index(i).Set(ev)
				case ev.Type().ConvertibleTo(elemType):
					dst.Index(i).Set(ev.Convert(elemType))
				default:
					// Fallback: zero value (avoids panic; strict typing can be validated by user code).
					dst.Index(i).Set(reflect.Zero(elemType))
				}
			}
			return dst
		}
	case reflect.Interface:
		return reflect.ValueOf(arg.Interface())
	}
	return reflect.ValueOf(arg.Interface())
}

func (vm *execVM) copyEnv() map[string]Value {
	cp := make(map[string]Value, len(vm.env))
	for k, v := range vm.env {
		cp[k] = v
	}
	return cp
}

// closure represents a lambda/closure value.
type closure struct {
	code          *bytecode
	env           map[string]Value
	params        []string         // Parameter names for binding
	builtins      map[string]Value // Builtins available to closure (includes custom functions)
	limits        *sandboxLimits   // Sandbox limits inherited from parent vm
	maxDepth      int              // Max call depth inherited from parent vm
	strictMode    bool             // Strict mode flag inherited from parent vm
	numericMode   NumericMode      // Numeric policy inherited from parent vm
	nonFiniteMode NonFiniteMode    // Non-finite policy inherited from parent vm
}

// checkContext returns [ErrOperationCancelled] when the Run context is done.
func (vm *execVM) checkContext() error {
	if vm == nil || vm.ctx == nil {
		return nil
	}
	select {
	case <-vm.ctx.Done():
		return ErrOperationCancelled
	default:
		return nil
	}
}

// checkLimits checks sandbox resource limits and context cancellation.
// Sandbox timeout and context deadline both apply; whichever is observed first wins.
func (vm *execVM) checkLimits() error {
	if err := vm.checkContext(); err != nil {
		return err
	}

	limits := vm.limits
	if limits == nil {
		return nil
	}

	// Check ops limit
	if limits.maxOps > 0 && vm.opsCount >= limits.maxOps {
		return sandboxErr(ErrOpsLimitExceeded, sandboxOpsLimit, ErrOpsLimitExceeded.Error(), limits.maxOps, vm.opsCount)
	}

	// Check timeout
	if !limits.deadline.IsZero() && time.Now().After(limits.deadline) {
		return sandboxErr(ErrTimeout, sandboxTimeout, ErrTimeout.Error(), 0, 0)
	}

	// Check stack depth
	if limits.maxStackDepth > 0 && vm.sp >= limits.maxStackDepth {
		return sandboxErr(ErrStackOverflow, sandboxStackOverflow, ErrStackOverflow.Error(), int64(limits.maxStackDepth), int64(vm.sp))
	}

	// Check memory limit (approximate based on stack and env size)
	if limits.maxMemory > 0 {
		// Memory estimation can be expensive (walks env recursively). Only do it periodically.
		// This still provides a strong safety net while keeping per-op overhead low.
		if (vm.opsCount & sandboxCheckMask) != 0 {
			return nil
		}
		// Estimate memory: vmValueBaseBytes per Value on stack + env entries
		estimatedMemory := int64(vm.sp) * vmValueBaseBytes
		for _, v := range vm.env {
			estimatedMemory += vm.estimateValueMemory(v)
			if estimatedMemory > limits.maxMemory {
				break
			}
		}
		if estimatedMemory > limits.maxMemory {
			return sandboxErr(ErrMemoryLimitExceeded, sandboxMemoryLimit, ErrMemoryLimitExceeded.Error(), limits.maxMemory, estimatedMemory)
		}
	}

	return nil
}

// estimateValueMemory estimates the memory usage of a Value in bytes.
// Uses depth limit to prevent stack overflow on cyclic structures.
func (vm *execVM) estimateValueMemory(v Value) int64 {
	return vm.estimateValueMemoryWithDepth(v, 0)
}

// estimateValueMemoryWithDepth estimates memory with depth tracking to prevent infinite recursion.
const maxMemoryEstimateDepth = 100

func (vm *execVM) estimateValueMemoryWithDepth(v Value, depth int) int64 {
	// Prevent infinite recursion on cyclic structures
	if depth > maxMemoryEstimateDepth {
		return vmValueBaseBytes
	}

	switch v.kind {
	case kindString:
		if s, ok := v.ptr.(string); ok {
			return vmValueBaseBytes + int64(len(s))
		}
	case kindArray:
		if arr, ok := v.ptr.([]any); ok {
			size := vmValueBaseBytes + int64(len(arr))*8 // Pointer per element
			for _, elem := range arr {
				size += vm.estimateValueMemoryWithDepth(NewValue(elem), depth+1)
			}
			return size
		}
	case kindMap:
		if m, ok := v.ptr.(map[string]any); ok {
			size := vmValueBaseBytes + int64(len(m))*32 // Rough estimate per entry
			for k, elem := range m {
				size += int64(len(k))
				size += vm.estimateValueMemoryWithDepth(NewValue(elem), depth+1)
			}
			return size
		}
	}
	return vmValueBaseBytes
}

// checkFunctionAllowed checks if a function is allowed by sandbox rules.
func (vm *execVM) checkFunctionAllowed(name string) error {
	limits := vm.limits
	if limits == nil {
		return nil
	}

	// Check denied list first (takes precedence)
	if limits.deniedFuncs != nil && limits.deniedFuncs[name] {
		return sandboxErr(ErrFunctionNotAllowed, sandboxFunctionDenied, ErrFunctionNotAllowed.Error()+": "+name, 0, 0)
	}

	// Check allowed list (if specified)
	if limits.allowedFuncs != nil && !limits.allowedFuncs[name] {
		return sandboxErr(ErrFunctionNotAllowed, sandboxFunctionDenied, ErrFunctionNotAllowed.Error()+": "+name, 0, 0)
	}

	return nil
}
