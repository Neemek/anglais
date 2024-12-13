package core

import (
	"testing"
)

type LexerTestData struct {
	source         string
	expectedTokens []TokenType
}

func GetLexerTestData() map[string]LexerTestData {
	return map[string]LexerTestData{
		"hello_world_string(1)": {
			"\"Hello world\"",
			[]TokenType{TokenString, TokenEOF},
		},
		"empty_string(1)": {
			"\"\"",
			[]TokenType{TokenString, TokenEOF},
		},
		"simple number(1)": {
			"1024",
			[]TokenType{TokenNumber, TokenEOF},
		},
		"simple_arithmetics(7)": {
			"1 + 23 / 4 * 3",
			[]TokenType{
				TokenNumber, TokenPlus, TokenNumber, TokenSlash,
				TokenNumber, TokenStar, TokenNumber, TokenEOF,
			},
		},
		"condition(3)": {
			"a <= 200",
			[]TokenType{TokenName, TokenLessThanOrEqual, TokenNumber, TokenEOF},
		},
		"if_statement(10)": {
			"if a >= 200 {\n    write(\"Hello world!\")\n}",
			[]TokenType{
				TokenIf, TokenName, TokenGreaterThanOrEqual, TokenNumber, TokenOpenBrace,
				TokenName, TokenOpenParenthesis, TokenString, TokenCloseParenthesis, TokenCloseBrace,
				TokenEOF,
			},
		},
		"if_else_statement(20)": {
			"if 23 * 2/3 > 32 {\n    write(\"It is larger!\")\n} else {\n    write(\"It is lower!\")\n}",
			[]TokenType{
				TokenIf, TokenNumber, TokenStar, TokenNumber, TokenSlash, TokenNumber, TokenGreaterThan, TokenNumber, TokenOpenBrace,
				TokenName, TokenOpenParenthesis, TokenString, TokenCloseParenthesis, TokenCloseBrace,
				TokenElse, TokenOpenBrace, TokenName, TokenOpenParenthesis, TokenString, TokenCloseParenthesis, TokenCloseBrace,
				TokenEOF,
			},
		},
		"empty_string": {
			"",
			[]TokenType{TokenEOF},
		},
		"full_arithmetic_equality": {
			"a + 2 == 10 * 2 / 3",
			[]TokenType{
				TokenName, TokenPlus, TokenNumber, TokenEquals,
				TokenNumber, TokenStar, TokenNumber, TokenSlash, TokenNumber,
				TokenEOF,
			},
		},
		"name": {
			"print",
			[]TokenType{TokenName, TokenEOF},
		},
		"bunch_of_parentheses": {
			"(((())))",
			[]TokenType{
				TokenOpenParenthesis, TokenOpenParenthesis, TokenOpenParenthesis, TokenOpenParenthesis,
				TokenCloseParenthesis, TokenCloseParenthesis, TokenCloseParenthesis, TokenCloseParenthesis,
				TokenEOF,
			},
		},
		"space_before_string": {
			"\n   	\"\"",
			[]TokenType{TokenString, TokenEOF},
		},
		"write_call": {
			"write(\"Hello world\")",
			[]TokenType{TokenName, TokenOpenParenthesis, TokenString, TokenCloseParenthesis, TokenEOF},
		},
		"complex_comparison": {
			"!(h__elo123  >= 1)",
			[]TokenType{
				TokenBang, TokenOpenParenthesis, TokenName, TokenGreaterThanOrEqual, TokenNumber, TokenCloseParenthesis,
				TokenEOF,
			},
		},
		"3assignments_1condition": {
			"a = 8 * 32\nb = a > 256\nc = a <= 256\n!b == c",
			[]TokenType{
				TokenName, TokenAssign, TokenNumber, TokenStar, TokenNumber,
				TokenName, TokenAssign, TokenName, TokenGreaterThan, TokenNumber,
				TokenName, TokenAssign, TokenName, TokenLessThanOrEqual, TokenNumber,
				TokenBang, TokenName, TokenEquals, TokenName, TokenEOF,
			},
		},
		"func": {
			"func sum(a, b) {\n    return a + b\n}",
			[]TokenType{
				TokenFunc, TokenName, TokenOpenParenthesis, TokenName, TokenComma, TokenName, TokenCloseParenthesis,
				TokenOpenBrace, TokenReturn, TokenName, TokenPlus, TokenName, TokenCloseBrace,
			},
		},
		"while_loop": {
			"while a < 5 {\n    a = a + 1\n}",
			[]TokenType{
				TokenWhile, TokenName, TokenLessThan, TokenNumber, TokenOpenBrace,
				TokenName, TokenAssign, TokenName, TokenPlus, TokenNumber, TokenCloseBrace,
			},
		},
		"lambda": {
			"sum := func(a, b) {\n" +
				"    return a + b\n" +
				"}",
			[]TokenType{
				TokenName, TokenDeclare, TokenFunc, TokenOpenParenthesis, TokenName, TokenComma, TokenName, TokenCloseParenthesis,
				TokenOpenBrace, TokenReturn, TokenName, TokenPlus, TokenName, TokenCloseBrace,
			},
		},
	}
}

func TestLexer_NextToken(t *testing.T) {
	data := GetLexerTestData()

	for name, tc := range data {
		t.Run(name, func(t *testing.T) {
			lex := NewLexer(tc.source)

			t.Logf("Testing source: '%s'", tc.source)

			for _, expectedType := range tc.expectedTokens {
				tok, err := lex.NextToken()

				if err != nil {
					t.Errorf("Unexpected error while parsing token '%s'. Error: %s", expectedType, err)
					continue
				}

				if tok.Type != expectedType {
					t.Errorf("Expected token type '%s' but got '%s'", expectedType, tok.Type)
				} else {
					t.Logf("Got expected token type '%s'", expectedType)
				}
			}
		})
	}
}

func TestNewLexer(t *testing.T) {
	lex := NewLexer("example source")

	if lex == nil {
		t.Fatalf("Lexer was not initialized correctly.")
	}

	if lex.start != 0 {
		t.Errorf("Lexer start position was not initialized correctly.")
	}

	if lex.current != 0 {
		t.Errorf("Lexer current position was not initialized correctly.")
	}

	if lex.src != "example source" {
		t.Errorf("Lexer lexer was not initialized correctly.")
	}

	t.Log("Successfully initialized lexer")
}

// lexer NextToken provides an error when it comes across an invalid token
func TestLexer_NextTokenErrors(t *testing.T) {
	invalid_codes := []string{
		// Invalid tokens
		"^", "@", "$&", "¨",
		// Non-ending string (in same line)
		"\"", "Hini minit \"mini moe", "\"this is some test\ncontent\"", "\n\"Hello world",
	}

	for _, code := range invalid_codes {
		lex := NewLexer(code)
		tok, err := lex.NextToken()

		for err == nil && tok.Type != TokenEOF {
			tok, err = lex.NextToken()
		}

		if err == nil {
			t.Errorf("Expected error for invalid code '%s'", code)
		} else {
			t.Logf("Got an expected error (%s) for invalid code '%s'", err.Error(), code)
		}
	}
}

func BenchmarkNewLexer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewLexer("example source")
	}
}

func BenchmarkLexer_NextToken(b *testing.B) {
	data := GetLexerTestData()

	for name, tc := range data {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				lex := NewLexer(tc.source)
				tok, err := lex.NextToken()

				for err == nil && tok.Type != TokenEOF {
					tok, err = lex.NextToken()
				}
			}
		})
	}

}
