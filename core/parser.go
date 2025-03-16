package core

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type ParsingError struct {
	Description string
	Causer      *Token
}

func (p *ParsingError) Error() string {
	return p.Description
}

// Format Print a rich and informative error
func (p *ParsingError) Format(src []rune) string {
	builder := strings.Builder{}

	lineNumber := 1
	lineBeginning := 0
	for i := 0; i < int(p.Causer.Start); i++ {
		if src[i] == '\n' {
			lineBeginning = i + 1
			lineNumber++
		}
	}

	lineEnd := len(src)
	for i := lineBeginning; i < len(src); i++ {
		if src[i] == '\n' {
			lineEnd = i
			break
		}
	}

	builder.WriteString(" \t v ")
	builder.WriteString(p.Description)
	builder.WriteRune('\n')

	builder.WriteString(fmt.Sprintf("  %d:%d\t | %s", lineNumber, int(p.Causer.Start)-lineBeginning+1, string(src[lineBeginning:lineEnd])))

	builder.WriteString("\n\t ^")
	for i := lineBeginning; i <= int(p.Causer.Start); i++ {
		builder.WriteRune(' ')
	}

	for i := 0; i < int(p.Causer.Length); i++ {
		builder.WriteRune('^')
	}
	builder.WriteRune('\n')

	return builder.String()
}

type Parser struct {
	tokens []Token
	prev   *Token
	curr   *Token
	pos    Pos
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

func (p *Parser) Parse() (Node, error) {
	// top level statements
	statements := make([]Node, 0)

	// initialize current
	p.advance()

	for int(p.pos) < len(p.tokens) && p.curr.Type != TokenEOF {
		b, err := p.block(true)

		if err != nil {
			return nil, err
		}

		statements = append(statements, b)
	}

	return &BlockNode{
		statements: statements,
	}, nil
}

func (p *Parser) accept(tokenType TokenType) bool {
	if p.curr == nil {
		log.Fatal("unexpected current token nil")
		return false
	}

	if (*p.curr).Type == tokenType {
		p.advance()
		return true
	}

	return false
}

func (p *Parser) expect(tokenType TokenType) error {
	if !p.accept(tokenType) {
		return p.error("Expected token "+tokenType.String()+", got "+p.curr.Type.String(), p.curr)
	}
	return nil
}

func (p *Parser) peek() (Token, error) {
	if p.pos >= Pos(len(p.tokens)) {
		return Token{}, errors.New("cannot peek beyond tokens")
	}

	return p.tokens[p.pos], nil
}

func (p *Parser) advance() {
	p.prev = p.curr

	if p.pos < Pos(len(p.tokens)) {
		p.curr = &p.tokens[p.pos]
		p.pos++
	} else {
		panic("no more tokens")
	}
}

func (p *Parser) error(error string, causer *Token) error {
	return &ParsingError{
		Description: error,
		Causer:      causer,
	}
}

func (p *Parser) factor() (Node, error) {
	switch (*p.curr).Type {
	case TokenString:
		p.advance()
		return &StringNode{
			(*p.prev).Lexeme[1 : len((*p.prev).Lexeme)-1],
			(*p.prev).Lexeme,
		}, nil

	case TokenNumber:
		p.advance()
		num, err := strconv.ParseFloat((*p.prev).Lexeme, NumberSize)

		if err != nil {
			return nil, p.error(fmt.Sprintf("Error parsing number: %v", err), p.prev)
		}

		return &NumberNode{
			num,
		}, nil

	case TokenTrue:
		p.advance()
		return &BooleanNode{
			true,
		}, nil
	case TokenFalse:
		p.advance()
		return &BooleanNode{
			false,
		}, nil

	case TokenNil:
		p.advance()
		return &NilNode{}, nil

	case TokenOpenBracket:
		p.advance()

		var values []Node
		for !p.accept(TokenCloseBracket) {
			if len(values) > 0 {
				if err := p.expect(TokenComma); err != nil {
					return nil, err
				}
			}

			value, err := p.condition()

			if err != nil {
				return nil, err
			}

			values = append(values, value)
		}

		return &ListNode{
			values,
		}, nil

	// unary minus
	case TokenMinus:
		p.advance()
		f, err := p.factor()
		if err != nil {
			return nil, err
		}
		return &BinaryNode{
			BinarySubtraction,
			&NumberNode{0},
			f,
		}, nil

	case TokenName:
		p.advance()
		name := (*p.prev).Lexeme

		if p.curr.Type == TokenOpenParenthesis {
			args, err := p.parseArgs()
			if err != nil {
				return nil, err
			}

			return &CallNode{
				&ReferenceNode{
					name,
				},
				args,
				true,
			}, nil
		}

		return &ReferenceNode{
			name,
		}, nil

	case TokenFunc:
		p.advance()
		params, err := p.parseParams()
		if err != nil {
			return nil, err
		}

		b, err := p.block(false)
		if err != nil {
			return nil, err
		}

		return &FunctionNode{
			"*",
			params,
			b,
		}, nil

	case TokenOpenParenthesis:
		p.advance()
		v, err := p.condition()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenCloseParenthesis); err != nil {
			return nil, err
		}

		return v, nil

	default:
		err := p.error("invalid factor", p.curr)
		p.advance()
		return nil, err
	}
}

