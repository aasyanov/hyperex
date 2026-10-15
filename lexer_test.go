package hyperex

import (
	"context"
	"testing"
)

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
		{"null", tokenIdent}, // not a keyword; use nil
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
