package hyperex

import (
	"strconv"
)

// parser transforms a sequence of tokens into an AST.
type parser struct {
	lexer *lexer
	curr  token
	prev  token
}

// newParser creates a new parser for the given input string.
func newParser(input string) *parser {
	p := &parser{lexer: newLexer(input)}
	p.advance() // initialize curr
	return p
}

// Parse parses the input and returns the root AST node.
func (p *parser) parse() (Node, error) {
	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.curr.Type != tokenEOF {
		return nil, &ParseError{
			Message: "unexpected token after expression",
			token:   p.curr,
		}
	}

	return node, nil
}

func (p *parser) advance() token {
	p.prev = p.curr
	p.curr = p.lexer.Next()
	return p.prev
}

func (p *parser) check(t tokenType) bool {
	return p.curr.Type == t
}

func (p *parser) match(types ...tokenType) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *parser) expect(t tokenType) (token, error) {
	if p.check(t) {
		return p.advance(), nil
	}
	return token{}, &ParseError{
		Message: "expected " + t.String(),
		token:   p.curr,
	}
}

// Expression parsing using Pratt parser (precedence climbing)
func (p *parser) parseExpression() (Node, error) {
	return p.parsePrecedence(1)
}

func (p *parser) parsePrecedence(minPrec int) (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		// Handle range operator (..) at precedence 13, left-associative.
		// Respect minPrec so binding matches the precedence table:
		//   1+2..4  →  1+(2..4)
		//   1..2+3  →  (1..2)+3
		//   1..2..3 →  (1..2)..3
		if p.curr.Type == tokenRange {
			if tokenRange.Precedence() < minPrec {
				break
			}
			p.advance()
			right, err := p.parsePrecedence(tokenRange.Precedence() + 1)
			if err != nil {
				return nil, err
			}
			left = &rangeExpr{Start: left, End: right}
			continue
		}

		// Handle ternary - ternary has very low precedence (lower than comparison operators)
		// Only parse ternary at low precedence levels (minPrec <= 1) to ensure
		// expressions like "x == y ? a : b" are parsed as "(x == y) ? a : b"
		// not as "x == (y ? a : b)"
		if p.curr.Type == tokenQuestion && minPrec <= 1 {
			p.advance()
			then, thenErr := p.parseExpression()
			if thenErr != nil {
				return nil, thenErr
			}
			if _, colonErr := p.expect(tokenColon); colonErr != nil {
				return nil, colonErr
			}
			elseExpr, elseErr := p.parsePrecedence(1) // Parse with low precedence
			if elseErr != nil {
				return nil, elseErr
			}
			left = &ternaryExpr{Cond: left, Then: then, Else: elseExpr}
			continue
		}

		// Handle 'not in' as a binary operator (special case before precedence check)
		if p.curr.Type == tokenNot && p.lexer.Peek().Type == tokenIn {
			p.advance() // consume 'not'
			p.advance() // consume 'in'
			right, err := p.parsePrecedence(tokenIn.Precedence() + 1)
			if err != nil {
				return nil, err
			}
			left = &binaryExpr{Op: opNotIn, Left: left, Right: right}
			continue
		}

		prec := p.curr.Type.Precedence()
		if prec < minPrec {
			break
		}

		op := p.advance()
		nextPrec := prec
		if !op.Type.IsRightAssociative() {
			nextPrec = prec + 1
		}

		// Handle coalesce
		if op.Type == tokenCoalesce {
			right, err := p.parsePrecedence(nextPrec)
			if err != nil {
				return nil, err
			}
			left = &coalesceExpr{Left: left, Right: right}
			continue
		}

		// Handle pipe operator: arr | func
		//
		// DISAMBIGUATION RULES for | operator (pipe vs bitwise OR):
		// The parser uses heuristics to determine intent (documented in LANGUAGE.md §9):
		//
		// 1. Integer/float literals on either side → never PIPE (always bitwise OR).
		//
		// 2. If left is clearly a collection/value (arrayLit, callExpr, memberExpr,
		//    indexExpr, sliceExpr, pipeExpr, rangeExpr, stringLit) AND right is
		//    ident or callExpr → PIPE
		//
		// 3. If left is ident AND right is callExpr → PIPE
		//    (e.g., "data | filter(...)" is a pipe)
		//
		// 4. If left is ident AND right is ident whose name matches a default
		//    builtin (same keys as defaultBuiltins: sum, map, filter, trim, len, …)
		//    → PIPE. Bitwise OR with those names is vanishingly rare; force with
		//    intermediate bindings (let a = xs; let b = mask; a | b) if needed.
		//
		// 5. Otherwise ident | ident → BITWISE OR (e.g., "x | y", "flags | MASK").
		//
		// To force pipe when bitwise OR would be chosen, use an explicit call:
		//    "x | transform()" instead of "x | transform"
		if op.Type == tokenPipe {
			right, err := p.parsePrecedence(nextPrec)
			if err != nil {
				return nil, err
			}
			isPipe := false
			if !isNumericLit(left) && !isNumericLit(right) {
				switch left.(type) {
				case *arrayLit, *callExpr, *memberExpr, *indexExpr, *sliceExpr, *pipeExpr, *rangeExpr, *stringLit:
					switch right.(type) {
					case *ident, *callExpr:
						isPipe = true
					}
				case *ident:
					switch r := right.(type) {
					case *callExpr:
						isPipe = true
					case *ident:
						// Builtin-name rule: xs | sum, data | unique, …
						if isDefaultBuiltinName(r.Name) {
							isPipe = true
						}
					}
				}
			}
			if isPipe {
				left = &pipeExpr{Left: left, Right: right}
			} else {
				left = &binaryExpr{Op: "|", Left: left, Right: right}
			}
			continue
		}

		opStr := tokenToOp(op.Type)

		right, err := p.parsePrecedence(nextPrec)
		if err != nil {
			return nil, err
		}

		left = &binaryExpr{Op: opStr, Left: left, Right: right}
	}

	return left, nil
}

