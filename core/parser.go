package core

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type FormatedError interface {
	Error() string
	Format() string
}

type ParsingError struct {
	Description string
	Causer      *Token
	Source      string
}

func (p ParsingError) Error() string {
	return p.Description
}

// Format Print a rich and informative error
func (p ParsingError) Format() string {
	src := []rune(p.Source)
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

	descriptor := fmt.Sprintf("%d:%d", lineNumber, int(p.Causer.Start)-lineBeginning+1)
	builder.WriteString(p.Description)
	builder.WriteRune('\n')

	builder.WriteString(descriptor)
	builder.WriteString(" | ")
	builder.WriteString(string(src[lineBeginning:lineEnd]))

	builder.WriteString("\n")
	builder.WriteString(strings.Repeat(" ", len(descriptor)))
	builder.WriteString("  ")
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
	source string
	tokens []Token
	prev   *Token
	curr   *Token
	pos    Pos
}

func NewParser(source string, tokens []Token) *Parser {
	return &Parser{
		source: source,
		tokens: tokens,
		pos:    0,
	}
}

type Program struct {
	Imports []string
	Block   *BlockNode
	Path    string
}

func (p *Program) String() string {
	builder := strings.Builder{}

	builder.WriteString("=== Imports ===\n")
	for _, i := range p.Imports {
		builder.WriteString(i)
		builder.WriteString("\n")
	}
	builder.WriteString("===============\n")

	builder.WriteString(p.Block.String())

	return builder.String()
}

func (p *Parser) Parse(path string) (*Program, error) {
	imports := make([]string, 0)

	// top level statements
	statements := make([]Node, 0)

	// initialize current
	p.advance()

	for int(p.pos) < len(p.tokens) && p.curr.Type != TokenEOF {
		if p.accept(TokenImport) {
			if err := p.expect(TokenString); err != nil {
				return nil, err
			}

			imports = append(imports, p.prev.Lexeme[1:len(p.prev.Lexeme)-1])
		}

		b, err := p.block(true)

		if err != nil {
			return nil, err
		}

		statements = append(statements, b)
	}

	return &Program{
		imports,
		&BlockNode{
			statements,
			0,
			p.curr.Start + p.curr.Length,
		},
		path,
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
	return ParsingError{
		Description: error,
		Causer:      causer,
		Source:      p.source,
	}
}

func (p *Parser) factor() (Node, error) {
	switch (*p.curr).Type {
	case TokenString:
		p.advance()
		return &StringNode{
			(*p.prev).Lexeme[1 : len((*p.prev).Lexeme)-1],
			(*p.prev).Lexeme,
			p.prev.Start,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenNumber:
		p.advance()
		num, err := strconv.ParseFloat((*p.prev).Lexeme, NumberSize)

		if err != nil {
			return nil, p.error(fmt.Sprintf("Error parsing number: %v", err), p.prev)
		}

		return &NumberNode{
			num,
			p.prev.Start,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenHexadecimal:
		p.advance()
		start := (*p.prev).Start
		num, err := strconv.ParseUint((*p.prev).Lexeme[2:], 16, NumberSize)
		if err != nil {
			return nil, err
		}

		return &NumberNode{
			float64(num),
			start,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenTrue:
		p.advance()
		return &BooleanNode{
			true,
			p.prev.Start,
			p.prev.Start + p.prev.Length,
		}, nil
	case TokenFalse:
		p.advance()
		return &BooleanNode{
			false,
			p.prev.Start,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenNil:
		p.advance()
		return &NilNode{}, nil

	case TokenOpenBracket:
		p.advance()

		start := p.prev.Start

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
			start,
			p.prev.Start + p.prev.Length,
		}, nil

	// unary minus
	case TokenMinus:
		p.advance()
		first := p.prev

		f, err := p.factor()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{
			UnaryNegate,
			f,
			first.Start,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenBang:
		p.advance()
		start := p.prev.Start

		v, err := p.factor()
		if err != nil {
			return nil, err
		}

		return &UnaryNode{
			UnaryNot,
			v,
			start,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenName:
		p.advance()
		name := (*p.prev).Lexeme
		start := p.prev.Start
		nameEnd := start + p.prev.Length

		if p.curr.Type == TokenOpenParenthesis {
			args, err := p.parseArgs()
			if err != nil {
				return nil, err
			}

			return &CallNode{
				&ReferenceNode{
					name,
					start,
					nameEnd,
				},
				args,
				true,
				start,
				p.prev.Start + p.prev.Length,
			}, nil
		}

		return &ReferenceNode{
			name,
			start,
			nameEnd,
		}, nil

	case TokenFunc:
		p.advance()
		start := p.prev.Start

		params, err := p.parseParams()
		if err != nil {
			return nil, err
		}

		var sig TypeSignature = &NilSignature{}
		if p.curr.Type != TokenOpenBrace {
			sig, err = p.parseSignature()
			if err != nil {
				return nil, err
			}
		}

		b, err := p.block(false)
		if err != nil {
			return nil, err
		}

		return &FunctionNode{
			"*",
			params,
			sig,
			b,
			start,
			p.prev.Start + p.prev.Length,
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
	start := p.curr.Start

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
			start,
			p.prev.Start + p.prev.Length,
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
				start,
				p.prev.Start + p.prev.Length,
			}
		}
	}

	return v, nil
}

func (p *Parser) product() (Node, error) {
	start := p.curr.Start
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
			start,
			p.prev.Start + p.prev.Length,
		}
	}

	return left, nil
}

func (p *Parser) term() (Node, error) {
	start := p.curr.Start

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
			start,
			p.prev.Start + p.prev.Length,
		}
	}

	return left, nil
}

func (p *Parser) comparison() (Node, error) {
	start := p.curr.Start
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
		start,
		p.prev.Start + p.prev.Length,
	}, nil
}

