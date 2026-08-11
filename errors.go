package hyperex

import (
	"errors"
	"fmt"
	"strings"
)

// Error types for HyperEx operations.

// LexerError represents an error during tokenization.
type LexerError struct {
	Message string
	Line    int
	Column  int
	Source  string // Original source code
}

func (e *LexerError) Error() string {
	if e.Source != "" {
		return formatErrorWithContext("lexer error", e.Message, e.Source, e.Line, e.Column)
	}
	return fmt.Sprintf("lexer error at %d:%d: %s", e.Line, e.Column, e.Message)
}

// ParseError represents an error during parsing.
type ParseError struct {
	Message string
	token   token
	Source  string // Original source code
}

// Line returns the 1-based source line of the unexpected token.
func (e *ParseError) Line() int {
	if e == nil {
		return 0
	}
	return e.token.Line
}

// Column returns the 1-based source column of the unexpected token.
func (e *ParseError) Column() int {
	if e == nil {
		return 0
	}
	return e.token.Column
}

func (e *ParseError) Error() string {
	if e.Source != "" {
		return formatErrorWithContext("parse error", e.Message, e.Source, e.token.Line, e.token.Column)
	}
	return fmt.Sprintf("parse error at %d:%d: %s (got %s)",
		e.token.Line, e.token.Column, e.Message, e.token.Type)
}

// WithSource adds source context to the error
func (e *ParseError) WithSource(source string) *ParseError {
	e.Source = source
	return e
}

// TypeError represents a type checking error.
type TypeError struct {
	Message  string
	Node     Node
	Expected string
	Got      string
}

func (e *TypeError) Error() string {
	if e.Expected != "" && e.Got != "" {
		return fmt.Sprintf("type error: %s (expected %s, got %s)", e.Message, e.Expected, e.Got)
	}
	return fmt.Sprintf("type error: %s", e.Message)
}

// TypeCheckErrors represents multiple type checking errors.
type TypeCheckErrors struct {
	Errors []error
}

func (e *TypeCheckErrors) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d type errors:\n", len(e.Errors))
	for i, err := range e.Errors {
		fmt.Fprintf(&sb, "  %d. %s\n", i+1, err.Error())
	}
	return sb.String()
}

// Unwrap returns the first error for errors.Is/As compatibility.
func (e *TypeCheckErrors) Unwrap() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// Len returns the number of errors.
func (e *TypeCheckErrors) Len() int {
	if e == nil {
		return 0
	}
	return len(e.Errors)
}

// RuntimeError represents an error during expression evaluation.
// When Cause is a package sentinel (via Unwrap), errors.Is(err, Err…) works
// without comparing Message strings.
type RuntimeError struct {
	Message string
	Op      string
	Left    string
	Right   string
	Source  string // Original expression (optional)
	cause   error  // wrapped sentinel or underlying error
}

func (e *RuntimeError) Error() string {
	var msg string
	switch {
	case e.Left != "" && e.Right != "":
		msg = fmt.Sprintf("runtime error: cannot %s %s and %s", e.Op, e.Left, e.Right)
	case e.Op != "":
		msg = fmt.Sprintf("runtime error in '%s': %s", e.Op, e.Message)
	default:
		msg = fmt.Sprintf("runtime error: %s", e.Message)
	}
	if e.Source != "" {
		msg += fmt.Sprintf(" in expression: %s", e.Source)
	}
	return msg
}

// Unwrap returns the underlying cause for errors.Is / errors.As.
func (e *RuntimeError) Unwrap() error {
	return e.cause
}

// WithSource returns a copy of the error with source context added.
// All fields including the wrapped cause are preserved.
func (e *RuntimeError) WithSource(source string) *RuntimeError {
	return &RuntimeError{
		Message: e.Message,
		Op:      e.Op,
		Left:    e.Left,
		Right:   e.Right,
		Source:  source,
		cause:   e.cause,
	}
}

// UndefinedError represents an undefined variable or function reference.
type UndefinedError struct {
	Name       string
	Kind       string   // "variable" or "function"
	Suggestion string   // Suggested similar name
	Available  []string // List of available names
}

func (e *UndefinedError) Error() string {
	msg := fmt.Sprintf("undefined %s: %s", e.Kind, e.Name)
	if e.Suggestion != "" {
		msg += fmt.Sprintf(" (did you mean '%s'?)", e.Suggestion)
	}
	return msg
}

// CompileError represents an error during compilation.
type CompileError struct {
	Message string
	Node    Node
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("compile error: %s", e.Message)
}