func (p *parser) parseUnary() (Node, error) {
	if p.match(tokenMinus, tokenBang, tokenNot, tokenTilde) {
		op := p.prev
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryExpr{Op: tokenToOp(op.Type), Operand: operand}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Node, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		switch {
		case p.match(tokenDot):
			// Member access: obj.field
			name, err := p.expect(tokenIdent)
			if err != nil {
				return nil, err
			}
			node = &memberExpr{
				Object:   node,
				Property: &ident{Name: name.Value},
				Computed: false,
			}

		case p.match(tokenOptChain):
			// Optional member access: obj?.field
			name, err := p.expect(tokenIdent)
			if err != nil {
				return nil, err
			}
			node = &memberExpr{
				Object:   node,
				Property: &ident{Name: name.Value},
				Computed: false,
				Optional: true,
			}

		case p.match(tokenLBracket):
			// Index access: arr[index] or slice: arr[low:high]
			index, err := p.parseSliceOrIndex(node)
			if err != nil {
				return nil, err
			}
			node = index

		case p.match(tokenOptIndex):
			// Optional index: arr?.[index]
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(tokenRBracket); err != nil {
				return nil, err
			}
			node = &indexExpr{Object: node, Index: index, Optional: true}

		case p.match(tokenLParen):
			// Function call: func(args)
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			node = &callExpr{Func: node, Args: args}

		default:
			return node, nil
		}
	}
}

func (p *parser) parseSliceOrIndex(obj Node) (Node, error) {
	// Check for [:] or [::] slice
	if p.check(tokenColon) {
		return p.parseSlice(obj, nil)
	}

	first, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Check for slice: [low:high] or [low:high:step]
	if p.check(tokenColon) {
		return p.parseSlice(obj, first)
	}

	// Simple index
	if _, err := p.expect(tokenRBracket); err != nil {
		return nil, err
	}
	return &indexExpr{Object: obj, Index: first}, nil
}

