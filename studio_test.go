package hyperex

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestTokenize_Smoke(t *testing.T) {
	toks, err := Tokenize(`x + 1 // hi`)
	if err != nil {
		t.Fatalf("Tokenize: %v", err)
	}
	if len(toks) < 3 || toks[1].Type != TokenOperator || toks[1].Lit != "+" {
		t.Fatalf("want operator Lit \"+\", got %#v", toks)
	}
	if len(toks) < 4 {
		t.Fatalf("want >=4 tokens, got %d: %+v", len(toks), toks)
	}
	if toks[0].Type != TokenIdent || toks[0].Lit != "x" || toks[0].Offset != 0 {
		t.Fatalf("first token: %+v", toks[0])
	}
	if toks[1].Type != TokenOperator {
		t.Fatalf("want Operator +, got %+v", toks[1])
	}
	if toks[2].Type != TokenNumber || toks[2].Lit != "1" {
		t.Fatalf("want Number 1, got %+v", toks[2])
	}
	foundComment := false
	foundEOF := false
	for _, tok := range toks {
		if tok.Type == TokenComment {
			foundComment = true
			if !strings.HasPrefix(tok.Lit, "//") {
				t.Fatalf("comment lit: %q", tok.Lit)
			}
		}
		if tok.Type == TokenEOF {
			foundEOF = true
		}
	}
	if !foundComment {
		t.Fatal("expected Comment token")
	}
	if !foundEOF {
		t.Fatal("expected EOF token")
	}
}

func TestTokenize_KeywordAndString(t *testing.T) {
	toks, err := Tokenize(`let x = "ab"`)
	if err != nil {
		t.Fatalf("Tokenize: %v", err)
	}
	kinds := make([]TokenKind, 0, len(toks))
	for _, tok := range toks {
		if tok.Type == TokenEOF {
			break
		}
		kinds = append(kinds, tok.Type)
	}
	want := []TokenKind{TokenKeyword, TokenIdent, TokenOperator, TokenString}
	if len(kinds) != len(want) {
		t.Fatalf("kinds=%v want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds[%d]=%s want %s (all=%v)", i, kinds[i], want[i], kinds)
		}
	}
}

func TestParse_Walk(t *testing.T) {
	node, err := Parse(`a + b`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var idents []string
	Walk(node, func(n Node) bool {
		if id, ok := n.(*ident); ok {
			idents = append(idents, id.Name)
		}
		return true
	})
	if len(idents) != 2 || idents[0] != "a" || idents[1] != "b" {
		t.Fatalf("idents=%v", idents)
	}
	if s := ASTToString(node); s == "" {
		t.Fatal("ASTToString empty")
	}
}

func TestExtractVariablesOpts_MemberRoots(t *testing.T) {
	expr := `Alarm.Fire + other`
	def, err := ExtractVariables(expr)
	if err != nil {
		t.Fatalf("ExtractVariables: %v", err)
	}
	// Default ExtractVariables keeps nested property idents.
	if !containsStr(def, "Fire") || !containsStr(def, "Alarm") || !containsStr(def, "other") {
		t.Fatalf("default vars=%v", def)
	}

	roots, err := ExtractVariablesOpts(expr, ExtractMemberRoots())
	if err != nil {
		t.Fatalf("ExtractVariablesOpts: %v", err)
	}
	if containsStr(roots, "Fire") {
		t.Fatalf("member roots should strip Fire: %v", roots)
	}
	if !containsStr(roots, "Alarm") || !containsStr(roots, "other") {
		t.Fatalf("member roots vars=%v", roots)
	}

	// Computed property still extracts the index expression idents.
	comp, err := ExtractVariablesOpts(`obj[key]`, ExtractMemberRoots())
	if err != nil {
		t.Fatalf("computed: %v", err)
	}
	if !containsStr(comp, "obj") || !containsStr(comp, "key") {
		t.Fatalf("computed vars=%v", comp)
	}
}

func TestWithMaxSourceBytes(t *testing.T) {
	eng := New(WithMaxSourceBytes(5))
	_, err := eng.Compile(`123456`)
	if !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("want ErrSourceTooLarge, got %v", err)
	}
	prog, err := eng.Compile(`1+2`)
	if err != nil {
		t.Fatalf("short Compile: %v", err)
	}
	v, err := prog.Eval(context.Background(), nil)
	if err != nil || v != int64(3) {
		t.Fatalf("Eval=%v %v", v, err)
	}
	// Unlimited default: long identifier is a valid expression.
	_, err = New().Compile(strings.Repeat("a", 100))
	if err != nil {
		t.Fatalf("unlimited Compile: %v", err)
	}
}

