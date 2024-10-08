package main

import (
	"errors"
	"fmt"
	"unicode"
)

type Token struct {
	Type   TokenType
	Start  Pos
	Length Pos
	Line   Pos
	Lexeme string
}

func (t Token) String() string {
	return fmt.Sprintf("token %s, '%s' %d -> %d, line %d", t.Type.String(), t.Lexeme, t.Start, t.Length, t.Line)
}

type TokenType uint64

const (
	TokenPlus TokenType = iota
	TokenMinus
	TokenStar
	TokenSlash
	TokenBang
	TokenSemicolon

	TokenNumber
	TokenString
	TokenName

	TokenOpenParenthesis
	TokenCloseParenthesis
	TokenOpenBrace
	TokenCloseBrace

	TokenTrue
	TokenFalse
	TokenNil

	TokenFunc
	TokenReturn
	TokenWhile
	TokenVar
	TokenIf
	TokenElse

	TokenComma
	TokenDot

	TokenAssign
	TokenDeclare
	TokenBangEquals
	TokenEquals
	TokenGreaterThan
	TokenLessThan
	TokenGreaterThanOrEqual
	TokenLessThanOrEqual

	TokenEOF
	TokenError
)

func (t TokenType) String() string {
	switch t {
	case TokenPlus:
		return "plus"
	case TokenMinus:
		return "minus"
	case TokenStar:
		return "star"
	case TokenSlash:
		return "slash"
	case TokenBang:
		return "bang"
	case TokenNumber:
		return "number"
	case TokenString:
		return "string"
	case TokenTrue:
		return "true"
	case TokenFalse:
		return "false"
	case TokenNil:
		return "nil"
	case TokenOpenParenthesis:
		return "open parenthesis"
	case TokenCloseParenthesis:
		return "close parenthesis"
	case TokenOpenBrace:
		return "open brace"
	case TokenCloseBrace:
		return "close brace"
	case TokenVar:
		return "var"
	case TokenIf:
		return "if"
	case TokenElse:
		return "else"
	case TokenAssign:
		return "equals"
	case TokenBangEquals:
		return "equals"
	case TokenEquals:
		return "double equals"
	case TokenGreaterThan:
		return "greater than"
	case TokenLessThan:
		return "less than"
	case TokenGreaterThanOrEqual:
		return "greater than or equal"
	case TokenLessThanOrEqual:
		return "less than or equal"
	case TokenName:
		return "name"
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "error"
	case TokenSemicolon:
		return "semicolon"
	case TokenDeclare:
		return "declare"
	case TokenFunc:
		return "func"
	case TokenReturn:
		return "return"
	case TokenWhile:
		return "while"
	case TokenComma:
		return "comma"
	case TokenDot:
		return "dot"
	}

	return "UNDEFINED TOKENTYPE STRING CONVERSION"
}

type Lexer struct {
	src     string
	start   Pos
	current Pos
	line    Pos
}

func NewLexer(src string) *Lexer {
	return &Lexer{
		src:     src,
		start:   0,
		current: 0,
		line:    0,
	}
}