func (p *parser) parseSlice(obj, low Node) (Node, error) {
	p.advance() // consume first ':'

	var high, step Node
	var err error

	// Parse high bound
	if !p.check(tokenRBracket) && !p.check(tokenColon) {
		high, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}

	// Parse step
	if p.match(tokenColon) {
		if !p.check(tokenRBracket) {
			step, err = p.parseExpression()
			if err != nil {
				return nil, err
			}
		}
	}

	if _, err := p.expect(tokenRBracket); err != nil {
		return nil, err
	}

	return &sliceExpr{Object: obj, Low: low, High: high, Step: step}, nil
}

func (p *parser) parseArguments() ([]Node, error) {
	var args []Node

	if !p.check(tokenRParen) {
		for {
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if !p.match(tokenComma) {
				break
			}
		}
	}

	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}

	return args, nil
}

func (p *parser) parsePrimary() (Node, error) {
	switch {
	case p.match(tokenInt):
		val, err := strconv.ParseInt(p.prev.Value, 0, 64)
		if err != nil {
			return nil, &ParseError{Message: "invalid integer", token: p.prev}
		}
		return &intLit{Value: val}, nil

	case p.match(tokenFloat):
		val, err := strconv.ParseFloat(p.prev.Value, 64)
		if err != nil {
			return nil, &ParseError{Message: "invalid float", token: p.prev}
		}
		return &floatLit{Value: val}, nil

	case p.match(tokenString):
		return &stringLit{Value: p.prev.Value}, nil

	case p.match(tokenBool):
		return &boolLit{Value: p.prev.Value == litTrue}, nil

	case p.match(tokenNil):
		return &nilLit{}, nil

	case p.match(tokenIdent):
		name := p.prev.Value

		// Check for lambda: x => expr or (x, y) => expr
		if p.check(tokenFatArrow) {
			p.advance()
			body, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			return &lambdaExpr{Params: []string{name}, Body: body}, nil
		}

		return &ident{Name: name}, nil

	case p.match(tokenLet):
		return p.parseLet()

	case p.match(tokenLParen):
		// Could be grouping or lambda params
		return p.parseParenOrLambda()

	case p.match(tokenLBracket):
		return p.parseArray()

	case p.match(tokenLCurly):
		return p.parseMap()

	default:
		return nil, &ParseError{
			Message: "unexpected token: " + p.curr.Type.String(),
			token:   p.curr,
		}
	}
}

func (p *parser) parseLet() (Node, error) {
	// Check for object destructuring: let {a, b} = ...
	if p.check(tokenLCurly) {
		return p.parseObjectDestructure()
	}

	// Check for array destructuring: let [a, b] = ...
	if p.check(tokenLBracket) {
		return p.parseArrayDestructure()
	}

	// Standard let: let name = value
	name, err := p.expect(tokenIdent)
	if err != nil {
		return nil, err
	}

	// Expect '=' after identifier
	if _, assignErr := p.expect(tokenAssign); assignErr != nil {
		return nil, &ParseError{
			Message: "expected '=' after variable name in let expression",
			token:   p.curr,
		}
	}

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Expect ';' before body
	if !p.match(tokenSemi) {
		// If no semicolon, the let is the final expression
		// Note: `let x = 5` returns 5 and binds x in the environment
		return &letExpr{Name: name.Value, Value: value, Body: value}, nil
	}

	body, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &letExpr{Name: name.Value, Value: value, Body: body}, nil
}

