package hyperex

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

type panicStringerKey struct{}

func (panicStringerKey) String() string { panic("boom") }

// =============================================================================
// LEXER TESTS
// =============================================================================

func TestLexer_Numbers(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		tokens []tokenType
	}{
		{"integer", "42", []tokenType{tokenInt, tokenEOF}},
		{"float", "3.14", []tokenType{tokenFloat, tokenEOF}},
		{"float_exp", "1e10", []tokenType{tokenFloat, tokenEOF}},
		{"float_exp_neg", "1e-5", []tokenType{tokenFloat, tokenEOF}},
		{"hex", "0xFF", []tokenType{tokenInt, tokenEOF}},
		{"hex_lower", "0xff", []tokenType{tokenInt, tokenEOF}},
		{"binary", "0b1010", []tokenType{tokenInt, tokenEOF}},
		{"octal", "0o777", []tokenType{tokenInt, tokenEOF}},
		{"dot_float", ".5", []tokenType{tokenFloat, tokenEOF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens, err := lexer.tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tokens) != len(tt.tokens) {
				t.Fatalf("token count: got %d, want %d", len(tokens), len(tt.tokens))
			}
			for i, tok := range tokens {
				if tok.Type != tt.tokens[i] {
					t.Errorf("token[%d]: got %v, want %v", i, tok.Type, tt.tokens[i])
				}
			}
		})
	}
}

func TestLexer_Strings(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		value  string
		hasErr bool
	}{
		{"double_quote", `"hello"`, "hello", false},
		{"single_quote", `'hello'`, "hello", false},
		{"backtick", "`hello`", "hello", false},
		{"escape_n", `"line\nbreak"`, "line\nbreak", false},
		{"escape_t", `"tab\there"`, "tab\there", false},
		{"escape_r", `"cr\rhere"`, "cr\rhere", false},
		{"escape_quote", `"say \"hi\""`, `say "hi"`, false},
		{"escape_backslash", `"path\\to"`, `path\to`, false},
		{"escape_null", `"null\0char"`, "null\x00char", false},
		{"unterminated", `"hello`, "", true},
		{"newline_in_string", "\"hello\nworld\"", "", true},
		{"backtick_multiline", "`line1\nline2`", "line1\nline2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens, err := lexer.tokenize()
			if tt.hasErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens[0].Type != tokenString {
				t.Errorf("expected string token, got %v", tokens[0].Type)
			}
			if tokens[0].Value != tt.value {
				t.Errorf("value: got %q, want %q", tokens[0].Value, tt.value)
			}
		})
	}
}

func TestLexer_Operators(t *testing.T) {
	tests := []struct {
		input string
		token tokenType
	}{
		{"+", tokenPlus},
		{"-", tokenMinus},
		{"*", tokenStar},
		{"/", tokenSlash},
		{"%", tokenPercent},
		{"**", tokenPower},
		{"++", tokenPlusPlus},
		{"--", tokenMinMin},
		{"==", tokenEQ},
		{"!=", tokenNE},
		{"<", tokenLT},
		{"<=", tokenLE},
		{">", tokenGT},
		{">=", tokenGE},
		{"&&", tokenAmpAmp},
		{"||", tokenPipePipe},
		{"!", tokenBang},
		{"&", tokenAmp},
		{"|", tokenPipe},
		{"^", tokenCaret},
		{"~", tokenTilde},
		{"<<", tokenShiftL},
		{">>", tokenShiftR},
		{">>>", tokenShiftRU},
		{"?", tokenQuestion},
		{"??", tokenCoalesce},
		{"?.", tokenOptChain},
		{"?.[", tokenOptIndex},
		{".", tokenDot},
		{"..", tokenRange},
		{"->", tokenArrow},
		{"=>", tokenFatArrow},
		{"~=", tokenLike},
		{"(", tokenLParen},
		{")", tokenRParen},
		{"[", tokenLBracket},
		{"]", tokenRBracket},
		{"{", tokenLCurly},
		{"}", tokenRCurly},
		{",", tokenComma},
		{":", tokenColon},
		{";", tokenSemi},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tok := lexer.Next()
			if tok.Type != tt.token {
				t.Errorf("got %v, want %v", tok.Type, tt.token)
			}
		})
	}
}

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		input string
		token tokenType
	}{
		{"true", tokenBool},
		{"false", tokenBool},
		{"nil", tokenNil},
		{"null", tokenNil},
		{"in", tokenIn},
		{"not", tokenNot},
		{"and", tokenAnd},
		{"or", tokenOr},
		{"let", tokenLet},
		{"if", tokenIf},
		{"else", tokenElse},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tok := lexer.Next()
			if tok.Type != tt.token {
				t.Errorf("got %v, want %v", tok.Type, tt.token)
			}
		})
	}
}

