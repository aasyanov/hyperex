package hyperex

import (
	"unicode"
	"unicode/utf8"
)

// lexer tokenizes an expression string into a sequence of tokens.
// It properly handles Unicode characters by working with runes.
type lexer struct {
	input  string
	pos    int // current byte position in input
	width  int // width of last rune read
	line   int // current line number (1-based)
	column int // current column (1-based, in runes)

	// Lookahead buffer
	tokens []token
	idx    int
}

// newLexer creates a new lexer for the given input string.
func newLexer(input string) *lexer {
	return &lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
		tokens: make([]token, 0, 32),
	}
}

// Tokenize processes the entire input and returns all tokens.
func (l *lexer) tokenize() ([]token, error) {
	for {
		tok := l.nextToken()
		l.tokens = append(l.tokens, tok)
		if tok.Type == tokenEOF {
			break
		}
		if tok.Type == tokenError {
			return l.tokens, &LexerError{
				Message: tok.Value,
				Line:    tok.Line,
				Column:  tok.Column,
				Source:  l.input, // Include source for better error messages
			}
		}
	}
	return l.tokens, nil
}

// Next returns the next token.
func (l *lexer) Next() token {
	if l.idx < len(l.tokens) {
		tok := l.tokens[l.idx]
		l.idx++
		return tok
	}

	tok := l.nextToken()
	l.tokens = append(l.tokens, tok)
	l.idx++
	return tok
}

// Peek returns the next token without consuming it.
func (l *lexer) Peek() token {
	if l.idx < len(l.tokens) {
		return l.tokens[l.idx]
	}

	tok := l.nextToken()
	l.tokens = append(l.tokens, tok)
	return tok
}

// PeekN returns the n-th token ahead without consuming any tokens.
func (l *lexer) PeekN(n int) token {
	target := l.idx + n - 1
	for len(l.tokens) <= target {
		tok := l.nextToken()
		l.tokens = append(l.tokens, tok)
		if tok.Type == tokenEOF || tok.Type == tokenError {
			break
		}
	}
	if target < len(l.tokens) {
		return l.tokens[target]
	}
	return l.tokens[len(l.tokens)-1]
}

// nextToken scans and returns the next token from input.
func (l *lexer) nextToken() token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return token{Type: tokenEOF, Line: l.line, Column: l.column}
	}

	startLine := l.line
	startCol := l.column

	ch := l.current()

	// Check for single-line comments
	if ch == '/' && l.peek() == '/' {
		l.skipLineComment()
		return l.nextToken()
	}

	// Check for multi-line comments
	if ch == '/' && l.peek() == '*' {
		if err := l.skipBlockComment(); err != nil {
			return token{Type: tokenError, Value: err.Error(), Line: startLine, Column: startCol}
		}
		return l.nextToken()
	}

	// Identifiers and keywords
	if isIdentStart(ch) {
		return l.scanIdent(startLine, startCol)
	}

	// Numbers
	if isDigit(ch) || (ch == '.' && isDigit(l.peek())) {
		return l.scanNumber(startLine, startCol)
	}

	// Strings
	if ch == '"' || ch == '\'' || ch == '`' {
		return l.scanString(ch, startLine, startCol)
	}

	// Operators and delimiters
	return l.scanOperator(startLine, startCol)
}

func (l *lexer) scanIdent(line, col int) token {
	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.current()) {
		l.advance()
	}
	value := l.input[start:l.pos]

	// Check for keywords
	if tokType, ok := keywords[value]; ok {
		if tokType == tokenTrue || tokType == tokenFalse {
			return token{Type: tokenBool, Value: value, Line: line, Column: col}
		}
		if tokType == tokenNil || tokType == tokenNull {
			return token{Type: tokenNil, Value: value, Line: line, Column: col}
		}
		return token{Type: tokType, Value: value, Line: line, Column: col}
	}

	return token{Type: tokenIdent, Value: value, Line: line, Column: col}
}

func (l *lexer) scanNumber(line, col int) token {
	start := l.pos
	isFloat := false

	// Integer part
	for l.pos < len(l.input) && isDigit(l.current()) {
		l.advance()
	}

	// Hex, octal, binary
	if l.pos == start+1 && l.input[start] == '0' && l.pos < len(l.input) {
		ch := l.current()
		if ch == 'x' || ch == 'X' {
			l.advance()
			for l.pos < len(l.input) && isHexDigit(l.current()) {
				l.advance()
			}
			return token{Type: tokenInt, Value: l.input[start:l.pos], Line: line, Column: col}
		}
		if ch == 'b' || ch == 'B' {
			l.advance()
			for l.pos < len(l.input) && (l.current() == '0' || l.current() == '1') {
				l.advance()
			}
			return token{Type: tokenInt, Value: l.input[start:l.pos], Line: line, Column: col}
		}
		if ch == 'o' || ch == 'O' {
			l.advance()
			for l.pos < len(l.input) && l.current() >= '0' && l.current() <= '7' {
				l.advance()
			}
			return token{Type: tokenInt, Value: l.input[start:l.pos], Line: line, Column: col}
		}
	}

	// Decimal part
	if l.pos < len(l.input) && l.current() == '.' && isDigit(l.peek()) {
		isFloat = true
		l.advance() // consume '.'
		for l.pos < len(l.input) && isDigit(l.current()) {
			l.advance()
		}
	}

	// Exponent part
	if l.pos < len(l.input) && (l.current() == 'e' || l.current() == 'E') {
		isFloat = true
		l.advance()
		if l.pos < len(l.input) && (l.current() == '+' || l.current() == '-') {
			l.advance()
		}
		for l.pos < len(l.input) && isDigit(l.current()) {
			l.advance()
		}
	}

	value := l.input[start:l.pos]
	if isFloat {
		return token{Type: tokenFloat, Value: value, Line: line, Column: col}
	}
	return token{Type: tokenInt, Value: value, Line: line, Column: col}
}