func (p *parser) parseObjectDestructure() (Node, error) {
	p.advance() // consume '{'

	var pattern []destructureField

	if !p.check(tokenRCurly) {
		for {
			// Parse field: name or name: alias or name = default
			key, err := p.expect(tokenIdent)
			if err != nil {
				return nil, err
			}

			field := destructureField{Key: key.Value, Name: key.Value}

			// Check for alias: key: alias
			if p.match(tokenColon) {
				alias, err := p.expect(tokenIdent)
				if err != nil {
					return nil, err
				}
				field.Name = alias.Value
			}

			// Check for default: name = default
			if p.match(tokenAssign) {
				def, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				field.Default = def
			}

			pattern = append(pattern, field)

			if !p.match(tokenComma) {
				break
			}
			if p.check(tokenRCurly) {
				break
			}
		}
	}

	if _, err := p.expect(tokenRCurly); err != nil {
		return nil, err
	}

	if _, err := p.expect(tokenAssign); err != nil {
		return nil, &ParseError{Message: "expected '=' after destructure pattern", token: p.curr}
	}

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Match plain let: if no ';', the destructure is the final expression
	// (Body = Value). If ';' is present, parse the body expression.
	if !p.match(tokenSemi) {
		return &destructureExpr{Pattern: pattern, IsArray: false, Value: value, Body: value}, nil
	}

	body, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &destructureExpr{Pattern: pattern, IsArray: false, Value: value, Body: body}, nil
}

func (p *parser) parseArrayDestructure() (Node, error) {
	p.advance() // consume '['

	var pattern []destructureField

	if !p.check(tokenRBracket) {
		for {
			name, err := p.expect(tokenIdent)
			if err != nil {
				return nil, err
			}

			field := destructureField{Name: name.Value}

			// Check for default
			if p.match(tokenAssign) {
				def, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				field.Default = def
			}

			pattern = append(pattern, field)

			if !p.match(tokenComma) {
				break
			}
			if p.check(tokenRBracket) {
				break
			}
		}
	}

	if _, err := p.expect(tokenRBracket); err != nil {
		return nil, err
	}

	if _, err := p.expect(tokenAssign); err != nil {
		return nil, &ParseError{Message: "expected '=' after destructure pattern", token: p.curr}
	}

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Match plain let: if no ';', the destructure is the final expression
	// (Body = Value). If ';' is present, parse the body expression.
	if !p.match(tokenSemi) {
		return &destructureExpr{Pattern: pattern, IsArray: true, Value: value, Body: value}, nil
	}

	body, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &destructureExpr{Pattern: pattern, IsArray: true, Value: value, Body: body}, nil
}

func (p *parser) parseParenOrLambda() (Node, error) {
	// Empty parens followed by => is a no-arg lambda
	if p.check(tokenRParen) {
		p.advance()
		if p.match(tokenFatArrow) {
			body, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			return &lambdaExpr{Params: nil, Body: body}, nil
		}
		// Empty parens with no arrow is invalid
		return nil, &ParseError{Message: "empty parentheses", token: p.curr}
	}

	// Parse first expression
	first, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Check for comma (lambda params)
	if p.check(tokenComma) {
		params, paramsErr := p.parseLambdaParams(first)
		if paramsErr != nil {
			return nil, paramsErr
		}
		if _, rparenErr := p.expect(tokenRParen); rparenErr != nil {
			return nil, rparenErr
		}
		if _, arrowErr := p.expect(tokenFatArrow); arrowErr != nil {
			return nil, arrowErr
		}
		body, bodyErr := p.parseExpression()
		if bodyErr != nil {
			return nil, bodyErr
		}
		return &lambdaExpr{Params: params, Body: body}, nil
	}

	// Regular grouping
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}

	// Check for lambda: (x) => expr
	if p.match(tokenFatArrow) {
		ident, ok := first.(*ident)
		if !ok {
			return nil, &ParseError{Message: "expected identifier in lambda", token: p.prev}
		}
		body, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &lambdaExpr{Params: []string{ident.Name}, Body: body}, nil
	}

	return first, nil
}