func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	// if at end of source
	if l.isAtEnd() {
		return l.makeToken(TokenEOF), nil
	}

	// skip comments
	if l.match('#') {
		for !l.match('\n') {
			l.advance()
		}

		return l.NextToken()
	}

	l.start = l.current

	var c = []rune(l.src)[l.current]
	l.advance()

	switch c {
	case '+':
		return l.makeToken(TokenPlus), nil
	case '-':
		return l.makeToken(TokenMinus), nil
	case '*':
		return l.makeToken(TokenStar), nil
	case '/':
		return l.makeToken(TokenSlash), nil
	case '(':
		return l.makeToken(TokenOpenParenthesis), nil
	case ')':
		return l.makeToken(TokenCloseParenthesis), nil
	case '{':
		return l.makeToken(TokenOpenBrace), nil
	case '}':
		return l.makeToken(TokenCloseBrace), nil
	case ';':
		return l.makeToken(TokenSemicolon), nil
	case ',':
		return l.makeToken(TokenComma), nil
	case '.':
		return l.makeToken(TokenDot), nil
	case ':':
		if !l.accept('=') {
			return l.makeToken(TokenError), errors.New("malformed token (got ':', expected '=' to follow)")
		}

		return l.makeToken(TokenDeclare), nil
	case '!':
		if l.accept('=') {
			return l.makeToken(TokenBangEquals), nil
		}

		return l.makeToken(TokenBang), nil
	case '=':
		if l.accept('=') {
			return l.makeToken(TokenEquals), nil
		}

		return l.makeToken(TokenAssign), nil
	case '>':
		if l.accept('=') {
			return l.makeToken(TokenGreaterThanOrEqual), nil
		}

		return l.makeToken(TokenGreaterThan), nil
	case '<':
		if l.accept('=') {
			return l.makeToken(TokenLessThanOrEqual), nil
		}

		return l.makeToken(TokenLessThan), nil

	case '"':
		// include ending quote
		for !l.accept('"') {
			if l.match('\n') {
				return l.makeToken(TokenError), errors.New("string did not end in current line")
			}

			if l.isAtEnd() {
				return l.makeToken(TokenError), errors.New("string did not before end of source")
			}

			l.advance()
		}

		return l.makeToken(TokenString), nil

	default:
		if unicode.IsLetter(c) || c == '_' {
			// assemble variable
			for l.isAlpha(l.peek()) {
				l.advance()
			}

			switch l.src[l.start:l.current] {
			case "true":
				return l.makeToken(TokenTrue), nil
			case "false":
				return l.makeToken(TokenFalse), nil
			case "nil":
				return l.makeToken(TokenNil), nil
			case "if":
				return l.makeToken(TokenIf), nil
			case "else":
				return l.makeToken(TokenElse), nil
			case "var":
				return l.makeToken(TokenVar), nil
			case "func":
				return l.makeToken(TokenFunc), nil
			case "while":
				return l.makeToken(TokenWhile), nil
			case "return":
				return l.makeToken(TokenReturn), nil
			default:
				return l.makeToken(TokenName), nil
			}
		} else if unicode.IsDigit(c) {
			for unicode.IsDigit(l.peek()) {
				l.advance()
			}

			return l.makeToken(TokenNumber), nil
		}

		return l.makeToken(TokenError), errors.New(fmt.Sprintf("invalid token %c", c))
	}
}

func NewToken(t TokenType, start Pos, length Pos, line Pos, lexeme string) Token {
	return Token{
		Type:   t,
		Start:  start,
		Length: length,
		Line:   line,
		Lexeme: lexeme,
	}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	tokens := make([]Token, 0)

	tok, err := l.NextToken()
	for ; err == nil; tok, err = l.NextToken() {
		tokens = append(tokens, tok)

		if tok.Type == TokenEOF {
			break
		}
	}

	return tokens, err
}

func (l *Lexer) makeToken(t TokenType) Token {
	return NewToken(t, l.start, l.current-l.start, l.line, l.src[l.start:l.current])
}

func (l *Lexer) peek() rune {
	if l.isAtEnd() {
		return 0
	}

	return []rune(l.src)[l.current]
}

func (l *Lexer) match(c rune) bool {
	return l.peek() == c
}

func (l *Lexer) accept(c rune) bool {
	if l.match(c) {
		l.advance()
		return true
	}

	return false
}

func (l *Lexer) isAlpha(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_'
}

func (l *Lexer) advance() {
	if l.isAtEnd() {
		return
	}

	if []rune(l.src)[l.current] == '\n' {
		l.line++
	}

	l.current++
}

func (l *Lexer) isAtEnd() bool {
	return l.current >= Pos(len([]rune(l.src)))
}

func (l *Lexer) skipWhitespace() {
	for !l.isAtEnd() && unicode.IsSpace(l.peek()) {
		l.advance()
	}
}