// formatErrorWithContext creates a nicely formatted error message with source context
func formatErrorWithContext(errType, message, source string, line, col int) string {
	var sb strings.Builder

	// Header
	fmt.Fprintf(&sb, "%s at line %d, column %d: %s\n", errType, line, col, message)

	// Get the relevant line from source
	lines := strings.Split(source, "\n")
	if line > 0 && line <= len(lines) {
		lineContent := lines[line-1]

		// Line number prefix
		lineNum := fmt.Sprintf("%4d | ", line)
		sb.WriteString(lineNum)
		sb.WriteString(lineContent)
		sb.WriteString("\n")

		// Caret indicator
		spaces := strings.Repeat(" ", len(lineNum)+col-1)
		sb.WriteString(spaces)
		sb.WriteString("^")

		// Underline if token is multi-character
		// (simplified - just show single caret)
	}

	return sb.String()
}

// Sentinel errors for common runtime cases.
var (
	// ErrDivisionByZero is returned when dividing or taking modulo by zero.
	ErrDivisionByZero = errors.New("hyperex: division by zero")

	// ErrIndexOutOfBounds is returned when an array/string index is out of range.
	ErrIndexOutOfBounds = errors.New("hyperex: index out of bounds")

	// ErrInvalidOperation is returned for operations that are not valid for the operand types.
	ErrInvalidOperation = errors.New("hyperex: invalid operation")

	// ErrNilDereference is returned when a nil value is dereferenced.
	ErrNilDereference = errors.New("hyperex: nil dereference")

	// ErrNotCallable is returned when attempting to call a non-callable value.
	ErrNotCallable = errors.New("hyperex: value is not callable")

	// ErrArgumentCount is returned when a function is called with the wrong number of arguments.
	ErrArgumentCount = errors.New("hyperex: wrong number of arguments")

	// ErrInvalidType is returned when a value has an unexpected type for an operation.
	ErrInvalidType = errors.New("hyperex: invalid type")

	// ErrNotIterable is returned when attempting to iterate a non-iterable value.
	ErrNotIterable = errors.New("hyperex: value is not iterable")

	// ErrNotIndexable is returned when attempting to index a non-indexable value.
	ErrNotIndexable = errors.New("hyperex: value is not indexable")

	// ErrKeyNotFound is returned when a map key does not exist.
	ErrKeyNotFound = errors.New("hyperex: key not found")

	// ErrInvalidSliceIndex is returned when slice bounds or step are invalid.
	ErrInvalidSliceIndex = errors.New("hyperex: invalid slice index")

	// ErrIntegerOverflow is returned when an integer arithmetic operation overflows.
	ErrIntegerOverflow = errors.New("hyperex: integer overflow")

	// ErrInvalidRegex is returned when a regular expression pattern fails to compile.
	ErrInvalidRegex = errors.New("hyperex: invalid regular expression")

	// ErrMaxCallDepth is returned when nested function/lambda calls exceed the configured limit.
	ErrMaxCallDepth = errors.New("hyperex: maximum call depth exceeded")

	// ErrOperationCancelled is returned when evaluation is cancelled (e.g. via context).
	ErrOperationCancelled = errors.New("hyperex: operation cancelled")

	// ErrInvalidConversion is returned in strict mode when a type conversion fails.
	ErrInvalidConversion = errors.New("hyperex: invalid type conversion")
)

// wrapRuntime wraps a cause (typically a package sentinel) in a RuntimeError.
// Message is the cause text with the "hyperex: " prefix trimmed for display.
func wrapRuntime(cause error) *RuntimeError {
	msg := cause.Error()
	msg = strings.TrimPrefix(msg, "hyperex: ")
	return &RuntimeError{
		Message: msg,
		cause:   cause,
	}
}

// isKnownRuntimeErr reports whether err matches a known runtime sentinel.
func isKnownRuntimeErr(err error) bool {
	return errors.Is(err, ErrDivisionByZero) ||
		errors.Is(err, ErrIndexOutOfBounds) ||
		errors.Is(err, ErrInvalidOperation) ||
		errors.Is(err, ErrNilDereference) ||
		errors.Is(err, ErrNotCallable) ||
		errors.Is(err, ErrArgumentCount) ||
		errors.Is(err, ErrInvalidType) ||
		errors.Is(err, ErrNotIterable) ||
		errors.Is(err, ErrNotIndexable) ||
		errors.Is(err, ErrKeyNotFound) ||
		errors.Is(err, ErrInvalidSliceIndex) ||
		errors.Is(err, ErrIntegerOverflow) ||
		errors.Is(err, ErrInvalidRegex) ||
		errors.Is(err, ErrMaxCallDepth) ||
		errors.Is(err, ErrOperationCancelled) ||
		errors.Is(err, ErrInvalidConversion)
}

// NewRuntimeErrorf creates a new RuntimeError with formatted message
func NewRuntimeErrorf(format string, args ...any) *RuntimeError {
	return &RuntimeError{Message: fmt.Sprintf(format, args...)}
}

// NewTypeError creates a new TypeError with expected/got info
func NewTypeError(msg, expected, got string) *TypeError {
	return &TypeError{Message: msg, Expected: expected, Got: got}
}