func (p *parser) parseLambdaParams(first Node) ([]string, error) {
	ident, ok := first.(*ident)
	if !ok {
		return nil, &ParseError{Message: "expected identifier in lambda parameters", token: p.prev}
	}

	params := []string{ident.Name}

	for p.match(tokenComma) {
		name, err := p.expect(tokenIdent)
		if err != nil {
			return nil, err
		}
		params = append(params, name.Value)
	}

	return params, nil
}

func (p *parser) parseArray() (Node, error) {
	var elements []Node

	if !p.check(tokenRBracket) {
		for {
			var elem Node
			var err error

			// Check for spread operator
			if p.match(tokenSpread) {
				spreadNode, spreadErr := p.parseExpression()
				if spreadErr != nil {
					return nil, spreadErr
				}
				elem = &spreadExpr{Expr: spreadNode}
			} else {
				elem, err = p.parseExpression()
				if err != nil {
					return nil, err
				}
			}
			elements = append(elements, elem)

			if !p.match(tokenComma) {
				break
			}

			// Allow trailing comma
			if p.check(tokenRBracket) {
				break
			}
		}
	}

	if _, err := p.expect(tokenRBracket); err != nil {
		return nil, err
	}

	return &arrayLit{Elements: elements}, nil
}

func (p *parser) parseMap() (Node, error) {
	var keys, values []Node

	if !p.check(tokenRCurly) {
		for {
			// Key can be string, identifier, or expression in brackets
			var key Node

			switch {
			case p.check(tokenIdent):
				// Identifier key: {foo: value}
				name := p.advance()
				key = &stringLit{Value: name.Value}
			case p.check(tokenString):
				// String key: {"foo": value}
				p.advance()
				key = &stringLit{Value: p.prev.Value}
			case p.check(tokenLBracket):
				// Computed key: {[expr]: value}
				p.advance()
				var keyErr error
				key, keyErr = p.parseExpression()
				if keyErr != nil {
					return nil, keyErr
				}
				if _, bracketErr := p.expect(tokenRBracket); bracketErr != nil {
					return nil, bracketErr
				}
			default:
				return nil, &ParseError{Message: "expected map key", token: p.curr}
			}

			if _, colonErr := p.expect(tokenColon); colonErr != nil {
				return nil, colonErr
			}

			value, valueErr := p.parseExpression()
			if valueErr != nil {
				return nil, valueErr
			}

			keys = append(keys, key)
			values = append(values, value)

			if !p.match(tokenComma) {
				break
			}

			// Allow trailing comma
			if p.check(tokenRCurly) {
				break
			}
		}
	}

	if _, err := p.expect(tokenRCurly); err != nil {
		return nil, err
	}

	return &mapLit{Keys: keys, Values: values}, nil
}

func tokenToOp(t tokenType) string {
	switch t {
	case tokenPlus:
		return "+"
	case tokenMinus:
		return "-"
	case tokenStar:
		return "*"
	case tokenSlash:
		return "/"
	case tokenPercent:
		return "%"
	case tokenPower:
		return "**"
	case tokenEQ:
		return "=="
	case tokenNE:
		return "!="
	case tokenLT:
		return "<"
	case tokenLE:
		return "<="
	case tokenGT:
		return ">"
	case tokenGE:
		return ">="
	case tokenAnd, tokenAmpAmp:
		return "&&"
	case tokenOr, tokenPipePipe:
		return "||"
	case tokenBang, tokenNot:
		return "!"
	case tokenAmp:
		return "&"
	case tokenPipe:
		return "|"
	case tokenCaret:
		return "^"
	case tokenTilde:
		return "~"
	case tokenShiftL:
		return "<<"
	case tokenShiftR:
		return ">>"
	case tokenShiftRU:
		return litOpShrU
	case tokenIn:
		return "in"
	case tokenLike:
		return "~="
	default:
		return t.String()
	}
}

// isNumericLit reports whether n is an integer or float literal.
// Used by the | pipe/bitwise disambiguation heuristic.
func isNumericLit(n Node) bool {
	switch n.(type) {
	case *intLit, *floatLit:
		return true
	default:
		return false
	}
}