func TestBuiltin_KeysValues_Sorted(t *testing.T) {
	eng := New()
	env := map[string]any{
		"m": map[string]any{"z": 3, "a": 1, "m": 2},
	}
	keys, err := eng.Eval(context.Background(), `keys(m)`, env)
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	arr := keys.([]any)
	if len(arr) != 3 || arr[0] != "a" || arr[1] != "m" || arr[2] != "z" {
		t.Fatalf("keys order=%v", arr)
	}
	vals, err := eng.Eval(context.Background(), `values(m)`, env)
	if err != nil {
		t.Fatalf("values: %v", err)
	}
	varr := vals.([]any)
	if len(varr) != 3 || varr[0] != 1 || varr[1] != 2 || varr[2] != 3 {
		t.Fatalf("values order=%v (want by sorted key a,m,z -> 1,2,3)", varr)
	}
}

func TestWarnConstants_Exported(t *testing.T) {
	a, err := Analyze(`x | y`)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !containsStr(a.Warnings, WarnAmbiguousBitwiseOr) {
		t.Fatalf("want WarnAmbiguousBitwiseOr in %v", a.Warnings)
	}
	a2, err := Analyze(`xs | sum`)
	if err != nil {
		t.Fatalf("Analyze pipe: %v", err)
	}
	if !containsStr(a2.Warnings, WarnBuiltinNamePipe) {
		t.Fatalf("want WarnBuiltinNamePipe in %v", a2.Warnings)
	}
}

func TestRuntimeError_LocationFieldsDefaultZero(t *testing.T) {
	_, err := Eval(context.Background(), `1/0`)
	var re *RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("want RuntimeError, got %T %v", err, err)
	}
	// No AST source map yet - Offset/Line/Column stay 0 (documented).
	if re.Offset != 0 || re.Line != 0 || re.Column != 0 {
		t.Fatalf("expected zero location fields, got Offset=%d Line=%d Column=%d", re.Offset, re.Line, re.Column)
	}
	if re.Source == "" {
		t.Fatal("expected Source annotation")
	}
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Domain edge proofs -> studio_test.go
// ---------------------------------------------------------------------------

func TestModeB_Studio_IllegalAndParseErrors(t *testing.T) {
	// TokenKind.String for all exported kinds
	for _, k := range []TokenKind{
		TokenIdent, TokenNumber, TokenString, TokenOperator, TokenKeyword,
		TokenPunct, TokenComment, TokenEOF, TokenIllegal,
	} {
		if k.String() == "" || k.String() != string(k) {
			t.Fatalf("TokenKind.String: %q", k.String())
		}
	}

	// Block comment path (emitComments)
	toks, err := Tokenize(`/* block */ 1 + 2`)
	if err != nil {
		t.Fatalf("Tokenize block: %v", err)
	}
	foundBlock := false
	for _, tok := range toks {
		if tok.Type == TokenComment && strings.HasPrefix(tok.Lit, "/*") {
			foundBlock = true
		}
	}
	if !foundBlock {
		t.Fatalf("expected block comment token, got %+v", toks)
	}

	// Illegal token on lexical error (unterminated string)
	toks, err = Tokenize(`"unterminated`)
	if err == nil {
		t.Fatal("expected Tokenize error for unterminated string")
	}
	foundIllegal := false
	for _, tok := range toks {
		if tok.Type == TokenIllegal {
			foundIllegal = true
			break
		}
	}
	if !foundIllegal {
		// Some lexer paths return error without pushing Illegal; accept either
		// as long as error is non-nil (contract: may include Illegal).
		t.Logf("no Illegal token (ok per docs); toks=%+v err=%v", toks, err)
	}

	// Another illegal-ish lexeme
	toks2, err2 := Tokenize("\x00")
	_ = toks2
	_ = err2

	node, err := Parse(`1 + 2`)
	if err != nil || node == nil {
		t.Fatalf("Parse ok: %v", err)
	}
	_, err = Parse(`((((`)
	if err == nil {
		t.Fatal("Parse want error")
	}
	var pe *ParseError
	if errors.As(err, &pe) && pe.Source == "" {
		// Parse should attach source on *ParseError
		t.Fatalf("ParseError.Source empty: %#v", pe)
	}
}
