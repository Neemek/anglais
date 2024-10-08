package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
)

type ParsingError struct {
	Description string
	Causer      *Token
}

// Print a rich and informative error
func (p *ParsingError) Print(src string) {
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

	print(" \t v ")
	println(p.Description)

	println(fmt.Sprintf("  %d:%d\t | %s", lineNumber, int(p.Causer.Start)-lineBeginning+1, src[lineBeginning:lineEnd]))

	print("\t ^")
	for i := lineBeginning; i <= int(p.Causer.Start); i++ {
		print(" ")
	}

	for i := 0; i < int(p.Causer.Length); i++ {
		print("^")
	}
	println()
}

type Parser struct {
	tokens []Token
	prev   *Token
	curr   *Token
	pos    Pos

	hadError bool
	errors   []ParsingError
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:   tokens,
		pos:      0,
		hadError: false,
		errors:   make([]ParsingError, 0),
	}
}

func (p *Parser) Parse() Node {
	// top level statements
	statements := make([]Node, 0)

	// initialize current
	p.advance()

	for int(p.pos) < len(p.tokens) && p.curr.Type != TokenEOF {
		statements = append(statements, p.block(true))
	}

	return &BlockNode{
		statements: statements,
	}
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

func (p *Parser) expect(tokenType TokenType) {
	if !p.accept(tokenType) {
		p.error("Expected token "+tokenType.String()+", got "+p.curr.Type.String(), p.curr)
		p.advance() // just move on
	}
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

func (p *Parser) error(error string, causer *Token) {
	p.hadError = true
	p.errors = append(p.errors, ParsingError{
		Description: error,
		Causer:      causer,
	})
}

func (p *Parser) factor() Node {
	switch (*p.curr).Type {
	case TokenString:
		p.advance()
		return &StringNode{
			(*p.prev).Lexeme[1 : len((*p.prev).Lexeme)-1],
			(*p.prev).Lexeme,
		}

	case TokenNumber:
		p.advance()
		num, err := strconv.ParseFloat((*p.prev).Lexeme, NumberSize)

		if err != nil {
			p.error(fmt.Sprintf("Error parsing number: %v", err), p.prev)
		}

		return &NumberNode{
			NumberValue(num),
		}

	case TokenTrue:
		p.advance()
		return &BooleanNode{
			true,
		}
	case TokenFalse:
		p.advance()
		return &BooleanNode{
			false,
		}

	case TokenNil:
		p.advance()
		return &NilNode{}

	// unary minus
	case TokenMinus:
		p.advance()
		return &BinaryNode{
			BinarySubtraction,
			&NumberNode{NumberValue(0)},
			p.factor(),
		}

	case TokenName:
		p.advance()
		name := (*p.prev).Lexeme

		if p.curr.Type == TokenOpenParenthesis {
			args := p.parseArgs()

			return &CallNode{
				name,
				args,
				true,
			}
		}

		return &ReferenceNode{
			name,
		}

	case TokenFunc:
		p.advance()
		params := p.parseParams()

		return &FunctionNode{
			"*",
			params,
			p.block(false),
		}

	case TokenOpenParenthesis:
		p.advance()
		v := p.condition()
		p.expect(TokenCloseParenthesis)

		return v

	default:
		p.error("invalid factor", p.curr)
		p.advance()
		return nil
	}
}

func (p *Parser) product() Node {
	left := p.factor()

	for p.accept(TokenStar) || p.accept(TokenSlash) {
		op := BinaryMultiplication

		if (*p.prev).Type == TokenSlash {
			op = BinaryDivision
		}

		left = &BinaryNode{
			op,
			left,
			p.factor(),
		}
	}

	return left
}

func (p *Parser) term() Node {
	left := p.product()

	for p.accept(TokenPlus) || p.accept(TokenMinus) {
		op := BinaryAddition

		if (*p.prev).Type == TokenMinus {
			op = BinarySubtraction
		}

		left = &BinaryNode{
			op,
			left,
			p.product(),
		}
	}

	return left
}

func (p *Parser) condition() Node {
	left := p.term()
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
		return left
	}

	p.advance()

	return &BinaryNode{
		op,
		left,
		p.term(),
	}
}

func (p *Parser) statement() Node {
	switch (*p.curr).Type {
	case TokenIf:
		p.advance()

		condition := p.condition()
		then := p.block(false)
		var otherwise Node

		if p.accept(TokenElse) {
			otherwise = p.block(false)
		}

		return &ConditionalNode{
			condition,
			then,
			otherwise,
		}

	case TokenName:
		p.advance()
		name := (*p.prev).Lexeme

		if p.curr.Type == TokenOpenParenthesis {
			args := p.parseArgs()

			return &CallNode{
				name,
				args,
				false,
			}
		} else if p.accept(TokenAssign) || p.accept(TokenDeclare) {
			isDeclaration := p.prev.Type == TokenDeclare

			return &AssignNode{
				name,
				p.condition(),
				isDeclaration,
			}
		} else {
			return p.condition()
		}

	case TokenFunc:
		p.advance()

		p.expect(TokenName)
		name := p.prev.Lexeme

		params := p.parseParams()

		return &AssignNode{
			name,
			&FunctionNode{
				name,
				params,
				p.block(false),
			},
			true,
		}

	case TokenReturn:
		p.advance()

		return &ReturnNode{
			p.condition(),
		}

	default:
		p.error("invalid statement", p.curr)
		p.advance()
		return nil
	}
}

func (p *Parser) block(canBeStatement bool) Node {
	if canBeStatement {
		if !p.accept(TokenOpenBrace) {
			return p.statement()
		}
	} else {
		p.expect(TokenOpenBrace)
	}

	statements := make([]Node, 0)

	for !p.accept(TokenCloseBrace) {
		statements = append(statements, p.statement())
	}

	return &BlockNode{
		statements,
	}
}

func (p *Parser) parseArgs() []Node {
	args := make([]Node, 0)

	p.expect(TokenOpenParenthesis)

	if !p.accept(TokenCloseParenthesis) {
		args = append(args, p.condition())
		for !p.accept(TokenCloseParenthesis) {
			p.expect(TokenComma)
			args = append(args, p.condition())
		}
	}

	return args
}

// parseParams parse parameters and parentheses
func (p *Parser) parseParams() []string {
	p.expect(TokenOpenParenthesis)
	params := make([]string, 0)

	if p.accept(TokenName) {
		name := (*p.prev).Lexeme
		params = append(params, name)
		for !p.accept(TokenCloseParenthesis) {
			p.expect(TokenComma)
			p.expect(TokenName)
			name = (*p.prev).Lexeme
			params = append(params, name)
		}
	} else {
		p.expect(TokenCloseParenthesis)
	}

	return params
}
