package hyperex

import "fmt"

// token represents a lexical token with its type, value, and position.
type token struct {
	Type   tokenType
	Value  string
	Line   int
	Column int
	Offset int // 0-based byte offset into the source
}

// String returns a human-readable representation of the token.
func (t token) String() string {
	if t.Value != "" {
		return fmt.Sprintf("%s(%q)", t.Type, t.Value)
	}
	return t.Type.String()
}

// tokenType represents the type of a lexical token.
type tokenType int

const (
	// Special tokens
	tokenEOF tokenType = iota
	tokenError
	tokenIdent
	tokenInt
	tokenFloat
	tokenString
	tokenBool
	tokenNil

	// Operators - arithmetic
	tokenPlus     // +
	tokenMinus    // -
	tokenStar     // *
	tokenSlash    // /
	tokenPercent  // %
	tokenPower    // **
	tokenPlusPlus // ++
	tokenMinMin   // --

	// Operators - assignment
	tokenAssign // =

	// Operators - comparison
	tokenEQ   // ==
	tokenNE   // !=
	tokenLT   // <
	tokenLE   // <=
	tokenGT   // >
	tokenGE   // >=
	tokenIn   // in
	tokenNot  // not (also !)
	tokenLike // ~= (regex match)

	// Operators - logical
	tokenAnd // && or "and"
	tokenOr  // || or "or"
	tokenBang

	// Operators - bitwise
	tokenAmp      // &
	tokenPipe     // |
	tokenCaret    // ^
	tokenTilde    // ~
	tokenShiftL   // <<
	tokenShiftR   // >>
	tokenShiftRU  // >>> (unsigned)
	tokenAmpAmp   // && (also logical and)
	tokenPipePipe // || (also logical or)
	tokenQuestion // ?
	tokenCoalesce // ??
	tokenOptChain // ?.
	tokenOptIndex // ?.[

	// Delimiters
	tokenLParen   // (
	tokenRParen   // )
	tokenLBracket // [
	tokenRBracket // ]
	tokenLCurly   // {
	tokenRCurly   // }
	tokenComma    // ,
	tokenDot      // .
	tokenColon    // :
	tokenSemi     // ;
	tokenArrow    // ->
	tokenFatArrow // =>
	tokenRange    // ..
	tokenSpread   // ...

	// keywords
	tokenLet
	tokenIf
	tokenElse
	tokenTrue
	tokenFalse

	// comment (emitted only when lexer.emitComments is set; skipped for parse)
	tokenComment
)

var tokenNames = map[tokenType]string{
	tokenEOF:      "EOF",
	tokenError:    "Error",
	tokenIdent:    tokIdent,
	tokenInt:      "Int",
	tokenFloat:    "Float",
	tokenString:   "String",
	tokenBool:     "Bool",
	tokenNil:      "Nil",
	tokenAssign:   "=",
	tokenPlus:     "+",
	tokenMinus:    "-",
	tokenStar:     "*",
	tokenSlash:    "/",
	tokenPercent:  "%",
	tokenPower:    "**",
	tokenPlusPlus: "++",
	tokenMinMin:   "--",
	tokenEQ:       "==",
	tokenNE:       "!=",
	tokenLT:       "<",
	tokenLE:       "<=",
	tokenGT:       ">",
	tokenGE:       ">=",
	tokenIn:       "in",
	tokenNot:      kwNot,
	tokenLike:     "~=",
	tokenAnd:      kwAnd,
	tokenOr:       "or",
	tokenBang:     "!",
	tokenAmp:      "&",
	tokenPipe:     "|",
	tokenCaret:    "^",
	tokenTilde:    "~",
	tokenShiftL:   "<<",
	tokenShiftR:   ">>",
	tokenShiftRU:  litOpShrU,
	tokenAmpAmp:   "&&",
	tokenPipePipe: "||",
	tokenQuestion: "?",
	tokenCoalesce: "??",
	tokenOptChain: "?.",
	tokenOptIndex: "?.[",
	tokenLParen:   "(",
	tokenRParen:   ")",
	tokenLBracket: "[",
	tokenRBracket: "]",
	tokenLCurly:   "{",
	tokenRCurly:   "}",
	tokenComma:    ",",
	tokenDot:      ".",
	tokenColon:    ":",
	tokenSemi:     ";",
	tokenArrow:    "->",
	tokenFatArrow: "=>",
	tokenRange:    "..",
	tokenSpread:   "...",
	tokenLet:      kwLet,
	tokenIf:       "if",
	tokenElse:     kwElse,
	tokenTrue:     litTrue,
	tokenFalse:    litFalse,
	tokenComment:  "Comment",
}

func (t tokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("token(%d)", t)
}

// keywords maps string literals to their corresponding keyword tokens.
var keywords = map[string]tokenType{
	litTrue:  tokenTrue,
	litFalse: tokenFalse,
	litNil:   tokenNil,
	"in":     tokenIn,
	kwNot:    tokenNot,
	kwAnd:    tokenAnd,
	"or":     tokenOr,
	kwLet:    tokenLet,
	"if":     tokenIf,
	kwElse:   tokenElse,
}

// Precedence returns the operator precedence for the given token.
// Higher values mean higher precedence (binds tighter).
func (t tokenType) Precedence() int {
	switch t {
	case tokenOr, tokenPipePipe:
		return 1
	case tokenAnd, tokenAmpAmp:
		return 2
	case tokenPipe: // bitwise OR
		return 3
	case tokenCaret: // bitwise XOR
		return 4
	case tokenAmp: // bitwise AND
		return 5
	case tokenEQ, tokenNE:
		return 6
	case tokenLT, tokenLE, tokenGT, tokenGE, tokenIn, tokenLike:
		return 7
	case tokenShiftL, tokenShiftR, tokenShiftRU:
		return 8
	case tokenPlus, tokenMinus:
		return 9
	case tokenStar, tokenSlash, tokenPercent:
		return 10
	case tokenPower:
		return 11
	case tokenCoalesce:
		return 12
	case tokenRange:
		return 13
	default:
		return 0
	}
}

// IsRightAssociative returns true if the operator is right-associative.
func (t tokenType) IsRightAssociative() bool {
	return t == tokenPower || t == tokenCoalesce
}
