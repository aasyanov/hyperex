package hyperex

// TokenKind is a stable public classification for syntax highlighting and tooling.
// Values are stable strings; do not rely on internal lexer token types.
type TokenKind string

// Public token kinds for [Tokenize].
const (
	TokenIdent    TokenKind = "Ident"
	TokenNumber   TokenKind = "Number"
	TokenString   TokenKind = "String"
	TokenOperator TokenKind = "Operator"
	TokenKeyword  TokenKind = "Keyword"
	TokenPunct    TokenKind = "Punct"
	TokenComment  TokenKind = "Comment"
	TokenEOF      TokenKind = "EOF"
	TokenIllegal  TokenKind = "Illegal"
)

// String returns the kind name.
func (k TokenKind) String() string { return string(k) }

// Token is a stable, opaque view of a lexical token for Studio / highlighting.
// Offset is a 0-based byte index into the source; Line and Column are 1-based.
type Token struct {
	Type   TokenKind
	Lit    string
	Line   int
	Column int
	Offset int
}

// Tokenize returns a public token stream for highlighting and tooling.
// Comments are included. The internal lexer type is not exported.
// On lexical error, the stream may include a trailing Illegal token and a non-nil error.
func Tokenize(src string) ([]Token, error) {
	l := newLexer(src)
	l.emitComments = true
	raw, err := l.tokenize()
	out := make([]Token, 0, len(raw))
	for _, tok := range raw {
		out = append(out, mapPublicToken(tok))
	}
	return out, err
}

// Parse parses an expression into an opaque [Node] suitable for [Walk] / [Transform] / [ASTToString].
// Concrete AST node types remain unexported.
func Parse(expr string) (Node, error) {
	parser := newParser(expr)
	ast, err := parser.parse()
	if err != nil {
		if pe, ok := err.(*ParseError); ok {
			pe.Source = expr
		}
		return nil, err
	}
	return ast, nil
}

func mapPublicToken(tok token) Token {
	lit := tok.Value
	if lit == "" {
		// Operators / punctuation store the glyph in tokenType, not Value.
		kind := publicTokenKind(tok.Type)
		if kind == TokenOperator || kind == TokenPunct || kind == TokenKeyword {
			if name, ok := tokenNames[tok.Type]; ok {
				lit = name
			}
		}
	}
	return Token{
		Type:   publicTokenKind(tok.Type),
		Lit:    lit,
		Line:   tok.Line,
		Column: tok.Column,
		Offset: tok.Offset,
	}
}

func publicTokenKind(t tokenType) TokenKind {
	switch t {
	case tokenEOF:
		return TokenEOF
	case tokenError:
		return TokenIllegal
	case tokenIdent:
		return TokenIdent
	case tokenInt, tokenFloat:
		return TokenNumber
	case tokenString:
		return TokenString
	case tokenBool, tokenNil, tokenLet, tokenIf, tokenElse, tokenIn, tokenNot, tokenAnd, tokenOr,
		tokenTrue, tokenFalse:
		return TokenKeyword
	case tokenLParen, tokenRParen, tokenLBracket, tokenRBracket, tokenLCurly, tokenRCurly,
		tokenComma, tokenDot, tokenColon, tokenSemi:
		return TokenPunct
	case tokenComment:
		return TokenComment
	default:
		// Remaining token types are operators / delimiters used as operators.
		return TokenOperator
	}
}