func (p *Parser) prop() (Node, error) {
	v, err := p.factor()
	if err != nil {
		return nil, err
	}

	// parse chains of prop-getting ( "".split().join().length.round() )
	for p.accept(TokenDot) {
		if err := p.expect(TokenName); err != nil {
			return nil, err
		}
		property := (*p.prev).Lexeme

		v = &AccessNode{
			v,
			property,
		}

		// if called, also add
		if (*p.curr).Type == TokenOpenParenthesis {
			args, err := p.parseArgs()
			if err != nil {
				return nil, err
			}

			v = &CallNode{
				v,
				args,
				true,
			}
		}
	}

	return v, nil
}

func (p *Parser) product() (Node, error) {
	left, err := p.prop()
	if err != nil {
		return nil, err
	}

	for p.accept(TokenStar) || p.accept(TokenSlash) {
		op := BinaryMultiplication

		if (*p.prev).Type == TokenSlash {
			op = BinaryDivision
		}

		f, err := p.prop()
		if err != nil {
			return nil, err
		}

		left = &BinaryNode{
			op,
			left,
			f,
		}
	}

	return left, nil
}

func (p *Parser) term() (Node, error) {
	left, err := p.product()
	if err != nil {
		return nil, err
	}

	for p.accept(TokenPlus) || p.accept(TokenMinus) {
		op := BinaryAddition

		if (*p.prev).Type == TokenMinus {
			op = BinarySubtraction
		}

		pr, err := p.product()
		if err != nil {
			return nil, err
		}

		left = &BinaryNode{
			op,
			left,
			pr,
		}
	}

	return left, nil
}

func (p *Parser) comparison() (Node, error) {
	left, err := p.term()

	if err != nil {
		return nil, err
	}

	op := BinaryEquality

	switch (*p.curr).Type {
	case TokenEquals:
		op = BinaryEquality
	case TokenBangEquals:
		op = BinaryInequality
	case TokenGreaterThan:
		op = BinaryGreater
	case TokenLessThan:
		op = BinaryLess
	case TokenLessThanOrEqual:
		op = BinaryLessEqual
	case TokenGreaterThanOrEqual:
		op = BinaryGreaterEqual
	default:
		return left, nil
	}

	p.advance()

	t, err := p.term()

	if err != nil {
		return nil, err
	}

	return &BinaryNode{
		op,
		left,
		t,
	}, nil
}

func (p *Parser) condition() (Node, error) {
	left, err := p.comparison()
	if err != nil {
		return nil, err
	}

	op := BinaryEquality

	switch (*p.curr).Type {
	case TokenDoubleAmpersand:
		op = BinaryAnd
	case TokenDoublePipe:
		op = BinaryOr
	default:
		return left, nil
	}

	p.advance()

	c, err := p.comparison()
	if err != nil {
		return left, err
	}

	return &BinaryNode{
		op,
		left,
		c,
	}, nil
}