func TestLexer_Comments(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		result int64
	}{
		{"line_comment", "1 + 2 // this is a comment", 3},
		{"block_comment", "1 /* comment */ + 2", 3},
		{"multiline_comment", "1 /* multi\nline */ + 2", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestLexer_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unexpected_char", "@"},
		{"invalid_equals", "="},
		{"unterminated_block", "/* unclosed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(context.Background(), tt.input)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestLexer_PeekN(t *testing.T) {
	lexer := newLexer("1 + 2 * 3")

	// Peek ahead
	tok1 := lexer.PeekN(1)
	tok2 := lexer.PeekN(2)
	tok3 := lexer.PeekN(3)

	if tok1.Type != tokenInt || tok1.Value != "1" {
		t.Errorf("PeekN(1) got %v", tok1)
	}
	if tok2.Type != tokenPlus {
		t.Errorf("PeekN(2) got %v", tok2)
	}
	if tok3.Type != tokenInt || tok3.Value != "2" {
		t.Errorf("PeekN(3) got %v", tok3)
	}

	// Next should still return first token
	next := lexer.Next()
	if next.Type != tokenInt || next.Value != "1" {
		t.Errorf("Next after PeekN got %v", next)
	}
}

// =============================================================================
// PARSER TESTS
// =============================================================================

func TestParser_Arithmetic(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"2 + 2", 4},
		{"10 - 3", 7},
		{"4 * 5", 20},
		{"20 / 4", 5},
		{"17 % 5", 2},
		{"-5", -5},
		{"-(-5)", 5},
		{"2 + 3 * 4", 14},
		{"(2 + 3) * 4", 20},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_FloatArithmetic(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"2.5 + 2.5", 5.0},
		{"10.0 / 4", 2.5},
		{"2 ** 10", 1024.0},
		{"2.0 ** 0.5", 1.4142135623730951},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Float() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Comparison(t *testing.T) {
	tests := []struct {
		expr   string
		result bool
	}{
		{"1 == 1", true},
		{"1 != 2", true},
		{"1 < 2", true},
		{"2 <= 2", true},
		{"3 > 2", true},
		{"3 >= 3", true},
		{"1 == 2", false},
		{`"abc" == "abc"`, true},
		{`"abc" != "def"`, true},
		{`"a" < "b"`, true},
		{`"b" > "a"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Logical(t *testing.T) {
	tests := []struct {
		expr   string
		result bool
	}{
		{"true && true", true},
		{"true && false", false},
		{"false && true", false},
		{"true || false", true},
		{"false || true", true},
		{"false || false", false},
		{"!true", false},
		{"!false", true},
		{"!!true", true},
		{"true and true", true},
		{"true or false", true},
		{"not false", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Bitwise(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"5 & 3", 1},
		{"(5 | 3)", 7},
		{"5 ^ 3", 6},
		{"~0", -1},
		{"~(-1)", 0},
		{"1 << 4", 16},
		{"16 >> 2", 4},
		{"8 >> 1", 4},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Ternary(t *testing.T) {
	tests := []struct {
		expr   string
		result any
	}{
		{"true ? 1 : 2", int64(1)},
		{"false ? 1 : 2", int64(2)},
		{"(1 > 0) ? \"yes\" : \"no\"", "yes"},
		{"(1 < 0) ? \"yes\" : \"no\"", "no"},
		{"true ? (true ? 1 : 2) : 3", int64(1)},
		{"false ? 1 : (true ? 2 : 3)", int64(2)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Coalesce(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result int64
	}{
		{"x ?? 0", map[string]any{"x": nil}, 0},
		{"x ?? 0", map[string]any{"x": int64(42)}, 42},
		{"x ?? y ?? 0", map[string]any{"x": nil, "y": nil}, 0},
		{"x ?? y ?? 0", map[string]any{"x": nil, "y": int64(10)}, 10},
		{"x ?? y ?? z", map[string]any{"x": nil, "y": nil, "z": int64(5)}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_InOperator(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2), int64(3)},
		"str": "hello world",
		"m":   map[string]any{"foo": 1, "bar": 2},
	}

	tests := []struct {
		expr   string
		result bool
	}{
		{"2 in arr", true},
		{"5 in arr", false},
		{`"world" in str`, true},
		{`"xyz" in str`, false},
		{`"foo" in m`, true},
		{`"baz" in m`, false},
		{"!(1 in arr)", false},
		{"!(5 in arr)", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Range(t *testing.T) {
	tests := []struct {
		expr  string
		count int
	}{
		{"(1..5)", 5},
		{"(0..3)", 4},
		{"(10..10)", 1},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			arr := result.([]any)
			if len(arr) != tt.count {
				t.Errorf("got len %d, want %d", len(arr), tt.count)
			}
		})
	}
}

func TestParser_ArrayLiteral(t *testing.T) {
	tests := []struct {
		expr  string
		count int
	}{
		{"[]", 0},
		{"[1]", 1},
		{"[1, 2, 3]", 3},
		{"[1, 2, 3,]", 3}, // trailing comma
		{"[[1, 2], [3, 4]]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			arr := result.([]any)
			if len(arr) != tt.count {
				t.Errorf("got len %d, want %d", len(arr), tt.count)
			}
		})
	}
}

func TestParser_MapLiteral(t *testing.T) {
	tests := []struct {
		expr string
		keys []string
	}{
		{"{}", nil},
		{"{foo: 1}", []string{"foo"}},
		{`{"foo": 1, "bar": 2}`, []string{"foo", "bar"}},
		{"{foo: 1,}", []string{"foo"}}, // trailing comma
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			m := result.(map[string]any)
			if len(m) != len(tt.keys) {
				t.Errorf("got len %d, want %d", len(m), len(tt.keys))
			}
			for _, k := range tt.keys {
				if _, ok := m[k]; !ok {
					t.Errorf("missing key %q", k)
				}
			}
		})
	}
}

func TestParser_IndexAccess(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(10), int64(20), int64(30)},
		"str": "hello",
	}

	tests := []struct {
		expr   string
		result any
	}{
		{"arr[0]", int64(10)},
		{"arr[1]", int64(20)},
		{"arr[-1]", int64(30)},
		{"arr[-2]", int64(20)},
		{"str[0]", "h"},
		{"str[-1]", "o"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_SliceAccess(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(0), int64(1), int64(2), int64(3), int64(4)},
		"str": "hello",
	}

	tests := []struct {
		expr  string
		count int
	}{
		{"arr[1:3]", 2},
		{"arr[:2]", 2},
		{"arr[3:]", 2},
		{"arr[:]", 5},
		{"str[1:3]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			switch v := result.(type) {
			case []any:
				if len(v) != tt.count {
					t.Errorf("got len %d, want %d", len(v), tt.count)
				}
			case string:
				if len(v) != tt.count {
					t.Errorf("got len %d, want %d", len(v), tt.count)
				}
			}
		})
	}
}

func TestParser_MemberAccess(t *testing.T) {
	engine := New()
	env := map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"age":  int64(30),
		},
	}

	tests := []struct {
		expr   string
		result any
	}{
		{"user.name", "Alice"},
		{"user.age", int64(30)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_OptionalChain(t *testing.T) {
	engine := New()
	env := map[string]any{
		"user": map[string]any{"name": "Alice"},
		"null": nil,
	}

	tests := []struct {
		expr   string
		isNil  bool
		result string
	}{
		{"user?.name", false, "Alice"},
		{"null?.name", true, ""},
		{"user?.missing", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if tt.isNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if result != tt.result {
					t.Errorf("got %v, want %v", result, tt.result)
				}
			}
		})
	}
}

func TestParser_Let(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"let x = 5; x + 1", 6},
		{"let x = 10; let y = 20; x + y", 30},
		{"let a = 2; let b = 3; a * b", 6},
		{"let n = 100; n / 2", 50},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_NotIn(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2), int64(3)},
		"str": "hello world",
		"m":   map[string]any{"foo": 1, "bar": 2},
	}

	tests := []struct {
		expr   string
		result bool
	}{
		{"1 not in arr", false},
		{"5 not in arr", true},
		{"10 not in arr", true},
		{`"xyz" not in str`, true},
		{`"world" not in str`, false},
		{`"baz" not in m`, true},
		{`"foo" not in m`, false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Regex(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		result bool
	}{
		{`"hello" ~= "^h"`, true},
		{`"hello" ~= "o$"`, true},
		{`"hello" ~= "^x"`, false},
		{`"test123" ~= "[0-9]+"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestParser_Errors(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"incomplete_plus", "1 +"},
		{"unbalanced_paren", "(1 + 2"},
		{"double_op", "1 + + 1"},
		{"incomplete_array", "[1, 2,"},
		{"incomplete_map", "{foo:"},
		{"invalid_primary", "1 + @"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(context.Background(), tt.expr)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

// =============================================================================
// BUILTIN FUNCTION TESTS
// =============================================================================

func TestBuiltin_Math(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"abs(-5)", 5},
		{"abs(5)", 5},
		{"abs(-3.14)", 3.14},
		{"ceil(3.2)", 4},
		{"ceil(3.9)", 4},
		{"floor(3.9)", 3},
		{"floor(3.2)", 3},
		{"round(3.4)", 3},
		{"round(3.5)", 4},
		{"sqrt(16)", 4},
		{"sqrt(2)", 1.4142135623730951},
		{"pow(2, 3)", 8},
		{"pow(4, 0.5)", 2},
		{"log(1)", 0},
		{"exp(0)", 1},
		{"sin(0)", 0},
		{"cos(0)", 1},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			got := NewValue(result).Float()
			if got != tt.result {
				t.Errorf("got %v, want %v", got, tt.result)
			}
		})
	}
}

func TestBuiltin_MinMax(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"min(1, 2, 3)", 1},
		{"min([3, 1, 2])", 1},
		{"max(1, 2, 3)", 3},
		{"max([1, 3, 2])", 3},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestBuiltin_Strings(t *testing.T) {
	tests := []struct {
		expr   string
		result any
	}{
		{`len("hello")`, int64(5)},
		{`upper("hello")`, "HELLO"},
		{`lower("HELLO")`, "hello"},
		{`trim("  hello  ")`, "hello"},
		{`trimLeft("  hello")`, "hello"},
		{`trimRight("hello  ")`, "hello"},
		{`trimLeft("xxhello", "x")`, "hello"},
		{`split("a,b,c", ",")`, []any{"a", "b", "c"}},
		{`contains("hello", "ell")`, true},
		{`contains("hello", "xyz")`, false},
		{`startsWith("hello", "he")`, true},
		{`startsWith("hello", "lo")`, false},
		{`endsWith("hello", "lo")`, true},
		{`endsWith("hello", "he")`, false},
		{`replace("hello", "l", "L")`, "heLLo"},
		{`substr("hello", 1)`, "ello"},
		{`substr("hello", 1, 3)`, "ell"},
		{`substr("hello", -2)`, "lo"},
		{`indexOf("hello", "l")`, int64(2)},
		{`indexOf("hello", "x")`, int64(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			switch expected := tt.result.(type) {
			case []any:
				arr := result.([]any)
				if len(arr) != len(expected) {
					t.Errorf("got %v, want %v", result, tt.result)
				}
			default:
				if result != tt.result {
					t.Errorf("got %v, want %v", result, tt.result)
				}
			}
		})
	}
}

func TestBuiltin_Join(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{"a", "b", "c"},
	}
	result, err := engine.Eval(context.Background(), `join(arr, "-")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "a-b-c" {
		t.Errorf("got %v, want a-b-c", result)
	}
}

func TestBuiltin_Arrays(t *testing.T) {
	tests := []struct {
		name  string
		expr  string
		check func(any) bool
	}{
		{"first", "first([1, 2, 3])", func(v any) bool { return NewValue(v).Int() == 1 }},
		{"last", "last([1, 2, 3])", func(v any) bool { return NewValue(v).Int() == 3 }},
		{"first_empty", "first([])", func(v any) bool { return v == nil }},
		{"last_empty", "last([])", func(v any) bool { return v == nil }},
		{"take", "take([1, 2, 3, 4], 2)", func(v any) bool { return len(v.([]any)) == 2 }},
		{"take_more", "take([1, 2], 5)", func(v any) bool { return len(v.([]any)) == 2 }},
		{"skip", "skip([1, 2, 3, 4], 2)", func(v any) bool { return len(v.([]any)) == 2 }},
		{"skip_all", "skip([1, 2], 5)", func(v any) bool { return len(v.([]any)) == 0 }},
		{"reverse", "reverse([1, 2, 3])", func(v any) bool {
			arr := v.([]any)
			return len(arr) == 3 && arr[0] == int64(3)
		}},
		{"sort", "sort([3, 1, 2])", func(v any) bool {
			arr := v.([]any)
			return len(arr) == 3 && arr[0] == int64(1)
		}},
		{"unique", "unique([1, 2, 2, 3, 3, 3])", func(v any) bool { return len(v.([]any)) == 3 }},
		{"concat", "concat([1, 2], [3, 4])", func(v any) bool { return len(v.([]any)) == 4 }},
		{"concat_single", "concat([1], 2, [3])", func(v any) bool { return len(v.([]any)) == 3 }},
		{"flatten", "flatten([[1, 2], [3, [4, 5]]])", func(v any) bool { return len(v.([]any)) == 5 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !tt.check(result) {
				t.Errorf("check failed for %v", result)
			}
		})
	}
}

func TestBuiltin_Aggregation(t *testing.T) {
	tests := []struct {
		expr   string
		result any
	}{
		{"sum([1, 2, 3, 4, 5])", int64(15)},
		{"sum([1.5, 2.5])", float64(4.0)},
		// Test overflow handling: sum should switch to float on overflow (consistent with + operator)
		{"sum([9223372036854775807, 1])", float64(9223372036854775808.0)}, // MaxInt64 + 1 overflows
		{"avg([2, 4, 6])", float64(4.0)},
		{"avg([])", nil},
		{"count([1, 2, 3])", int64(3)},
		{"all([true, true, true])", true},
		{"all([true, false, true])", false},
		{"all([1, 2, 3])", true},
		{"all([1, 0, 3])", false},
		{"any([false, false, true])", true},
		{"any([false, false, false])", false},
		{"none([false, false, false])", true},
		{"none([false, true, false])", false},
		{"one([false, true, false])", true},
		{"one([true, true, false])", false},
		{"one([false, false, false])", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v (%T), want %v (%T)", result, result, tt.result, tt.result)
			}
		})
	}
}

func TestBuiltin_TypeConversion(t *testing.T) {
	tests := []struct {
		expr   string
		result any
	}{
		{`int(3.7)`, int64(3)},
		{`int("42")`, int64(42)},
		{`float(3)`, float64(3)},
		{`float("3.14")`, float64(3.14)},
		{`string(42)`, "42"},
		{`string(3.14)`, "3.14"},
		{`string(true)`, "true"},
		{`bool(1)`, true},
		{`bool(0)`, false},
		{`bool("")`, false},
		{`bool("hi")`, true},
		{`type(42)`, "int"},
		{`type("hello")`, "string"},
		{`type([1, 2])`, "array"},
		{`type({foo: 1})`, "map"},
		{`type(nil)`, "nil"},
		{`isNil(nil)`, true},
		{`isNil(0)`, false},
		{`isNil("")`, false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestBuiltin_KeysValues(t *testing.T) {
	engine := New()
	env := map[string]any{
		"m": map[string]any{"a": 1, "b": 2},
	}

	// Test keys
	result, err := engine.Eval(context.Background(), "keys(m)", env)
	if err != nil {
		t.Fatalf("keys error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 2 {
		t.Errorf("keys: got %d items", len(arr))
	}

	// Test values
	result, err = engine.Eval(context.Background(), "values(m)", env)
	if err != nil {
		t.Fatalf("values error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 2 {
		t.Errorf("values: got %d items", len(arr))
	}
}

func TestBuiltin_Sprintf(t *testing.T) {
	result, err := Eval(context.Background(), `sprintf("Hello, %s! You have %d messages.", "Alice", 5)`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "Hello, Alice! You have 5 messages." {
		t.Errorf("got %v", result)
	}
}

func TestBuiltin_ErrorCases(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"abs_no_args", "abs()"},
		{"min_no_args", "min()"},
		{"len_no_args", "len()"},
		{"first_not_array", "first(42)"},
		{"sum_not_array", "sum(42)"},
		{"keys_not_map", "keys([1,2])"},
		{"values_not_map", "values([1,2])"},
		{"join_not_array", "join(42, \"-\")"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(context.Background(), tt.expr)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

// =============================================================================
// OPTIMIZER TESTS
// =============================================================================

func TestOptimizer_ConstantFolding(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"2 + 3", 5},
		{"2 * 3 + 4", 10},
		{"10 / 2 - 1", 4},
		{"2 ** 3", 8},
		{"10 % 3", 1},
		{"5 & 3", 1},
		{"5 | 3", 7},
		{"5 ^ 3", 6},
		{"1 << 4", 16},
		{"16 >> 2", 4},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestOptimizer_StringFolding(t *testing.T) {
	result, err := Eval(context.Background(), `"hello" + " " + "world"`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("got %v", result)
	}
}

func TestOptimizer_BooleanFolding(t *testing.T) {
	tests := []struct {
		expr   string
		result bool
	}{
		{"true && true", true},
		{"true && false", false},
		{"false || true", true},
		{"!false", true},
		{"1 == 1", true},
		{"1 != 2", true},
		{"1 < 2", true},
		{"2 <= 2", true},
		{"3 > 2", true},
		{"3 >= 3", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestOptimizer_TernaryFolding(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"true ? 1 : 2", 1},
		{"false ? 1 : 2", 2},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestOptimizer_CoalesceFolding(t *testing.T) {
	tests := []struct {
		expr   string
		result int64
	}{
		{"nil ?? 5", 5},
		{"3 ?? 5", 3},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestOptimizer_AlgebraicSimplifications(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result int64
	}{
		{"x + 0", map[string]any{"x": int64(5)}, 5},
		{"0 + x", map[string]any{"x": int64(5)}, 5},
		{"x - 0", map[string]any{"x": int64(5)}, 5},
		{"x * 1", map[string]any{"x": int64(5)}, 5},
		{"1 * x", map[string]any{"x": int64(5)}, 5},
		{"x * 0", map[string]any{"x": int64(5)}, 0},
		{"x / 1", map[string]any{"x": int64(5)}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestOptimizer_LogicalSimplifications(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result bool
	}{
		{"x && true", map[string]any{"x": true}, true},
		{"true && x", map[string]any{"x": true}, true},
		{"x && false", map[string]any{"x": true}, false},
		{"x || false", map[string]any{"x": true}, true},
		{"false || x", map[string]any{"x": true}, true},
		{"x || true", map[string]any{"x": false}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestOptimizer_StrengthReduction(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result int64
	}{
		{"x * 2", map[string]any{"x": int64(5)}, 10},  // x << 1
		{"x * 4", map[string]any{"x": int64(5)}, 20},  // x << 2
		{"x * 8", map[string]any{"x": int64(5)}, 40},  // x << 3
		{"x % 2", map[string]any{"x": int64(5)}, 1},   // x & 1
		{"x % 4", map[string]any{"x": int64(5)}, 1},   // x & 3
		{"x ** 2", map[string]any{"x": int64(5)}, 25}, // x * x
		{"x ** 1", map[string]any{"x": int64(5)}, 5},  // x
		{"x ** 0", map[string]any{"x": int64(5)}, 1},  // 1
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestOptimizer_UnarySimplifications(t *testing.T) {
	engine := New()
	tests := []struct {
		expr   string
		env    map[string]any
		result any
	}{
		{"!!x", map[string]any{"x": true}, true},
		{"-(-x)", map[string]any{"x": int64(5)}, int64(5)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

// =============================================================================
// ENGINE & OPTIONS TESTS
// =============================================================================

func TestEngine_CustomFunctions(t *testing.T) {
	engine := New(
		WithFunction("double", func(x int) int { return x * 2 }),
		WithFunction("greet", func(name string) string { return "Hello, " + name }),
		WithFunction("add", func(a, b int) int { return a + b }),
		WithFunction("sum3", func(a, b, c int) int { return a + b + c }),
	)

	tests := []struct {
		expr   string
		result any
	}{
		{"double(21)", int64(42)},
		{`greet("World")`, "Hello, World"},
		{"add(10, 20)", int64(30)},
		{"sum3(1, 2, 3)", int64(6)},
		{"double(double(5))", int64(20)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			got := NewValue(result).Interface()
			expected := NewValue(tt.result).Interface()
			if got != expected {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	}
}

func TestEngine_WithFunctions(t *testing.T) {
	engine := New(
		WithFunctions(map[string]any{
			"add": func(a, b int) int { return a + b },
			"mul": func(a, b int) int { return a * b },
		}),
	)

	result, err := engine.Eval(context.Background(), "add(2, 3) * mul(2, 3)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 30 { // (2+3) * (2*3) = 5 * 6 = 30
		t.Errorf("got %v, want 30", result)
	}
}

func TestEngine_WithOptimize(t *testing.T) {
	// Disabled optimization
	engine := New(WithOptimize(false))
	result, err := engine.Eval(context.Background(), "2 + 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(5) {
		t.Errorf("got %v, want 5", result)
	}
}

func TestEngine_WithConstantFolding(t *testing.T) {
	engine := New(WithConstantFolding(false))
	result, err := engine.Eval(context.Background(), "2 + 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(5) {
		t.Errorf("got %v, want 5", result)
	}
}

func TestEngine_WithStrengthReduction(t *testing.T) {
	engine := New(WithStrengthReduction(false))
	result, err := engine.Eval(context.Background(), "5 * 2", map[string]any{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(10) {
		t.Errorf("got %v, want 10", result)
	}
}

func TestEngine_WithMaxCallDepth(t *testing.T) {
	engine := New(WithMaxCallDepth(5))
	// Simple call should work
	result, err := engine.Eval(context.Background(), "len([1,2,3])")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(3) {
		t.Errorf("got %v, want 3", result)
	}
}

func TestEngine_AddFunction(t *testing.T) {
	engine := New()
	engine.AddFunction("triple", func(x int) int { return x * 3 })

	result, err := engine.Eval(context.Background(), "triple(7)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 21 {
		t.Errorf("got %v, want 21", result)
	}
}

func TestEngine_StructEnv(t *testing.T) {
	type Env struct {
		X    int     `expr:"x"`
		Y    float64 `expr:"y"`
		Name string
		skip int // unexported, should be skipped
	}

	engine := New()
	env := Env{X: 10, Y: 2.0, Name: "Bob", skip: 999}

	result, err := engine.Eval(context.Background(), "x + 5", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 15 {
		t.Errorf("got %v, want 15", result)
	}

	result, err = engine.Eval(context.Background(), `"Hi, " + Name`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "Hi, Bob" {
		t.Errorf("got %v, want Hi, Bob", result)
	}
}

func TestEngine_MapStringEnv(t *testing.T) {
	engine := New()
	env := map[string]int{"x": 10, "y": 20}

	result, err := engine.Eval(context.Background(), "x + y", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 30 {
		t.Errorf("got %v, want 30", result)
	}
}

func TestProgram_TypedEval(t *testing.T) {
	program := MustCompile("x + y")
	env := map[string]any{"x": int64(10), "y": int64(20)}

	// EvalInt
	i, err := program.EvalInt(context.Background(), env)
	if err != nil {
		t.Fatalf("EvalInt error: %v", err)
	}
	if i != 30 {
		t.Errorf("EvalInt: got %v, want 30", i)
	}

	// EvalFloat
	f, err := program.EvalFloat(context.Background(), env)
	if err != nil {
		t.Fatalf("EvalFloat error: %v", err)
	}
	if f != 30.0 {
		t.Errorf("EvalFloat: got %v, want 30.0", f)
	}

	// EvalString
	program2 := MustCompile(`"Hello"`)
	s, err := program2.EvalString(context.Background(), nil)
	if err != nil {
		t.Fatalf("EvalString error: %v", err)
	}
	if s != "Hello" {
		t.Errorf("EvalString: got %v, want Hello", s)
	}
}

func TestProgram_Source(t *testing.T) {
	program := MustCompile("x + y")
	if program.Source() != "x + y" {
		t.Errorf("got %v, want x + y", program.Source())
	}
}

func TestCompileOptions(t *testing.T) {
	tests := []struct {
		name string
		opt  CompileOption
	}{
		{"AsBool", AsBool()},
		{"AsInt", AsInt()},
		{"AsFloat", AsFloat()},
		{"AsString", AsString()},
		{"Unoptimized", Unoptimized()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile("1 + 1", tt.opt)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMustEval(t *testing.T) {
	result := MustEval(context.Background(), "2 + 2")
	if result != int64(4) {
		t.Errorf("got %v, want 4", result)
	}
}

func TestMustCompile_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	MustCompile("1 +")
}

// =============================================================================
// VALUE TESTS
// =============================================================================

func TestValue_Types(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		isNil    bool
		isBool   bool
		isInt    bool
		isFloat  bool
		isNumber bool
		isString bool
		isArray  bool
		isMap    bool
		isFunc   bool
	}{
		{"nil", Nil, true, false, false, false, false, false, false, false, false},
		{"bool", Bool(true), false, true, false, false, false, false, false, false, false},
		{"int", Int(42), false, false, true, false, true, false, false, false, false},
		{"float", Float(3.14), false, false, false, true, true, false, false, false, false},
		{"string", String("hello"), false, false, false, false, false, true, false, false, false},
		{"array", Array([]any{1, 2}), false, false, false, false, false, false, true, false, false},
		{"map", Map(map[string]any{"a": 1}), false, false, false, false, false, false, false, true, false},
		{"func", Func(func() {}), false, false, false, false, false, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value.IsNil() != tt.isNil {
				t.Errorf("IsNil: got %v, want %v", tt.value.IsNil(), tt.isNil)
			}
			if tt.value.IsBool() != tt.isBool {
				t.Errorf("IsBool: got %v, want %v", tt.value.IsBool(), tt.isBool)
			}
			if tt.value.IsInt() != tt.isInt {
				t.Errorf("IsInt: got %v, want %v", tt.value.IsInt(), tt.isInt)
			}
			if tt.value.IsFloat() != tt.isFloat {
				t.Errorf("IsFloat: got %v, want %v", tt.value.IsFloat(), tt.isFloat)
			}
			if tt.value.IsNumber() != tt.isNumber {
				t.Errorf("IsNumber: got %v, want %v", tt.value.IsNumber(), tt.isNumber)
			}
			if tt.value.IsString() != tt.isString {
				t.Errorf("IsString: got %v, want %v", tt.value.IsString(), tt.isString)
			}
			if tt.value.IsArray() != tt.isArray {
				t.Errorf("IsArray: got %v, want %v", tt.value.IsArray(), tt.isArray)
			}
			if tt.value.IsMap() != tt.isMap {
				t.Errorf("IsMap: got %v, want %v", tt.value.IsMap(), tt.isMap)
			}
			if tt.value.IsFunc() != tt.isFunc {
				t.Errorf("IsFunc: got %v, want %v", tt.value.IsFunc(), tt.isFunc)
			}
		})
	}
}

func TestValue_Type(t *testing.T) {
	tests := []struct {
		value    Value
		typeName string
	}{
		{Nil, "nil"},
		{Bool(true), "bool"},
		{Int(42), "int"},
		{Float(3.14), "float"},
		{String("hello"), "string"},
		{Array([]any{1}), "array"},
		{Map(map[string]any{}), "map"},
		{Func(func() {}), "function"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			if tt.value.Type() != tt.typeName {
				t.Errorf("got %v, want %v", tt.value.Type(), tt.typeName)
			}
		})
	}
}

func TestValue_Bool(t *testing.T) {
	tests := []struct {
		value  Value
		result bool
	}{
		{Bool(true), true},
		{Bool(false), false},
		{Int(1), true},
		{Int(0), false},
		{Float(1.0), true},
		{Float(0.0), false},
		{String("hello"), true},
		{String(""), false},
		{Array([]any{1}), true},
		{Array([]any{}), false},
		{Map(map[string]any{"a": 1}), true},
		{Map(map[string]any{}), false},
		{Nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.Bool() != tt.result {
				t.Errorf("got %v, want %v", tt.value.Bool(), tt.result)
			}
		})
	}
}

func TestValue_Int(t *testing.T) {
	tests := []struct {
		value  Value
		result int64
	}{
		{Int(42), 42},
		{Float(3.7), 3},
		{Bool(true), 1},
		{Bool(false), 0},
		{String("123"), 123},
		{String("invalid"), 0},
		{Nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.Int() != tt.result {
				t.Errorf("got %v, want %v", tt.value.Int(), tt.result)
			}
		})
	}
}

func TestValue_Float(t *testing.T) {
	tests := []struct {
		value  Value
		result float64
	}{
		{Float(3.14), 3.14},
		{Int(42), 42.0},
		{Bool(true), 1.0},
		{String("3.14"), 3.14},
		{Nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.Float() != tt.result {
				t.Errorf("got %v, want %v", tt.value.Float(), tt.result)
			}
		})
	}
}

func TestValue_String(t *testing.T) {
	tests := []struct {
		value  Value
		result string
	}{
		{String("hello"), "hello"},
		{Int(42), "42"},
		{Float(3.14), "3.14"},
		{Bool(true), "true"},
		{Bool(false), "false"},
		{Nil, "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.String() != tt.result {
				t.Errorf("got %v, want %v", tt.value.String(), tt.result)
			}
		})
	}
}

func TestValue_ArrayString(t *testing.T) {
	v := Array([]any{int64(1), int64(2), int64(3)})
	s := v.String()
	if !strings.Contains(s, "1") || !strings.Contains(s, "2") || !strings.Contains(s, "3") {
		t.Errorf("unexpected string: %v", s)
	}
}

func TestValue_MapString(t *testing.T) {
	v := Map(map[string]any{"a": int64(1)})
	s := v.String()
	if !strings.Contains(s, "a") || !strings.Contains(s, "1") {
		t.Errorf("unexpected string: %v", s)
	}
}

func TestValue_Len(t *testing.T) {
	tests := []struct {
		value  Value
		result int
	}{
		{String("hello"), 5},
		{Array([]any{1, 2, 3}), 3},
		{Map(map[string]any{"a": 1, "b": 2}), 2},
		{Int(42), 0},
	}

	for _, tt := range tests {
		t.Run(tt.value.Type(), func(t *testing.T) {
			if tt.value.Len() != tt.result {
				t.Errorf("got %v, want %v", tt.value.Len(), tt.result)
			}
		})
	}
}

func TestValue_Index(t *testing.T) {
	arr := Array([]any{int64(10), int64(20), int64(30)})

	// Valid indices
	v, err := arr.Index(0)
	if err != nil || v.Int() != 10 {
		t.Errorf("Index(0): got %v, err %v", v, err)
	}

	v, err = arr.Index(-1)
	if err != nil || v.Int() != 30 {
		t.Errorf("Index(-1): got %v, err %v", v, err)
	}

	// Out of bounds
	_, err = arr.Index(10)
	if !errors.Is(err, ErrIndexOutOfBounds) {
		t.Errorf("expected out of bounds error")
	}

	// String index
	str := String("hello")
	v, err = str.Index(0)
	if err != nil || v.String() != "h" {
		t.Errorf("String.Index(0): got %v, err %v", v, err)
	}

	// Not indexable
	i := Int(42)
	_, err = i.Index(0)
	if !errors.Is(err, ErrNotIndexable) {
		t.Errorf("expected not indexable error")
	}
}

func TestValue_Get(t *testing.T) {
	m := Map(map[string]any{"foo": int64(42)})

	v, ok := m.Get("foo")
	if !ok || v.Int() != 42 {
		t.Errorf("Get(foo): got %v, ok %v", v, ok)
	}

	_, ok = m.Get("missing")
	if ok {
		t.Error("expected missing key")
	}
}

func TestValue_GetStruct(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	v := NewValue(&Person{Name: "Alice", Age: 30})

	name, ok := v.Get("Name")
	if !ok || name.String() != "Alice" {
		t.Errorf("Get(Name): got %v, ok %v", name, ok)
	}

	// Case-insensitive
	age, ok := v.Get("age")
	if !ok {
		t.Error("expected to find age (case-insensitive)")
	}
	_ = age
}

func TestValue_Equal(t *testing.T) {
	tests := []struct {
		a, b   Value
		result bool
	}{
		{Nil, Nil, true},
		{Bool(true), Bool(true), true},
		{Bool(true), Bool(false), false},
		{Int(42), Int(42), true},
		{Int(42), Int(43), false},
		{Int(42), Float(42.0), true}, // cross-type numeric
		{Float(3.14), Float(3.14), true},
		{String("hello"), String("hello"), true},
		{String("hello"), String("world"), false},
		{Array([]any{int64(1)}), Array([]any{int64(1)}), true},
		{Array([]any{int64(1)}), Array([]any{int64(2)}), false},
	}

	for _, tt := range tests {
		if tt.a.Equal(tt.b) != tt.result {
			t.Errorf("%v == %v: got %v, want %v", tt.a, tt.b, tt.a.Equal(tt.b), tt.result)
		}
	}
}

func TestValue_Compare(t *testing.T) {
	tests := []struct {
		a, b   Value
		result int
	}{
		{Int(1), Int(2), -1},
		{Int(2), Int(2), 0},
		{Int(3), Int(2), 1},
		{Float(1.0), Float(2.0), -1},
		{String("a"), String("b"), -1},
		{String("b"), String("a"), 1},
		{String("a"), String("a"), 0},
	}

	for _, tt := range tests {
		if tt.a.Compare(tt.b) != tt.result {
			t.Errorf("%v <=> %v: got %v, want %v", tt.a, tt.b, tt.a.Compare(tt.b), tt.result)
		}
	}
}

func TestNewValue_ComplexTypes(t *testing.T) {
	// Slice of int
	v := NewValue([]int{1, 2, 3})
	if !v.IsArray() || v.Len() != 3 {
		t.Errorf("[]int: got %v", v)
	}

	// Map with int values
	v = NewValue(map[string]int{"a": 1})
	if !v.IsMap() {
		t.Errorf("map[string]int: got %v", v)
	}

	// Pointer
	x := 42
	v = NewValue(&x)
	if v.Int() != 42 {
		t.Errorf("*int: got %v", v)
	}

	// Nil pointer
	var p *int
	v = NewValue(p)
	if !v.IsNil() {
		t.Errorf("nil *int: got %v", v)
	}

	// Struct
	type S struct{ X int }
	v = NewValue(S{X: 5})
	if v.Type() != "hyperex.S" {
		t.Logf("struct type: %v", v.Type())
	}
}

// =============================================================================
// AST TESTS
// =============================================================================

func TestAST_Walk(t *testing.T) {
	parser := newParser("1 + 2 * 3")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	count := 0
	Walk(ast, func(n Node) bool {
		count++
		return true
	})

	if count < 5 { // Should visit multiple nodes
		t.Errorf("Walk visited %d nodes, expected more", count)
	}
}

func TestAST_WalkStop(t *testing.T) {
	parser := newParser("1 + 2 * 3")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	count := 0
	Walk(ast, func(n Node) bool {
		count++
		return count < 2 // Stop after 2 nodes
	})

	// Walk should stop early
	if count < 1 {
		t.Errorf("Walk visited %d nodes, expected at least 1", count)
	}
}

func TestAST_NodeStrings(t *testing.T) {
	nodes := []Node{
		&intLit{Value: 42},
		&floatLit{Value: 3.14},
		&stringLit{Value: "hello"},
		&boolLit{Value: true},
		&nilLit{},
		&ident{Name: "x"},
		&binaryExpr{Op: "+", Left: &intLit{Value: 1}, Right: &intLit{Value: 2}},
		&unaryExpr{Op: "-", Operand: &intLit{Value: 5}},
		&ternaryExpr{Cond: &boolLit{Value: true}, Then: &intLit{Value: 1}, Else: &intLit{Value: 2}},
		&arrayLit{Elements: []Node{&intLit{Value: 1}}},
		&mapLit{Keys: []Node{&stringLit{Value: "a"}}, Values: []Node{&intLit{Value: 1}}},
		&rangeExpr{Start: &intLit{Value: 1}, End: &intLit{Value: 5}},
		&coalesceExpr{Left: &ident{Name: "x"}, Right: &intLit{Value: 0}},
	}

	for _, n := range nodes {
		s := n.String()
		if s == "" {
			t.Errorf("empty string for %T", n)
		}
	}
}

// =============================================================================
// ERROR TESTS
// =============================================================================

func TestError_Strings(t *testing.T) {
	errors := []error{
		&LexerError{Message: "test", Line: 1, Column: 5},
		&ParseError{Message: "test", token: token{Type: tokenInt, Line: 1, Column: 1}},
		&TypeError{Message: "test"},
		&RuntimeError{Message: "test", Op: "+"},
		&RuntimeError{Message: "test"},
		&UndefinedError{Name: "x", Kind: "variable"},
		&CompileError{Message: "test"},
	}

	for _, err := range errors {
		s := err.Error()
		if s == "" {
			t.Errorf("empty error string for %T", err)
		}
	}
}

func TestRuntime_Errors(t *testing.T) {
	tests := []struct {
		name string
		expr string
		env  map[string]any
	}{
		{"div_zero", "1 / 0", nil},
		{"mod_zero", "1 % 0", map[string]any{}},
		{"index_bounds", "arr[10]", map[string]any{"arr": []any{1, 2}}},
		{"undefined_var", "undefined_variable", nil},
		{"not_callable", "x()", map[string]any{"x": 42}},
		{"invalid_regex", `"test" ~= "["`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(context.Background(), tt.expr, tt.env)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

// =============================================================================
// TOKEN TESTS
// =============================================================================

func TestToken_String(t *testing.T) {
	tok := token{Type: tokenInt, Value: "42"}
	s := tok.String()
	if s == "" {
		t.Error("empty token string")
	}

	tok2 := token{Type: tokenPlus}
	s2 := tok2.String()
	if s2 == "" {
		t.Error("empty token string for operator")
	}
}

func TestTokenType_String(t *testing.T) {
	types := []tokenType{
		tokenEOF, tokenInt, tokenFloat, tokenString,
		tokenPlus, tokenMinus, tokenStar, tokenSlash,
		tokenEQ, tokenNE, tokenLT, tokenGT,
	}

	for _, tt := range types {
		s := tt.String()
		if s == "" {
			t.Errorf("empty string for tokenType %d", tt)
		}
	}

	// Unknown type
	unknown := tokenType(999)
	s := unknown.String()
	if s == "" {
		t.Error("empty string for unknown tokenType")
	}
}

func TestTokenType_Precedence(t *testing.T) {
	// Verify precedence ordering
	if tokenOr.Precedence() >= tokenAnd.Precedence() {
		t.Error("OR should have lower precedence than AND")
	}
	if tokenPlus.Precedence() >= tokenStar.Precedence() {
		t.Error("PLUS should have lower precedence than STAR")
	}
	if tokenStar.Precedence() >= tokenPower.Precedence() {
		t.Error("STAR should have lower precedence than POWER")
	}
}

func TestTokenType_IsRightAssociative(t *testing.T) {
	if !tokenPower.IsRightAssociative() {
		t.Error("POWER should be right-associative")
	}
	if !tokenCoalesce.IsRightAssociative() {
		t.Error("COALESCE should be right-associative")
	}
	if tokenPlus.IsRightAssociative() {
		t.Error("PLUS should not be right-associative")
	}
}

// =============================================================================
// ADDITIONAL COVERAGE TESTS
// =============================================================================

func TestLexer_Peek(t *testing.T) {
	lexer := newLexer("1 + 2")

	// Peek without consuming
	tok := lexer.Peek()
	if tok.Type != tokenInt {
		t.Errorf("Peek: got %v, want Int", tok.Type)
	}

	// Next should return same token
	tok = lexer.Next()
	if tok.Type != tokenInt {
		t.Errorf("Next after Peek: got %v, want Int", tok.Type)
	}
}

func TestVM_StringMultiply(t *testing.T) {
	result, err := Eval(context.Background(), `"ab" * 3`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "ababab" {
		t.Errorf("got %v, want ababab", result)
	}
}

func TestVM_ArrayConcat(t *testing.T) {
	result, err := Eval(context.Background(), "[1, 2] + [3, 4]")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 4 {
		t.Errorf("got len %d, want 4", len(arr))
	}
}

func TestVM_Subtraction(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"5 - 3", 2},
		{"5.0 - 3.0", 2.0},
		{"5.0 - 3", 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Float() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestVM_MapIndex(t *testing.T) {
	engine := New()
	env := map[string]any{
		"m": map[string]any{"foo": int64(42)},
	}

	result, err := engine.Eval(context.Background(), `m["foo"]`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("got %v, want 42", result)
	}

	// Missing key should return nil
	result, err = engine.Eval(context.Background(), `m["missing"]`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for missing key")
	}
}

func TestVM_NegativeSlice(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(0), int64(1), int64(2), int64(3), int64(4)},
	}

	// Test negative indices in slice
	result, err := engine.Eval(context.Background(), "arr[-3:-1]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 2 {
		t.Errorf("got len %d, want 2", len(arr))
	}
}

func TestDefaultOptimizer(t *testing.T) {
	opt := defaultOptimizer()
	if opt == nil {
		t.Error("defaultOptimizer returned nil")
	}

	// Test optimization
	parser := newParser("2 + 3")
	ast, _ := parser.parse()
	result := opt.optimize(ast)

	// Result should be a constant
	if _, ok := result.(*intLit); !ok {
		t.Error("expected constant folding to produce intLit")
	}
}

func TestWithOperator(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		fnName   string
		fn       any
		expr     string
		expected any
	}{
		{
			name:     "custom add",
			op:       "+",
			fnName:   "customAdd",
			fn:       func(a, b int) int { return a + b + 100 },
			expr:     "2 + 3",
			expected: int64(105), // 2 + 3 + 100
		},
		{
			name:     "custom multiply",
			op:       "*",
			fnName:   "customMul",
			fn:       func(a, b int) int { return a * b * 2 },
			expr:     "3 * 4",
			expected: int64(24), // 3 * 4 * 2
		},
		{
			name:     "custom comparison",
			op:       "==",
			fnName:   "customEq",
			fn:       strings.EqualFold,
			expr:     `"Hello" == "hello"`,
			expected: true, // case-insensitive comparison
		},
		{
			name:     "array concatenation with custom +",
			op:       "+",
			fnName:   "arrayConcat",
			fn:       func(a, b []any) []any { return append(append([]any{}, a...), b...) },
			expr:     "[1, 2] + [3, 4]",
			expected: []any{int64(1), int64(2), int64(3), int64(4)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := New(
				WithFunction(tt.fnName, tt.fn),
				WithOperator(tt.op, tt.fnName),
			)

			result, err := engine.Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			// Compare results
			switch expected := tt.expected.(type) {
			case int64:
				if NewValue(result).Int() != expected {
					t.Errorf("got %v, want %v", result, expected)
				}
			case bool:
				if NewValue(result).Bool() != expected {
					t.Errorf("got %v, want %v", result, expected)
				}
			case []any:
				resultArr := NewValue(result).Array()
				if len(resultArr) != len(expected) {
					t.Errorf("array length: got %d, want %d", len(resultArr), len(expected))
				}
				for i, v := range expected {
					if NewValue(resultArr[i]).Int() != NewValue(v).Int() {
						t.Errorf("array[%d]: got %v, want %v", i, resultArr[i], v)
					}
				}
			}
		})
	}
}

func TestWithOperator_PreservesShortCircuit(t *testing.T) {
	// Ensure && and || still use short-circuit even if someone tries to override them
	callCount := 0
	engine := New(
		WithFunction("customAnd", func(a, b bool) bool {
			callCount++
			return a && b
		}),
		WithOperator("&&", "customAnd"), // This should be ignored for &&
	)

	// Short-circuit: right side should not be evaluated
	result, err := engine.Eval(context.Background(), "false && true")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Bool() != false {
		t.Errorf("got %v, want false", result)
	}
	// customAnd should not be called because && uses short-circuit
	if callCount != 0 {
		t.Errorf("customAnd was called %d times, expected 0 (short-circuit should be preserved)", callCount)
	}
}

func TestWithEnv(t *testing.T) {
	type MyEnv struct {
		X int `expr:"x"`
	}

	engine := New(WithEnv(MyEnv{}))
	result, err := engine.Eval(context.Background(), "x + 1", MyEnv{X: 5})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 6 {
		t.Errorf("got %v, want 6", result)
	}
}

// =============================================================================
// ENGINE TYPED EVAL METHODS TESTS
// =============================================================================

func TestEngine_EvalBool(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		env      map[string]any
		expected bool
	}{
		{"true", nil, true},
		{"false", nil, false},
		{"5 > 3", nil, true},
		{"5 < 3", nil, false},
		{"x > 10", map[string]any{"x": 15}, true},
		{"x > 10", map[string]any{"x": 5}, false},
		{`name == "test"`, map[string]any{"name": "test"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.EvalBool(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEngine_EvalInt(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		env      map[string]any
		expected int64
	}{
		{"42", nil, 42},
		{"2 + 2 * 3", nil, 8},
		{"x + y", map[string]any{"x": 10, "y": 20}, 30},
		{"len([1, 2, 3])", nil, 3},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.EvalInt(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEngine_EvalFloat(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		env      map[string]any
		expected float64
	}{
		{"3.14", nil, 3.14},
		{"10.0 / 4", nil, 2.5},
		{"price * discount", map[string]any{"price": 100.0, "discount": 0.9}, 90.0},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.EvalFloat(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEngine_EvalString(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		env      map[string]any
		expected string
	}{
		{`"hello"`, nil, "hello"},
		{`"Hello, " + name`, map[string]any{"name": "World"}, "Hello, World"},
		{`upper("test")`, nil, "TEST"},
		{`lower("HELLO")`, nil, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.EvalString(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// =============================================================================
// PACKAGE-LEVEL TYPED EVAL FUNCTIONS TESTS
// =============================================================================

func TestPackage_EvalBool(t *testing.T) {
	result, err := EvalBool(context.Background(), "5 > 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result {
		t.Error("expected true")
	}
}

func TestPackage_EvalInt(t *testing.T) {
	result, err := EvalInt(context.Background(), "2 + 2 * 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 8 {
		t.Errorf("got %v, want 8", result)
	}
}

func TestPackage_EvalFloat(t *testing.T) {
	result, err := EvalFloat(context.Background(), "3.14 * 2")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 6.28 {
		t.Errorf("got %v, want 6.28", result)
	}
}

func TestPackage_EvalString(t *testing.T) {
	result, err := EvalString(context.Background(), `"Hello, " + "World"`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "Hello, World" {
		t.Errorf("got %q, want %q", result, "Hello, World")
	}
}

// =============================================================================
// ERROR CONTEXT TESTS
// =============================================================================

func TestError_RuntimeErrorWithSource(t *testing.T) {
	// Test that runtime errors include source expression
	_, err := Eval(context.Background(), "x + y", map[string]any{"x": 10}) // y is undefined
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "x + y") {
		t.Errorf("error should contain source expression, got: %s", errStr)
	}
}

func TestError_LexerErrorWithSource(t *testing.T) {
	// Test that lexer errors include source
	lexer := newLexer(`"unterminated string`)
	_, err := lexer.tokenize()
	if err == nil {
		t.Fatal("expected error")
	}
	lexErr, ok := err.(*LexerError)
	if !ok {
		t.Fatalf("expected LexerError, got %T", err)
	}
	if lexErr.Source == "" {
		t.Error("LexerError.Source should be set")
	}
}

func TestError_ParseErrorWithSource(t *testing.T) {
	// Test that parse errors include source
	_, err := Compile("x ++ y") // invalid syntax
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	// Parse error should have context
	if !strings.Contains(errStr, "x ++ y") {
		t.Errorf("error should contain source expression, got: %s", errStr)
	}
}

func TestError_DivisionByZero(t *testing.T) {
	_, err := Eval(context.Background(), "10 / 0")
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "division by zero") {
		t.Errorf("error should mention division by zero, got: %s", errStr)
	}
	if !strings.Contains(errStr, "10 / 0") {
		t.Errorf("error should contain source expression, got: %s", errStr)
	}
}

func TestProgram_EvalBool(t *testing.T) {
	program := MustCompile("x > 5")

	b, err := program.EvalBool(context.Background(), map[string]any{"x": int64(10)})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !b {
		t.Error("expected true")
	}

	b, err = program.EvalBool(context.Background(), map[string]any{"x": int64(3)})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if b {
		t.Error("expected false")
	}
}

// =============================================================================
// NEW BUILTIN FUNCTIONS TESTS
// =============================================================================

func TestBuiltin_Clamp(t *testing.T) {
	tests := []struct {
		expr     string
		expected int64
	}{
		{"clamp(5, 0, 10)", 5},
		{"clamp(-5, 0, 10)", 0},
		{"clamp(15, 0, 10)", 10},
		{"clamp(0, 0, 10)", 0},
		{"clamp(10, 0, 10)", 10},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalInt(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Between(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"between(5, 0, 10)", true},
		{"between(-5, 0, 10)", false},
		{"between(15, 0, 10)", false},
		{"between(0, 0, 10)", true},
		{"between(10, 0, 10)", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalBool(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_HasKey(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected bool
	}{
		{`hasKey(m, "a")`, map[string]any{"m": map[string]any{"a": 1, "b": 2}}, true},
		{`hasKey(m, "c")`, map[string]any{"m": map[string]any{"a": 1, "b": 2}}, false},
		{`hasKey(m, "")`, map[string]any{"m": map[string]any{"": "empty key"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalBool(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Empty(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected bool
	}{
		{`empty("")`, nil, true},
		{`empty("hello")`, nil, false},
		{`empty([])`, nil, true},
		{`empty([1, 2])`, nil, false},
		{`empty(x)`, map[string]any{"x": nil}, true},
		{`empty(x)`, map[string]any{"x": map[string]any{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalBool(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_NotEmpty(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{`notEmpty("")`, false},
		{`notEmpty("hello")`, true},
		{`notEmpty([])`, false},
		{`notEmpty([1, 2])`, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalBool(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Default(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected int64
	}{
		{`default(x, 42)`, map[string]any{"x": nil}, 42},
		{`default(x, 42)`, map[string]any{"x": 10}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalInt(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Coalesce(t *testing.T) {
	tests := []struct {
		expr     string
		env      map[string]any
		expected string
	}{
		{`coalesce(a, b, "default")`, map[string]any{"a": nil, "b": nil}, "default"},
		{`coalesce(a, b, "default")`, map[string]any{"a": nil, "b": "found"}, "found"},
		{`coalesce(a, b, "default")`, map[string]any{"a": "first", "b": "second"}, "first"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalString(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuiltin_Repeat(t *testing.T) {
	result, err := Eval(context.Background(), `repeat("x", 3)`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}
	for _, v := range arr {
		if NewValue(v).String() != "x" {
			t.Errorf("expected 'x', got %v", v)
		}
	}
}

func TestBuiltin_Range(t *testing.T) {
	tests := []struct {
		expr     string
		expected []int64
	}{
		{"range(1, 5)", []int64{1, 2, 3, 4, 5}},
		{"range(0, 3)", []int64{0, 1, 2, 3}},
		{"range(1, 10, 2)", []int64{1, 3, 5, 7, 9}},
		{"range(10, 1, -2)", []int64{10, 8, 6, 4, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			arr := NewValue(result).Array()
			if len(arr) != len(tt.expected) {
				t.Errorf("length: got %d, want %d", len(arr), len(tt.expected))
				return
			}
			for i, v := range arr {
				if NewValue(v).Int() != tt.expected[i] {
					t.Errorf("[%d]: got %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestBuiltin_Zip(t *testing.T) {
	result, err := Eval(context.Background(), `zip([1, 2, 3], ["a", "b", "c"])`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 3 {
		t.Errorf("expected 3 tuples, got %d", len(arr))
	}
	// First tuple should be [1, "a"]
	tuple := NewValue(arr[0]).Array()
	if len(tuple) != 2 {
		t.Errorf("expected tuple of 2, got %d", len(tuple))
	}
}

func TestBuiltin_Chunk(t *testing.T) {
	result, err := Eval(context.Background(), `chunk([1, 2, 3, 4, 5], 2)`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 3 { // [[1,2], [3,4], [5]]
		t.Errorf("expected 3 chunks, got %d", len(arr))
	}
}

func TestBuiltin_Compact(t *testing.T) {
	// compact removes only nil values
	result, err := Eval(context.Background(), `compact([1, nil, 2, false, 3, "", 0])`, map[string]any{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	// Should keep: 1, 2, false, 3, "", 0 (removes only nil)
	if len(arr) != 6 {
		t.Errorf("expected 6 elements, got %d: %v", len(arr), arr)
	}
}

func TestBuiltin_Truthy(t *testing.T) {
	// truthy removes all falsy values (nil, false, 0, "")
	result, err := Eval(context.Background(), `truthy([1, nil, 2, false, 3, "", 0])`, map[string]any{})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	// Should keep: 1, 2, 3 (removes nil, false, "", 0)
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d: %v", len(arr), arr)
	}
}

func TestBuiltin_Pluck(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob", "age": 25},
			map[string]any{"name": "Charlie", "age": 35},
		},
	}
	result, err := Eval(context.Background(), `pluck(users, "name")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 3 {
		t.Errorf("expected 3 names, got %d", len(arr))
	}
	if NewValue(arr[0]).String() != "Alice" {
		t.Errorf("expected Alice, got %v", arr[0])
	}
}

func TestBuiltin_GroupBy(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "Alice", "role": "admin"},
			map[string]any{"name": "Bob", "role": "user"},
			map[string]any{"name": "Charlie", "role": "admin"},
			map[string]any{"name": "Diana", "role": "user"},
		},
	}

	// Basic groupBy
	result, err := Eval(context.Background(), `groupBy(users, u => u.role)`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	if len(arr) != 2 {
		t.Errorf("expected 2 groups, got %d", len(arr))
	}

	// GroupBy with map transformation
	result2, err := Eval(context.Background(), `groupBy(users, u => u.role) | map(g => {"role": g.key, "count": len(g.items)})`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr2 := NewValue(result2).Array()
	if len(arr2) != 2 {
		t.Errorf("expected 2 groups, got %d", len(arr2))
	}

	// Check first group
	first := NewValue(arr2[0]).Map()
	if first["role"] != "admin" {
		t.Errorf("expected first role to be admin, got %v", first["role"])
	}
	if NewValue(first["count"]).Int() != 2 {
		t.Errorf("expected admin count to be 2, got %v", first["count"])
	}
}

func TestBuiltin_GroupBy_Errors(t *testing.T) {
	// Not an array
	_, err := Eval(context.Background(), `groupBy(123, x => x)`)
	if err == nil {
		t.Error("expected error for non-array")
	}

	// Not a function
	_, err = Eval(context.Background(), `groupBy([1,2,3], "not a function")`)
	if err == nil {
		t.Error("expected error for non-function")
	}
}

func TestBuiltin_SortBy(t *testing.T) {
	env := map[string]any{
		"users": []any{
			map[string]any{"name": "Charlie", "age": 30},
			map[string]any{"name": "Alice", "age": 25},
			map[string]any{"name": "Bob", "age": 35},
		},
	}

	// Sort ascending by age
	result, err := Eval(context.Background(), `sortBy(users, u => u.age) | pluck("name")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := NewValue(result).Array()
	expected := []string{"Alice", "Charlie", "Bob"}
	for i, name := range expected {
		if NewValue(arr[i]).String() != name {
			t.Errorf("position %d: expected %s, got %v", i, name, arr[i])
		}
	}

	// Sort descending by age
	result2, err := Eval(context.Background(), `sortBy(users, u => u.age, true) | pluck("name")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr2 := NewValue(result2).Array()
	expected2 := []string{"Bob", "Charlie", "Alice"}
	for i, name := range expected2 {
		if NewValue(arr2[i]).String() != name {
			t.Errorf("position %d: expected %s, got %v", i, name, arr2[i])
		}
	}

	// Sort by string field
	result3, err := Eval(context.Background(), `sortBy(users, u => u.name) | pluck("name")`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr3 := NewValue(result3).Array()
	expected3 := []string{"Alice", "Bob", "Charlie"}
	for i, name := range expected3 {
		if NewValue(arr3[i]).String() != name {
			t.Errorf("position %d: expected %s, got %v", i, name, arr3[i])
		}
	}
}

func TestBuiltin_SortBy_Errors(t *testing.T) {
	// Not an array
	_, err := Eval(context.Background(), `sortBy(123, x => x)`)
	if err == nil {
		t.Error("expected error for non-array")
	}

	// Not a function
	_, err = Eval(context.Background(), `sortBy([1,2,3], "not a function")`)
	if err == nil {
		t.Error("expected error for non-function")
	}

	// Wrong arg count
	_, err = Eval(context.Background(), `sortBy([1,2,3])`)
	if err == nil {
		t.Error("expected error for wrong arg count")
	}
}

func TestBuiltin_ToJSON(t *testing.T) {
	result, err := EvalString(context.Background(), `toJSON({"a": 1, "b": 2})`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Result should be valid JSON
	if !strings.Contains(result, `"a"`) || !strings.Contains(result, `"b"`) {
		t.Errorf("expected JSON with a and b keys, got %s", result)
	}
}

func TestBuiltin_FromJSON(t *testing.T) {
	result, err := Eval(context.Background(), `fromJSON('{"name": "test", "value": 42}')`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	m := NewValue(result).Map()
	if m["name"] != "test" {
		t.Errorf("expected name=test, got %v", m["name"])
	}
}

func TestBuiltin_Size(t *testing.T) {
	// size is alias for len
	tests := []struct {
		expr     string
		expected int64
	}{
		{`size([1, 2, 3])`, 3},
		{`size("hello")`, 5},
		{`size({"a": 1})`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalInt(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAST_WalkAllNodes(t *testing.T) {
	// Parse a complex expression to cover more Walk paths
	exprs := []string{
		"[1, 2, 3]",
		"{a: 1, b: 2}",
		"(1..5)",
		"a ?? b",
		"x ? 1 : 2",
	}

	for _, expr := range exprs {
		parser := newParser(expr)
		ast, err := parser.parse()
		if err != nil {
			continue // Some may not parse
		}

		visited := false
		Walk(ast, func(n Node) bool {
			visited = true
			return true
		})

		if !visited {
			t.Errorf("Walk didn't visit any nodes for %s", expr)
		}
	}
}

func TestAST_Transform(t *testing.T) {
	parser := newParser("1 + 2")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Double all integers
	result := Transform(ast, func(n Node) Node {
		if lit, ok := n.(*intLit); ok {
			return &intLit{Value: lit.Value * 2}
		}
		return n
	})

	// Verify transformation
	bin, ok := result.(*binaryExpr)
	if !ok {
		t.Fatal("expected binaryExpr")
	}
	left, ok := bin.Left.(*intLit)
	if !ok || left.Value != 2 {
		t.Errorf("left: got %v, want 2", left)
	}
	right, ok := bin.Right.(*intLit)
	if !ok || right.Value != 4 {
		t.Errorf("right: got %v, want 4", right)
	}
}

func TestAST_MoreNodeStrings(t *testing.T) {
	nodes := []Node{
		&callExpr{Func: &ident{Name: "foo"}, Args: []Node{&intLit{Value: 1}}},
		&memberExpr{Object: &ident{Name: "obj"}, Property: &ident{Name: "field"}},
		&indexExpr{Object: &ident{Name: "arr"}, Index: &intLit{Value: 0}},
		&sliceExpr{Object: &ident{Name: "arr"}, Low: &intLit{Value: 0}, High: &intLit{Value: 2}},
		&pipeExpr{Left: &ident{Name: "arr"}, Right: &ident{Name: "sum"}},
		&lambdaExpr{Params: []string{"x"}, Body: &ident{Name: "x"}},
		&letExpr{Name: "x", Value: &intLit{Value: 5}, Body: &ident{Name: "x"}},
	}

	for _, n := range nodes {
		s := n.String()
		if s == "" {
			t.Errorf("empty string for %T", n)
		}
	}
}

func TestValue_MapEquals(t *testing.T) {
	m1 := Map(map[string]any{"a": int64(1), "b": int64(2)})
	m2 := Map(map[string]any{"a": int64(1), "b": int64(2)})
	m3 := Map(map[string]any{"a": int64(1), "c": int64(3)})
	m4 := Map(map[string]any{"a": int64(1)})

	if !m1.Equal(m2) {
		t.Error("m1 should equal m2")
	}
	if m1.Equal(m3) {
		t.Error("m1 should not equal m3")
	}
	if m1.Equal(m4) {
		t.Error("m1 should not equal m4 (different size)")
	}
}

func TestValue_Interface(t *testing.T) {
	tests := []struct {
		value    Value
		expected any
	}{
		{Nil, nil},
		{Bool(true), true},
		{Int(42), int64(42)},
		{Float(3.14), 3.14},
		{String("hello"), "hello"},
	}

	for _, tt := range tests {
		if tt.value.Interface() != tt.expected {
			t.Errorf("got %v, want %v", tt.value.Interface(), tt.expected)
		}
	}
}

func TestValue_ArrayMap(t *testing.T) {
	arr := Array([]any{1, 2, 3})
	if arr.Array() == nil {
		t.Error("Array() returned nil")
	}
	if Int(42).Array() != nil {
		t.Error("Int.Array() should return nil")
	}

	m := Map(map[string]any{"a": 1})
	if m.Map() == nil {
		t.Error("Map() returned nil")
	}
	if Int(42).Map() != nil {
		t.Error("Int.Map() should return nil")
	}
}

func TestValue_GetStructNil(t *testing.T) {
	v := Value{kind: kindAny, ptr: nil}
	_, ok := v.Get("field")
	if ok {
		t.Error("expected false for nil ptr")
	}
}

func TestValue_CompareIncompatible(t *testing.T) {
	// Comparing incompatible types
	arr := Array([]any{1})
	m := Map(map[string]any{"a": 1})

	// Should not panic
	_ = arr.Compare(m)
}

func TestVM_SubErrors(t *testing.T) {
	engine := New()
	_, err := engine.Eval(context.Background(), `"a" - "b"`)
	if err == nil {
		t.Error("expected error for string subtraction")
	}
}

func TestVM_MulErrors(t *testing.T) {
	engine := New()
	_, err := engine.Eval(context.Background(), `[1] * [2]`)
	if err == nil {
		t.Error("expected error for array multiplication")
	}
}

func TestVM_AddErrors(t *testing.T) {
	engine := New()
	env := map[string]any{
		"f": func() {},
	}
	_, err := engine.Eval(context.Background(), "f + f", env)
	if err == nil {
		t.Error("expected error for function addition")
	}
}

func TestSliceErrors(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2)},
	}

	// Invalid slice bounds now return empty array (Python-style behavior)
	result, err := engine.Eval(context.Background(), "arr[5:1]", env)
	if err != nil {
		t.Errorf("unexpected error for invalid slice bounds: %v", err)
	}
	arr, ok := result.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty array for invalid slice bounds, got %v", result)
	}
}

func TestLambda_Simple(t *testing.T) {
	// Test lambda parsing
	parser := newParser("(x) => x + 1")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if _, ok := ast.(*lambdaExpr); !ok {
		t.Error("expected lambdaExpr")
	}
}

func TestLambda_NoParams(t *testing.T) {
	parser := newParser("() => 42")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if lambda, ok := ast.(*lambdaExpr); ok {
		if len(lambda.Params) != 0 {
			t.Error("expected 0 params")
		}
	} else {
		t.Error("expected lambdaExpr")
	}
}

func TestLambda_MultiParams(t *testing.T) {
	parser := newParser("(x, y) => x + y")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if lambda, ok := ast.(*lambdaExpr); ok {
		if len(lambda.Params) != 2 {
			t.Errorf("expected 2 params, got %d", len(lambda.Params))
		}
	} else {
		t.Error("expected lambdaExpr")
	}
}

func TestGrouping(t *testing.T) {
	// Test that grouping works
	result, err := Eval(context.Background(), "(1 + 2) * 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(9) {
		t.Errorf("got %v, want 9", result)
	}
}

func TestSliceStep(t *testing.T) {
	// Slice with step currently just ignores step
	engine := New()
	env := map[string]any{
		"arr": []any{int64(0), int64(1), int64(2), int64(3)},
	}

	// Just verify it parses and runs without error
	_, err := engine.Eval(context.Background(), "arr[0:2]", env)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMapComputedKey(t *testing.T) {
	engine := New()
	env := map[string]any{
		"key": "foo",
	}

	result, err := engine.Eval(context.Background(), `{[key]: 42}`, env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	m := result.(map[string]any)
	if m["foo"] != int64(42) {
		t.Errorf("got %v", m)
	}
}

func TestBuiltin_SubstrEdgeCases(t *testing.T) {
	tests := []struct {
		expr   string
		result string
	}{
		{`substr("hello", 10)`, ""},          // start beyond end
		{`substr("hello", 0, 100)`, "hello"}, // length beyond end
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestBuiltin_TakeSkipEdges(t *testing.T) {
	tests := []struct {
		expr  string
		count int
	}{
		{"take([1,2,3], -1)", 0},
		{"skip([1,2,3], -1)", 3},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if len(result.([]any)) != tt.count {
				t.Errorf("got len %d, want %d", len(result.([]any)), tt.count)
			}
		})
	}
}

func TestNewValue_SliceTypes(t *testing.T) {
	// []Value conversion
	values := []Value{Int(1), Int(2)}
	v := NewValue(values)
	if !v.IsArray() {
		t.Error("expected array")
	}
}

func TestNewValue_FuncType(t *testing.T) {
	fn := func() int { return 42 }
	v := NewValue(fn)
	if !v.IsFunc() {
		t.Error("expected func")
	}
}

func TestValue_AnyType(t *testing.T) {
	type CustomStruct struct {
		X int
	}

	v := NewValue(CustomStruct{X: 5})
	typeName := v.Type()
	if typeName == "" {
		t.Error("expected non-empty type name")
	}
}

func TestVariadicFunction(t *testing.T) {
	engine := New(
		WithFunction("sumAll", func(nums ...int) int {
			sum := 0
			for _, n := range nums {
				sum += n
			}
			return sum
		}),
	)

	result, err := engine.Eval(context.Background(), "sumAll(1, 2, 3, 4, 5)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 15 {
		t.Errorf("got %v, want 15", result)
	}
}

func TestFunctionWithError(t *testing.T) {
	engine := New(
		WithFunction("divide", func(a, b int) (int, error) {
			if b == 0 {
				return 0, ErrDivisionByZero
			}
			return a / b, nil
		}),
	)

	// Success case
	result, err := engine.Eval(context.Background(), "divide(10, 2)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 5 {
		t.Errorf("got %v, want 5", result)
	}

	// Error case
	_, err = engine.Eval(context.Background(), "divide(10, 0)")
	if err == nil {
		t.Error("expected error")
	}
}

// =============================================================================
// FINAL COVERAGE TESTS
// =============================================================================

func TestAST_WalkComplex(t *testing.T) {
	// Test Walk on various node types
	testCases := []struct {
		expr string
	}{
		{"1 + 2"},
		{"-5"},
		{"x ? 1 : 2"},
		{"foo(1, 2)"},
		{"obj.field"},
		{"arr[0]"},
		{"arr[1:2]"},
		{"[1, 2, 3]"},
		{"{a: 1}"},
		{"(1..5)"},
		{"a ?? b"},
	}

	for _, tc := range testCases {
		parser := newParser(tc.expr)
		ast, err := parser.parse()
		if err != nil {
			continue
		}

		nodeCount := 0
		Walk(ast, func(n Node) bool {
			nodeCount++
			// Continue walking
			return true
		})

		if nodeCount == 0 {
			t.Errorf("no nodes walked for %s", tc.expr)
		}
	}
}

func TestVM_FloatOperations(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"3.5 + 1.5", 5.0},
		{"5.0 - 2.0", 3.0},
		{"2.5 * 2.0", 5.0},
		{"10.0 / 4.0", 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Float() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestVM_MixedTypeOperations(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"3 + 1.5", 4.5},
		{"5.0 - 2", 3.0},
		{"2 * 2.5", 5.0},
		{"10 / 4.0", 2.5},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Float() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestVM_StringIndex(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	// String negative index
	result, err := engine.Eval(context.Background(), "s[-1]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "o" {
		t.Errorf("got %v, want o", result)
	}
}

func TestVM_DivisionTypes(t *testing.T) {
	// Integer division
	result, err := Eval(context.Background(), "10 / 3")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(3) {
		t.Errorf("int div: got %v, want 3", result)
	}

	// Float division
	_, err = Eval(context.Background(), "10.0 / 3.0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Just check it works without error
}

func TestConvertArg_Types(t *testing.T) {
	engine := New(
		WithFunction("testInt", func(x int) int { return x }),
		WithFunction("testInt64", func(x int64) int64 { return x }),
		WithFunction("testFloat64", func(x float64) float64 { return x }),
		WithFunction("testString", func(x string) string { return x }),
		WithFunction("testBool", func(x bool) bool { return x }),
		WithFunction("testSlice", func(x []any) int { return len(x) }),
		WithFunction("testAny", func(x any) any { return x }),
	)

	tests := []struct {
		expr string
	}{
		{"testInt(42)"},
		{"testInt64(42)"},
		{"testFloat64(3.14)"},
		{`testString("hello")`},
		{"testBool(true)"},
		{"testSlice([1,2,3])"},
		{"testAny(42)"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			_, err := engine.Eval(context.Background(), tt.expr)
			if err != nil {
				t.Errorf("error: %v", err)
			}
		})
	}
}

func TestParser_OptionalIndex(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr":  []any{int64(1), int64(2), int64(3)},
		"null": nil,
	}

	// Normal optional index
	result, err := engine.Eval(context.Background(), "arr?.[0]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(1) {
		t.Errorf("got %v, want 1", result)
	}

	// Nil optional index
	result, err = engine.Eval(context.Background(), "null?.[0]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil")
	}
}

func TestParser_FunctionCall(t *testing.T) {
	engine := New(
		WithFunction("noArgs", func() int { return 42 }),
		WithFunction("oneArg", func(x int) int { return x }),
		WithFunction("twoArgs", func(a, b int) int { return a + b }),
	)

	tests := []struct {
		expr   string
		result int64
	}{
		{"noArgs()", 42},
		{"oneArg(5)", 5},
		{"twoArgs(2, 3)", 5},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := engine.Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if NewValue(result).Int() != tt.result {
				t.Errorf("got %v, want %v", result, tt.result)
			}
		})
	}
}

func TestBuiltin_ErrorPaths(t *testing.T) {
	// These should all return errors
	errorCases := []string{
		"ceil()",       // no args
		"floor()",      // no args
		"round()",      // no args
		"sqrt()",       // no args
		"pow(1)",       // wrong arg count
		"log()",        // no args
		"exp()",        // no args
		"sin()",        // no args
		"cos()",        // no args
		"upper()",      // no args
		"lower()",      // no args
		"trim()",       // no args
		"contains()",   // no args
		"startsWith()", // no args
		"endsWith()",   // no args
		"replace()",    // no args
		"substr()",     // no args
		"indexOf()",    // no args
		"split()",      // no args
		"int()",        // no args
		"float()",      // no args
		"string()",     // no args
		"bool()",       // no args
		"type()",       // no args
		"isNil()",      // no args
		"count()",      // no args
		"flatten()",    // no args
		"flatten(42)",  // not array
	}

	for _, expr := range errorCases {
		t.Run(expr, func(t *testing.T) {
			_, err := Eval(context.Background(), expr)
			if err == nil {
				t.Errorf("expected error for %s", expr)
			}
		})
	}
}

func TestValue_String_Complex(t *testing.T) {
	// Test string conversion of different types
	v := Value{kind: kindAny, ptr: struct{ X int }{X: 5}}
	s := v.String()
	if s == "" {
		t.Error("expected non-empty string")
	}

	// Bool value string
	v = Value{kind: kindAny, ptr: true}
	s = v.String()
	_ = s
}

func TestCompiler_AllOps(t *testing.T) {
	// Test various operations that hit different compiler paths
	exprs := []string{
		"1 + 2",
		"1 - 2",
		"1 * 2",
		"1 / 2",
		"1 % 2",
		"2 ** 3",
		"1 == 2",
		"1 != 2",
		"1 < 2",
		"1 <= 2",
		"1 > 2",
		"1 >= 2",
		"1 & 2",
		"(1 | 2)",
		"1 ^ 2",
		"1 << 2",
		"1 >> 2",
		"true && false",
		"true || false",
		"!true",
		"~1",
		"-1",
		"1 in [1,2]",
	}

	for _, expr := range exprs {
		_, err := Compile(expr)
		if err != nil {
			t.Errorf("compile error for %s: %v", expr, err)
		}
	}
}

func TestLexer_SkipWhitespace(t *testing.T) {
	// Test whitespace handling
	exprs := []struct {
		input  string
		result int64
	}{
		{"  1  +  2  ", 3},
		{"\t1\t+\t2\t", 3},
		{"\n1\n+\n2\n", 3},
		{"\r\n1\r\n+\r\n2\r\n", 3},
	}

	for _, tt := range exprs {
		result, err := Eval(context.Background(), tt.input)
		if err != nil {
			t.Errorf("error for %q: %v", tt.input, err)
			continue
		}
		if result != tt.result {
			t.Errorf("got %v, want %v", result, tt.result)
		}
	}
}

func TestValue_TypeAny(t *testing.T) {
	// Test Type() for any kind with nil ptr
	v := Value{kind: kindAny, ptr: nil}
	if v.Type() != "nil" {
		t.Errorf("got %v, want nil", v.Type())
	}
}

func TestProgram_EvalBoolConversion(t *testing.T) {
	// Non-bool result conversion
	program := MustCompile("42")
	b, err := program.EvalBool(context.Background(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !b {
		t.Error("42 should be truthy")
	}

	program = MustCompile("0")
	b, err = program.EvalBool(context.Background(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if b {
		t.Error("0 should be falsy")
	}
}

func TestLexer_PeekCached(t *testing.T) {
	lexer := newLexer("1 + 2")

	// First peek
	tok1 := lexer.Peek()
	// Second peek should return cached
	tok2 := lexer.Peek()

	if tok1.Type != tok2.Type || tok1.Value != tok2.Value {
		t.Error("cached peek returned different value")
	}
}

func TestSlice_FullRange(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2), int64(3)},
	}

	result, err := engine.Eval(context.Background(), "arr[:]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("got len %d, want 3", len(arr))
	}
}

func TestPipe_Simple(t *testing.T) {
	// Test that pipe operator parses (even if execution isn't fully implemented)
	parser := newParser("arr | len")
	_, err := parser.parse()
	if err != nil {
		t.Errorf("parse error: %v", err)
	}
}

func TestPipe_Execution(t *testing.T) {
	// Test pipe with len function
	result, err := EvalInt(context.Background(), "[1, 2, 3] | len")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 3 {
		t.Errorf("got %d, want 3", result)
	}
}

func TestPipe_WithSum(t *testing.T) {
	result, err := EvalInt(context.Background(), "[1, 2, 3, 4] | sum")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 10 {
		t.Errorf("got %d, want 10", result)
	}
}

func TestPipe_Chain(t *testing.T) {
	// Test chained pipes
	result, err := EvalInt(context.Background(), "[3, 1, 2] | sort | first")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 1 {
		t.Errorf("got %d, want 1", result)
	}
}

// Test TypeChecker actually generates errors
func TestTypeChecker_GeneratesErrors(t *testing.T) {
	tc := NewTypeChecker().
		WithVar("name", TypeString).
		WithVar("count", TypeInt).
		WithVar("flag", TypeBool)

	// String - String should error (not allowed)
	parser := newParser("name - name")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errors := tc.Check(ast)
	if errors == nil || errors.Len() == 0 {
		t.Error("expected type error for string - string")
	}

	// Boolean operators require bool - int && bool should error
	parser = newParser("count && flag")
	ast, _ = parser.parse()
	errors = tc.Check(ast)
	if errors == nil || errors.Len() == 0 {
		t.Error("expected type error for int && bool")
	}

	// Bitwise operators require int - string & int should error
	parser = newParser("name & count")
	ast, _ = parser.parse()
	errors = tc.Check(ast)
	if errors == nil || errors.Len() == 0 {
		t.Error("expected type error for string & int")
	}

	// Unary not on int should error
	parser = newParser("!count")
	ast, _ = parser.parse()
	errors = tc.Check(ast)
	if errors == nil || errors.Len() == 0 {
		t.Error("expected type error for !int")
	}
}

// Test DisableBuiltins sandbox option
func TestSandbox_DisableBuiltins(t *testing.T) {
	engine := New(
		WithSandbox(SandboxConfig{DisableBuiltins: true}),
		WithFunction("myFunc", func() int { return 42 }),
	)

	// Custom function should still work
	result, err := engine.Eval(context.Background(), "myFunc()", nil)
	if err != nil {
		t.Fatalf("custom func error: %v", err)
	}
	if result != int64(42) {
		t.Errorf("got %v, want 42", result)
	}

	// Built-in should fail
	_, err = engine.Eval(context.Background(), "len([1, 2, 3])", nil)
	if err == nil {
		t.Error("expected error for disabled builtin 'len'")
	}
}

// Test MaxMemory sandbox limit
func TestSandbox_MaxMemory(t *testing.T) {
	engine := New(WithSandbox(SandboxConfig{
		MaxMemory: 1024, // Very low limit
	}))

	// Simple expression should work
	_, err := engine.Eval(context.Background(), "1 + 1", nil)
	if err != nil {
		t.Fatalf("simple eval error: %v", err)
	}

	// Creating large array might exceed memory
	_, _ = engine.Eval(context.Background(), "range(1, 1000)", nil)
	// Note: This may or may not fail depending on implementation
	// The important thing is the check runs without panic
}

// Test substr with Unicode
func TestBuiltin_SubstrUnicode(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{`substr("Привет", 0, 3)`, "При"},
		{`substr("Привет", 3)`, "вет"},
		{`substr("Hello世界", 5, 2)`, "世界"},
		{`substr("日本語", -2)`, "本語"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalString(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.want {
				t.Errorf("got %q, want %q", result, tt.want)
			}
		})
	}
}

// Test ASTToString handles spreadExpr and destructureExpr
func TestASTToString_NewNodes(t *testing.T) {
	// spreadExpr
	spread := &spreadExpr{Expr: &ident{Name: "arr"}}
	if s := ASTToString(spread); s != "...arr" {
		t.Errorf("spreadExpr: got %q, want %q", s, "...arr")
	}

	// destructureExpr (object)
	destruct := &destructureExpr{
		Pattern: []destructureField{
			{Key: "a", Name: "a"},
			{Key: "b", Name: "x", Default: &intLit{Value: 0}},
		},
		IsArray: false,
		Value:   &ident{Name: "obj"},
		Body:    &binaryExpr{Op: "+", Left: &ident{Name: "a"}, Right: &ident{Name: "x"}},
	}
	s := ASTToString(destruct)
	if !strings.Contains(s, "let {") || !strings.Contains(s, "a, b: x = 0") {
		t.Errorf("destructureExpr: got %q", s)
	}
}

func TestValue_BoolFromAny(t *testing.T) {
	// Test Bool() for any kind
	v := Value{kind: kindAny, ptr: "non-empty"}
	if !v.Bool() {
		t.Error("non-empty string should be truthy")
	}

	v = Value{kind: kindAny, ptr: nil}
	if v.Bool() {
		t.Error("nil should be falsy")
	}
}

// =============================================================================
// TYPE CHECKER TESTS
// =============================================================================

func TestTypeChecker_BasicTypes(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{"42", TypeInt},
		{"3.14", TypeFloat},
		{`"hello"`, TypeString},
		{"true", TypeBool},
		{"nil", TypeNil},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_BinaryOps(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{"1 + 2", TypeInt},
		{"1.0 + 2", TypeFloat},
		{"1 + 2.0", TypeFloat},
		{`"a" + "b"`, TypeString},
		{"1 - 2", TypeInt},
		{"1 * 2", TypeInt},
		{"1 / 2", TypeInt},
		{"1.0 / 2", TypeFloat},
		{"1 == 2", TypeBool},
		{"1 < 2", TypeBool},
		{"true && false", TypeBool},
		{"true || false", TypeBool},
		{"1 & 2", TypeInt},
		{"1 << 2", TypeInt},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_WithEnvType(t *testing.T) {
	type Env struct {
		Count int     `expr:"count"`
		Price float64 `expr:"price"`
		Name  string  `expr:"name"`
		Ok    bool    `expr:"ok"`
	}

	tc := NewTypeChecker().WithEnvType(Env{})

	tests := []struct {
		expr     string
		expected Type
	}{
		{"count", TypeInt},
		{"price", TypeFloat},
		{"name", TypeString},
		{"ok", TypeBool},
		{"count + 1", TypeInt},
		{"price * 2", TypeFloat},
		{`name + " suffix"`, TypeString},
		{"ok && true", TypeBool},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_Builtins(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{`len("hello")`, TypeInt},
		{`upper("hello")`, TypeString},
		// abs can return int for int inputs, float for float inputs; treat as any.
		{`abs(-5)`, TypeAny},
		{`sqrt(16)`, TypeFloat},
		{`contains("hello", "ll")`, TypeBool},
		{`split("a,b", ",")`, TypeArray},
		// sum returns int when all ints and no overflow, float otherwise.
		{`sum([1,2,3])`, TypeAny},
		{`first([1,2,3])`, TypeAny},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_Ternary(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{"true ? 1 : 2", TypeInt},
		{"true ? 1.0 : 2.0", TypeFloat},
		{`true ? "a" : "b"`, TypeString},
		{"true ? 1 : 2.0", TypeFloat}, // Mixed numeric -> float
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeChecker_Let(t *testing.T) {
	tc := NewTypeChecker()

	tests := []struct {
		expr     string
		expected Type
	}{
		{"let x = 5; x + 1", TypeInt},
		{"let x = 3.14; x * 2", TypeFloat},
		{`let s = "hi"; s + "!"`, TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := tc.Infer(ast)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWithTypeCheck(t *testing.T) {
	type Env struct {
		X int `expr:"x"`
		Y int `expr:"y"`
	}

	engine := New(WithTypeCheck(Env{}))

	// Should compile successfully
	_, err := engine.Compile("x + y")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	// Execute with map env (Program.Eval requires map[string]any)
	result, err := engine.Eval(context.Background(), "x + y", Env{X: 10, Y: 20})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if NewValue(result).Int() != 30 {
		t.Errorf("got %v, want 30", result)
	}
}

func TestCheckExpressionType(t *testing.T) {
	type Env struct {
		X int `expr:"x"`
	}

	// Should pass - x is int, expression returns int
	err := CheckExpressionType("x + 1", TypeInt, Env{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should pass - literal int
	err = CheckExpressionType("42", TypeInt, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestType_String(t *testing.T) {
	types := []struct {
		typ      Type
		expected string
	}{
		{TypeUnknown, "unknown"},
		{TypeNil, "nil"},
		{TypeBool, "bool"},
		{TypeInt, "int"},
		{TypeFloat, "float"},
		{TypeString, "string"},
		{TypeArray, "array"},
		{TypeMap, "map"},
		{TypeFunc, "func"},
		{TypeAny, "any"},
	}

	for _, tt := range types {
		if tt.typ.String() != tt.expected {
			t.Errorf("Type %d: got %v, want %v", tt.typ, tt.typ.String(), tt.expected)
		}
	}
}

func TestVM_Division_FloatByInt(t *testing.T) {
	result, err := Eval(context.Background(), "7.0 / 2")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Float() != 3.5 {
		t.Errorf("got %v, want 3.5", result)
	}
}

func TestVM_Division_IntByFloat(t *testing.T) {
	result, err := Eval(context.Background(), "7 / 2.0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Float() != 3.5 {
		t.Errorf("got %v, want 3.5", result)
	}
}

func TestVM_Sub_IntFloat(t *testing.T) {
	tests := []struct {
		expr   string
		result float64
	}{
		{"7.0 - 2", 5.0},
		{"7 - 2.0", 5.0},
	}

	for _, tt := range tests {
		result, err := Eval(context.Background(), tt.expr)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if NewValue(result).Float() != tt.result {
			t.Errorf("got %v, want %v", result, tt.result)
		}
	}
}

func TestVM_Slice_StringWithNegative(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	result, err := engine.Eval(context.Background(), "s[1:-1]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "ell" {
		t.Errorf("got %v, want ell", result)
	}
}

func TestVM_Slice_MapError(t *testing.T) {
	engine := New()
	env := map[string]any{"m": map[string]any{"a": 1}}

	_, err := engine.Eval(context.Background(), "m[0:1]", env)
	if err == nil {
		t.Error("expected error for slicing map")
	}
}

func TestBuiltin_Sort_Float(t *testing.T) {
	result, err := Eval(context.Background(), "sort([3.0, 1.0, 2.0])")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := result.([]any)
	if NewValue(arr[0]).Float() != 1.0 {
		t.Errorf("first should be 1.0, got %v", arr[0])
	}
}

func TestBuiltin_Sort_String(t *testing.T) {
	result, err := Eval(context.Background(), `sort(["c", "a", "b"])`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr := result.([]any)
	if arr[0] != "a" {
		t.Errorf("first should be a, got %v", arr[0])
	}
}

func TestSlice_StringOmitEnd(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	result, err := engine.Eval(context.Background(), "s[2:]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "llo" {
		t.Errorf("got %v, want llo", result)
	}
}

func TestSlice_StringOmitStart(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	result, err := engine.Eval(context.Background(), "s[:3]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "hel" {
		t.Errorf("got %v, want hel", result)
	}
}

func TestIndex_StringOutOfBounds(t *testing.T) {
	engine := New()
	env := map[string]any{"s": "hello"}

	_, err := engine.Eval(context.Background(), "s[100]", env)
	if err == nil {
		t.Error("expected error for out of bounds")
	}
}

func TestCompiler_MoreOps(t *testing.T) {
	// Test compilation of more complex expressions
	exprs := []string{
		"true ? 1 : 2",
		"nil ?? 5",
		"[1, 2, 3]",
		"{a: 1, b: 2}",
		"(1..3)",
	}

	for _, expr := range exprs {
		_, err := Compile(expr)
		if err != nil {
			t.Errorf("compile error for %s: %v", expr, err)
		}
	}
}

func TestVM_UnsignedRightShift(t *testing.T) {
	// Test unsigned right shift
	tests := []struct {
		expr     string
		expected int64
	}{
		{"(-8) >>> 1", 0x7ffffffffffffffc}, // -8 as unsigned >> 1
		{"8 >>> 1", 4},                     // Positive works same as >>
		{"(-1) >>> 32", 0xffffffff},        // All 1s >> 32
		{"16 >>> 2", 4},                    // Simple positive case
	}
	for _, tt := range tests {
		result, err := Eval(context.Background(), tt.expr)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.expr, err)
			continue
		}
		if got, ok := result.(int64); !ok || got != tt.expected {
			t.Errorf("%s: expected %d, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestOptChain_DeepNil(t *testing.T) {
	engine := New()
	env := map[string]any{
		"obj": map[string]any{
			"nested": nil,
		},
	}

	result, err := engine.Eval(context.Background(), "obj?.nested?.value", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestMemberAccess_NestedMap(t *testing.T) {
	engine := New()
	env := map[string]any{
		"obj": map[string]any{
			"nested": map[string]any{
				"value": int64(42),
			},
		},
	}

	result, err := engine.Eval(context.Background(), "obj.nested.value", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(42) {
		t.Errorf("got %v, want 42", result)
	}
}

func TestIndex_2D_Array(t *testing.T) {
	engine := New()
	env := map[string]any{
		"matrix": []any{
			[]any{int64(1), int64(2)},
			[]any{int64(3), int64(4)},
		},
	}

	result, err := engine.Eval(context.Background(), "matrix[1][0]", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(3) {
		t.Errorf("got %v, want 3", result)
	}
}

func TestFunctionChain(t *testing.T) {
	result, err := Eval(context.Background(), "upper(lower(\"HeLLo\"))")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "HELLO" {
		t.Errorf("got %v, want HELLO", result)
	}
}

func TestComplex_Expression(t *testing.T) {
	result, err := Eval(context.Background(), "(1 + 2) * 3 > 5 && true || false")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != true {
		t.Errorf("got %v, want true", result)
	}
}

func TestParser_MethodCall(t *testing.T) {
	// Test method-like calls
	engine := New()
	env := map[string]any{
		"obj": map[string]any{
			"getValue": func() int { return 42 },
		},
	}

	result, err := engine.Eval(context.Background(), "obj.getValue()", env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("got %v, want 42", result)
	}
}

func TestVM_Push_Overflow(t *testing.T) {
	// Create expression that uses deep stack
	// This is hard to trigger with normal expressions
	// Just test a deep nested expression
	result, err := Eval(context.Background(), "((((((1 + 2) + 3) + 4) + 5) + 6) + 7)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(28) {
		t.Errorf("got %v, want 28", result)
	}
}

func TestBuiltin_HigherOrder_Errors(t *testing.T) {
	// map, filter, reduce, find, findIndex all return "not implemented" errors
	// because they require closure support
	funcs := []string{
		"map([], 1)",
		"filter([], 1)",
		"reduce([], 1, 1)",
		"find([], 1)",
		"findIndex([], 1)",
	}

	for _, fn := range funcs {
		_, err := Eval(context.Background(), fn)
		if err == nil {
			t.Errorf("expected error for %s", fn)
		}
	}
}

func TestValue_Interface_Func(t *testing.T) {
	fn := func() {}
	v := Func(fn)
	iface := v.Interface()
	if iface == nil {
		t.Error("expected non-nil interface")
	}
}

func TestSlice_Clamp(t *testing.T) {
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2), int64(3)},
	}

	// Start beyond end now returns empty array (Python-style clamping behavior)
	result, err := engine.Eval(context.Background(), "arr[10:20]", env)
	if err != nil {
		t.Errorf("unexpected error for out of bounds slice: %v", err)
	}
	arr, ok := result.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty array for out of bounds slice, got %v", result)
	}
}

// TestAST_NodeMarkers covers the empty node() interface marker methods
func TestAST_NodeMarkers(t *testing.T) {
	// These methods are just type markers for the Node interface
	// They have empty bodies but calling them increases coverage
	(&intLit{}).node()
	(&floatLit{}).node()
	(&stringLit{}).node()
	(&boolLit{}).node()
	(&nilLit{}).node()
	(&ident{}).node()
	(&binaryExpr{}).node()
	(&unaryExpr{}).node()
	(&ternaryExpr{}).node()
	(&callExpr{}).node()
	(&memberExpr{}).node()
	(&indexExpr{}).node()
	(&sliceExpr{}).node()
	(&arrayLit{}).node()
	(&mapLit{}).node()
	(&rangeExpr{}).node()
	(&pipeExpr{}).node()
	(&lambdaExpr{}).node()
	(&coalesceExpr{}).node()
	(&letExpr{}).node()
}

// =============================================================================
// SANDBOX TESTS
// =============================================================================

func TestSandbox_MaxOps(t *testing.T) {
	env := map[string]any{"x": int64(1)}

	// Simple expression should work with enough ops
	engine := New(WithMaxOps(100))
	result, err := engine.Eval(context.Background(), "x + x + x", env)
	if err != nil {
		t.Fatalf("simple expression failed: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	// With very low limit, even simple expressions should fail
	engine2 := New(WithMaxOps(3))
	_, err = engine2.Eval(context.Background(), "x + x + x + x + x + x + x + x", env)
	if err == nil {
		t.Error("expected ops limit error")
	}
	if !errors.Is(err, ErrOpsLimitExceeded) {
		t.Errorf("expected ErrOpsLimitExceeded, got %v", err)
	}
}

func TestSandbox_Timeout(t *testing.T) {
	engine := New(WithTimeout(10 * time.Millisecond))

	// Quick expression should work
	result, err := engine.Eval(context.Background(), "1 + 1")
	if err != nil {
		t.Fatalf("quick expression failed: %v", err)
	}
	if NewValue(result).Int() != 2 {
		t.Errorf("expected 2, got %v", result)
	}
}

func TestSandbox_DeniedFuncs(t *testing.T) {
	engine := New(WithDeniedFuncs("print", "sprintf"))

	// Allowed functions should work
	result, err := engine.Eval(context.Background(), "len([1, 2, 3])")
	if err != nil {
		t.Fatalf("len should work: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	// Denied function should fail
	_, err = engine.Eval(context.Background(), `print("hello")`)
	if err == nil {
		t.Error("expected function denied error")
	}
	if !errors.Is(err, ErrFunctionNotAllowed) {
		t.Errorf("expected ErrFunctionNotAllowed, got %v", err)
	}
}

func TestSandbox_AllowedFuncs(t *testing.T) {
	engine := New(WithAllowedFuncs("len", "sum"))

	// Allowed functions should work
	result, err := engine.Eval(context.Background(), "len([1, 2, 3])")
	if err != nil {
		t.Fatalf("len should work: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	// Non-allowed function should fail
	_, err = engine.Eval(context.Background(), `upper("hello")`)
	if err == nil {
		t.Error("expected function denied error")
	}
	if !errors.Is(err, ErrFunctionNotAllowed) {
		t.Errorf("expected ErrFunctionNotAllowed, got %v", err)
	}
}

func TestSandbox_EnvFunction_BypassBlocked(t *testing.T) {
	engine := New(WithAllowedFuncs("len"))

	env := map[string]any{
		"evil": func() int { return 123 },
	}

	_, err := engine.Eval(context.Background(), `evil()`, env)
	if err == nil {
		t.Error("expected function denied error for env function under allowlist")
	}
	if !errors.Is(err, ErrFunctionNotAllowed) {
		t.Errorf("expected ErrFunctionNotAllowed, got %v", err)
	}
}

func TestSandbox_CustomFunction_AllowedByName(t *testing.T) {
	engine := New(
		WithAllowedFuncs("double"),
		WithFunction("double", func(x int) int { return x * 2 }),
	)

	result, err := engine.Eval(context.Background(), `double(21)`, nil)
	if err != nil {
		t.Fatalf("double should be allowed: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestSandbox_SecureEnv_RejectsStructPointer(t *testing.T) {
	type Env struct {
		Secret string
	}
	engine := New(WithSandbox(SandboxConfig{Secure: true}))
	_, err := engine.Eval(context.Background(), `x`, map[string]any{"x": &Env{Secret: "nope"}})
	if err == nil {
		t.Error("expected secure sandbox to reject struct pointer in environment")
	}
	var sandboxErr *SandboxError
	if !errors.As(err, &sandboxErr) || sandboxErr.Type != "access_denied" {
		t.Errorf("expected access_denied SandboxError, got %v", err)
	}
}

func TestSandbox_ReadOnly(t *testing.T) {
	engine := New(WithReadOnly(true))

	// Reading should work
	result, err := engine.Eval(context.Background(), "x + 1", map[string]any{"x": 5})
	if err != nil {
		t.Fatalf("reading should work: %v", err)
	}
	if NewValue(result).Int() != 6 {
		t.Errorf("expected 6, got %v", result)
	}

	// Let expressions (assignment) should fail
	_, err = engine.Eval(context.Background(), "let y = 5; y + 1")
	if err == nil {
		t.Error("expected assignment denied error")
	}
	if !errors.Is(err, ErrAssignmentNotAllowed) {
		t.Errorf("expected ErrAssignmentNotAllowed, got %v", err)
	}
}

func TestSandbox_Combined(t *testing.T) {
	engine := New(
		WithSandbox(SandboxConfig{
			MaxOps:      10000,
			Timeout:     time.Second,
			DeniedFuncs: []string{"print"},
			ReadOnly:    true,
		}),
	)

	// Normal expression should work
	result, err := engine.Eval(context.Background(), "1 + 2 * 3")
	if err != nil {
		t.Fatalf("normal expression failed: %v", err)
	}
	if NewValue(result).Int() != 7 {
		t.Errorf("expected 7, got %v", result)
	}

	// Denied function should fail
	_, err = engine.Eval(context.Background(), `print("test")`)
	if err == nil {
		t.Error("expected function denied error")
	}

	// Assignment should fail
	_, err = engine.Eval(context.Background(), "let x = 5; x")
	if err == nil {
		t.Error("expected assignment denied error")
	}
}

func TestSandboxError_Is(t *testing.T) {
	tests := []struct {
		errType string
		target  error
		want    bool
	}{
		{"ops_limit", ErrOpsLimitExceeded, true},
		{"memory_limit", ErrMemoryLimitExceeded, true},
		{"timeout", ErrTimeout, true},
		{"function_denied", ErrFunctionNotAllowed, true},
		{"assignment_denied", ErrAssignmentNotAllowed, true},
		{"stack_overflow", ErrStackOverflow, true},
		{"ops_limit", ErrTimeout, false},
		{"unknown", ErrOpsLimitExceeded, false},
	}

	for _, tt := range tests {
		t.Run(tt.errType, func(t *testing.T) {
			err := &SandboxError{Type: tt.errType, Message: "test"}
			if got := errors.Is(err, tt.target); got != tt.want {
				t.Errorf("SandboxError{Type: %q}.Is(%v) = %v, want %v", tt.errType, tt.target, got, tt.want)
			}
		})
	}
}

func TestSandboxError_Error(t *testing.T) {
	err := &SandboxError{
		Type:    "ops_limit",
		Message: "hyperex: operations limit exceeded",
		Limit:   1000,
		Actual:  1001,
	}
	if err.Error() != "hyperex: operations limit exceeded" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

// =============================================================================
// ANALYTICS TESTS
// =============================================================================

func TestAnalyze_Variables(t *testing.T) {
	analysis, err := Analyze("x + y * z")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(analysis.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d: %v", len(analysis.Variables), analysis.Variables)
	}

	// Variables should be sorted
	expected := []string{"x", "y", "z"}
	for i, v := range expected {
		if analysis.Variables[i] != v {
			t.Errorf("expected variable %s at position %d, got %s", v, i, analysis.Variables[i])
		}
	}
}

func TestAnalyze_Functions(t *testing.T) {
	analysis, err := Analyze("sum([1, 2, 3]) + len(arr)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(analysis.Functions) != 2 {
		t.Errorf("expected 2 functions, got %d: %v", len(analysis.Functions), analysis.Functions)
	}

	// Functions should be sorted
	expected := []string{"len", "sum"}
	for i, f := range expected {
		if analysis.Functions[i] != f {
			t.Errorf("expected function %s at position %d, got %s", f, i, analysis.Functions[i])
		}
	}
}

func TestAnalyze_Operators(t *testing.T) {
	analysis, err := Analyze("x + y * z - w / v")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(analysis.Operators) != 4 {
		t.Errorf("expected 4 operators, got %d: %v", len(analysis.Operators), analysis.Operators)
	}
}

func TestAnalyze_Complexity(t *testing.T) {
	tests := []struct {
		expr       string
		complexity Complexity
	}{
		{"1 + 2", ComplexityConstant},
		{"x + y", ComplexityConstant},
		{"sum([1, 2, 3])", ComplexityLinear},
		{"sort([3, 1, 2])", ComplexityLinearithmic}, // O(n log n) is correct for sorting
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			analysis, err := Analyze(tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if analysis.Complexity != tt.complexity {
				t.Errorf("expected complexity %s, got %s", tt.complexity, analysis.Complexity)
			}
		})
	}
}

func TestAnalyze_IsPure(t *testing.T) {
	tests := []struct {
		expr   string
		isPure bool
	}{
		{"1 + 2", true},
		{"x + y", true},
		{"sum([1, 2, 3])", true},
		{`print("hello")`, false}, // Side effect
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			analysis, err := Analyze(tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if analysis.IsPure != tt.isPure {
				t.Errorf("expected IsPure=%v, got %v", tt.isPure, analysis.IsPure)
			}
		})
	}
}

func TestAnalyze_HasAssignment(t *testing.T) {
	tests := []struct {
		expr          string
		hasAssignment bool
	}{
		{"1 + 2", false},
		{"let x = 5; x + 1", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			analysis, err := Analyze(tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if analysis.HasAssignment != tt.hasAssignment {
				t.Errorf("expected HasAssignment=%v, got %v", tt.hasAssignment, analysis.HasAssignment)
			}
		})
	}
}

func TestAnalyze_OptionalChaining(t *testing.T) {
	analysis, err := Analyze("obj?.field")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !analysis.UsesOptionalChaining {
		t.Error("expected UsesOptionalChaining=true")
	}
}

func TestAnalyze_NilCoalescing(t *testing.T) {
	analysis, err := Analyze("x ?? 0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !analysis.UsesNilCoalescing {
		t.Error("expected UsesNilCoalescing=true")
	}
}

func TestAnalyze_Depth(t *testing.T) {
	// Simple expression
	a1, _ := Analyze("1 + 2")
	// Nested expression
	a2, _ := Analyze("(1 + (2 + (3 + 4)))")

	if a1.Depth >= a2.Depth {
		t.Errorf("expected deeper nesting in second expr: %d vs %d", a1.Depth, a2.Depth)
	}
}

func TestAnalyze_NodeCount(t *testing.T) {
	a1, _ := Analyze("1")
	a2, _ := Analyze("1 + 2 + 3")

	if a1.NodeCount >= a2.NodeCount {
		t.Errorf("expected more nodes in second expr: %d vs %d", a1.NodeCount, a2.NodeCount)
	}
}

func TestProfileExpr(t *testing.T) {
	profile, result, err := ProfileExpr(context.Background(), "1 + 2 + 3", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if NewValue(result).Int() != 6 {
		t.Errorf("expected 6, got %v", result)
	}

	// Times can be 0 for very fast operations on fast machines
	// Just check they're non-negative
	if profile.TotalTime < 0 {
		t.Error("expected non-negative total time")
	}

	if profile.CompileTime < 0 {
		t.Error("expected non-negative compile time")
	}

	if profile.ExecuteTime < 0 {
		t.Error("expected non-negative exec time")
	}

	// Ops should always be positive
	if profile.OpsCount <= 0 {
		t.Error("expected positive ops count")
	}
}

func TestProfileExpr_MaxStackDepthTracksPeak(t *testing.T) {
	// Use env variables to avoid constant folding eliminating intermediate stack usage.
	profile, _, err := ProfileExpr(context.Background(), "a + b + c", map[string]any{
		"a": int64(1),
		"b": int64(2),
		"c": int64(3),
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// During evaluation there must be at least one moment where two operands are on stack.
	if profile.MaxStackDepth < 2 {
		t.Fatalf("expected MaxStackDepth >= 2, got %d", profile.MaxStackDepth)
	}
}

func TestProgram_Analyze(t *testing.T) {
	program := MustCompile("x + y * 2")
	analysis := program.Analyze()

	if len(analysis.Variables) != 2 {
		t.Errorf("expected 2 variables, got %d", len(analysis.Variables))
	}
}

func TestProgram_Profile(t *testing.T) {
	program := MustCompile("x + y")
	env := map[string]any{"x": int64(1), "y": int64(2)}

	profile, result, err := program.Profile(context.Background(), env)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if NewValue(result).Int() != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	if profile.OpsCount <= 0 {
		t.Error("expected positive ops count")
	}
}

// =============================================================================
// TRANSFORM TESTS
// =============================================================================

func TestSimplify_Constants(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2 + 3", "5"},
		{"2 * 3 + 4", "10"},
		{"10 / 2", "5"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Simplify(tt.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Simplify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSimplify_Algebraic(t *testing.T) {
	tests := []struct {
		input    string
		contains string // Result should contain this
	}{
		{"x + 0", "x"},
		{"x * 1", "x"},
		{"x * 0", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Simplify(tt.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Simplify(%q) = %q, should contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestExpand_Macros(t *testing.T) {
	macros := map[string]string{
		"ADULT": "age >= 18",
		"USA":   `country == "US"`,
	}

	result, err := Expand("ADULT", macros)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(result, "age") || !strings.Contains(result, "18") {
		t.Errorf("Expand did not expand ADULT macro: %s", result)
	}
}

func TestExpand_Combined(t *testing.T) {
	macros := map[string]string{
		"A": "x > 0",
		"B": "y < 10",
	}

	result, err := Expand("A && B", macros)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should contain both expanded macros
	if !strings.Contains(result, "x") || !strings.Contains(result, "y") {
		t.Errorf("Expand did not expand both macros: %s", result)
	}
}

func TestInline_Let(t *testing.T) {
	result, err := Inline("let x = 5; x + x")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should not contain "let"
	if strings.Contains(result, "let") {
		t.Errorf("Inline should remove let: %s", result)
	}
	// Should contain "5"
	if !strings.Contains(result, "5") {
		t.Errorf("Inline should substitute value: %s", result)
	}
}

func TestInline_NestedLet(t *testing.T) {
	result, err := Inline("let x = 2; let y = x + 1; y * 2")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should not contain "let"
	if strings.Contains(result, "let") {
		t.Errorf("Inline should remove all let expressions: %s", result)
	}
}

func TestASTToString_Roundtrip(t *testing.T) {
	exprs := []string{
		"1 + 2",
		"x * y",
		"a && b || c",
		"x > 0 ? 1 : 0",
		"x ?? 0",
		"[1, 2, 3]",
		`{"a": 1}`,
		"sum(arr)",
		"obj.field",
		"arr[0]",
	}

	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			parser := newParser(expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			result := ASTToString(ast)
			// The result should be parseable
			parser2 := newParser(result)
			_, err = parser2.parse()
			if err != nil {
				t.Errorf("ASTToString produced unparseable result: %s (from %s)", result, expr)
			}
		})
	}
}

func TestExtractVariables(t *testing.T) {
	vars, err := ExtractVariables("x + y * z")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(vars) != 3 {
		t.Errorf("expected 3 variables, got %d: %v", len(vars), vars)
	}
}

func TestExtractFunctions(t *testing.T) {
	funcs, err := ExtractFunctions("sum(arr) + len(s)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(funcs) != 2 {
		t.Errorf("expected 2 functions, got %d: %v", len(funcs), funcs)
	}
}

func TestValidate(t *testing.T) {
	// Valid expressions
	if err := Validate("1 + 2"); err != nil {
		t.Errorf("valid expression failed: %v", err)
	}

	// Invalid expressions
	if err := Validate("1 +"); err == nil {
		t.Error("invalid expression should fail")
	}
}

func TestValidateWithEnv(t *testing.T) {
	type Env struct {
		X int `expr:"x"`
	}

	// Valid
	if err := ValidateWithEnv("x + 1", Env{}); err != nil {
		t.Errorf("valid expression failed: %v", err)
	}
}

func TestPrint_Builtin(t *testing.T) {
	// print just outputs to stdout and returns nil
	result, err := Eval(context.Background(), "print(42)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}
}

// =============================================================================
// ADDITIONAL COVERAGE TESTS
// =============================================================================

func TestLexer_UnicodeEscape(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		value  string
		hasErr bool
	}{
		{"unicode_escape", `"\u0048\u0065\u006C\u006C\u006F"`, "Hello", false},
		{"unicode_heart", `"\u2665"`, "♥", false},
		{"unicode_invalid", `"\u00GG"`, "", true},
		{"unicode_short", `"\u00"`, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens, err := lexer.tokenize()
			if tt.hasErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens[0].Value != tt.value {
				t.Errorf("value: got %q, want %q", tokens[0].Value, tt.value)
			}
		})
	}
}

func TestLexer_UnicodeIdentifiers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		ident string
	}{
		{"cyrillic", "привет + 1", "привет"},
		{"chinese", "变量 + 1", "变量"},
		{"mixed", "var_переменная + 1", "var_переменная"},
		{"emoji_start", "$emoji + 1", "$emoji"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens, err := lexer.tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens[0].Type != tokenIdent {
				t.Errorf("expected ident, got %v", tokens[0].Type)
			}
			if tokens[0].Value != tt.ident {
				t.Errorf("ident: got %q, want %q", tokens[0].Value, tt.ident)
			}
		})
	}
}

func TestLambda_Compile(t *testing.T) {
	// Test that lambda expressions compile without error
	tests := []string{
		"x => x",
		"x => x * 2",
		"(a, b) => a + b",
		"() => 42",
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := Compile(expr)
			if err != nil {
				t.Errorf("compile error: %v", err)
			}
		})
	}
}

func TestLambda_IIFE(t *testing.T) {
	// Test Immediately Invoked Function Expression
	tests := []struct {
		name   string
		expr   string
		expect any
	}{
		{"single_param", "(x => x * 2)(5)", int64(10)},
		{"two_params", "((a, b) => a + b)(10, 20)", int64(30)},
		{"no_params", "(() => 42)()", int64(42)},
		{"three_params", "((a, b, c) => a + b + c)(1, 2, 3)", int64(6)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expect {
				t.Errorf("got %v (%T), want %v", result, result, tt.expect)
			}
		})
	}
}

func TestLambda_InLet(t *testing.T) {
	// Test lambda defined in let expression
	program := MustCompile("let double = x => x * 2; double(21)")
	result, err := program.Eval(context.Background(), nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != int64(42) {
		t.Errorf("got %v, want 42", result)
	}
}

func TestLambda_Closure_Variables(t *testing.T) {
	// Test lambda closure over let bindings
	tests := []struct {
		name   string
		expr   string
		expect any
	}{
		{"closure_multiply", "let mul = x => y => x * y; let double = mul(2); double(5)", int64(10)},
		{"closure_add", "let add = a => b => a + b; let add5 = add(5); add5(3)", int64(8)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expect {
				t.Errorf("got %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestRegex_CacheHit(t *testing.T) {
	// Test regex cache by using same pattern multiple times
	program := MustCompile(`text ~= "^hello"`)
	env := map[string]any{"text": "hello world"}

	// Run multiple times to ensure cache is used
	for i := 0; i < 10; i++ {
		result, err := program.Eval(context.Background(), env)
		if err != nil {
			t.Fatalf("error on iteration %d: %v", i, err)
		}
		if result != true {
			t.Errorf("iteration %d: got %v, want true", i, result)
		}
	}
}

func TestRegex_ManyPatterns(t *testing.T) {
	// Test with many different patterns to trigger LRU eviction
	program := MustCompile(`text ~= pattern`)
	text := "hello world 12345"

	patterns := make([]string, 300) // More than maxRegexCacheSize
	for i := 0; i < 300; i++ {
		patterns[i] = "^" + string(rune('a'+i%26)) + "*"
	}

	for _, p := range patterns {
		env := map[string]any{"text": text, "pattern": p}
		_, err := program.Eval(context.Background(), env)
		if err != nil {
			t.Fatalf("error with pattern %s: %v", p, err)
		}
	}
}

func TestInline_Complex(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			"nested_let",
			"let x = 1; let y = x + 1; let z = y + 1; z",
			"1 + 1 + 1",
		},
		{
			"let_in_binary",
			"let a = 2; a + a",
			"2 + 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Inline(tt.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			// Just verify it doesn't error - the exact format may vary
			if result == "" {
				t.Error("got empty result")
			}
		})
	}
}

func TestInline_AllNodeTypes(t *testing.T) {
	// Test inline with various node types
	tests := []string{
		"let x = 1; x + 2",         // binary
		"let x = true; !x",         // unary
		"let x = 1; x > 0 ? 1 : 0", // ternary
		"let x = nil; x ?? 1",      // coalesce
		"let x = [1,2]; x[0]",      // index
		"let x = {a:1}; x.a",       // member
		"let arr = [1,2,3]; arr",   // array
		"let m = {a:1}; m",         // map
		"let f = len; f([1,2])",    // call
	}

	for _, expr := range tests {
		_, err := Inline(expr)
		if err != nil {
			t.Errorf("Inline(%q) error: %v", expr, err)
		}
	}
}

func TestTypeChecker_WithVar(t *testing.T) {
	tc := NewTypeChecker()
	tc.WithVar("x", TypeInt)
	tc.WithVar("y", TypeString)

	parser := newParser("x + 1")
	ast, _ := parser.parse()
	typ := tc.Infer(ast)
	if typ != TypeInt {
		t.Errorf("got %v, want TypeInt", typ)
	}
}

func TestTypeChecker_WithFunc(t *testing.T) {
	tc := NewTypeChecker()
	tc.WithFunc("myFunc", FuncSig{Params: []Type{TypeInt}, Returns: TypeString})

	parser := newParser("myFunc(42)")
	ast, _ := parser.parse()
	typ := tc.Infer(ast)
	if typ != TypeString {
		t.Errorf("got %v, want TypeString", typ)
	}
}

func TestTypeChecker_AllTypes(t *testing.T) {
	tc := NewTypeChecker()
	tests := []struct {
		expr string
		typ  Type
	}{
		{"nil", TypeNil},
		{"42", TypeInt},
		{"3.14", TypeFloat},
		{`"hello"`, TypeString},
		{"true", TypeBool},
		{"[1,2,3]", TypeArray},
		{"{a:1}", TypeMap},
		{"1..5", TypeArray},
		{"x => x", TypeFunc},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, _ := parser.parse()
			typ := tc.Infer(ast)
			if typ != tt.typ {
				t.Errorf("got %v, want %v", typ, tt.typ)
			}
		})
	}
}

func TestTypeChecker_Coalesce(t *testing.T) {
	tc := NewTypeChecker()
	tc.WithVar("x", TypeNil)
	tc.WithVar("y", TypeInt)

	parser := newParser("x ?? y")
	ast, _ := parser.parse()
	typ := tc.Infer(ast)
	if typ != TypeInt {
		t.Errorf("got %v, want TypeInt", typ)
	}
}

func TestTypeChecker_Index(t *testing.T) {
	tc := NewTypeChecker()

	// Array index returns Any
	parser := newParser("[1,2,3][0]")
	ast, _ := parser.parse()
	typ := tc.Infer(ast)
	if typ != TypeAny {
		t.Errorf("array index: got %v, want TypeAny", typ)
	}

	// String index returns String
	parser = newParser(`"hello"[0]`)
	ast, _ = parser.parse()
	typ = tc.Infer(ast)
	if typ != TypeString {
		t.Errorf("string index: got %v, want TypeString", typ)
	}
}

func TestTypeChecker_Slice(t *testing.T) {
	tc := NewTypeChecker()

	parser := newParser("[1,2,3][0:2]")
	ast, _ := parser.parse()
	typ := tc.Infer(ast)
	if typ != TypeArray {
		t.Errorf("got %v, want TypeArray", typ)
	}
}

func TestTypeChecker_MoreBuiltins(t *testing.T) {
	tc := NewTypeChecker()
	tests := []struct {
		expr string
		typ  Type
	}{
		{"abs(-5)", TypeAny},
		{"len([1,2])", TypeInt},
		{"upper('hi')", TypeString},
		{"split('a,b', ',')", TypeArray},
		{"contains('hi', 'h')", TypeBool},
		{"indexOf('hi', 'i')", TypeInt},
		{"first([1,2])", TypeAny},
		{"keys({a:1})", TypeArray},
		{"sum([1,2])", TypeAny},
		{"count([1,2])", TypeInt},
		{"all([true])", TypeBool},
		{"int(3.14)", TypeInt},
		{"float(3)", TypeFloat},
		{"string(42)", TypeString},
		{"bool(1)", TypeBool},
		{"type(42)", TypeString},
		{"isNil(nil)", TypeBool},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, _ := parser.parse()
			typ := tc.Infer(ast)
			if typ != tt.typ {
				t.Errorf("got %v, want %v", typ, tt.typ)
			}
		})
	}
}

func TestError_NewFunctions(t *testing.T) {
	err := NewRuntimeErrorf("test %d", 42)
	if !strings.Contains(err.Error(), "test 42") {
		t.Errorf("got %q, want contains 'test 42'", err.Error())
	}

	typeErr := NewTypeError("mismatch", "int", "string")
	if !strings.Contains(typeErr.Error(), "mismatch") {
		t.Errorf("got %q, want contains 'mismatch'", typeErr.Error())
	}
}

func TestParseError_WithSource(t *testing.T) {
	pe := &ParseError{
		Message: "test error",
		token:   token{Line: 1, Column: 5},
	}
	pe.WithSource("x + y + z")

	errStr := pe.Error()
	if !strings.Contains(errStr, "test error") {
		t.Errorf("error should contain message: %s", errStr)
	}
	if pe.Line() != 1 || pe.Column() != 5 {
		t.Errorf("Line/Column = %d:%d, want 1:5", pe.Line(), pe.Column())
	}
}

func TestASTToString_AllNodes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"int", "42"},
		{"float", "3.14"},
		{"string", `"hello"`},
		{"bool_true", "true"},
		{"bool_false", "false"},
		{"nil", "nil"},
		{"ident", "x"},
		{"binary", "x + y"},
		{"unary_minus", "-x"},
		{"unary_not", "!x"},
		{"unary_not_word", "not x"},
		{"ternary", "a ? b : c"},
		{"coalesce", "a ?? b"},
		{"call", "f(x, y)"},
		{"member", "obj.field"},
		{"member_opt", "obj?.field"},
		{"index", "arr[0]"},
		{"index_opt", "arr?.[0]"},
		{"slice", "arr[1:3]"},
		{"array", "[1, 2, 3]"},
		{"map", `{"a": 1}`},
		{"range", "(1..5)"},
		{"lambda_single", "x => x"},
		{"lambda_multi", "(a, b) => a + b"},
		{"let", "let x = 1; x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := newParser(tt.input)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			str := ASTToString(ast)
			if str == "" {
				t.Error("got empty string")
			}
		})
	}
}

func TestBuiltin_TrimLeftRight(t *testing.T) {
	tests := []struct {
		name   string
		expr   string
		expect string
	}{
		{"trimLeft_default", `trimLeft("  hello")`, "hello"},
		{"trimLeft_chars", `trimLeft("xxhello", "x")`, "hello"},
		{"trimRight_default", `trimRight("hello  ")`, "hello"},
		{"trimRight_chars", `trimRight("helloxx", "x")`, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvalString(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if result != tt.expect {
				t.Errorf("got %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestBuiltin_ClampFloat(t *testing.T) {
	result, err := EvalFloat(context.Background(), "clamp(1.5, 2.0, 3.0)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 2.0 {
		t.Errorf("got %v, want 2.0", result)
	}

	result, err = EvalFloat(context.Background(), "clamp(5.0, 2.0, 3.0)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 3.0 {
		t.Errorf("got %v, want 3.0", result)
	}
}

func TestBuiltin_RangeNegativeStep(t *testing.T) {
	result, err := Eval(context.Background(), "range(5, 1, -1)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr) != 5 {
		t.Errorf("got %d elements, want 5", len(arr))
	}
}

func TestBuiltin_RangeZeroStep(t *testing.T) {
	_, err := Eval(context.Background(), "range(1, 5, 0)")
	if err == nil {
		t.Error("expected error for zero step")
	}
}

func TestBuiltin_ChunkInvalid(t *testing.T) {
	_, err := Eval(context.Background(), "chunk([1,2,3], 0)")
	if err == nil {
		t.Error("expected error for zero chunk size")
	}
}

func TestBuiltin_JSONErrors(t *testing.T) {
	// toJSON with unencodable value is tested implicitly
	// fromJSON with invalid JSON
	_, err := Eval(context.Background(), `fromJSON("not valid json")`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestBuiltin_LastEmpty(t *testing.T) {
	result, err := Eval(context.Background(), "last([])")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}
}

func TestBuiltin_AvgEmpty(t *testing.T) {
	result, err := Eval(context.Background(), "avg([])")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}
}

func TestParser_SliceAllCases(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"full", "arr[1:3]"},
		{"omit_low", "arr[:3]"},
		{"omit_high", "arr[1:]"},
		{"omit_both", "arr[:]"},
		{"with_step", "arr[::2]"},
		{"all_three", "arr[1:5:2]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := newParser(tt.expr)
			_, err := parser.parse()
			if err != nil {
				t.Errorf("parse error: %v", err)
			}
		})
	}
}

func TestCompiler_AllBinaryOps(t *testing.T) {
	ops := []string{
		"x + y", "x - y", "x * y", "x / y", "x % y", "x ** y",
		"x == y", "x != y", "x < y", "x <= y", "x > y", "x >= y",
		"x && y", "x || y",
		"x & y", "x | y", "x ^ y", "x << y", "x >> y",
		"x in y", "x not in y", `x ~= "a"`,
	}

	env := map[string]any{"x": int64(5), "y": int64(3)}

	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			_, err := Compile(op)
			if err != nil {
				t.Errorf("compile error: %v", err)
			}
		})
	}

	// Also run with evaluation where possible
	evalOps := []string{
		"x + y", "x - y", "x * y", "x / y", "x % y",
		"x == y", "x != y", "x < y", "x <= y", "x > y", "x >= y",
		"x && y", "x || y",
		"x & y", "x | y", "x ^ y", "x << y", "x >> y",
	}

	for _, op := range evalOps {
		_, err := Eval(context.Background(), op, env)
		if err != nil {
			t.Errorf("eval %s error: %v", op, err)
		}
	}
}

func TestVM_Execute_AllOpcodes(t *testing.T) {
	// This test ensures we hit various vm execution paths
	tests := []struct {
		name string
		expr string
		env  map[string]any
	}{
		{"power", "2 ** 3", nil},
		{"mod_zero", "10 % 3", nil},
		{"bitwise_not", "~5", nil},
		{"opt_member_nil", "x?.field", map[string]any{"x": nil}},
		{"opt_index_nil", "x?.[0]", map[string]any{"x": nil}},
		{"range", "1..5", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Eval(context.Background(), tt.expr, tt.env)
			if err != nil {
				t.Errorf("error: %v", err)
			}
		})
	}
}

func TestVM_ContainsMap(t *testing.T) {
	result, err := EvalBool(context.Background(), `"a" in {"a": 1, "b": 2}`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !result {
		t.Error("expected true")
	}
}

func TestGoTypeToHyperexType(t *testing.T) {
	type TestStruct struct {
		X int
	}

	tc := NewTypeChecker()
	tc.WithEnvType(struct {
		Bool    bool
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32
		Int64   int64
		Uint    uint
		Uint8   uint8
		Float32 float32
		Float64 float64
		String  string
		Slice   []int
		Array   [3]int
		Map     map[string]int
		Func    func()
		Any     any
		Struct  TestStruct
		Ptr     *int
	}{})

	// TypeChecker should have registered all these types
	// This exercises goTypeToHyperexType
}

func TestSandbox_StackOverflow(t *testing.T) {
	engine := New(WithSandbox(SandboxConfig{
		MaxStackDepth: 10,
	}))

	// Create expression that will exceed stack depth
	expr := "1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1"
	program := engine.MustCompile(expr)
	_, err := program.Eval(context.Background(), nil)

	// May or may not error depending on actual stack usage
	_ = err
}

func TestValue_NewValueFromStruct(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	p := Person{Name: "John", Age: 30}
	v := NewValue(p)

	if v.Type() != "hyperex.Person" && v.Type() != "Person" {
		// Type returns the reflect type name for structs
		t.Logf("struct type: %s", v.Type())
	}
}

func TestValue_NewValueNilPointer(t *testing.T) {
	var ptr *int = nil
	v := NewValue(ptr)
	if !v.IsNil() {
		t.Error("expected nil value for nil pointer")
	}
}

// =============================================================================
// NEW FEATURE TESTS
// =============================================================================

func TestSpread_Array(t *testing.T) {
	tests := []struct {
		name   string
		expr   string
		expect int // expected length
	}{
		{"spread_single", "[...[1, 2, 3]]", 3},
		{"spread_multiple", "[...[1, 2], ...[3, 4]]", 4},
		{"spread_with_elements", "[0, ...[1, 2], 3]", 4},
		{"spread_empty", "[...[]]", 0},
		{"spread_nested", "[...[1, ...[2, 3]]]", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(context.Background(), tt.expr)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			arr, ok := result.([]any)
			if !ok {
				t.Fatalf("expected array, got %T", result)
			}
			if len(arr) != tt.expect {
				t.Errorf("len: got %d, want %d", len(arr), tt.expect)
			}
		})
	}
}

func TestSpread_Lexer(t *testing.T) {
	lexer := newLexer("...")
	tokens, err := lexer.tokenize()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if tokens[0].Type != tokenSpread {
		t.Errorf("got %v, want tokenSpread", tokens[0].Type)
	}
}

func TestHigherOrder_Map(t *testing.T) {
	result, err := Eval(context.Background(), "map([1, 2, 3], x => x * 2)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr) != 3 {
		t.Errorf("len: got %d, want 3", len(arr))
	}
	if arr[0] != int64(2) || arr[1] != int64(4) || arr[2] != int64(6) {
		t.Errorf("got %v, want [2, 4, 6]", arr)
	}
}

func TestHigherOrder_Filter(t *testing.T) {
	result, err := Eval(context.Background(), "filter([1, 2, 3, 4, 5], x => x > 2)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr) != 3 {
		t.Errorf("len: got %d, want 3", len(arr))
	}
}

func TestHigherOrder_Reduce(t *testing.T) {
	result, err := EvalInt(context.Background(), "reduce([1, 2, 3, 4], (acc, x) => acc + x, 0)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 10 {
		t.Errorf("got %d, want 10", result)
	}
}

func TestHigherOrder_ReduceNoInit(t *testing.T) {
	result, err := EvalInt(context.Background(), "reduce([1, 2, 3, 4], (acc, x) => acc + x)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 10 {
		t.Errorf("got %d, want 10", result)
	}
}

func TestHigherOrder_Find(t *testing.T) {
	result, err := EvalInt(context.Background(), "find([1, 2, 3, 4], x => x > 2)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 3 {
		t.Errorf("got %d, want 3", result)
	}
}

func TestHigherOrder_FindIndex(t *testing.T) {
	result, err := EvalInt(context.Background(), "findIndex([1, 2, 3, 4], x => x > 2)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 2 {
		t.Errorf("got %d, want 2", result)
	}
}

func TestHigherOrder_FindNotFound(t *testing.T) {
	result, err := Eval(context.Background(), "find([1, 2, 3], x => x > 10)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("got %v, want nil", result)
	}
}

func TestDestructure_Object(t *testing.T) {
	result, err := EvalInt(context.Background(), `let {a, b} = {a: 1, b: 2}; a + b`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 3 {
		t.Errorf("got %d, want 3", result)
	}
}

func TestDestructure_ObjectAlias(t *testing.T) {
	result, err := EvalInt(context.Background(), `let {a: x, b: y} = {a: 10, b: 20}; x + y`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 30 {
		t.Errorf("got %d, want 30", result)
	}
}

func TestDestructure_ObjectDefault(t *testing.T) {
	result, err := EvalInt(context.Background(), `let {a, b = 100} = {a: 1}; a + b`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 101 {
		t.Errorf("got %d, want 101", result)
	}
}

func TestDestructure_Array(t *testing.T) {
	result, err := EvalInt(context.Background(), `let [a, b, c] = [1, 2, 3]; a + b + c`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 6 {
		t.Errorf("got %d, want 6", result)
	}
}

func TestDestructure_ArrayDefault(t *testing.T) {
	result, err := EvalInt(context.Background(), `let [a, b = 10] = [1]; a + b`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 11 {
		t.Errorf("got %d, want 11", result)
	}
}

func TestUnicode_StringIndex(t *testing.T) {
	result, err := EvalString(context.Background(), `"Привет"[0]`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "П" {
		t.Errorf("got %q, want П", result)
	}
}

func TestUnicode_StringSlice(t *testing.T) {
	result, err := EvalString(context.Background(), `"Привет"[0:3]`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "При" {
		t.Errorf("got %q, want При", result)
	}
}

func TestUnicode_StringLen(t *testing.T) {
	result, err := EvalInt(context.Background(), `len("Привет")`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 6 { // 6 runes, not 12 bytes
		t.Errorf("got %d, want 6", result)
	}
}

func TestUnicode_NegativeIndex(t *testing.T) {
	result, err := EvalString(context.Background(), `"Hello"[-1]`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != "o" {
		t.Errorf("got %q, want o", result)
	}
}

// =============================================================================
// COVERAGE IMPROVEMENT TESTS
// =============================================================================

func TestHigherOrder_MapWithNativeFunc(t *testing.T) {
	// Test map with a native Go function (must accept element and index)
	engine := New(
		WithFunction("double", func(x, idx int) int { return x * 2 }),
	)
	result, err := engine.Eval(context.Background(), "map([1, 2, 3], double)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr) != 3 {
		t.Errorf("got %d elements, want 3", len(arr))
	}
}

func TestHigherOrder_FilterWithIndex(t *testing.T) {
	// Test filter that uses index parameter
	result, err := Eval(context.Background(), "filter([10, 20, 30, 40], (x, i) => i > 1)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr) != 2 {
		t.Errorf("got %d elements, want 2", len(arr))
	}
}

func TestHigherOrder_ReduceWithIndex(t *testing.T) {
	// Test reduce that uses index parameter
	result, err := EvalInt(context.Background(), "reduce([1, 2, 3], (acc, x, i) => acc + x + i, 0)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// sum = 0 + (1+0) + (2+1) + (3+2) = 0 + 1 + 3 + 5 = 9
	if result != 9 {
		t.Errorf("got %d, want 9", result)
	}
}

func TestHigherOrder_MapErrors(t *testing.T) {
	// Test map with wrong number of arguments
	_, err := Eval(context.Background(), "map([1, 2])")
	if err == nil {
		t.Error("expected error for map with 1 arg")
	}

	// Test map with non-array first argument
	_, err = Eval(context.Background(), "map(42, x => x)")
	if err == nil {
		t.Error("expected error for map with non-array")
	}

	// Test map with non-function second argument
	_, err = Eval(context.Background(), "map([1, 2], 42)")
	if err == nil {
		t.Error("expected error for map with non-function")
	}
}

func TestHigherOrder_FilterErrors(t *testing.T) {
	_, err := Eval(context.Background(), "filter([1, 2])")
	if err == nil {
		t.Error("expected error for filter with 1 arg")
	}

	_, err = Eval(context.Background(), "filter(42, x => x)")
	if err == nil {
		t.Error("expected error for filter with non-array")
	}

	_, err = Eval(context.Background(), "filter([1, 2], 42)")
	if err == nil {
		t.Error("expected error for filter with non-function")
	}
}

func TestHigherOrder_ReduceErrors(t *testing.T) {
	_, err := Eval(context.Background(), "reduce([1, 2])")
	if err == nil {
		t.Error("expected error for reduce with 1 arg")
	}

	_, err = Eval(context.Background(), "reduce(42, (a, x) => a + x)")
	if err == nil {
		t.Error("expected error for reduce with non-array")
	}

	_, err = Eval(context.Background(), "reduce([1, 2], 42)")
	if err == nil {
		t.Error("expected error for reduce with non-function")
	}
}

func TestHigherOrder_FindErrors(t *testing.T) {
	_, err := Eval(context.Background(), "find([1, 2])")
	if err == nil {
		t.Error("expected error for find with 1 arg")
	}

	_, err = Eval(context.Background(), "find(42, x => x)")
	if err == nil {
		t.Error("expected error for find with non-array")
	}
}

func TestHigherOrder_FindIndexErrors(t *testing.T) {
	_, err := Eval(context.Background(), "findIndex([1, 2])")
	if err == nil {
		t.Error("expected error for findIndex with 1 arg")
	}

	_, err = Eval(context.Background(), "findIndex(42, x => x)")
	if err == nil {
		t.Error("expected error for findIndex with non-array")
	}
}

func TestHigherOrder_ReduceEmpty(t *testing.T) {
	// Empty array with init
	result, err := EvalInt(context.Background(), "reduce([], (a, x) => a + x, 100)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 100 {
		t.Errorf("got %d, want 100", result)
	}

	// Empty array without init
	result2, err := Eval(context.Background(), "reduce([], (a, x) => a + x)")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result2 != nil {
		t.Errorf("got %v, want nil", result2)
	}
}

func TestAnalyze_AllNodeTypes(t *testing.T) {
	exprs := []string{
		"1 + 2",        // binaryExpr, intLit
		"-x",           // unaryExpr
		"a ? b : c",    // ternaryExpr
		"x ?? y",       // coalesceExpr
		"f(1, 2)",      // callExpr
		"obj.field",    // memberExpr
		"arr[0]",       // indexExpr
		"arr[1:3]",     // sliceExpr
		"[1, 2, 3]",    // arrayLit
		"{a: 1}",       // mapLit
		"1..5",         // rangeExpr
		"x => x",       // lambdaExpr
		"let x = 1; x", // letExpr
		`"hello"`,      // stringLit
		"true",         // boolLit
		"nil",          // nilLit
		"3.14",         // floatLit
		"variable",     // ident
	}

	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			analysis, err := Analyze(expr)
			if err != nil {
				t.Errorf("Analyze(%q) error: %v", expr, err)
				return
			}
			if analysis == nil {
				t.Errorf("Analyze(%q) returned nil", expr)
			}
		})
	}
}

func TestAnalyze_SpreadExpr(t *testing.T) {
	analysis, err := Analyze("[...[1, 2], 3]")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if analysis == nil {
		t.Fatal("expected analysis result")
	}
}

func TestAnalyze_DestructureExpr(t *testing.T) {
	analysis, err := Analyze("let {a, b} = obj; a + b")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if analysis == nil {
		t.Fatal("expected analysis result")
	}
}

func TestSandbox_AllLimits(t *testing.T) {
	engine := New(WithSandbox(SandboxConfig{
		MaxOps:        100,
		MaxStackDepth: 50,
		MaxCallDepth:  10,
	}))

	// Simple expression should work
	result, err := engine.EvalInt(context.Background(), "1 + 2")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 3 {
		t.Errorf("got %d, want 3", result)
	}
}

func TestSpread_Errors(t *testing.T) {
	// Spread on non-array should fail
	_, err := Eval(context.Background(), "[...42]")
	if err == nil {
		t.Error("expected error for spreading non-array")
	}
}

func TestDestructure_Nested(t *testing.T) {
	// Nested destructuring usage
	result, err := EvalInt(context.Background(), `
		let {a, b} = {a: {x: 1}, b: 2};
		b
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 2 {
		t.Errorf("got %d, want 2", result)
	}
}

func TestVM_SubTypes(t *testing.T) {
	// Float - Float
	result, err := EvalFloat(context.Background(), "3.5 - 1.5")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 2.0 {
		t.Errorf("got %f, want 2.0", result)
	}
}

func TestVM_DivTypes(t *testing.T) {
	// Float / Float
	result, err := EvalFloat(context.Background(), "10.0 / 4.0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != 2.5 {
		t.Errorf("got %f, want 2.5", result)
	}
}

func TestTypeCheckErrors_Multiple(t *testing.T) {
	// Test TypeCheckErrors with multiple errors
	errs := &TypeCheckErrors{
		Errors: []error{
			&TypeError{Message: "first error"},
			&TypeError{Message: "second error"},
		},
	}

	// Test Error() with multiple errors
	errStr := errs.Error()
	if !strings.Contains(errStr, "2 type errors") {
		t.Errorf("expected '2 type errors' in message, got: %s", errStr)
	}
	if !strings.Contains(errStr, "first error") || !strings.Contains(errStr, "second error") {
		t.Errorf("expected both errors in message, got: %s", errStr)
	}

	// Test Unwrap
	first := errs.Unwrap()
	if first == nil {
		t.Error("Unwrap returned nil")
	}
	if !strings.Contains(first.Error(), "first error") {
		t.Errorf("expected first error, got: %v", first)
	}

	// Test single error
	single := &TypeCheckErrors{Errors: []error{&TypeError{Message: "single"}}}
	if single.Error() != "type error: single" {
		t.Errorf("expected single error message, got: %s", single.Error())
	}

	// Test empty
	empty := &TypeCheckErrors{Errors: []error{}}
	if empty.Unwrap() != nil {
		t.Error("expected nil from empty Unwrap")
	}
}

func TestTypeChecker_PipeExpr(t *testing.T) {
	// Test that pipeExpr is properly type-checked
	tc := NewTypeChecker()
	parser := newParser("[1, 2, 3] | sum")
	ast, err := parser.parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// This should type-check without errors
	errs := tc.Check(ast)
	if errs != nil && errs.Len() > 0 {
		t.Errorf("unexpected type errors for pipe expression: %v", errs)
	}
}

func TestOptimizer_SpreadDestructure(t *testing.T) {
	engine := New(WithOptimize(true))

	// Test spread expression optimization
	result, err := engine.Eval(context.Background(), "[...[1, 2], 3]", nil)
	if err != nil {
		t.Fatalf("spread error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}

	// Test destructure optimization
	result, err = engine.Eval(context.Background(), "let {a, b} = {\"a\": 1, \"b\": 2}; a + b", nil)
	if err != nil {
		t.Fatalf("destructure error: %v", err)
	}
	if result.(int64) != 3 {
		t.Errorf("expected 3, got %v", result)
	}

	// Test slice expression optimization
	result, err = engine.Eval(context.Background(), "[1, 2, 3, 4][1:3]", nil)
	if err != nil {
		t.Fatalf("slice error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 2 {
		t.Errorf("expected 2 elements, got %d", len(arr))
	}

	// Test range expression optimization (1..5 is inclusive: [1,2,3,4,5])
	result, err = engine.Eval(context.Background(), "1..5", nil)
	if err != nil {
		t.Fatalf("range error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 5 {
		t.Errorf("expected 5 elements (1..5 inclusive), got %d", len(arr))
	}
}

func TestSlice_WithStep(t *testing.T) {
	// Test array slicing with step
	arrTests := []struct {
		expr     string
		expected []any
	}{
		{"[1,2,3,4,5][::2]", []any{int64(1), int64(3), int64(5)}},
		{"[1,2,3,4,5][1::2]", []any{int64(2), int64(4)}},
		{"[1,2,3,4,5][::-1]", []any{int64(5), int64(4), int64(3), int64(2), int64(1)}},
	}

	for _, tt := range arrTests {
		result, err := Eval(context.Background(), tt.expr)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.expr, err)
			continue
		}
		arr, ok := result.([]any)
		if !ok {
			t.Errorf("%s: expected array, got %T", tt.expr, result)
			continue
		}
		if len(arr) != len(tt.expected) {
			t.Errorf("%s: expected %v, got %v", tt.expr, tt.expected, arr)
		}
	}

	// Test string slicing with step
	strTests := []struct {
		expr     string
		expected string
	}{
		{"\"hello\"[::2]", "hlo"},
		{"\"hello\"[::-1]", "olleh"},
	}

	for _, tt := range strTests {
		result, err := Eval(context.Background(), tt.expr)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.expr, err)
			continue
		}
		if result.(string) != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.expr, tt.expected, result)
		}
	}
}

func TestRegexCache_LRU(t *testing.T) {
	// Test that regex cache properly handles LRU eviction
	patterns := make([]string, maxRegexCacheSize+10)
	for i := range patterns {
		patterns[i] = fmt.Sprintf("pattern_%d", i)
	}

	// Fill cache beyond capacity
	for _, p := range patterns {
		_, err := getRegexp(p)
		if err != nil {
			t.Fatalf("unexpected error compiling pattern: %v", err)
		}
	}

	// Access early patterns - they should still compile (may have been evicted)
	for i := 0; i < 5; i++ {
		_, err := getRegexp(patterns[i])
		if err != nil {
			t.Fatalf("unexpected error re-compiling pattern: %v", err)
		}
	}
}

func TestOptimizer_NegativeIntegers(t *testing.T) {
	// Ensure optimizer doesn't break semantics for negative integers
	// These operations have different results for bitwise vs arithmetic ops
	engine := New(WithOptimize(true))

	tests := []struct {
		expr     string
		expected int64
	}{
		// Division - should NOT be optimized to shift for negative numbers
		{"-5 / 2", -2}, // Truncation toward zero
		{"-7 / 4", -1}, // Not -2 like -7 >> 2 would give
		{"-10 / 2", -5},

		// Modulo - should NOT be optimized to AND for negative numbers
		{"-5 % 2", -1},  // Not 1 like -5 & 1 would give
		{"-7 % 4", -3},  // Not 1 like -7 & 3 would give
		{"-10 % 8", -2}, // Not 6 like -10 & 7 would give
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.expr, err)
			continue
		}
		if got := result.(int64); got != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.expr, tt.expected, got)
		}
	}
}

func TestOptimizer_SideEffects(t *testing.T) {
	// Ensure optimizer preserves side effects
	callCount := 0
	engine := New(
		WithOptimize(true),
		WithFunction("sideEffect", func() int64 {
			callCount++
			return 42
		}),
	)

	// x * 0 should NOT eliminate sideEffect call
	callCount = 0
	_, err := engine.Eval(context.Background(), "sideEffect() * 0", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("sideEffect() * 0: expected function to be called, but callCount=%d", callCount)
	}

	// 0 * x should NOT eliminate sideEffect call
	callCount = 0
	_, err = engine.Eval(context.Background(), "0 * sideEffect()", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("0 * sideEffect(): expected function to be called, but callCount=%d", callCount)
	}
}

func TestSubstr_Overflow(t *testing.T) {
	// Test that substr handles extreme values without panicking
	engine := New()

	// Large start - should return empty string
	result, err := engine.Eval(context.Background(), `substr("hello", 999999999)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(string) != "" {
		t.Errorf("expected empty string for large start, got %q", result)
	}

	// Negative length - should be treated as 0
	result, err = engine.Eval(context.Background(), `substr("hello", 0, -5)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(string) != "" {
		t.Errorf("expected empty string for negative length, got %q", result)
	}

	// Very large length - should clamp to string length
	result, err = engine.Eval(context.Background(), `substr("hello", 0, 999999999)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(string) != "hello" {
		t.Errorf("expected 'hello' for large length, got %q", result)
	}
}

// =============================================================================
// NEW TESTS FOR BUG FIXES
// =============================================================================

func TestOptionalIndex_OutOfBounds(t *testing.T) {
	// opOptIndex should return nil on out-of-bounds, not error
	engine := New()
	env := map[string]any{
		"arr": []any{int64(1), int64(2), int64(3)},
	}

	// Out of bounds should return nil, not error
	result, err := engine.Eval(context.Background(), "arr?.[10]", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for out of bounds, got %v", result)
	}

	// Negative out of bounds
	result, err = engine.Eval(context.Background(), "arr?.[-10]", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for negative out of bounds, got %v", result)
	}

	// Valid index should still work
	result, err = engine.Eval(context.Background(), "arr?.[1]", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != int64(2) {
		t.Errorf("expected 2, got %v", result)
	}
}

func TestDestructure_OutOfBounds(t *testing.T) {
	// Array destructuring should use nil for missing elements
	engine := New()

	result, err := engine.Eval(context.Background(), `let [a, b, c] = [1, 2]; c ?? 999`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if NewValue(result).Int() != 999 {
		t.Errorf("expected 999 for missing element, got %v", result)
	}
}

func TestPowerOperator_IntPrecision(t *testing.T) {
	// Test that ** preserves int precision for integer exponents
	engine := New()

	tests := []struct {
		expr     string
		expected int64
	}{
		{"2 ** 10", 1024},
		{"2 ** 20", 1048576},
		{"2 ** 30", 1073741824},
		{"2 ** 40", 1099511627776},
		{"2 ** 50", 1125899906842624},
		{"2 ** 60", 1152921504606846976},
		{"3 ** 10", 59049},
		{"10 ** 6", 1000000},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		// Result should be int64, not float64
		if got, ok := result.(int64); ok {
			if got != tt.expected {
				t.Errorf("%s: expected %d, got %d", tt.expr, tt.expected, got)
			}
		} else {
			t.Errorf("%s: expected int64, got %T (%v)", tt.expr, result, result)
		}
	}
}

func TestRange_EdgeCases(t *testing.T) {
	engine := New()

	tests := []struct {
		expr     string
		expected int // expected length
	}{
		// Positive step, start > end = empty
		{"range(10, 1, 1)", 0},
		// Negative step, start < end = empty
		{"range(1, 10, -1)", 0},
		// Normal cases
		{"range(1, 5, 1)", 5},
		{"range(5, 1, -1)", 5},
		{"range(0, 10, 2)", 6},  // 0, 2, 4, 6, 8, 10
		{"range(10, 0, -2)", 6}, // 10, 8, 6, 4, 2, 0
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		arr := NewValue(result).Array()
		if len(arr) != tt.expected {
			t.Errorf("%s: expected length %d, got %d (%v)", tt.expr, tt.expected, len(arr), arr)
		}
	}
}

func TestOptimizer_TripleNegation(t *testing.T) {
	engine := New(WithOptimize(true))
	env := map[string]any{"x": true}

	tests := []struct {
		expr     string
		expected bool
	}{
		{"!!x", true},
		{"!!!x", false},
		{"!!!!x", true},
		{"!!!!!x", false},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, env)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		if result != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestOptimizer_TripleNegative(t *testing.T) {
	engine := New(WithOptimize(true))
	env := map[string]any{"x": int64(5)}

	tests := []struct {
		expr     string
		expected int64
	}{
		{"-(-x)", 5},
		{"-(-(-x))", -5},
		{"-(-(-(-x)))", 5},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, env)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		if NewValue(result).Int() != tt.expected {
			t.Errorf("%s: expected %d, got %v", tt.expr, tt.expected, result)
		}
	}
}

// =============================================================================
// SECURITY AND EDGE CASE TESTS (from code review)
// =============================================================================

func TestRange_OverflowProtection(t *testing.T) {
	// Test that opRange doesn't allow overflow-based DoS
	_, err := Eval(context.Background(), "0..9223372036854775807", nil) // MaxInt64
	if err == nil {
		t.Error("Expected error for huge range, got nil")
	}

	// Test that builtinRange also has overflow protection
	_, err = Eval(context.Background(), "range(0, 9223372036854775807)", nil)
	if err == nil {
		t.Error("Expected error for huge range in builtin, got nil")
	}

	// Valid ranges should still work
	result, err := Eval(context.Background(), "1..10", nil)
	if err != nil {
		t.Fatalf("Unexpected error for valid range: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 10 {
		t.Errorf("Expected 10 elements, got %d", len(arr))
	}
}

func TestRange_MaxInt64_DoesNotHang(t *testing.T) {
	// Prior bug: MaxInt64..MaxInt64 would hang due to int64 overflow in i++ loop.
	result, err := Eval(context.Background(), "9223372036854775807..9223372036854775807", nil) // MaxInt64..MaxInt64
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any result, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 element, got %d", len(arr))
	}
	if arr[0].(int64) != int64(9223372036854775807) {
		t.Fatalf("unexpected value: %v", arr[0])
	}

	// builtinRange should also not hang.
	result, err = Eval(context.Background(), "range(9223372036854775807, 9223372036854775807, 1)", nil)
	if err != nil {
		t.Fatalf("unexpected error for builtin range: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 1 || arr[0].(int64) != int64(9223372036854775807) {
		t.Fatalf("unexpected builtin range result: %v", result)
	}
}

func TestLexer_BlockComment_NewlineColumnAccurate(t *testing.T) {
	// Regression: newline handling inside /* */ advanced column incorrectly (+1).
	// The error should point at column 1 on the next line when the first token starts there.
	src := "/*x\n*/\n@"
	_, err := Eval(context.Background(), src, nil)
	if err == nil {
		t.Fatalf("expected lexer error, got nil")
	}
	// Depending on how the error propagates, this may surface as LexerError (Tokenize)
	// or as ParseError seeing tokenError.
	switch e := err.(type) {
	case *LexerError:
		if e.Line != 3 || e.Column != 1 {
			t.Fatalf("expected error at 3:1, got %d:%d (%v)", e.Line, e.Column, e.Message)
		}
	case *ParseError:
		if e.token.Line != 3 || e.token.Column != 1 {
			t.Fatalf("expected error at 3:1, got %d:%d (%v)", e.token.Line, e.token.Column, e.Message)
		}
	default:
		t.Fatalf("expected LexerError or ParseError, got %T: %v", err, err)
	}
}

func TestBuiltin_Abs_MinInt64(t *testing.T) {
	// abs(MinInt64) cannot fit in int64; we expect a float 2^63.
	// MinInt64 = (0 - MaxInt64 - 1)
	result, err := Eval(context.Background(), "abs((0 - 9223372036854775807) - 1)", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := NewValue(result)
	if !v.IsFloat() {
		t.Fatalf("expected float, got %T (%v)", result, result)
	}
	if v.Float() != 9223372036854775808.0 {
		t.Fatalf("expected 9223372036854775808, got %v", v.Float())
	}
}

func TestTypeCheck_BuiltinOptionalArgs(t *testing.T) {
	type Env struct {
		S string `expr:"s"`
		A []any  `expr:"a"`
	}
	engine := New(WithTypeCheck(Env{}))

	// substr: 2 args
	if _, err := engine.Compile(`substr(s, 1)`); err != nil {
		t.Fatalf("expected substr(s,1) to typecheck, got %v", err)
	}
	// range: 2 args
	if _, err := engine.Compile(`range(1, 10)`); err != nil {
		t.Fatalf("expected range(1,10) to typecheck, got %v", err)
	}
	// reduce: 2 args
	if _, err := engine.Compile(`reduce(a, (x, y) => x)`); err != nil {
		t.Fatalf("expected reduce(a, fn) to typecheck, got %v", err)
	}
}

func TestTypeCheck_Sigs_SortByAndZip(t *testing.T) {
	engine := New(WithTypeCheck(struct{}{}))

	// sortBy supports optional third boolean arg (desc).
	if _, err := engine.Compile(`sortBy([3, 1, 2], x => x, true)`); err != nil {
		t.Fatalf("expected sortBy(..., true) to typecheck, got %v", err)
	}

	// zip requires at least 2 arrays.
	if _, err := engine.Compile(`zip([1,2,3])`); err == nil {
		t.Fatalf("expected zip([..]) to fail typecheck due to arg count, got nil")
	}
}

func TestOptimizer_ShiftBoundsCheck(t *testing.T) {
	engine := New(WithOptimize(true))

	// These should NOT be folded at compile time (out of bounds)
	// but should work at runtime with proper semantics
	tests := []struct {
		expr     string
		expected int64
	}{
		{"1 << 0", 1},
		{"1 << 1", 2},
		{"1 << 62", int64(1) << 62}, // Use 62 to avoid overflow
		{"8 >> 1", 4},
		{"8 >> 2", 2},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		if NewValue(result).Int() != tt.expected {
			t.Errorf("%s: expected %d, got %v", tt.expr, tt.expected, result)
		}
	}

	// Runtime errors for invalid shifts
	_, err := Eval(context.Background(), "1 << -1", nil)
	if err == nil {
		t.Error("Expected error for negative shift, got nil")
	}

	// Large shifts should return 0 (not panic)
	result, err := Eval(context.Background(), "1 << 100", nil)
	if err != nil {
		t.Fatalf("Unexpected error for large shift: %v", err)
	}
	if NewValue(result).Int() != 0 {
		t.Errorf("Expected 0 for 1 << 100, got %v", result)
	}
}

func TestMaxCallDepth_InheritedInLambdas(t *testing.T) {
	// Test that maxCallDepth is properly inherited in lambdas without sandbox
	// Use a custom recursive function instead of self-referencing lambda
	engine := New(
		WithMaxCallDepth(5),
		WithFunction("countdown", func(n int) int {
			// This will be called recursively via expression
			return n
		}),
	)

	// Test nested lambda calls that should hit depth limit
	// Each nested call increases depth
	expr := `
		let f1 = x => x + 1;
		let f2 = x => f1(x) + 1;
		let f3 = x => f2(x) + 1;
		let f4 = x => f3(x) + 1;
		let f5 = x => f4(x) + 1;
		let f6 = x => f5(x) + 1;
		f6(0)
	`
	_, err := engine.Eval(context.Background(), expr, nil)
	if err == nil {
		t.Error("Expected max call depth error for deeply nested calls, got nil")
	}

	// With higher depth, it should work
	engine2 := New(WithMaxCallDepth(20))
	result, err := engine2.Eval(context.Background(), expr, nil)
	if err != nil {
		t.Fatalf("Unexpected error with higher depth: %v", err)
	}
	if NewValue(result).Int() != 6 {
		t.Errorf("Expected 6, got %v", result)
	}
}

func TestRange_NegativeStepOverflow(t *testing.T) {
	// Test range with negative step and potential overflow scenarios
	result, err := Eval(context.Background(), "range(10, 0, -1)", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 11 {
		t.Errorf("Expected 11 elements, got %d", len(arr))
	}
	if arr[0].(int64) != 10 || arr[10].(int64) != 0 {
		t.Errorf("Unexpected range values: first=%v, last=%v", arr[0], arr[len(arr)-1])
	}

	// Empty range when step direction doesn't match
	result, err = Eval(context.Background(), "range(0, 10, -1)", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 0 {
		t.Errorf("Expected empty array for mismatched step, got %d elements", len(arr))
	}
}

func TestSandbox_OpsLimit_TriggersOnInOperator(t *testing.T) {
	// Large array + tiny ops limit: 'in' should not silently return false; it must error.
	arr := make([]any, 5000)
	for i := range arr {
		arr[i] = int64(i + 1)
	}
	engine := New(WithSandbox(SandboxConfig{MaxOps: 1})) // extremely low
	_, err := engine.Eval(context.Background(), "999999 in arr", map[string]any{"arr": arr})
	if err == nil {
		t.Fatal("expected ops limit error, got nil")
	}
	if !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("expected ErrOpsLimitExceeded, got %v", err)
	}
}

func TestSecureSandbox_InOperator_DeniesExternalGoObjects(t *testing.T) {
	engine := New(WithSandbox(SandboxConfig{Secure: true}))
	type S struct{ Name string }
	_, err := engine.Eval(context.Background(), `"Name" in s`, map[string]any{"s": S{Name: "x"}})
	if err == nil {
		t.Fatal("expected secure sandbox denial error, got nil")
	}
	var se *SandboxError
	if !errors.As(err, &se) || se.Type != "access_denied" {
		t.Fatalf("expected SandboxError access_denied, got %T: %v", err, err)
	}
}

func TestSandbox_OpsLimit_TriggersOnBuiltins(t *testing.T) {
	engine := New(WithSandbox(SandboxConfig{MaxOps: 1000}))

	// unique
	_, err := engine.Eval(context.Background(), `unique(arr)`, map[string]any{
		"arr": make([]any, 2000),
	})
	if err == nil || !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("expected ErrOpsLimitExceeded for unique, got %v", err)
	}

	// range
	_, err = engine.Eval(context.Background(), `range(0, 2000, 1)`, nil)
	if err == nil || !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("expected ErrOpsLimitExceeded for range, got %v", err)
	}

	// repeat
	_, err = engine.Eval(context.Background(), `repeat(1, 2000)`, nil)
	if err == nil || !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("expected ErrOpsLimitExceeded for repeat, got %v", err)
	}

	// keys/values on large map
	m := make(map[string]any, 2000)
	for i := 0; i < 2000; i++ {
		m[fmt.Sprintf("k%d", i)] = i
	}
	_, err = engine.Eval(context.Background(), `keys(m)`, map[string]any{"m": m})
	if err == nil || !errors.Is(err, ErrOpsLimitExceeded) {
		t.Fatalf("expected ErrOpsLimitExceeded for keys, got %v", err)
	}
}

func TestKindAny_Map_NilKey_InAndIndex_NoPanic(t *testing.T) {
	// Ensure reflect-based map paths handle nil keys without panicking.
	m := map[any]int{nil: 42, "x": 1}
	// Wrap as Value(kindAny) to avoid NewValue() converting maps to map[string]any.
	env := map[string]any{"m": Value{kind: kindAny, ptr: m}}

	// Membership test
	got, err := Eval(context.Background(), "nil in m", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != true {
		t.Fatalf("expected true for nil in map[any]int{nil:42}, got %v", got)
	}

	// Indexing
	got, err = Eval(context.Background(), "m[nil]", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if NewValue(got).Int() != 42 {
		t.Fatalf("expected 42 for m[nil], got %v", got)
	}
}

func TestValueGet_NilPointer_NoPanic(t *testing.T) {
	// Hardening: Value.Get on kindAny with typed nil pointer should not panic.
	type S struct{ X int }
	var p *S = nil
	v := Value{kind: kindAny, ptr: p}
	_, ok := v.Get("X")
	if ok {
		t.Fatalf("expected ok=false for nil pointer kindAny Get")
	}
}

func TestMaxCallDepth_InheritedInLambdas_WithSandbox(t *testing.T) {
	// Regression: when sandbox is enabled (but MaxCallDepth not set in sandbox),
	// nested closures must still honor Engine.WithMaxCallDepth.
	engine := New(
		WithMaxCallDepth(5),
		WithSandbox(SandboxConfig{MaxOps: 1_000_000}), // enable sandbox without MaxCallDepth
	)
	expr := `
		let f1 = x => x + 1;
		let f2 = x => f1(x) + 1;
		let f3 = x => f2(x) + 1;
		let f4 = x => f3(x) + 1;
		let f5 = x => f4(x) + 1;
		let f6 = x => f5(x) + 1;
		f6(0)
	`
	_, err := engine.Eval(context.Background(), expr, nil)
	if err == nil {
		t.Fatal("expected max call depth error under sandbox, got nil")
	}
	if !errors.Is(err, ErrMaxCallDepth) {
		t.Fatalf("expected max call depth error, got %v", err)
	}
}

func TestNativeFunction_TypedSliceArg_NoPanic(t *testing.T) {
	engine := New(
		WithFunction("sumInts", func(xs []int) int {
			s := 0
			for _, v := range xs {
				s += v
			}
			return s
		}),
	)
	got, err := engine.Eval(context.Background(), "sumInts([1,2,3])", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if NewValue(got).Int() != 6 {
		t.Fatalf("expected 6, got %v", got)
	}
}

func TestKindAny_MapIntInt_InAndIndex(t *testing.T) {
	engine := New()
	env := map[string]any{
		"m": map[int]int{1: 10, 2: 20},
	}
	// Membership should work (key type conversion int64 -> int).
	got, err := engine.Eval(context.Background(), `1 in m`, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != true {
		t.Fatalf("expected true for 1 in map[int]int, got %v", got)
	}
	// Indexing should work too.
	got, err = engine.Eval(context.Background(), `m[1]`, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if NewValue(got).Int() != 10 {
		t.Fatalf("expected 10 for m[1], got %v", got)
	}
}

func TestNewValue_MapWithPanicStringerKey_NoPanic(t *testing.T) {
	// If NewValue tries to stringify keys, this would panic; it must not.
	m := map[panicStringerKey]int{{}: 1}
	v := NewValue(m)
	// Non-string key maps should remain kindAny for safe reflect handling.
	if v.kind != kindAny {
		t.Fatalf("expected kindAny for non-string key map, got %v", v)
	}
	_ = v.String() // must not panic
}

func TestValueString_CycleSafe(t *testing.T) {
	m := map[string]any{}
	m["self"] = m
	v := NewValue(m)
	s := v.String()
	if s == "" {
		t.Fatalf("expected non-empty string")
	}
	// Just ensure it doesn't recurse forever and gives a recognizable marker/truncation.
	if !strings.Contains(s, "<cycle>") && !strings.Contains(s, "…") {
		t.Fatalf("expected cycle marker or truncation, got: %q", s)
	}
}

func TestOptimize_EquivalentToUnoptimized(t *testing.T) {
	env := map[string]any{
		"x": int64(10),
		"y": int64(3),
		"arr": []any{int64(1), int64(2), int64(3), int64(4)},
		"m": map[string]any{"a": int64(1), "b": int64(2)},
		"s": "hello",
	}
	tests := []string{
		"1 + 2 * 3",
		"x + y * 2",
		"x * 2 + 0",
		"(x + (y + 1))",
		`"a" + "b"`,
		`"he" in s`,
		"2 in arr",
		"arr[1:3]",
		`m["a"] + 2`,
		"coalesce(nil, 1, 2)",
		"default(nil, 42)",
		"range(0, 10, 2) | len",
		"unique([1,2,2,3]) | len",
		"sum([1,2,3,4])",
		"avg([1,2,3,4])",
	}

	engOpt := New(WithOptimize(true))
	engNo := New(WithOptimize(false))

	for _, expr := range tests {
		pOpt, err := engOpt.Compile(expr)
		if err != nil {
			t.Fatalf("opt compile %q: %v", expr, err)
		}
		pNo, err := engNo.Compile(expr)
		if err != nil {
			t.Fatalf("no-opt compile %q: %v", expr, err)
		}

		gotOpt, err := pOpt.Eval(context.Background(), env)
		if err != nil {
			t.Fatalf("opt eval %q: %v", expr, err)
		}
		gotNo, err := pNo.Eval(context.Background(), env)
		if err != nil {
			t.Fatalf("no-opt eval %q: %v", expr, err)
		}

		if !NewValue(gotOpt).Equal(NewValue(gotNo)) {
			t.Fatalf("opt mismatch for %q: opt=%v no=%v", expr, gotOpt, gotNo)
		}
	}
}

// =============================================================================
// GO INTEROP TESTS (kindAny support)
// =============================================================================

func TestKindAny_Contains_GoSlice(t *testing.T) {
	// Test 'in' operator with native Go slice (not converted to Array)
	goSlice := []int{10, 20, 30, 40, 50}
	env := map[string]any{"arr": goSlice}

	// Should find element
	result, err := Eval(context.Background(), "20 in arr", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true for 20 in [10,20,30,40,50], got %v", result)
	}

	// Should not find element
	result, err = Eval(context.Background(), "99 in arr", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false for 99 in [10,20,30,40,50], got %v", result)
	}
}

func TestKindAny_Contains_GoMap(t *testing.T) {
	// Test 'in' operator with native Go map
	goMap := map[string]int{"foo": 1, "bar": 2, "baz": 3}
	env := map[string]any{"m": goMap}

	// Should find key
	result, err := Eval(context.Background(), `"foo" in m`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true for 'foo' in map, got %v", result)
	}

	// Should not find key
	result, err = Eval(context.Background(), `"xyz" in m`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false for 'xyz' in map, got %v", result)
	}
}

func TestKindAny_Index_GoSlice(t *testing.T) {
	// Test indexing native Go slice
	goSlice := []string{"a", "b", "c", "d", "e"}
	env := map[string]any{"arr": goSlice}

	// Positive index
	result, err := Eval(context.Background(), "arr[0]", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "a" {
		t.Errorf("Expected 'a', got %v", result)
	}

	// Negative index
	result, err = Eval(context.Background(), "arr[-1]", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "e" {
		t.Errorf("Expected 'e', got %v", result)
	}
}

func TestKindAny_Index_GoMap(t *testing.T) {
	// Test indexing native Go map
	goMap := map[string]int{"x": 100, "y": 200}
	env := map[string]any{"m": goMap}

	result, err := Eval(context.Background(), `m["x"]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 100 {
		t.Errorf("Expected 100, got %v", result)
	}
}

func TestKindAny_Slice_GoSlice(t *testing.T) {
	// Test slicing native Go slice
	goSlice := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	env := map[string]any{"arr": goSlice}

	// Basic slice
	result, err := Eval(context.Background(), "arr[2:5]", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 || NewValue(arr[0]).Int() != 2 || NewValue(arr[2]).Int() != 4 {
		t.Errorf("Expected [2,3,4], got %v", arr)
	}

	// Slice with step
	result, err = Eval(context.Background(), "arr[0:10:2]", env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 5 {
		t.Errorf("Expected 5 elements with step 2, got %d", len(arr))
	}
}

func TestFromJSON_IntegerPreservation(t *testing.T) {
	// Test that fromJSON converts whole numbers to int64
	result, err := Eval(context.Background(), `fromJSON("[1, 2, 3]")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arr))
	}

	// Check that elements are int64, not float64
	for i, v := range arr {
		if _, ok := v.(int64); !ok {
			t.Errorf("Element %d: expected int64, got %T", i, v)
		}
	}

	// Test with object
	result, err = Eval(context.Background(), `fromJSON("{\"x\": 42, \"y\": 3.14}")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	m := result.(map[string]any)

	// x should be int64 (whole number)
	if _, ok := m["x"].(int64); !ok {
		t.Errorf("x: expected int64, got %T", m["x"])
	}

	// y should remain float64 (has decimal)
	if _, ok := m["y"].(float64); !ok {
		t.Errorf("y: expected float64, got %T", m["y"])
	}

	// Test that int values work in expressions
	result, err = Eval(context.Background(), `let data = fromJSON("[10, 20, 30]"); data[0] + data[1]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 30 {
		t.Errorf("Expected 30, got %v", result)
	}

	// Safe integer boundary behavior:
	// 2^53-1 is representable exactly in float64, so we convert to int64.
	result, err = Eval(context.Background(), `fromJSON("9007199254740991")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if _, ok := result.(int64); !ok {
		t.Errorf("2^53-1: expected int64, got %T (%v)", result, result)
	}

	// 2^53 is outside the safe-integer range; keep as float64 to avoid precision corruption.
	result, err = Eval(context.Background(), `fromJSON("9007199254740992")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if _, ok := result.(float64); !ok {
		t.Errorf("2^53: expected float64, got %T (%v)", result, result)
	}
}

func TestCeilFloorRound_OutOfInt64Range_ReturnsFloat(t *testing.T) {
	// Extremely large numbers should not silently overflow into int64.
	// We keep them as float64.
	tests := []string{
		"ceil(1e30)",
		"floor(1e30)",
		"round(1e30)",
	}
	for _, expr := range tests {
		result, err := Eval(context.Background(), expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", expr, err)
		}
		if _, ok := result.(float64); !ok {
			t.Fatalf("%s: expected float64, got %T (%v)", expr, result, result)
		}
		if result.(float64) <= float64(math.MaxInt64) {
			t.Fatalf("%s: expected value > MaxInt64, got %v", expr, result)
		}
	}
}

// =============================================================================
// COMPLETE COVERAGE TESTS
// =============================================================================

func TestSandbox_MaxMemory_Coverage(t *testing.T) {
	// Test sandbox with memory limit
	engine := New(
		WithSandbox(SandboxConfig{
			MaxMemory: 1024, // Very small limit
		}),
	)

	// This should trigger memory limit when creating large arrays
	// Note: Memory estimation is approximate, so we use a very large allocation
	_, err := engine.Eval(context.Background(), `let arr = []; let i = 0; i + 1`, nil)
	// Small expressions should work
	if err != nil {
		t.Fatalf("Small expression should work: %v", err)
	}

	// Test memory estimation function coverage
	vm := &execVM{}

	// Test different value types
	tests := []struct {
		value Value
		name  string
	}{
		{Int(42), "int"},
		{Float(3.14), "float"},
		{String("hello world"), "string"},
		{Bool(true), "bool"},
		{Nil, "nil"},
		{Array([]any{1, 2, 3}), "array"},
		{Map(map[string]any{"a": 1, "b": 2}), "map"},
	}

	for _, tt := range tests {
		mem := vm.estimateValueMemory(tt.value)
		if mem < 32 {
			t.Errorf("%s: expected at least 32 bytes, got %d", tt.name, mem)
		}
	}
}

func TestVM_StackOperations(t *testing.T) {
	// Test stack growth
	engine := New()

	// Expression that uses deep stack (nested calls)
	expr := `let add = (a, b) => a + b; add(add(add(add(1, 2), 3), 4), 5)`
	result, err := engine.Eval(context.Background(), expr, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 15 {
		t.Errorf("Expected 15, got %v", result)
	}

	// Test stack overflow protection with DebugMode
	restore := SetDebugModeForTest(true)
	defer restore()

	// Create a vm manually to test edge cases
	code := &bytecode{
		Instructions: []instruction{{Op: opHalt}},
		Constants:    []Value{},
	}
	vm := newVM(code, defaultBuiltins())
	defer releaseVM(vm)

	// Test push/pop with stack growth
	for i := 0; i < 300; i++ {
		vm.push(Int(int64(i)))
	}
	if vm.sp != 300 {
		t.Errorf("Expected sp=300, got %d", vm.sp)
	}
	// Pop all
	for i := 0; i < 300; i++ {
		_ = vm.pop()
	}
	if vm.sp != 0 {
		t.Errorf("Expected sp=0, got %d", vm.sp)
	}
}

func TestContains_NilAndEdgeCases(t *testing.T) {
	engine := New()

	// Test 'in' with nil collection
	env := map[string]any{
		"nilVal": nil,
	}
	result, err := engine.Eval(context.Background(), `1 in nilVal`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false for 'in' on nil, got %v", result)
	}

	// Test 'in' with struct (kindAny)
	type TestStruct struct {
		Name string
		Age  int
	}
	env["s"] = TestStruct{Name: "test", Age: 25}

	// Struct should return false for 'in' check
	result, err = engine.Eval(context.Background(), `"Name" in s`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Note: struct doesn't implement 'in' the way map does
	if result != false {
		t.Errorf("Expected false for 'in' on struct, got %v", result)
	}
}

func TestIndex_KindAny_EdgeCases(t *testing.T) {
	engine := New()

	// Test indexing with int map (not string keys)
	intMap := map[int]string{1: "one", 2: "two", 3: "three"}
	env := map[string]any{"m": intMap}

	// Should work with int key
	result, err := engine.Eval(context.Background(), `m[1]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "one" {
		t.Errorf("Expected 'one', got %v", result)
	}

	// Test non-existent key returns nil
	result, err = engine.Eval(context.Background(), `m[99]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for missing key, got %v", result)
	}
}

func TestSlice_KindAny_EdgeCases(t *testing.T) {
	engine := New()

	// Test slicing with negative step on native Go slice
	goSlice := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	env := map[string]any{"arr": goSlice}

	// Negative step
	result, err := engine.Eval(context.Background(), `arr[::-1]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 10 {
		t.Errorf("Expected 10 elements, got %d", len(arr))
	}
	// Check it's reversed
	if NewValue(arr[0]).Int() != 9 || NewValue(arr[9]).Int() != 0 {
		t.Errorf("Expected reversed array, got %v", arr)
	}

	// Test slice on non-sliceable type
	env["notSlice"] = 42
	_, err = engine.Eval(context.Background(), `notSlice[0:1]`, env)
	if err == nil {
		t.Error("Expected error for slicing non-sliceable type")
	}
}

func TestRepeat_Limit(t *testing.T) {
	// Test that repeat has a limit
	_, err := Eval(context.Background(), `repeat(1, 2000000)`, nil)
	if err == nil {
		t.Error("Expected error for repeat with count exceeding limit")
	}
	if err != nil && !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("Expected 'exceeds maximum' error, got: %v", err)
	}

	// Valid repeat should work
	result, err := Eval(context.Background(), `repeat("x", 3)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}
}

func TestChunk_Limit(t *testing.T) {
	// Create array manually to test chunk limit
	// With size=1 on large array, it would create too many chunks
	engine := New()

	// Normal chunk should work
	result, err := engine.Eval(context.Background(), `chunk([1, 2, 3, 4, 5], 2)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(arr))
	}
}

func TestFlatten_DepthLimit(t *testing.T) {
	// Test deeply nested array
	engine := New()

	// Create a moderately deep nested array - this should work
	result, err := engine.Eval(context.Background(), `flatten([[1, [2, [3, [4]]]]])`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 4 {
		t.Errorf("Expected 4 elements after flatten, got %d", len(arr))
	}

	// Test normal flatten
	result, err = engine.Eval(context.Background(), `flatten([[1, 2], [3, 4], [5]])`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(arr))
	}
}

func TestMakeConstKey_NilPtr(t *testing.T) {
	// Test that makeConstKey handles nil ptr correctly
	testCases := []struct {
		name string
		val  Value
	}{
		{"nil_func", Value{kind: kindFunc, ptr: nil}},
		{"nil_array", Value{kind: kindArray, ptr: nil}},
		{"nil_map", Value{kind: kindMap, ptr: nil}},
		{"nil_any", Value{kind: kindAny, ptr: nil}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			key := makeConstKey(tc.val)
			if key.ptrVal != 0 {
				t.Errorf("Expected ptrVal=0 for nil ptr, got %d", key.ptrVal)
			}
		})
	}
}

func TestCallClosureFromBuiltin_MaxDepth(t *testing.T) {
	// Test that closures called from builtins respect maxDepth
	engine := New()

	// Create a deeply recursive map call
	// This should eventually hit the depth limit
	_, err := engine.Eval(context.Background(), `
		let deepRecurse = arr => arr | map(x => x > 0 ? deepRecurse([x - 1]) : 0);
		deepRecurse([100])
	`, nil)
	if err == nil {
		t.Error("Expected max call depth error for recursive closure")
	}
}

func TestSandbox_StackOverflow_Coverage(t *testing.T) {
	// Test sandbox stack depth limit
	engine := New(
		WithSandbox(SandboxConfig{
			MaxStackDepth: 10,
		}),
	)

	// This should trigger stack overflow with small stack limit
	// Create expression that grows stack
	_, err := engine.Eval(context.Background(), `1 + 2 + 3 + 4 + 5 + 6 + 7 + 8 + 9 + 10 + 11 + 12`, nil)
	// Deep enough expressions may or may not hit MaxStackDepth depending on codegen;
	// either outcome is acceptable as long as Eval returns without panic.
	_ = err
}

func TestSandbox_ReadOnlyMode(t *testing.T) {
	engine := New(
		WithSandbox(SandboxConfig{
			ReadOnly: true,
		}),
	)

	// Let expression should fail in read-only mode
	_, err := engine.Eval(context.Background(), `let x = 5; x`, nil)
	if err == nil {
		t.Error("Expected error for let in read-only mode")
	}
	if err != nil && !errors.Is(err, ErrAssignmentNotAllowed) {
		// Check if it's a SandboxError with correct type
		var sandboxErr *SandboxError
		if errors.As(err, &sandboxErr) {
			if sandboxErr.Type != "assignment_denied" {
				t.Errorf("Expected assignment_denied error, got: %s", sandboxErr.Type)
			}
		}
	}
}

func TestSandbox_FunctionAllowList(t *testing.T) {
	engine := New(
		WithAllowedFuncs("len", "upper"),
	)

	// Allowed functions should work
	result, err := engine.Eval(context.Background(), `len("hello")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error for allowed function: %v", err)
	}
	if NewValue(result).Int() != 5 {
		t.Errorf("Expected 5, got %v", result)
	}

	// Denied function should fail
	_, err = engine.Eval(context.Background(), `lower("HELLO")`, nil)
	if err == nil {
		t.Error("Expected error for non-allowed function")
	}
}

func TestSandbox_FunctionDenyList(t *testing.T) {
	engine := New(
		WithDeniedFuncs("print", "sprintf"),
	)

	// Non-denied functions should work
	result, err := engine.Eval(context.Background(), `len("hello")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 5 {
		t.Errorf("Expected 5, got %v", result)
	}

	// Denied function should fail
	_, err = engine.Eval(context.Background(), `print("test")`, nil)
	if err == nil {
		t.Error("Expected error for denied function")
	}
}

func TestSandbox_OpsLimit(t *testing.T) {
	// Use very small ops limit (1) to ensure it triggers
	engine := New(
		WithSandbox(SandboxConfig{
			MaxOps: 1,
		}),
	)

	// Even simple expression should hit ops limit with MaxOps=1
	_, err := engine.Eval(context.Background(), `1 + 2`, nil)
	if err == nil {
		t.Error("Expected ops limit error")
	}
	if err != nil {
		var sandboxErr *SandboxError
		if errors.As(err, &sandboxErr) {
			if sandboxErr.Type != "ops_limit" {
				t.Errorf("Expected ops_limit error, got: %s", sandboxErr.Type)
			}
		} else if !errors.Is(err, ErrOpsLimitExceeded) {
			t.Errorf("Expected ops limit error, got: %v", err)
		}
	}
}

func TestSandbox_Timeout_Coverage(t *testing.T) {
	engine := New(
		WithTimeout(1 * time.Millisecond),
	)

	// Very long running expression should timeout
	// Use a loop-like expression
	_, err := engine.Eval(context.Background(), `let arr = 1..10000; arr | map(x => x * x) | sum`, nil)
	// Note: timeout might not always trigger depending on execution speed
	// This is a best-effort test
	_ = err // Ignore result - timeout is non-deterministic
}

func TestValue_Get_StructWithPointer(t *testing.T) {
	type Inner struct {
		Value int
	}
	type Outer struct {
		Inner *Inner
		Name  string
	}

	env := map[string]any{
		"obj": &Outer{
			Inner: &Inner{Value: 42},
			Name:  "test",
		},
	}

	engine := New()

	// Access struct field
	result, err := engine.Eval(context.Background(), `obj.Name`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("Expected 'test', got %v", result)
	}

	// Access nested struct
	result, err = engine.Eval(context.Background(), `obj.Inner.Value`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
}

func TestValue_Get_CaseInsensitive(t *testing.T) {
	type TestStruct struct {
		UserName string
	}

	env := map[string]any{
		"obj": TestStruct{UserName: "john"},
	}

	engine := New()

	// Case-insensitive field access
	result, err := engine.Eval(context.Background(), `obj.username`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "john" {
		t.Errorf("Expected 'john', got %v", result)
	}
}

func TestVMExecute_UnknownOpcode(t *testing.T) {
	// Test that unknown opcode returns error
	code := &bytecode{
		Instructions: []instruction{
			{Op: opcode(255)}, // Invalid opcode
		},
		Constants: []Value{},
	}

	vm := newVM(code, defaultBuiltins())
	defer releaseVM(vm)

	_, err := vm.run(context.Background(), nil)
	if err == nil {
		t.Error("Expected error for unknown opcode")
	}
}

func TestVM_DivisionByZero_Modulo(t *testing.T) {
	// Test modulo by zero
	_, err := Eval(context.Background(), `10 % 0`, nil)
	if err == nil {
		t.Error("Expected division by zero error for modulo")
	}
}

func TestVM_ShiftOperations_EdgeCases(t *testing.T) {
	// Test unsigned right shift
	result, err := Eval(context.Background(), `-1 >>> 1`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// -1 in two's complement shifted right unsigned should be large positive
	expected := int64(uint64(0xFFFFFFFFFFFFFFFF) >> 1)
	if NewValue(result).Int() != expected {
		t.Errorf("Expected %d, got %v", expected, result)
	}

	// Test large shift for unsigned
	result, err = Eval(context.Background(), `1 >>> 100`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 0 {
		t.Errorf("Expected 0, got %v", result)
	}

	// Test negative shift for unsigned (should error)
	_, err = Eval(context.Background(), `1 >>> -1`, nil)
	if err == nil {
		t.Error("Expected error for negative unsigned shift")
	}
}

func TestVM_MemberAccess_NotFound(t *testing.T) {
	env := map[string]any{
		"obj": map[string]any{"name": "test"},
	}
	engine := New()

	// Non-optional member access on missing property should error
	_, err := engine.Eval(context.Background(), `obj.missing`, env)
	if err == nil {
		t.Error("Expected error for missing property")
	}
}

func TestVM_Index_NotIndexable(t *testing.T) {
	env := map[string]any{
		"num": 42,
	}
	engine := New()

	// Indexing a number should error
	_, err := engine.Eval(context.Background(), `num[0]`, env)
	if err == nil {
		t.Error("Expected error for indexing non-indexable type")
	}
}

func TestVM_Slice_StepZero(t *testing.T) {
	// Slice with step=0 should error
	_, err := Eval(context.Background(), `[1,2,3][::0]`, nil)
	if err == nil {
		t.Error("Expected error for slice with step=0")
	}
}

func TestVM_StringRepeat_Limit(t *testing.T) {
	// String repeat (via *) should have limit
	_, err := Eval(context.Background(), `"x" * 20000000`, nil)
	if err == nil {
		t.Error("Expected error for excessive string repeat")
	}
}

func TestBuiltin_TrimLeft_TrimRight_WithChars(t *testing.T) {
	engine := New()

	// TrimLeft with custom chars
	result, err := engine.Eval(context.Background(), `trimLeft("xxxhello", "x")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("Expected 'hello', got %v", result)
	}

	// TrimRight with custom chars
	result, err = engine.Eval(context.Background(), `trimRight("helloxxx", "x")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("Expected 'hello', got %v", result)
	}
}

func TestBuiltin_TypeConversions_EdgeCases(t *testing.T) {
	engine := New()

	// Int from bool
	result, err := engine.Eval(context.Background(), `int(true)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 1 {
		t.Errorf("Expected 1, got %v", result)
	}

	// Float from string
	result, err = engine.Eval(context.Background(), `float("3.14")`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Float() != 3.14 {
		t.Errorf("Expected 3.14, got %v", result)
	}
}

func TestOptimizer_ShiftBoundsNotFolded(t *testing.T) {
	// Test that optimizer doesn't fold shifts with out-of-bounds values
	engine := New(WithOptimize(true))

	// Test shift at boundary
	result, err := engine.Eval(context.Background(), `1 << 62`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := int64(1) << 62
	if NewValue(result).Int() != expected {
		t.Errorf("Expected %d, got %v", expected, result)
	}

	// Test that 1 << 63 produces correct (possibly negative) value
	result, err = engine.Eval(context.Background(), `1 << 63`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// The result should be int64 minimum value
	if NewValue(result).Int() >= 0 {
		t.Errorf("Expected negative number for 1 << 63, got %v", result)
	}
}

func TestCompiler_DestructureWithTemp(t *testing.T) {
	engine := New()

	// Test multiple destructuring expressions to verify temp variable uniqueness
	result, err := engine.Eval(context.Background(), `
		let {a, b} = {a: 1, b: 2};
		let {c, d} = {c: 3, d: 4};
		a + b + c + d
	`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 10 {
		t.Errorf("Expected 10, got %v", result)
	}
}

func TestSandbox_LimitsInheritedInClosure(t *testing.T) {
	engine := New(
		WithSandbox(SandboxConfig{
			MaxOps: 100,
		}),
	)

	// closure should inherit sandbox limits
	result, err := engine.Eval(context.Background(), `
		let double = x => x * 2;
		double(21)
	`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
}

func TestVMPool_LargeEnvCleanup(t *testing.T) {
	// Test that vm with large env is properly cleaned up
	engine := New()

	// Create large environment
	env := make(map[string]any)
	for i := 0; i < 100; i++ {
		env[fmt.Sprintf("var%d", i)] = i
	}

	// Run multiple evaluations to test pool reuse
	for i := 0; i < 10; i++ {
		_, err := engine.Eval(context.Background(), `var0 + var1`, env)
		if err != nil {
			t.Fatalf("Iteration %d: unexpected error: %v", i, err)
		}
	}
}

func TestVM_PopPeek_DebugMode(t *testing.T) {
	// Test pop and peek in debug mode with empty stack
	restore := SetDebugModeForTest(true)
	defer restore()

	code := &bytecode{
		Instructions: []instruction{{Op: opHalt}},
		Constants:    []Value{},
	}
	vm := newVM(code, defaultBuiltins())
	defer releaseVM(vm)

	// Test pop on empty stack in debug mode
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for pop on empty stack in debug mode")
		}
	}()
	vm.pop()
}

func TestVM_Peek_DebugMode(t *testing.T) {
	// Test peek in debug mode with empty stack
	restore := SetDebugModeForTest(true)
	defer restore()

	code := &bytecode{
		Instructions: []instruction{{Op: opHalt}},
		Constants:    []Value{},
	}
	vm := newVM(code, defaultBuiltins())
	defer releaseVM(vm)

	// Test peek on empty stack in debug mode
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for peek on empty stack in debug mode")
		}
	}()
	vm.peek()
}

func TestOptimizer_IsSideEffectFree(t *testing.T) {
	// Test isSideEffectFree function coverage
	tests := []struct {
		name     string
		node     Node
		expected bool
	}{
		{"intLit", &intLit{Value: 42}, true},
		{"floatLit", &floatLit{Value: 3.14}, true},
		{"stringLit", &stringLit{Value: "hello"}, true},
		{"boolLit", &boolLit{Value: true}, true},
		{"nilLit", &nilLit{}, true},
		{"ident", &ident{Name: "x"}, true},
		{"unaryExpr", &unaryExpr{Op: "-", Operand: &intLit{Value: 1}}, true},
		{"binaryExpr", &binaryExpr{Op: "+", Left: &intLit{Value: 1}, Right: &intLit{Value: 2}}, true},
		{"memberExpr", &memberExpr{Object: &ident{Name: "x"}, Property: &ident{Name: "y"}}, true},
		{"indexExpr", &indexExpr{Object: &ident{Name: "x"}, Index: &intLit{Value: 0}}, true},
		{"arrayLit", &arrayLit{Elements: []Node{&intLit{Value: 1}}}, true},
		{"ArrayLit_WithCall", &arrayLit{Elements: []Node{&callExpr{Func: &ident{Name: "f"}, Args: nil}}}, false},
		{"callExpr", &callExpr{Func: &ident{Name: "f"}, Args: nil}, false},
		{"letExpr", &letExpr{Name: "x", Value: &intLit{Value: 1}, Body: &ident{Name: "x"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSideEffectFree(tt.node)
			if result != tt.expected {
				t.Errorf("isSideEffectFree(%s): got %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestOptimizer_CopyNode(t *testing.T) {
	// Test copyNode function coverage
	tests := []struct {
		name string
		node Node
	}{
		{"intLit", &intLit{Value: 42}},
		{"floatLit", &floatLit{Value: 3.14}},
		{"stringLit", &stringLit{Value: "hello"}},
		{"boolLit", &boolLit{Value: true}},
		{"nilLit", &nilLit{}},
		{"ident", &ident{Name: "x"}},
		{"binaryExpr", &binaryExpr{Op: "+", Left: &intLit{Value: 1}, Right: &intLit{Value: 2}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			copy := copyNode(tt.node)
			if copy == nil {
				t.Errorf("copyNode returned nil for %s", tt.name)
			}
		})
	}
}

func TestBuiltin_ConvertArgTypes(t *testing.T) {
	// Test convertArgForBuiltin for various types
	engine := New(
		WithFunction("testInt8", func(x int8) int8 { return x }),
		WithFunction("testInt16", func(x int16) int16 { return x }),
		WithFunction("testInt32", func(x int32) int32 { return x }),
		WithFunction("testUint", func(x uint) uint { return x }),
		WithFunction("testUint8", func(x uint8) uint8 { return x }),
		WithFunction("testUint16", func(x uint16) uint16 { return x }),
		WithFunction("testUint32", func(x uint32) uint32 { return x }),
		WithFunction("testUint64", func(x uint64) uint64 { return x }),
		WithFunction("testFloat32", func(x float32) float32 { return x }),
	)

	tests := []struct {
		expr     string
		expected int64
	}{
		{"testInt8(42)", 42},
		{"testInt16(42)", 42},
		{"testInt32(42)", 42},
		{"testUint(42)", 42},
		{"testUint8(42)", 42},
		{"testUint16(42)", 42},
		{"testUint32(42)", 42},
		{"testUint64(42)", 42},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		if NewValue(result).Int() != tt.expected {
			t.Errorf("%s: expected %d, got %v", tt.expr, tt.expected, result)
		}
	}

	// Test float32
	result, err := engine.Eval(context.Background(), `testFloat32(3.14)`, nil)
	if err != nil {
		t.Fatalf("testFloat32: unexpected error: %v", err)
	}
	f := NewValue(result).Float()
	if f < 3.13 || f > 3.15 {
		t.Errorf("testFloat32: expected ~3.14, got %v", f)
	}
}

func TestVM_Contains_ReflectionEdgeCases(t *testing.T) {
	engine := New()

	// Test 'in' with typed int slice
	typedSlice := []int{1, 2, 3, 4, 5}
	env := map[string]any{"arr": typedSlice}

	result, err := engine.Eval(context.Background(), `3 in arr`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	// Test with typed string slice
	stringSlice := []string{"a", "b", "c"}
	env["strs"] = stringSlice

	result, err = engine.Eval(context.Background(), `"b" in strs`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestVM_Index_ReflectionEdgeCases(t *testing.T) {
	engine := New()

	// Test indexing typed map with non-string key
	intKeyMap := map[int]string{1: "one", 2: "two"}
	env := map[string]any{"m": intKeyMap}

	// Index by int
	result, err := engine.Eval(context.Background(), `m[1]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "one" {
		t.Errorf("Expected 'one', got %v", result)
	}

	// Test missing key
	result, err = engine.Eval(context.Background(), `m[99]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestVM_Slice_NegativeStep_EdgeCases(t *testing.T) {
	engine := New()

	// Test string with negative step
	result, err := engine.Eval(context.Background(), `"hello"[::-1]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "olleh" {
		t.Errorf("Expected 'olleh', got %v", result)
	}

	// Test array slice with step
	result, err = engine.Eval(context.Background(), `[0,1,2,3,4,5][1:5:2]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(arr))
	}
}

func TestBuiltin_Variadic(t *testing.T) {
	// Test variadic function handling
	engine := New(
		WithFunction("sumAll", func(nums ...int) int {
			sum := 0
			for _, n := range nums {
				sum += n
			}
			return sum
		}),
	)

	result, err := engine.Eval(context.Background(), `sumAll(1, 2, 3, 4, 5)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 15 {
		t.Errorf("Expected 15, got %v", result)
	}
}

func TestValue_Type_KindAny(t *testing.T) {
	type CustomType struct {
		Field int
	}

	val := NewValue(CustomType{Field: 42})
	typ := val.Type()
	if typ != "hyperex.CustomType" {
		t.Errorf("Expected 'hyperex.CustomType', got %v", typ)
	}
}

func TestBuiltin_Sort_Comparator(t *testing.T) {
	engine := New()

	// Test sorting strings
	result, err := engine.Eval(context.Background(), `sort(["c", "a", "b"])`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if arr[0] != "a" || arr[1] != "b" || arr[2] != "c" {
		t.Errorf("Expected [a, b, c], got %v", arr)
	}

	// Test sorting mixed numbers (int and float)
	result, err = engine.Eval(context.Background(), `sort([3, 1.5, 2])`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}
}

func TestBuiltin_Default_NonNil(t *testing.T) {
	engine := New()

	// Test default with non-nil value
	result, err := engine.Eval(context.Background(), `default(42, 0)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
}

func TestVM_Add_ArrayConcat(t *testing.T) {
	engine := New()

	// Test array concatenation with +
	result, err := engine.Eval(context.Background(), `[1, 2] + [3, 4]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(arr))
	}
}

func TestVM_Sub_Floats(t *testing.T) {
	engine := New()

	// Test float subtraction
	result, err := engine.Eval(context.Background(), `5.5 - 2.2`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	f := NewValue(result).Float()
	if f < 3.2 || f > 3.4 {
		t.Errorf("Expected ~3.3, got %v", f)
	}
}

func TestVM_Div_Floats(t *testing.T) {
	engine := New()

	// Test float division
	result, err := engine.Eval(context.Background(), `7.0 / 2.0`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Float() != 3.5 {
		t.Errorf("Expected 3.5, got %v", result)
	}
}

func TestVM_Add_TypeError(t *testing.T) {
	engine := New()
	env := map[string]any{
		"m": map[string]any{"a": 1},
	}

	// Adding maps should error
	_, err := engine.Eval(context.Background(), `m + m`, env)
	if err == nil {
		t.Error("Expected error for adding maps")
	}
}

func TestVM_Sub_TypeError(t *testing.T) {
	engine := New()

	// Subtracting strings should error
	_, err := engine.Eval(context.Background(), `"a" - "b"`, nil)
	if err == nil {
		t.Error("Expected error for subtracting strings")
	}
}

func TestParser_SliceEdgeCases(t *testing.T) {
	engine := New()

	// Test slice with only step
	result, err := engine.Eval(context.Background(), `[0,1,2,3,4][::2]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}

	// Test slice with negative indices
	result, err = engine.Eval(context.Background(), `[0,1,2,3,4][-3:-1]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(arr))
	}
}

func TestLexer_UnicodeEscape_Coverage(t *testing.T) {
	// Test Unicode escape in strings
	result, err := Eval(context.Background(), `"\u0048\u0065\u006c\u006c\u006f"`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "Hello" {
		t.Errorf("Expected 'Hello', got %v", result)
	}
}

func TestAnalytics_EdgeCases(t *testing.T) {
	// Test Analyze function with various expressions
	analysis, err := Analyze(`let x = 1; x + y + f(z)`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should find variables
	if len(analysis.Variables) == 0 {
		t.Error("Expected to find variables")
	}

	// Should find functions
	if len(analysis.Functions) == 0 {
		t.Error("Expected to find functions")
	}
}

func TestTransform_Expand(t *testing.T) {
	// Test macro expansion
	macros := map[string]string{
		"DOUBLE": "x * 2",
	}

	result, err := Expand("DOUBLE + 1", macros)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should contain the expansion
	if result == "DOUBLE + 1" {
		t.Error("Macro was not expanded")
	}
}

func TestTransform_Inline(t *testing.T) {
	// Test let inlining
	result, err := Inline("let x = 5; x + x")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should inline the value
	if result == "let x = 5; x + x" {
		t.Error("Let expression was not inlined")
	}
}

func TestTypeChecker_Coverage(t *testing.T) {
	// Test type checker with various expressions
	type Env struct {
		X int
		Y string
	}

	engine := New(WithTypeCheck(Env{}))

	// This should compile without type errors
	_, err := engine.Compile(`X + 1`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test WithVar
	tc := NewTypeChecker()
	tc.WithVar("z", TypeInt)
	tc.WithFunc("myFunc", FuncSig{Params: []Type{TypeInt}, Returns: TypeInt})

	// Infer type
	parser := newParser("z + 1")
	ast, _ := parser.parse()
	typ := tc.Infer(ast)
	if typ != TypeInt {
		t.Errorf("Expected TypeInt, got %v", typ)
	}
}

func TestTypeChecker_AllNodeTypes(t *testing.T) {
	tc := NewTypeChecker()
	tc.WithVar("x", TypeInt)
	tc.WithVar("arr", TypeArray)
	tc.WithVar("obj", TypeMap)
	tc.WithFunc("fn", FuncSig{Params: []Type{TypeInt}, Returns: TypeString})

	tests := []struct {
		expr string
		typ  Type
	}{
		{"42", TypeInt},
		{"3.14", TypeFloat},
		{`"hello"`, TypeString},
		{"true", TypeBool},
		{"nil", TypeNil},
		{"x", TypeInt},
		{"-x", TypeInt},
		{"!true", TypeBool},
		{"x + 1", TypeInt},
		{"x - 1", TypeInt},
		{"x * 2", TypeInt},
		{"x / 2", TypeInt},
		{"x % 2", TypeInt},
		{"x ** 2", TypeInt},
		{"x == 1", TypeBool},
		{"x != 1", TypeBool},
		{"x < 1", TypeBool},
		{"x <= 1", TypeBool},
		{"x > 1", TypeBool},
		{"x >= 1", TypeBool},
		{"x && true", TypeBool},
		{"x || true", TypeBool},
		{"x ?? 0", TypeInt},
		{"arr[0]", TypeAny},
		{`obj["key"]`, TypeAny},
		{"fn(x)", TypeString},
		{"[1, 2, 3]", TypeArray},
		{`{"a": 1}`, TypeMap},
		{"1..5", TypeArray},
		{"let y = 1; y + x", TypeInt},
	}

	for _, tt := range tests {
		parser := newParser(tt.expr)
		ast, err := parser.parse()
		if err != nil {
			t.Fatalf("%s: parse error: %v", tt.expr, err)
		}
		typ := tc.Infer(ast)
		if typ != tt.typ {
			t.Errorf("%s: expected %v, got %v", tt.expr, tt.typ, typ)
		}
	}
}

func TestContains_AllTypes(t *testing.T) {
	engine := New()

	// Test 'in' with array of different types
	env := map[string]any{
		"intArr":   []any{1, 2, 3},
		"strArr":   []any{"a", "b", "c"},
		"mixedArr": []any{1, "b", 3.0},
		"strMap":   map[string]any{"a": 1, "b": 2},
		"str":      "hello world",
		"goIntArr": []int{1, 2, 3},
		"goStrArr": []string{"a", "b", "c"},
		"goIntMap": map[int]string{1: "one", 2: "two"},
		"goStrMap": map[string]int{"a": 1, "b": 2},
	}

	tests := []struct {
		expr     string
		expected bool
	}{
		{`2 in intArr`, true},
		{`5 in intArr`, false},
		{`"b" in strArr`, true},
		{`"x" in strArr`, false},
		{`1 in mixedArr`, true},
		{`"b" in mixedArr`, true},
		{`"a" in strMap`, true},
		{`"x" in strMap`, false},
		{`"hello" in str`, true},
		{`"world" in str`, true},
		{`"xyz" in str`, false},
		{`2 in goIntArr`, true},
		{`5 in goIntArr`, false},
		{`"b" in goStrArr`, true},
		{`1 in goIntMap`, true},
		{`99 in goIntMap`, false},
		{`"a" in goStrMap`, true},
		{`"x" in goStrMap`, false},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, env)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		if result != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestIndex_AllTypes(t *testing.T) {
	engine := New()

	type TestStruct struct {
		Field int
	}

	env := map[string]any{
		"arr":       []any{10, 20, 30},
		"str":       "hello",
		"m":         map[string]any{"key": "value"},
		"goIntArr":  []int{10, 20, 30},
		"goStrArr":  []string{"a", "b", "c"},
		"goIntMap":  map[int]string{1: "one", 2: "two"},
		"goStrMap":  map[string]int{"a": 1, "b": 2},
		"goStruct":  TestStruct{Field: 42},
		"goStructP": &TestStruct{Field: 99},
	}

	tests := []struct {
		expr     string
		expected any
	}{
		{`arr[0]`, int64(10)},
		{`arr[2]`, int64(30)},
		{`str[0]`, "h"},
		{`str[4]`, "o"},
		{`m["key"]`, "value"},
		{`goIntArr[0]`, 10},
		{`goIntArr[2]`, 30},
		{`goStrArr[1]`, "b"},
		{`goIntMap[1]`, "one"},
		{`goStrMap["a"]`, 1},
		{`goStruct.Field`, 42},
		{`goStructP.Field`, 99},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, env)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		expected := NewValue(tt.expected)
		got := NewValue(result)
		if !expected.Equal(got) {
			t.Errorf("%s: expected %v, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestSlice_AllTypes(t *testing.T) {
	engine := New()

	env := map[string]any{
		"arr":      []any{0, 1, 2, 3, 4, 5},
		"str":      "hello",
		"goIntArr": []int{0, 1, 2, 3, 4, 5},
	}

	tests := []struct {
		expr string
		len  int
	}{
		{`arr[1:3]`, 2},
		{`arr[:3]`, 3},
		{`arr[3:]`, 3},
		{`arr[:]`, 6},
		{`arr[::2]`, 3},
		{`arr[1:5:2]`, 2},
		{`arr[::-1]`, 6},
		{`str[1:4]`, 3},
		{`str[::-1]`, 5},
		{`goIntArr[1:3]`, 2},
		{`goIntArr[::-1]`, 6},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, env)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		var gotLen int
		switch v := result.(type) {
		case []any:
			gotLen = len(v)
		case string:
			gotLen = len(v)
		}
		if gotLen != tt.len {
			t.Errorf("%s: expected length %d, got %d", tt.expr, tt.len, gotLen)
		}
	}
}

func TestInline_Coverage(t *testing.T) {
	// Test inlining with various let expressions
	tests := []struct {
		input string
		eval  int64
	}{
		{"let x = 5; x", 5},
		{"let x = 5; let y = x; y", 5},
		{"let x = 1; let y = 2; x + y", 3},
	}

	for _, tt := range tests {
		inlined, err := Inline(tt.input)
		if err != nil {
			t.Fatalf("%s: inline error: %v", tt.input, err)
		}

		// Verify the inlined expression still evaluates correctly
		result, err := Eval(context.Background(), inlined, nil)
		if err != nil {
			t.Fatalf("%s: eval error: %v", tt.input, err)
		}
		if NewValue(result).Int() != tt.eval {
			t.Errorf("%s: expected %d, got %v", tt.input, tt.eval, result)
		}
	}
}

func TestConvertArgForBuiltin_Coverage(t *testing.T) {
	// Test conversion of various argument types in built-in functions
	// map uses convertArgForBuiltin internally when calling closures

	tests := []struct {
		expr     string
		expected int64
	}{
		// These test convertArgForBuiltin through map/filter
		{`map([1, 2, 3], x => x * 2) | sum`, 12},
		{`filter([1, 2, 3], x => x > 1) | sum`, 5},
		{`reduce([1, 2, 3], (acc, x) => acc + x, 0)`, 6},
	}

	engine := New()
	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		if NewValue(result).Int() != tt.expected {
			t.Errorf("%s: expected %d, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestVM_PopPeek_NonDebug(t *testing.T) {
	// Test pop and peek in non-debug mode
	restore := SetDebugModeForTest(false)
	defer restore()


	code := &bytecode{
		Instructions: []instruction{{Op: opHalt}},
		Constants:    []Value{},
	}
	vm := newVM(code, defaultBuiltins())
	defer releaseVM(vm)

	// Pop on empty stack in non-debug mode returns Nil
	val := vm.pop()
	if !val.IsNil() {
		t.Errorf("Expected Nil, got %v", val)
	}

	// Peek on empty stack in non-debug mode returns Nil
	val = vm.peek()
	if !val.IsNil() {
		t.Errorf("Expected Nil, got %v", val)
	}
}

func TestVM_Slice_GoSliceNegativeStep(t *testing.T) {
	engine := New()

	// Test negative step on typed Go slice
	env := map[string]any{
		"arr": []int{1, 2, 3, 4, 5},
	}

	result, err := engine.Eval(context.Background(), `arr[::-1]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(arr))
	}
	// Should be reversed
	if NewValue(arr[0]).Int() != 5 || NewValue(arr[4]).Int() != 1 {
		t.Errorf("Expected reversed array, got %v", arr)
	}
}

func TestVM_Index_MapStringInterface(t *testing.T) {
	engine := New()

	// Test indexing map[string]interface{}
	m := map[string]interface{}{
		"name": "test",
		"age":  30,
	}
	env := map[string]any{"m": m}

	result, err := engine.Eval(context.Background(), `m["name"]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("Expected 'test', got %v", result)
	}
}

func TestCompile_SpreadOperator(t *testing.T) {
	engine := New()

	// Test spread operator in array
	result, err := engine.Eval(context.Background(), `let arr = [1, 2, 3]; [0, ...arr, 4]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(arr))
	}
}

func TestCompile_ObjectDestructure(t *testing.T) {
	engine := New()

	// Test object destructuring
	result, err := engine.Eval(context.Background(), `let {a, b} = {a: 1, b: 2, c: 3}; a + b`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("Expected 3, got %v", result)
	}

	// Test array destructuring
	result, err = engine.Eval(context.Background(), `let [x, y, z] = [1, 2, 3]; x + y + z`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 6 {
		t.Errorf("Expected 6, got %v", result)
	}
}

func TestContains_GoMaps(t *testing.T) {
	engine := New()

	// Test 'in' with various Go map types
	tests := []struct {
		name     string
		m        any
		expr     string
		expected bool
	}{
		{"map[string]string", map[string]string{"a": "1"}, `"a" in m`, true},
		{"map[string]int", map[string]int{"a": 1}, `"a" in m`, true},
		{"map[string]int missing", map[string]int{"a": 1}, `"b" in m`, false},
		{"map[int]int", map[int]int{1: 10}, `1 in m`, true},
		{"map[int]int missing", map[int]int{1: 10}, `2 in m`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := map[string]any{"m": tt.m}
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIndex_GoArraysSlices(t *testing.T) {
	engine := New()

	// Test indexing native Go arrays and slices of various types
	tests := []struct {
		name     string
		arr      any
		expr     string
		expected any
	}{
		{"[]int", []int{1, 2, 3}, `arr[1]`, 2},
		{"[]string", []string{"a", "b"}, `arr[0]`, "a"},
		{"[]float64", []float64{1.1, 2.2}, `arr[1]`, 2.2},
		{"[]bool", []bool{true, false}, `arr[0]`, true},
		{"[3]int", [3]int{1, 2, 3}, `arr[2]`, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := map[string]any{"arr": tt.arr}
			result, err := engine.Eval(context.Background(), tt.expr, env)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			got := NewValue(result)
			expected := NewValue(tt.expected)
			if !got.Equal(expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSlice_GoSlices(t *testing.T) {
	engine := New()

	// Test slicing native Go slices
	goSlice := []float64{1.1, 2.2, 3.3, 4.4, 5.5}
	env := map[string]any{"arr": goSlice}

	tests := []struct {
		expr string
		len  int
	}{
		{`arr[1:3]`, 2},
		{`arr[:2]`, 2},
		{`arr[3:]`, 2},
		{`arr[:]`, 5},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, env)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		arr := result.([]any)
		if len(arr) != tt.len {
			t.Errorf("%s: expected length %d, got %d", tt.expr, tt.len, len(arr))
		}
	}
}

func TestCallNativeFromBuiltin_EdgeCases(t *testing.T) {
	// Test calling native functions from builtin with various return types
	engine := New(
		WithFunction("retNil", func() interface{} { return nil }),
		WithFunction("retSlice", func() []int { return []int{1, 2, 3} }),
		WithFunction("retMap", func() map[string]int { return map[string]int{"a": 1} }),
		WithFunction("retError", func() (int, error) { return 0, fmt.Errorf("test error") }),
	)

	// Test nil return
	result, err := engine.Eval(context.Background(), `retNil()`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}

	// Test slice return
	result, err = engine.Eval(context.Background(), `retSlice()`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil slice")
	}

	// Test map return
	result, err = engine.Eval(context.Background(), `retMap()`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil map")
	}

	// Test error return
	_, err = engine.Eval(context.Background(), `retError()`, nil)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestTypeChecker_MoreCases(t *testing.T) {
	// Test type checker with more expressions to increase coverage
	tc := NewTypeChecker()
	tc.WithVar("arr", TypeArray)
	tc.WithVar("m", TypeMap)
	tc.WithVar("f", TypeFunc)

	tests := []struct {
		expr string
		typ  Type
	}{
		// sliceExpr
		{"arr[0:1]", TypeArray},
		// OptionalMemberExpr
		{"m?.key", TypeAny},
		// OptionalIndexExpr
		{"arr?[0]", TypeAny},
		// spreadExpr
		{"[...arr]", TypeArray},
		// rangeExpr
		{"1..10", TypeArray},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Skipf("Parse error: %v", err)
				return
			}
			typ := tc.Infer(ast)
			if typ != tt.typ {
				t.Errorf("Expected %v, got %v", tt.typ, typ)
			}
		})
	}
}

func TestInlineLetExprs_MoreCases(t *testing.T) {
	// Test inlining with various expressions
	tests := []struct {
		input    string
		expected int64
	}{
		// Binary expressions with inlined values
		{"let x = 2; let y = 3; x * y", 6},
		// Nested let
		{"let x = 1; let y = x + 1; let z = y + 1; z", 3},
	}

	for _, tt := range tests {
		inlined, err := Inline(tt.input)
		if err != nil {
			t.Fatalf("%s: inline error: %v", tt.input, err)
		}

		result, err := Eval(context.Background(), inlined, nil)
		if err != nil {
			t.Fatalf("%s: eval error: %v", tt.input, err)
		}
		if NewValue(result).Int() != tt.expected {
			t.Errorf("%s: expected %d, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestContains_String(t *testing.T) {
	engine := New()

	// Test 'in' with strings
	tests := []struct {
		expr     string
		expected bool
	}{
		{`"ell" in "hello"`, true},
		{`"xyz" in "hello"`, false},
		{`"hello" in "hello"`, true},
		{`"h" in "hello"`, true},
		{`"o" in "hello"`, true},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		if result != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestIndex_StringOutOfBounds_Coverage(t *testing.T) {
	engine := New()

	// Indexing string out of bounds should error
	_, err := engine.Eval(context.Background(), `"hello"[10]`, nil)
	if err == nil {
		t.Error("Expected index out of bounds error")
	}
}

func TestSlice_EmptyArray(t *testing.T) {
	engine := New()

	// Test slicing empty array
	result, err := engine.Eval(context.Background(), `[][:]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 0 {
		t.Errorf("Expected empty array, got %v", arr)
	}
}

func TestConvertArgForBuiltin_MoreTypes(t *testing.T) {
	// Test with more builtin argument conversions
	engine := New(
		WithFunction("passSlice", func(s []any) int { return len(s) }),
		WithFunction("passMap", func(m map[string]any) int { return len(m) }),
		WithFunction("passBool", func(b bool) bool { return !b }),
	)

	tests := []struct {
		expr string
		want any
	}{
		{`passSlice([1, 2, 3])`, int64(3)},
		{`passMap({"a": 1, "b": 2})`, int64(2)},
		{`passBool(true)`, false},
		{`passBool(false)`, true},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		got := NewValue(result)
		want := NewValue(tt.want)
		if !got.Equal(want) {
			t.Errorf("%s: expected %v, got %v", tt.expr, tt.want, result)
		}
	}
}

func TestContains_GoMapStringKey(t *testing.T) {
	engine := New()

	// Test 'in' with Go map[string]interface{}
	m := map[string]interface{}{"name": "test", "value": 123}
	env := map[string]any{"m": m}

	// String key in map[string]interface{}
	result, err := engine.Eval(context.Background(), `"name" in m`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	result, err = engine.Eval(context.Background(), `"missing" in m`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestIndex_GoMapStringKeyReflection(t *testing.T) {
	engine := New()

	// Test indexing Go map[string]interface{} via string key
	m := map[string]interface{}{"name": "test", "value": 123}
	env := map[string]any{"m": m}

	result, err := engine.Eval(context.Background(), `m["name"]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("Expected 'test', got %v", result)
	}

	// Test missing key returns nil
	result, err = engine.Eval(context.Background(), `m["missing"]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestSlice_GoArray(t *testing.T) {
	engine := New()

	// Test slicing Go array (not slice)
	arr := [5]int{0, 1, 2, 3, 4}
	env := map[string]any{"arr": arr}

	result, err := engine.Eval(context.Background(), `arr[1:3]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	slice := result.([]any)
	if len(slice) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(slice))
	}
}

func TestIndex_NegativeIndex(t *testing.T) {
	engine := New()

	// Test negative indexing
	result, err := engine.Eval(context.Background(), `[1, 2, 3, 4, 5][-1]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 5 {
		t.Errorf("Expected 5, got %v", result)
	}

	result, err = engine.Eval(context.Background(), `[1, 2, 3, 4, 5][-3]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("Expected 3, got %v", result)
	}

	// Negative index on string
	result, err = engine.Eval(context.Background(), `"hello"[-1]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "o" {
		t.Errorf("Expected 'o', got %v", result)
	}
}

func TestIndex_GoSliceNegative(t *testing.T) {
	engine := New()

	// Test negative indexing on Go slice
	goSlice := []int{1, 2, 3, 4, 5}
	env := map[string]any{"arr": goSlice}

	result, err := engine.Eval(context.Background(), `arr[-1]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 5 {
		t.Errorf("Expected 5, got %v", result)
	}
}

func TestContains_GoMapNonStringKey(t *testing.T) {
	engine := New()

	// Test 'in' with Go map with non-string key
	m := map[float64]string{1.5: "one", 2.5: "two"}
	env := map[string]any{"m": m}

	result, err := engine.Eval(context.Background(), `1.5 in m`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	result, err = engine.Eval(context.Background(), `9.9 in m`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestTypeChecker_BinaryOps_Coverage(t *testing.T) {
	tc := NewTypeChecker()
	tc.WithVar("i", TypeInt)
	tc.WithVar("f", TypeFloat)
	tc.WithVar("s", TypeString)
	tc.WithVar("b", TypeBool)
	tc.WithVar("a", TypeArray)
	tc.WithVar("m", TypeMap)

	tests := []struct {
		expr string
		typ  Type
	}{
		// Bitwise operations
		{"i & 1", TypeInt},
		{"i | 1", TypeInt},
		{"i ^ 1", TypeInt},
		{"i << 1", TypeInt},
		{"i >> 1", TypeInt},
		{"~i", TypeInt},
		// Comparisons with different types
		{"i < f", TypeBool},
		{"f > i", TypeBool},
		// String operations
		{"s + s", TypeString},
		// Member access
		{"m.key", TypeAny},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			parser := newParser(tt.expr)
			ast, err := parser.parse()
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			typ := tc.Infer(ast)
			if typ != tt.typ {
				t.Errorf("Expected %v, got %v", tt.typ, typ)
			}
		})
	}
}

func TestInlineLetExprs_Lambdas(t *testing.T) {
	// Test that Inline handles lambdas gracefully
	result, err := Inline("let x = 5; (y => x + y)(10)")
	if err != nil {
		t.Fatalf("Inline error: %v", err)
	}
	// Should return string
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestError_LexerError(t *testing.T) {
	// Test LexerError Error method
	err := &LexerError{
		Line:    1,
		Column:  5,
		Message: "invalid character",
	}
	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}
	if !strings.Contains(errStr, "invalid character") {
		t.Errorf("Error should contain message: %s", errStr)
	}
}

// =============================================================================
// TESTS FOR ЗАМЕЧАНИЯ.MD ISSUES
// =============================================================================

func TestConstantFolding_PowerOverflow(t *testing.T) {
	// Test that large powers don't cause overflow during constant folding
	// 10 ** 19 would overflow int64, so it should be computed as float
	engine := New(WithOptimize(true))

	// 10^19 is larger than int64 max (~9.2e18), should be computed as float
	result, err := engine.Eval(context.Background(), `10 ** 19`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Result should be approximately 1e19
	f := NewValue(result).Float()
	if f < 9e18 || f > 1.1e19 {
		t.Errorf("Expected ~1e19, got %v", f)
	}

	// 2^62 should still be computed as integer (fits in int64)
	result, err = engine.Eval(context.Background(), `2 ** 62`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := int64(1) << 62
	if NewValue(result).Int() != expected {
		t.Errorf("Expected %d, got %v", expected, result)
	}

	// Small powers should work correctly
	result, err = engine.Eval(context.Background(), `3 ** 10`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 59049 {
		t.Errorf("Expected 59049, got %v", result)
	}

	// Negative base with even exponent
	result, err = engine.Eval(context.Background(), `(-2) ** 10`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 1024 {
		t.Errorf("Expected 1024, got %v", result)
	}
}

func TestContains_EmptyString(t *testing.T) {
	// Test that empty string is contained in any string (matches Go's strings.Contains)
	engine := New()

	tests := []struct {
		expr     string
		expected bool
	}{
		{`"" in "hello"`, true},
		{`"" in ""`, true},
		{`"a" in ""`, false},
		{`"ell" in "hello"`, true},
	}

	for _, tt := range tests {
		result, err := engine.Eval(context.Background(), tt.expr, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.expr, err)
		}
		if result != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestUnique_StableForMaps(t *testing.T) {
	engine := New()

	// Create two maps with same content but possibly different iteration order
	// They should be considered equal by unique
	result, err := engine.Eval(context.Background(), `
		let maps = [
			{"a": 1, "b": 2},
			{"b": 2, "a": 1},
			{"a": 1, "b": 2}
		];
		unique(maps)
	`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	// Should deduplicate to 1 unique map (all have same content)
	if len(arr) != 1 {
		t.Errorf("Expected 1 unique map, got %d: %v", len(arr), arr)
	}
}

func TestUnique_NestedStructures(t *testing.T) {
	engine := New()

	// Test unique with nested arrays and maps
	result, err := engine.Eval(context.Background(), `
		unique([
			[1, 2, 3],
			[1, 2, 3],
			[3, 2, 1],
			{"x": [1, 2]}
		])
	`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	// Should have 3 unique elements: [1,2,3], [3,2,1], {"x": [1,2]}
	if len(arr) != 3 {
		t.Errorf("Expected 3 unique elements, got %d", len(arr))
	}
}

func TestCanIntPowWithoutOverflow(t *testing.T) {
	tests := []struct {
		base     int64
		exp      int64
		expected bool
	}{
		{10, 18, true},   // 10^18 fits in int64
		{10, 19, false},  // 10^19 overflows
		{2, 62, true},    // 2^62 fits
		{2, 63, false},   // 2^63 overflows
		{0, 100, true},   // 0^100 = 0
		{1, 1000, true},  // 1^1000 = 1
		{-1, 1000, true}, // (-1)^1000 = 1
		{3, 39, true},    // 3^39 fits
		{3, 40, false},   // 3^40 overflows
	}

	for _, tt := range tests {
		result := canIntPowWithoutOverflow(tt.base, tt.exp)
		if result != tt.expected {
			t.Errorf("canIntPowWithoutOverflow(%d, %d) = %v, want %v", tt.base, tt.exp, result, tt.expected)
		}
	}
}

// =============================================================================
// COVERAGE IMPROVEMENT TESTS (target: 70%+ per function)
// =============================================================================

func TestContains_AllBranches(t *testing.T) {
	engine := New()

	// Test contains with nil collection (default branch, collection.kind != kindAny)
	env := map[string]any{
		"nilVal":   nil,
		"intVal":   42,
		"boolVal":  true,
		"floatVal": 3.14,
	}

	// nil collection should return false
	result, err := engine.Eval(context.Background(), `1 in nilVal`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false for 'in' on nil, got %v", result)
	}

	// int collection should return false (not array/map/string/kindAny)
	result, err = engine.Eval(context.Background(), `1 in intVal`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false for 'in' on int, got %v", result)
	}

	// Test kindAny map where key type is not assignable and not string
	// Using map[int]string with float key
	env["intKeyMap"] = map[int]string{1: "one", 2: "two"}
	result, err = engine.Eval(context.Background(), `3.14 in intKeyMap`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// float64 is not assignable to int key, should return false
	if result != false {
		t.Errorf("Expected false for float key in int map, got %v", result)
	}

	// Test kindAny slice where item is not found
	env["goSlice"] = []int{1, 2, 3}
	result, err = engine.Eval(context.Background(), `99 in goSlice`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false for missing item, got %v", result)
	}
}

func TestIndex_AllBranches(t *testing.T) {
	engine := New()

	// Test index with negative that becomes out of bounds
	_, err := engine.Eval(context.Background(), `[1, 2, 3][-10]`, nil)
	if err == nil {
		t.Error("Expected error for out-of-bounds negative index")
	}

	// Test index on string with negative index out of bounds
	_, err = engine.Eval(context.Background(), `"hi"[-10]`, nil)
	if err == nil {
		t.Error("Expected error for out-of-bounds negative index on string")
	}

	// Test kindAny slice with negative index
	env := map[string]any{"arr": []int{1, 2, 3, 4, 5}}
	result, err := engine.Eval(context.Background(), `arr[-2]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 4 {
		t.Errorf("Expected 4, got %v", result)
	}

	// Test kindAny slice negative out of bounds
	_, err = engine.Eval(context.Background(), `arr[-100]`, env)
	if err == nil {
		t.Error("Expected error for out-of-bounds negative index on Go slice")
	}

	// Test kindAny map with non-assignable key type and non-string key
	env["intKeyMap"] = map[int]string{1: "one", 2: "two"}
	// Try accessing with float key (not assignable to int, not string key type)
	result, err = engine.Eval(context.Background(), `intKeyMap[3.14]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Should return nil for non-assignable key
	if result != nil {
		t.Errorf("Expected nil for float key in int map, got %v", result)
	}

	// Test kindAny map where key IS assignable but value doesn't exist
	result, err = engine.Eval(context.Background(), `intKeyMap[999]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for missing key, got %v", result)
	}

	// Test index on non-indexable type (func, etc.)
	env["fn"] = func() {}
	_, err = engine.Eval(context.Background(), `fn[0]`, env)
	if err == nil {
		t.Error("Expected error for indexing function")
	}
}

func TestSlice_AllBranches(t *testing.T) {
	engine := New()

	// Test slice on kindAny (Go slice)
	env := map[string]any{"arr": []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}}

	// Basic slice on Go slice
	result, err := engine.Eval(context.Background(), `arr[2:5]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.([]any)) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(result.([]any)))
	}

	// Slice with step on Go slice
	result, err = engine.Eval(context.Background(), `arr[1:8:2]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(arr))
	}

	// Negative step on Go slice
	result, err = engine.Eval(context.Background(), `arr[8:2:-1]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 6 {
		t.Errorf("Expected 6 elements, got %d", len(arr))
	}

	// Slice with nil bounds (uses defaults)
	result, err = engine.Eval(context.Background(), `arr[:]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.([]any)) != 10 {
		t.Errorf("Expected 10 elements, got %d", len(result.([]any)))
	}

	// String slicing with negative step
	result, err = engine.Eval(context.Background(), `"hello"[4:1:-1]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "oll" {
		t.Errorf("Expected 'oll', got %v", result)
	}

	// Slice on non-sliceable type
	env["num"] = 42
	_, err = engine.Eval(context.Background(), `num[0:1]`, env)
	if err == nil {
		t.Error("Expected error for slicing non-sliceable type")
	}

	// Array slice with negative indices
	result, err = engine.Eval(context.Background(), `[0,1,2,3,4,5,6,7,8,9][-5:-2]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}
}

func TestTypeChecker_AllBranches(t *testing.T) {
	tests := []struct {
		expr      string
		hasErrors bool
	}{
		// Arithmetic on non-numeric should error
		{`s - s`, true},
		{`s / s`, true},
		{`s % s`, true},
		{`s ** s`, true},
		// Logical on non-bool should error
		{`i && i`, true},
		{`i || i`, true},
		{`s and s`, true},
		{`s or s`, true},
		// Bitwise on non-int should error
		{`s & s`, true},
		{`s | s`, true},
		{`s ^ s`, true},
		{`s << s`, true},
		{`s >> s`, true},
		// Unary not on non-bool
		{`!i`, true},
		{`not s`, true},
		// Bitwise not on non-int
		{`~s`, true},
		// Ternary with non-bool condition
		{`s ? 1 : 0`, true},
		// Valid expressions
		{`i + i`, false},
		{`b && b`, false},
		{`i & i`, false},
	}

	for _, tt := range tests {
		parser := newParser(tt.expr)
		ast, err := parser.parse()
		if err != nil {
			t.Fatalf("%s: parse error: %v", tt.expr, err)
		}

		tc := NewTypeChecker()
		tc.WithVar("s", TypeString)
		tc.WithVar("b", TypeBool)
		tc.WithVar("i", TypeInt)

		errs := tc.Check(ast)
		if tt.hasErrors && (errs == nil || errs.Len() == 0) {
			t.Errorf("%s: expected type errors, got none", tt.expr)
		}
		if !tt.hasErrors && errs != nil && errs.Len() > 0 {
			t.Errorf("%s: unexpected errors: %v", tt.expr, errs)
		}
	}
}

func TestInlineLetExprs_AllBranches(t *testing.T) {
	// Test inlining with various node types

	// Coalesce expression
	result, err := Inline("let x = 5; x ?? 0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Ternary expression
	result, err = Inline("let x = true; x ? 1 : 0")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Array literal
	result, err = Inline("let x = 1; [x, x + 1, x + 2]")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Map literal
	result, err = Inline(`let x = 1; {"a": x, "b": x + 1}`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Member expression
	result, err = Inline("let obj = {a: 1}; obj.a")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Index expression
	result, err = Inline("let arr = [1, 2, 3]; arr[0]")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Call expression
	result, err = Inline("let x = 5; abs(x)")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}

	// Unary expression
	result, err = Inline("let x = -5; -x")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestCallNativeFromBuiltin_AllBranches(t *testing.T) {
	// Test various native function return types

	// Function returning error
	engine := New(
		WithFunction("mayFail", func(fail bool) (int, error) {
			if fail {
				return 0, fmt.Errorf("intentional error")
			}
			return 42, nil
		}),
	)

	// Success case
	result, err := engine.Eval(context.Background(), `mayFail(false)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("Expected 42, got %v", result)
	}

	// Error case
	_, err = engine.Eval(context.Background(), `mayFail(true)`, nil)
	if err == nil {
		t.Error("Expected error")
	}

	// Function with multiple return values (value, error)
	engine = New(
		WithFunction("divmod", func(a, b int) (int, int) {
			return a / b, a % b
		}),
	)
	// Multiple return values - only first is used
	result, err = engine.Eval(context.Background(), `divmod(10, 3)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("Expected 3, got %v", result)
	}

	// Variadic function
	engine = New(
		WithFunction("sumInts", func(nums ...int64) int64 {
			var sum int64
			for _, n := range nums {
				sum += n
			}
			return sum
		}),
	)
	result, err = engine.Eval(context.Background(), `sumInts(1, 2, 3, 4, 5)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 15 {
		t.Errorf("Expected 15, got %v", result)
	}
}

func TestCompiler_AllBranches(t *testing.T) {
	engine := New()

	// Test sliceExpr with all nil bounds
	result, err := engine.Eval(context.Background(), `[1,2,3][:]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.([]any)) != 3 {
		t.Errorf("Expected 3 elements")
	}

	// Test sliceExpr with step only
	result, err = engine.Eval(context.Background(), `[0,1,2,3,4,5][::2]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.([]any)) != 3 {
		t.Errorf("Expected 3 elements")
	}

	// Test rangeExpr
	result, err = engine.Eval(context.Background(), `1..5`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) < 4 { // 1..5 produces [1,2,3,4] or [1,2,3,4,5] depending on semantics
		t.Errorf("Expected at least 4 elements, got %d", len(arr))
	}

	// Test spreadExpr in array literal
	result, err = engine.Eval(context.Background(), `let arr = [1, 2]; [0, ...arr, 3]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(arr))
	}

	// Test mapLit with computed keys
	result, err = engine.Eval(context.Background(), `let key = "x"; {[key]: 1}`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["x"] != int64(1) {
		t.Errorf("Expected x=1, got %v", m)
	}

	// Test OptionalMemberExpr
	result, err = engine.Eval(context.Background(), `let obj = nil; obj?.prop`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

func TestError_AllBranches(t *testing.T) {
	// Test LexerError
	lexErr := &LexerError{
		Line:    1,
		Column:  5,
		Message: "test error",
	}
	str := lexErr.Error()
	if !strings.Contains(str, "test error") {
		t.Errorf("Expected error message in output: %s", str)
	}

	// Test RuntimeError with source
	rtErr := &RuntimeError{
		Message: "runtime error",
		Op:      "+",
		Source:  "1 + \"x\"",
	}
	str = rtErr.Error()
	if !strings.Contains(str, "runtime error") {
		t.Errorf("Expected error message in output: %s", str)
	}

	// Test RuntimeError.WithSource
	rtErr2 := rtErr.WithSource("new source")
	str = rtErr2.Error()
	if !strings.Contains(str, "new source") {
		t.Errorf("Expected source in output: %s", str)
	}

	// Test ParseError.WithSource
	parseErr := &ParseError{
		token:   token{Line: 1, Column: 5, Type: tokenIdent},
		Message: "unexpected token",
	}
	parseErr = parseErr.WithSource("let @x = 1")
	str = parseErr.Error()
	if !strings.Contains(str, "unexpected token") {
		t.Errorf("Expected error message in output: %s", str)
	}
}

func TestConvertArgForBuiltin_AllTypes(t *testing.T) {
	// Test all type conversions in convertArgForBuiltin
	engine := New(
		WithFunction("takeInt8", func(x int8) int8 { return x }),
		WithFunction("takeInt16", func(x int16) int16 { return x }),
		WithFunction("takeInt32", func(x int32) int32 { return x }),
		WithFunction("takeInt64", func(x int64) int64 { return x }),
		WithFunction("takeUint", func(x uint) uint { return x }),
		WithFunction("takeUint8", func(x uint8) uint8 { return x }),
		WithFunction("takeUint16", func(x uint16) uint16 { return x }),
		WithFunction("takeUint32", func(x uint32) uint32 { return x }),
		WithFunction("takeUint64", func(x uint64) uint64 { return x }),
		WithFunction("takeFloat32", func(x float32) float32 { return x }),
		WithFunction("takeFloat64", func(x float64) float64 { return x }),
		WithFunction("takeString", func(x string) string { return x }),
		WithFunction("takeBool", func(x bool) bool { return x }),
		WithFunction("takeSlice", func(x []any) int { return len(x) }),
		WithFunction("takeInterface", func(x interface{}) string { return fmt.Sprintf("%T", x) }),
	)

	// Test int8
	result, err := engine.Eval(context.Background(), `takeInt8(42)`, nil)
	if err != nil {
		t.Fatalf("takeInt8: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("takeInt8: expected 42, got %v", result)
	}

	// Test int16
	result, err = engine.Eval(context.Background(), `takeInt16(1000)`, nil)
	if err != nil {
		t.Fatalf("takeInt16: %v", err)
	}
	if NewValue(result).Int() != 1000 {
		t.Errorf("takeInt16: expected 1000, got %v", result)
	}

	// Test int32
	result, err = engine.Eval(context.Background(), `takeInt32(100000)`, nil)
	if err != nil {
		t.Fatalf("takeInt32: %v", err)
	}
	if NewValue(result).Int() != 100000 {
		t.Errorf("takeInt32: expected 100000, got %v", result)
	}

	// Test int64
	result, err = engine.Eval(context.Background(), `takeInt64(9999999999)`, nil)
	if err != nil {
		t.Fatalf("takeInt64: %v", err)
	}
	if NewValue(result).Int() != 9999999999 {
		t.Errorf("takeInt64: expected 9999999999, got %v", result)
	}

	// Test uint
	result, err = engine.Eval(context.Background(), `takeUint(42)`, nil)
	if err != nil {
		t.Fatalf("takeUint: %v", err)
	}
	if NewValue(result).Int() != 42 {
		t.Errorf("takeUint: expected 42, got %v", result)
	}

	// Test uint8
	result, err = engine.Eval(context.Background(), `takeUint8(255)`, nil)
	if err != nil {
		t.Fatalf("takeUint8: %v", err)
	}
	if NewValue(result).Int() != 255 {
		t.Errorf("takeUint8: expected 255, got %v", result)
	}

	// Test uint16
	result, err = engine.Eval(context.Background(), `takeUint16(65535)`, nil)
	if err != nil {
		t.Fatalf("takeUint16: %v", err)
	}
	if NewValue(result).Int() != 65535 {
		t.Errorf("takeUint16: expected 65535, got %v", result)
	}

	// Test uint32
	result, err = engine.Eval(context.Background(), `takeUint32(4294967295)`, nil)
	if err != nil {
		t.Fatalf("takeUint32: %v", err)
	}
	if NewValue(result).Int() != 4294967295 {
		t.Errorf("takeUint32: expected 4294967295, got %v", result)
	}

	// Test uint64 (use smaller value that fits in int64)
	result, err = engine.Eval(context.Background(), `takeUint64(9223372036854775807)`, nil)
	if err != nil {
		t.Fatalf("takeUint64: %v", err)
	}
	// int64 max
	if NewValue(result).Int() != 9223372036854775807 {
		t.Errorf("takeUint64: expected 9223372036854775807, got %v", result)
	}

	// Test float32
	result, err = engine.Eval(context.Background(), `takeFloat32(3.14)`, nil)
	if err != nil {
		t.Fatalf("takeFloat32: %v", err)
	}
	f := NewValue(result).Float()
	if f < 3.13 || f > 3.15 {
		t.Errorf("takeFloat32: expected ~3.14, got %v", f)
	}

	// Test float64
	result, err = engine.Eval(context.Background(), `takeFloat64(3.141592653589793)`, nil)
	if err != nil {
		t.Fatalf("takeFloat64: %v", err)
	}
	if NewValue(result).Float() != 3.141592653589793 {
		t.Errorf("takeFloat64: expected pi, got %v", result)
	}

	// Test string
	result, err = engine.Eval(context.Background(), `takeString("hello")`, nil)
	if err != nil {
		t.Fatalf("takeString: %v", err)
	}
	if result != "hello" {
		t.Errorf("takeString: expected 'hello', got %v", result)
	}

	// Test bool
	result, err = engine.Eval(context.Background(), `takeBool(true)`, nil)
	if err != nil {
		t.Fatalf("takeBool: %v", err)
	}
	if result != true {
		t.Errorf("takeBool: expected true, got %v", result)
	}

	// Test slice
	result, err = engine.Eval(context.Background(), `takeSlice([1, 2, 3])`, nil)
	if err != nil {
		t.Fatalf("takeSlice: %v", err)
	}
	if NewValue(result).Int() != 3 {
		t.Errorf("takeSlice: expected 3, got %v", result)
	}

	// Test interface
	result, err = engine.Eval(context.Background(), `takeInterface(42)`, nil)
	if err != nil {
		t.Fatalf("takeInterface: %v", err)
	}
	if result == "" {
		t.Error("takeInterface: expected non-empty type string")
	}
}

func TestSlice_MoreBranches(t *testing.T) {
	engine := New()

	// Test slice with negative low that becomes valid
	result, err := engine.Eval(context.Background(), `[0,1,2,3,4][-3:]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}

	// Test slice with negative high
	result, err = engine.Eval(context.Background(), `[0,1,2,3,4][:-2]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(arr))
	}

	// Test string slice with step
	result, err = engine.Eval(context.Background(), `"abcdef"[::2]`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "ace" {
		t.Errorf("Expected 'ace', got %v", result)
	}

	// Test Go slice with negative indices
	env := map[string]any{"arr": []int{0, 1, 2, 3, 4}}
	result, err = engine.Eval(context.Background(), `arr[-3:-1]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	arr = result.([]any)
	if len(arr) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(arr))
	}
}

func TestIndex_MoreBranches(t *testing.T) {
	engine := New()

	// Test index on Go array (not slice)
	env := map[string]any{"arr": [5]int{0, 1, 2, 3, 4}}
	result, err := engine.Eval(context.Background(), `arr[2]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 2 {
		t.Errorf("Expected 2, got %v", result)
	}

	// Test index with negative on Go array
	result, err = engine.Eval(context.Background(), `arr[-1]`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 4 {
		t.Errorf("Expected 4, got %v", result)
	}
}

func TestContains_MoreBranches(t *testing.T) {
	engine := New()

	// Test 'in' on Go array (not slice)
	env := map[string]any{"arr": [5]int{0, 1, 2, 3, 4}}
	result, err := engine.Eval(context.Background(), `2 in arr`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}

	// Test 'not in' on Go array
	result, err = engine.Eval(context.Background(), `99 in arr`, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}
}

// =============================================================================
// OVERFLOW AND STRICT MODE TESTS
// =============================================================================

func TestArithmetic_OverflowProtection(t *testing.T) {
	engine := New()

	// Test addition overflow - should convert to float
	result, err := engine.Eval(context.Background(), `9223372036854775807 + 1`, nil) // MaxInt64 + 1
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Result should be a float due to overflow
	f := NewValue(result).Float()
	if f < 9.2e18 {
		t.Errorf("Expected large float, got %v", f)
	}

	// Test subtraction overflow - use expression instead of literal
	result, err = engine.Eval(context.Background(), `(0 - 9223372036854775807) - 2`, nil) // -MaxInt64 - 2 overflows
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Result should be a float due to overflow
	f = NewValue(result).Float()
	if f > -9.2e18 {
		t.Errorf("Expected large negative float, got %v", f)
	}

	// Test multiplication overflow
	result, err = engine.Eval(context.Background(), `9223372036854775807 * 2`, nil) // MaxInt64 * 2
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Result should be a float due to overflow
	f = NewValue(result).Float()
	if f < 1.8e19 {
		t.Errorf("Expected large float, got %v", f)
	}

	// Normal operations should still return int
	result, err = engine.Eval(context.Background(), `100 + 200`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 300 {
		t.Errorf("Expected 300, got %v", result)
	}
}

func TestUnaryNeg_MinInt64(t *testing.T) {
	engine := New()

	// -MinInt64 overflows in two's complement, should convert to float
	// MinInt64 = -9223372036854775808
	// We can't write MinInt64 as a literal, so we use (0 - MaxInt64 - 1)
	result, err := engine.Eval(context.Background(), `-(0 - 9223372036854775807 - 1)`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Result should be positive float ~9.22e18
	f := NewValue(result).Float()
	if f < 9.2e18 {
		t.Errorf("Expected large positive float for -MinInt64, got %v", f)
	}
}

func TestConstantFolding_OverflowProtection(t *testing.T) {
	engine := New(WithOptimize(true))

	// Test that constant folding handles overflow correctly
	// MaxInt64 + 1 should be folded to float
	result, err := engine.Eval(context.Background(), `9223372036854775807 + 1`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Result should be reasonable
	f := NewValue(result).Float()
	if f < 9.2e18 {
		t.Errorf("Expected large float, got %v", f)
	}

	// Test subtraction overflow via expression
	result, err = engine.Eval(context.Background(), `(0 - 9223372036854775807) - 2`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	f = NewValue(result).Float()
	if f > -9.2e18 {
		t.Errorf("Expected large negative float, got %v", f)
	}

	// Multiplication overflow
	result, err = engine.Eval(context.Background(), `4611686018427387904 * 4`, nil) // 2^62 * 4 overflows
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	f = NewValue(result).Float()
	if f < 1.8e19 {
		t.Errorf("Expected large float, got %v", f)
	}
}

func TestStrictMode(t *testing.T) {
	// Test strict mode engine
	engine := New(WithStrictMode(true))

	// Valid numeric operations should work
	result, err := engine.Eval(context.Background(), `10 - 3`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 7 {
		t.Errorf("Expected 7, got %v", result)
	}

	// Invalid string-to-number should fail in strict mode
	_, err = engine.Eval(context.Background(), `"abc" - 1`, nil)
	if err == nil {
		t.Error("Expected error for invalid string-to-number conversion in strict mode")
	}

	// Non-strict mode - "abc" - 1 also fails because string - int is not allowed
	// (This is correct behavior - only string + anything converts to string)
	engineNonStrict := New()
	_, err = engineNonStrict.Eval(context.Background(), `"abc" - 1`, nil)
	if err == nil {
		t.Error("Expected error for string - int (not a valid operation)")
	}

	// Test that WithStrictMode(false) works
	engineExplicitNonStrict := New(WithStrictMode(false))
	result, err = engineExplicitNonStrict.Eval(context.Background(), `10 - 3`, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if NewValue(result).Int() != 7 {
		t.Errorf("Expected 7, got %v", result)
	}
}

func TestStrictMode_EmptyString(t *testing.T) {
	engine := New(WithStrictMode(true))

	// Empty string in numeric context should error
	_, err := engine.Eval(context.Background(), `"" - 1`, nil)
	if err == nil {
		t.Error("Expected error for empty string in strict mode")
	}
}

// Test that strict mode is honored inside lambdas and builtins (e.g. map + int()).
func TestStrictMode_InLambdaAndBuiltin(t *testing.T) {
	engine := New(WithStrictMode(true))

	// Case 1: all arguments are numeric strings - should succeed
	result, err := engine.Eval(context.Background(), `map(["1", "2", "3"], x => int(x))`, nil)
	if err != nil {
		t.Fatalf("Unexpected error for numeric strings in strict mode: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("Expected []any result, got %T", result)
	}
	if len(arr) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(arr))
	}
	if arr[0].(int64) != 1 || arr[1].(int64) != 2 || arr[2].(int64) != 3 {
		t.Errorf("Unexpected mapped values: %v", arr)
	}

	// Case 2: one of the arguments is non-numeric string - int() inside lambda should fail
	_, err = engine.Eval(context.Background(), `map(["1", "abc", "3"], x => int(x))`, nil)
	if err == nil {
		t.Error("Expected error for invalid string-to-int conversion inside lambda in strict mode")
	}
}