func (l *lexer) scanString(quote rune, line, col int) token {
	l.advance() // consume opening quote
	start := l.pos
	var result []byte

	for l.pos < len(l.input) {
		ch := l.current()
		if ch == quote {
			if result != nil {
				l.advance()
				return token{Type: tokenString, Value: string(result), Line: line, Column: col}
			}
			value := l.input[start:l.pos]
			l.advance()
			return token{Type: tokenString, Value: value, Line: line, Column: col}
		}

		if ch == '\\' && quote != '`' {
			if result == nil {
				result = make([]byte, 0, l.pos-start+16)
				result = append(result, l.input[start:l.pos]...)
			}
			l.advance()
			if l.pos >= len(l.input) {
				return token{Type: tokenError, Value: "unterminated string escape", Line: line, Column: col}
			}
			escaped := l.current()
			switch escaped {
			case 'n':
				result = append(result, '\n')
			case 'r':
				result = append(result, '\r')
			case 't':
				result = append(result, '\t')
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			case '\'':
				result = append(result, '\'')
			case '0':
				result = append(result, 0)
			case 'u':
				// Unicode escape: \uXXXX
				l.advance()
				if l.pos+4 > len(l.input) {
					return token{Type: tokenError, Value: "invalid unicode escape", Line: line, Column: col}
				}
				hex := l.input[l.pos : l.pos+4]
				var r rune
				for _, c := range hex {
					r <<= 4
					switch {
					case c >= '0' && c <= '9':
						r += c - '0'
					case c >= 'a' && c <= 'f':
						r += c - 'a' + 10
					case c >= 'A' && c <= 'F':
						r += c - 'A' + 10
					default:
						return token{Type: tokenError, Value: "invalid unicode escape", Line: line, Column: col}
					}
				}
				result = append(result, string(r)...)
				l.pos += 3 // Will advance one more below
				l.column += 3
			default:
				// For unknown escapes, include the character as-is
				result = append(result, string(escaped)...)
			}
			l.advance()
			start = l.pos
			continue
		}

		if ch == '\n' && quote != '`' {
			return token{Type: tokenError, Value: "unterminated string", Line: line, Column: col}
		}

		if result != nil {
			result = append(result, string(ch)...)
		}
		l.advance()
	}

	return token{Type: tokenError, Value: "unterminated string", Line: line, Column: col}
}