func (p *Parser) statement() (Node, error) {
	switch (*p.curr).Type {
	case TokenIf:
		p.advance()

		condition, err := p.condition()
		if err != nil {
			return nil, err
		}
		then, err := p.block(false)

		if err != nil {
			return nil, err
		}

		var otherwise Node

		if p.accept(TokenElse) {
			// allow else if
			if p.curr.Type == TokenIf {
				otherwise, err = p.statement()
			} else {
				otherwise, err = p.block(false)
			}
			if err != nil {
				return nil, err
			}
		}

		return &ConditionalNode{
			condition,
			then,
			otherwise,
		}, nil

	case TokenName:
		p.advance()
		name := (*p.prev).Lexeme

		if (*p.curr).Type == TokenDot {
			var v Node = &ReferenceNode{
				name,
			}

			// parse chains of prop-getting ( "".split().join().length.round() )
			for p.accept(TokenDot) {
				if err := p.expect(TokenName); err != nil {
					return nil, err
				}
				property := (*p.prev).Lexeme

				v = &AccessNode{
					v,
					property,
				}

				// if called, also add
				if (*p.curr).Type == TokenOpenParenthesis {
					args, err := p.parseArgs()
					if err != nil {
						return nil, err
					}

					v = &CallNode{
						v,
						args,
						(*p.curr).Type == TokenDot, // if the chain is continued, keep the value.
					}
				}
			}

			return v, nil
		} else if p.curr.Type == TokenOpenParenthesis {
			args, err := p.parseArgs()
			if err != nil {
				return nil, err
			}

			return &CallNode{
				&ReferenceNode{
					name,
				},
				args,
				false,
			}, nil
		} else if p.accept(TokenAssign) || p.accept(TokenDeclare) {
			isDeclaration := p.prev.Type == TokenDeclare
			c, err := p.condition()
			if err != nil {
				return nil, err
			}

			return &AssignNode{
				name,
				c,
				isDeclaration,
			}, nil
		} else {
			return p.condition()
		}

	case TokenImport:
		p.advance()

		if err := p.expect(TokenString); err != nil {
			return nil, err
		}

		path := p.prev.Lexeme[1 : len(p.prev.Lexeme)-1]

		return &ImportNode{
			path,
		}, nil

	case TokenFunc:
		p.advance()

		if err := p.expect(TokenName); err != nil {
			return nil, err
		}
		name := p.prev.Lexeme

		params, err := p.parseParams()
		if err != nil {
			return nil, err
		}

		b, err := p.block(false)
		if err != nil {
			return nil, err
		}

		return &AssignNode{
			name,
			&FunctionNode{
				name,
				params,
				b,
			},
			true,
		}, nil

	case TokenWhile:
		p.advance()

		c, err := p.condition()
		if err != nil {
			return nil, err
		}

		b, err := p.block(false)
		if err != nil {
			return nil, err
		}

		return &LoopNode{
			c,
			b,
		}, nil

	case TokenReturn:
		p.advance()

		c, err := p.condition()
		if err != nil {
			return nil, err
		}

		return &ReturnNode{
			c,
		}, nil

	case TokenBreakpoint:
		p.advance()

		return &BreakpointNode{}, nil

	default:
		err := p.error("invalid statement", p.curr)
		p.advance()
		return nil, err
	}
}

func (p *Parser) block(canBeStatement bool) (Node, error) {
	if canBeStatement {
		if !p.accept(TokenOpenBrace) {
			return p.statement()
		}
	} else {
		if err := p.expect(TokenOpenBrace); err != nil {
			return nil, err
		}
	}

	statements := make([]Node, 0)

	for !p.accept(TokenCloseBrace) {
		s, err := p.statement()

		if err != nil {
			return nil, err
		}

		statements = append(statements, s)
	}

	return &BlockNode{
		statements,
	}, nil
}

func (p *Parser) parseArgs() ([]Node, error) {
	args := make([]Node, 0)

	if err := p.expect(TokenOpenParenthesis); err != nil {
		return nil, err
	}

	if !p.accept(TokenCloseParenthesis) {
		c, err := p.condition()
		if err != nil {
			return nil, err
		}
		args = append(args, c)
		for !p.accept(TokenCloseParenthesis) {
			if err := p.expect(TokenComma); err != nil {
				return nil, err
			}
			c, err = p.condition()
			if err != nil {
				return nil, err
			}
			args = append(args, c)
		}
	}

	return args, nil
}

// parseParams parse parameters and parentheses
func (p *Parser) parseParams() ([]string, error) {
	if err := p.expect(TokenOpenParenthesis); err != nil {
		return nil, err
	}
	params := make([]string, 0)

	if p.accept(TokenName) {
		name := (*p.prev).Lexeme
		params = append(params, name)
		for !p.accept(TokenCloseParenthesis) {
			if err := p.expect(TokenComma); err != nil {
				return nil, err
			}
			if err := p.expect(TokenName); err != nil {
				return nil, err
			}
			name = (*p.prev).Lexeme
			params = append(params, name)
		}
	} else {
		if err := p.expect(TokenCloseParenthesis); err != nil {
			return nil, err
		}
	}

	return params, nil
}