func (p *Parser) condition() (Node, error) {
	start := p.curr.Start
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
		start,
		p.prev.Start + p.prev.Length,
	}, nil
}

func (p *Parser) statement() (Node, error) {
	switch (*p.curr).Type {
	case TokenIf:
		start := p.curr.Start
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
			start,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenName:
		p.advance()
		start := p.prev.Start
		name := (*p.prev).Lexeme

		if (*p.curr).Type == TokenDot {
			var v Node = &ReferenceNode{
				name,
				start,
				p.prev.Start + p.prev.Length,
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
					start,
					p.prev.Start + p.prev.Length,
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
						start,
						p.prev.Start + p.prev.Length,
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
					start,
					start + Pos(len(name)),
				},
				args,
				false,
				start,
				p.prev.Start + p.prev.Length,
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
				start,
				p.prev.Start + p.prev.Length,
			}, nil
		} else {
			return p.condition()
		}

	case TokenFunc:
		p.advance()

		funcStart := p.prev.Start

		if err := p.expect(TokenName); err != nil {
			return nil, err
		}
		name := p.prev.Lexeme

		params, err := p.parseParams()
		if err != nil {
			return nil, err
		}

		var yield TypeSignature = &NilSignature{}
		if p.curr.Type != TokenOpenBrace {
			yield, err = p.parseSignature()
			if err != nil {
				return nil, err
			}
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
				yield,
				b,
				funcStart,
				p.prev.Start + p.prev.Length,
			},
			true,
			funcStart,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenWhile:
		p.advance()
		start := p.prev.Start

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
			start,
			p.prev.Start + p.prev.Length,
		}, nil

	case TokenReturn:
		p.advance()
		start := p.prev.Start

		c, err := p.condition()
		if err != nil {
			return nil, err
		}

		return &ReturnNode{
			c,
			start,
			p.prev.Start + p.prev.Length,
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

	start := p.prev.Start

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
		start,
		p.prev.Start + p.prev.Length,
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
func (p *Parser) parseParams() ([]FunctionParameter, error) {
	if err := p.expect(TokenOpenParenthesis); err != nil {
		return nil, err
	}
	params := make([]FunctionParameter, 0)

	if p.accept(TokenName) {
		name := (*p.prev).Lexeme
		if err := p.expect(TokenColon); err != nil {
			return nil, err
		}

		t, err := p.parseSignature()
		if err != nil {
			return nil, err
		}

		params = append(params, FunctionParameter{
			name,
			t,
		})
		for !p.accept(TokenCloseParenthesis) {
			if err := p.expect(TokenComma); err != nil {
				return nil, err
			}
			if err := p.expect(TokenName); err != nil {
				return nil, err
			}
			name = (*p.prev).Lexeme
			if err := p.expect(TokenColon); err != nil {
				return nil, err
			}

			t, err := p.parseSignature()
			if err != nil {
				return nil, err
			}

			params = append(params, FunctionParameter{
				name,
				t,
			})
		}
	} else {
		if err := p.expect(TokenCloseParenthesis); err != nil {
			return nil, err
		}
	}

	return params, nil
}

func (p *Parser) parseSignature() (TypeSignature, error) {
	var s TypeSignature

	if p.accept(TokenFunc) {
		if err := p.expect(TokenOpenParenthesis); err != nil {
			return nil, err
		}

		var in []TypeSignature

		for !p.accept(TokenCloseParenthesis) {
			if len(in) > 0 {
				if err := p.expect(TokenComma); err != nil {
					return nil, err
				}
			}

			sig, err := p.parseSignature()

			if err != nil {
				return nil, err
			}

			in = append(in, sig)
		}

		out, err := p.parseSignature()
		if err != nil {
			return nil, err
		}

		s = &FunctionSignature{
			in,
			out,
		}
	} else {
		if err := p.expect(TokenName); err != nil {
			return nil, err
		}
		name := (*p.prev).Lexeme

		switch name {
		case "string":
			s = &StringSignature{}
		case "number":
			s = &NumberSignature{}
		case "boolean":
			s = &BooleanSignature{}
		case "list":
			if err := p.expect(TokenOpenBracket); err != nil {
				return nil, err
			}

			contents, err := p.parseSignature()
			if err != nil {
				return nil, err
			}

			if err := p.expect(TokenCloseBracket); err != nil {
				return nil, err
			}

			s = &ListSignature{
				contents,
			}

		case "any":
			s = &AnySignature{}

		default:
			return nil, p.error("unsupported type: "+name, p.prev)
		}
	}

	if p.accept(TokenPipe) {
		other, err := p.parseSignature()
		if err != nil {
			return nil, err
		}
		return &CompositeSignature{
			s,
			other,
		}, nil
	}

	return s, nil
}