func (l *lexer) scanOperator(line, col int) token {
	ch := l.current()
	l.advance()

	switch ch {
	case '+':
		if l.current() == '+' {
			l.advance()
			return token{Type: tokenPlusPlus, Line: line, Column: col}
		}
		return token{Type: tokenPlus, Line: line, Column: col}

	case '-':
		if l.current() == '-' {
			l.advance()
			return token{Type: tokenMinMin, Line: line, Column: col}
		}
		if l.current() == '>' {
			l.advance()
			return token{Type: tokenArrow, Line: line, Column: col}
		}
		return token{Type: tokenMinus, Line: line, Column: col}

	case '*':
		if l.current() == '*' {
			l.advance()
			return token{Type: tokenPower, Line: line, Column: col}
		}
		return token{Type: tokenStar, Line: line, Column: col}

	case '/':
		return token{Type: tokenSlash, Line: line, Column: col}

	case '%':
		return token{Type: tokenPercent, Line: line, Column: col}

	case '=':
		if l.current() == '=' {
			l.advance()
			return token{Type: tokenEQ, Line: line, Column: col}
		}
		if l.current() == '>' {
			l.advance()
			return token{Type: tokenFatArrow, Line: line, Column: col}
		}
		return token{Type: tokenAssign, Line: line, Column: col}

	case '!':
		if l.current() == '=' {
			l.advance()
			return token{Type: tokenNE, Line: line, Column: col}
		}
		return token{Type: tokenBang, Line: line, Column: col}

	case '<':
		if l.current() == '=' {
			l.advance()
			return token{Type: tokenLE, Line: line, Column: col}
		}
		if l.current() == '<' {
			l.advance()
			return token{Type: tokenShiftL, Line: line, Column: col}
		}
		return token{Type: tokenLT, Line: line, Column: col}

	case '>':
		if l.current() == '=' {
			l.advance()
			return token{Type: tokenGE, Line: line, Column: col}
		}
		if l.current() == '>' {
			l.advance()
			if l.current() == '>' {
				l.advance()
				return token{Type: tokenShiftRU, Line: line, Column: col}
			}
			return token{Type: tokenShiftR, Line: line, Column: col}
		}
		return token{Type: tokenGT, Line: line, Column: col}

	case '&':
		if l.current() == '&' {
			l.advance()
			return token{Type: tokenAmpAmp, Line: line, Column: col}
		}
		return token{Type: tokenAmp, Line: line, Column: col}

	case '|':
		if l.current() == '|' {
			l.advance()
			return token{Type: tokenPipePipe, Line: line, Column: col}
		}
		return token{Type: tokenPipe, Line: line, Column: col}

	case '^':
		return token{Type: tokenCaret, Line: line, Column: col}

	case '~':
		if l.current() == '=' {
			l.advance()
			return token{Type: tokenLike, Line: line, Column: col}
		}
		return token{Type: tokenTilde, Line: line, Column: col}

	case '?':
		if l.current() == '?' {
			l.advance()
			return token{Type: tokenCoalesce, Line: line, Column: col}
		}
		if l.current() == '.' {
			l.advance()
			if l.current() == '[' {
				l.advance()
				return token{Type: tokenOptIndex, Line: line, Column: col}
			}
			return token{Type: tokenOptChain, Line: line, Column: col}
		}
		return token{Type: tokenQuestion, Line: line, Column: col}

	case '.':
		if l.current() == '.' {
			l.advance()
			if l.current() == '.' {
				l.advance()
				return token{Type: tokenSpread, Line: line, Column: col}
			}
			return token{Type: tokenRange, Line: line, Column: col}
		}
		return token{Type: tokenDot, Line: line, Column: col}

	case '(':
		return token{Type: tokenLParen, Line: line, Column: col}
	case ')':
		return token{Type: tokenRParen, Line: line, Column: col}
	case '[':
		return token{Type: tokenLBracket, Line: line, Column: col}
	case ']':
		return token{Type: tokenRBracket, Line: line, Column: col}
	case '{':
		return token{Type: tokenLCurly, Line: line, Column: col}
	case '}':
		return token{Type: tokenRCurly, Line: line, Column: col}
	case ',':
		return token{Type: tokenComma, Line: line, Column: col}
	case ':':
		return token{Type: tokenColon, Line: line, Column: col}
	case ';':
		return token{Type: tokenSemi, Line: line, Column: col}
	}

	return token{Type: tokenError, Value: "unexpected character: " + string(ch), Line: line, Column: col}
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.current()
		switch {
		case ch == ' ' || ch == '\t' || ch == '\r':
			l.advance()
		case ch == '\n':
			l.advance()
			l.line++
			l.column = 1
		case unicode.IsSpace(ch):
			// Handle other Unicode whitespace characters
			l.advance()
		default:
			return
		}
	}
}

func (l *lexer) skipLineComment() {
	for l.pos < len(l.input) && l.current() != '\n' {
		l.advance()
	}
}

func (l *lexer) skipBlockComment() error {
	l.advance() // skip /
	l.advance() // skip *

	for l.pos < len(l.input) {
		if l.current() == '*' && l.peek() == '/' {
			l.advance()
			l.advance()
			return nil
		}
		if l.current() == '\n' {
			l.line++
			// columns are 1-based; advance() below increments column once,
			// so set to 0 here to land on 1 at the start of the next line.
			l.column = 0
		}
		l.advance()
	}
	return &LexerError{Message: "unterminated block comment", Line: l.line, Column: l.column}
}

// current returns the current rune without advancing.
func (l *lexer) current() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return 0
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	return r
}

// peek returns the next rune without advancing.
func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	_, w := utf8.DecodeRuneInString(l.input[l.pos:])
	if l.pos+w >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos+w:])
	return r
}

// advance moves to the next rune.
func (l *lexer) advance() {
	if l.pos < len(l.input) {
		_, w := utf8.DecodeRuneInString(l.input[l.pos:])
		l.pos += w
		l.column++
	}
}

// isIdentStart checks if a rune can start an identifier.
// Supports Unicode letters (e.g., Cyrillic, Chinese, etc.)
func isIdentStart(ch rune) bool {
	return ch == '_' || ch == '$' || ch == '#' || unicode.IsLetter(ch)
}

// isIdentChar checks if a rune can be part of an identifier.
func isIdentChar(ch rune) bool {
	return isIdentStart(ch) || unicode.IsDigit(ch)
}

// isDigit checks if a rune is an ASCII digit.
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// isHexDigit checks if a rune is a hexadecimal digit.
func isHexDigit(ch rune) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
