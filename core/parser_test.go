package core

import (
	"strconv"
	"testing"
)

func TestNewParser(t *testing.T) {
	tokens := make([]Token, 0)

	p := NewParser(tokens)

	if p == nil {
		t.Fatal("parser should not be nil")
	}

	if p.pos != 0 {
		t.Error("parser should initialize position at 0")
	}

	if len(p.tokens) != len(tokens) {
		t.Error("parser should have the passed token list")
	}

	for i, v := range tokens {
		if p.tokens[i] != v {
			t.Error("parser should have the passed token list")
		}
	}
}

func BenchmarkNewParser(b *testing.B) {
	tokens := make([]Token, 0)
	for i := 0; i < b.N; i++ {
		_ = NewParser(tokens)
	}
}

type TokenTestData struct {
	tokens []Token
	tree   Node
}

func GetTokenTestData() map[string]TokenTestData {
	return map[string]TokenTestData{
		"empty": {
			[]Token{
				NewToken(TokenEOF, 0, 0, 0, ""),
			},
			&BlockNode{},
		},
		"addition": {
			[]Token{
				NewToken(TokenName, 0, 1, 0, "_"),
				NewToken(TokenAssign, 1, 1, 0, "="),
				NewToken(TokenNumber, 3, 1, 0, "1"),
				NewToken(TokenPlus, 4, 1, 0, "+"),
				NewToken(TokenNumber, 5, 1, 0, "2"),
				NewToken(TokenEOF, 6, 0, 0, ""),
			},
			&BlockNode{
				[]Node{
					&AssignNode{
						"_",
						&BinaryNode{
							BinaryAddition,
							&NumberNode{
								1,
							},
							&NumberNode{
								2,
							},
						},
						false,
					},
				},
			},
		},
		"assignment": {
			[]Token{
				NewToken(TokenName, 0, 5, 0, "hello"),
				NewToken(TokenAssign, 5, 1, 0, "="),
				NewToken(TokenString, 6, 12, 0, "\"Hello world!\""),
				NewToken(TokenEOF, 18, 0, 0, ""),
			},
			&BlockNode{
				[]Node{
					&AssignNode{
						"hello",
						&StringNode{
							"Hello world!",
							"\"Hello world!\"",
						},
						false,
					},
				},
			},
		},
		"declaration": {
			[]Token{
				NewToken(TokenName, 0, 1, 0, "a"),
				NewToken(TokenDeclare, 1, 2, 0, ":="),
				NewToken(TokenNumber, 3, 1, 0, "1"),
				NewToken(TokenPlus, 4, 1, 0, "+"),
				NewToken(TokenName, 5, 1, 0, "b"),
				NewToken(TokenEOF, 6, 0, 0, ""),
			},
			&BlockNode{
				[]Node{
					&AssignNode{
						"a",
						&BinaryNode{
							BinaryAddition,
							&NumberNode{
								1,
							},
							&ReferenceNode{
								"b",
							},
						},
						true,
					},
				},
			},
		},
		// (2 + 1) * 5 + 3 / (6 - 2) - 10 / 2
		"arithmetic_order": {
			[]Token{
				NewToken(TokenName, 0, 1, 0, "_"),
				NewToken(TokenAssign, 1, 2, 0, "="),

				NewToken(TokenOpenParenthesis, 3, 1, 0, "("),
				NewToken(TokenNumber, 4, 1, 0, "2"),
				NewToken(TokenPlus, 5, 1, 0, "+"),
				NewToken(TokenNumber, 6, 1, 0, "1"),
				NewToken(TokenCloseParenthesis, 7, 1, 0, ")"),
				NewToken(TokenStar, 8, 1, 0, "*"),
				NewToken(TokenNumber, 9, 1, 0, "5"),

				NewToken(TokenPlus, 10, 1, 0, "+"),

				NewToken(TokenNumber, 11, 1, 0, "3"),
				NewToken(TokenSlash, 12, 1, 0, "/"),
				NewToken(TokenOpenParenthesis, 13, 1, 0, "("),
				NewToken(TokenNumber, 14, 1, 0, "6"),
				NewToken(TokenMinus, 15, 1, 0, "-"),
				NewToken(TokenNumber, 16, 1, 0, "2"),
				NewToken(TokenCloseParenthesis, 17, 1, 0, ")"),

				NewToken(TokenMinus, 18, 1, 0, "-"),

				NewToken(TokenNumber, 19, 2, 0, "10"),
				NewToken(TokenSlash, 20, 1, 0, "/"),
				NewToken(TokenNumber, 21, 1, 0, "2"),
				NewToken(TokenEOF, 22, 0, 0, ""),
			},
			// (2 + 1) * 5 + 3 / (6 - 2)  -  10 / 2
			&BlockNode{
				[]Node{
					&AssignNode{
						"_",
						&BinaryNode{
							BinarySubtraction,
							&BinaryNode{
								BinaryAddition,
								&BinaryNode{
									BinaryMultiplication,
									&BinaryNode{
										BinaryAddition,
										&NumberNode{2},
										&NumberNode{1},
									},
									&NumberNode{5},
								},
								&BinaryNode{
									BinaryDivision,
									&NumberNode{3},
									&BinaryNode{
										BinarySubtraction,
										&NumberNode{6},
										&NumberNode{2},
									},
								},
							},
							&BinaryNode{
								BinaryDivision,
								&NumberNode{10},
								&NumberNode{2},
							},
						},
						false,
					},
				},
			},
		},
		"condition_equal": {
			[]Token{
				NewToken(TokenName, 0, 1, 0, "_"),
				NewToken(TokenAssign, 1, 1, 0, "="),
				NewToken(TokenNumber, 2, 2, 0, "20"),
				NewToken(TokenEquals, 4, 2, 0, "=="),
				NewToken(TokenNumber, 6, 2, 0, "15"),
				NewToken(TokenEOF, 8, 0, 0, ""),
			},
			&BlockNode{
				[]Node{
					&AssignNode{
						"_",
						&BinaryNode{
							BinaryEquality,
							&NumberNode{
								20,
							},
							&NumberNode{
								15,
							},
						},
						false,
					},
				},
			},
		},
		"if_statement": {
			[]Token{
				NewToken(TokenIf, 0, 2, 0, "if"),
				NewToken(TokenName, 2, 1, 0, "a"),
				NewToken(TokenEquals, 3, 2, 0, "=="),
				NewToken(TokenNumber, 5, 1, 0, "0"),
				NewToken(TokenOpenBrace, 6, 1, 0, "{"),
				NewToken(TokenName, 7, 1, 1, "b"),
				NewToken(TokenAssign, 8, 1, 1, "="),
				NewToken(TokenNumber, 9, 1, 1, "1"),
				NewToken(TokenCloseBrace, 10, 1, 2, "}"),
				NewToken(TokenEOF, 11, 0, 2, ""),
			},
			&BlockNode{
				[]Node{
					&ConditionalNode{
						condition: &BinaryNode{
							BinaryEquality,
							&ReferenceNode{
								"a",
							},
							&NumberNode{
								0,
							},
						},
						do: &BlockNode{
							[]Node{
								&AssignNode{
									"b",
									&NumberNode{
										1,
									},
									false,
								},
							},
						},
					},
				},
			},
		},
		"if_else_statement": {
			[]Token{
				NewToken(TokenIf, 0, 2, 0, "if"),
				NewToken(TokenName, 2, 1, 0, "a"),
				NewToken(TokenEquals, 3, 2, 0, "=="),
				NewToken(TokenNumber, 5, 1, 0, "0"),
				NewToken(TokenOpenBrace, 6, 1, 0, "{"),
				NewToken(TokenName, 7, 1, 1, "b"),
				NewToken(TokenAssign, 8, 1, 1, "="),
				NewToken(TokenNumber, 9, 1, 1, "1"),
				NewToken(TokenCloseBrace, 10, 1, 2, "}"),
				NewToken(TokenElse, 11, 4, 2, "else"),
				NewToken(TokenOpenBrace, 15, 1, 2, "{"),
				NewToken(TokenName, 16, 1, 2, "b"),
				NewToken(TokenAssign, 17, 1, 2, "="),
				NewToken(TokenNumber, 18, 1, 2, "0"),
				NewToken(TokenCloseBrace, 19, 1, 2, "}"),
				NewToken(TokenEOF, 20, 0, 2, ""),
			},
			&BlockNode{
				[]Node{
					&ConditionalNode{
						condition: &BinaryNode{
							BinaryEquality,
							&ReferenceNode{
								"a",
							},
							&NumberNode{
								0,
							},
						},
						do: &BlockNode{
							[]Node{
								&AssignNode{
									"b",
									&NumberNode{
										1,
									},
									false,
								},
							},
						},
						otherwise: &BlockNode{
							[]Node{
								&AssignNode{
									"b",
									&NumberNode{
										0,
									},
									false,
								},
							},
						},
					},
				},
			},
		},
		"empty_block": {
			[]Token{
				NewToken(TokenOpenBrace, 0, 1, 0, "{"),
				NewToken(TokenCloseBrace, 1, 1, 0, "}"),
				NewToken(TokenEOF, 2, 0, 0, ""),
			},
			&BlockNode{
				[]Node{
					&BlockNode{
						[]Node{},
					},
				},
			},
		},
		"lambda": { // a := func(a, b) { return a + b }
			[]Token{
				NewToken(TokenName, 0, 1, 0, "a"),
				NewToken(TokenDeclare, 1, 2, 0, ":="),
				NewToken(TokenFunc, 3, 4, 0, "func"),
				NewToken(TokenOpenParenthesis, 7, 1, 0, "("),
				NewToken(TokenName, 8, 1, 0, "a"),
				NewToken(TokenComma, 9, 1, 0, ","),
				NewToken(TokenName, 10, 1, 0, "b"),
				NewToken(TokenCloseParenthesis, 11, 1, 0, ")"),

				NewToken(TokenOpenBrace, 12, 1, 1, "{"),
				NewToken(TokenReturn, 13, 6, 1, "return"),
				NewToken(TokenName, 19, 1, 1, "a"),
				NewToken(TokenPlus, 20, 1, 1, "+"),
				NewToken(TokenName, 21, 1, 1, "b"),
				NewToken(TokenCloseBrace, 22, 1, 2, "}"),

				NewToken(TokenEOF, 23, 0, 2, ""),
			},
			&BlockNode{
				[]Node{
					&AssignNode{
						"a",
						&FunctionNode{
							"*",
							[]string{"a", "b"},
							&BlockNode{
								[]Node{
									&ReturnNode{
										&BinaryNode{
											BinaryAddition,
											&ReferenceNode{
												"a",
											},
											&ReferenceNode{
												"b",
											},
										},
									},
								},
							},
						},
						true,
					},
				},
			},
		},
		"function_declaration": {
			[]Token{
				NewToken(TokenFunc, 0, 4, 0, "func"),
				NewToken(TokenName, 4, 3, 0, "a"),
				NewToken(TokenOpenParenthesis, 7, 1, 0, "("),
				NewToken(TokenName, 8, 1, 0, "a"),
				NewToken(TokenComma, 9, 1, 0, ","),
				NewToken(TokenName, 10, 1, 0, "b"),
				NewToken(TokenCloseParenthesis, 11, 1, 0, ")"),

				NewToken(TokenOpenBrace, 12, 1, 1, "{"),
				NewToken(TokenReturn, 13, 6, 1, "return"),
				NewToken(TokenName, 19, 1, 1, "a"),
				NewToken(TokenPlus, 20, 1, 1, "+"),
				NewToken(TokenName, 21, 1, 1, "b"),
				NewToken(TokenCloseBrace, 22, 1, 2, "}"),

				NewToken(TokenEOF, 23, 0, 2, ""),
			},
			&BlockNode{
				[]Node{
					&AssignNode{
						"a",
						&FunctionNode{
							"a",
							[]string{"a", "b"},
							&BlockNode{
								[]Node{
									&ReturnNode{
										&BinaryNode{
											BinaryAddition,
											&ReferenceNode{
												"a",
											},
											&ReferenceNode{
												"b",
											},
										},
									},
								},
							},
						},
						true,
					},
				},
			},
		},
		"prop_getting": {
			[]Token{
				NewToken(TokenName, 0, 1, 0, "p"),
				NewToken(TokenDeclare, 1, 2, 0, ":="),
				NewToken(TokenName, 3, 1, 0, "a"),
				NewToken(TokenDot, 4, 1, 0, "."),
				NewToken(TokenName, 5, 1, 0, "b"),

				NewToken(TokenEOF, 23, 0, 2, ""),
			},
			&BlockNode{
				[]Node{
					&AssignNode{
						"p",
						&AccessNode{
							&ReferenceNode{
								"a",
							},
							"b",
						},
						true,
					},
				},
			},
		},
		"list_init": {
			[]Token{
				NewToken(TokenName, 0, 4, 0, "data"),
				NewToken(TokenDeclare, 4, 2, 0, ":="),

				NewToken(TokenOpenBracket, 6, 1, 0, "["),
				NewToken(TokenName, 8, 1, 0, "a"),
				NewToken(TokenComma, 11, 1, 0, ","),

				NewToken(TokenNumber, 8, 1, 0, "3.141"),
				NewToken(TokenComma, 11, 1, 0, ","),

				NewToken(TokenString, 6, 1, 0, "\"Hello world!\""),
				NewToken(TokenComma, 11, 1, 0, ","),

				NewToken(TokenTrue, 6, 1, 0, "true"),
				NewToken(TokenComma, 11, 1, 0, ","),

				NewToken(TokenOpenBracket, 6, 1, 0, "["),
				NewToken(TokenNumber, 6, 1, 0, "2"),
				NewToken(TokenComma, 11, 1, 0, ","),

				NewToken(TokenNumber, 6, 1, 0, "3"),
				NewToken(TokenCloseBracket, 6, 1, 0, "]"),

				NewToken(TokenCloseBracket, 6, 1, 0, "]"),

				NewToken(TokenEOF, 23, 0, 2, ""),
			},
			&BlockNode{
				[]Node{
					&AssignNode{
						"data",
						&ListNode{
							[]Node{
								&ReferenceNode{
									"a",
								},
								&NumberNode{
									3.141,
								},
								&StringNode{
									"Hello world!",
									"\"Hello world!\"",
								},
								&BooleanNode{
									true,
								},
								&ListNode{
									[]Node{
										&NumberNode{2}, &NumberNode{3},
									},
								},
							},
						},
						true,
					},
				},
			},
		},
	}
}

func NodeEquality(t *testing.T, n1 Node, n2 Node) {
	if n1 == n2 {
		return
	}

	if n1 == nil || n2 == nil {
		t.Fatalf("one of the nodes are nil (1: %s; 2: %s)", n1, n2)
	}

	if n1.Type() != n2.Type() {
		t.Fatalf("node types (%s and %s) don't match", n1.Type(), n2.Type())
	}

	t.Logf("Nodes have same non-nil type (%s)", n1.Type())

	switch n1.Type() {
	case NilNodeType:
	case StringNodeType:
		if n1.(*StringNode).value != n2.(*StringNode).value {
			t.Errorf("String node values don't match (%s and %s)", n1.(*StringNode).value, n2.(*StringNode).value)
		} else {
			t.Logf("String node values match (%s)", n1.(*StringNode).value)
		}

		if n1.(*StringNode).quoted != n2.(*StringNode).quoted {
			t.Errorf("String node quoted values don't match (%s and %s)", n1.(*StringNode).quoted, n2.(*StringNode).quoted)
		} else {
			t.Logf("String node quoted values match (%s)", n1.(*StringNode).value)
		}

	case NumberNodeType:
		if n1.(*NumberNode).value != n2.(*NumberNode).value {
			t.Errorf("Number node values don't match (%f and %f)", n1.(*NumberNode).value, n2.(*NumberNode).value)
		} else {
			t.Logf("Number node values match (%f)", n1.(*NumberNode).value)
		}
	case ReferenceNodeType:
		if n1.(*ReferenceNode).name != n2.(*ReferenceNode).name {
			t.Errorf("Reference node values don't match (%s and %s)", n1.(*ReferenceNode).name, n2.(*ReferenceNode).name)
		} else {
			t.Logf("Reference node values match (%s)", n1.(*ReferenceNode).name)
		}
	case BinaryNodeType:
		if n1.(*BinaryNode).BinaryOperation != n2.(*BinaryNode).BinaryOperation {
			t.Errorf("Binary node operation not same (%s and %s)", n1.(*BinaryNode).BinaryOperation, n2.(*BinaryNode).BinaryOperation)
		} else {
			t.Logf("Binary node operation matches (%s)", n1.(*BinaryNode).BinaryOperation)
		}

		t.Log("Checking equality of binary left side")
		NodeEquality(t, n1.(*BinaryNode).Left, n2.(*BinaryNode).Left)
		t.Log("Checking equality of binary right side")
		NodeEquality(t, n1.(*BinaryNode).Right, n2.(*BinaryNode).Right)

	case BooleanNodeType:
		if n1.(*BooleanNode).value != n2.(*BooleanNode).value {
			t.Errorf("Boolean node values don't match (%s and %s)", strconv.FormatBool(n1.(*BooleanNode).value), strconv.FormatBool(n2.(*BooleanNode).value))
		} else {
			t.Logf("Boolean node values match (%s)", strconv.FormatBool(n1.(*BooleanNode).value))
		}
	case BlockNodeType:
		if len(n1.(*BlockNode).statements) != len(n2.(*BlockNode).statements) {
			t.Errorf("Block node statement count is not equal (%d and %d)", len(n1.(*BlockNode).statements), len(n2.(*BlockNode).statements))
		} else {
			t.Logf("Block node statement count is equal (%d) ", len(n1.(*BlockNode).statements))
		}

		for i, n := range n1.(*BlockNode).statements {
			t.Logf("Checking equality of statements at %d", i)
			NodeEquality(t, n, n2.(*BlockNode).statements[i])
		}

	case ConditionalNodeType:
		t.Log("Checking equality of conditions")
		NodeEquality(t, n1.(*ConditionalNode).condition, n2.(*ConditionalNode).condition)
		t.Log("Checking equality of do statement(s)")
		NodeEquality(t, n1.(*ConditionalNode).do, n2.(*ConditionalNode).do)
		t.Log("Checking equality of else statement(s)")
		NodeEquality(t, n1.(*ConditionalNode).otherwise, n2.(*ConditionalNode).otherwise)

	case LoopNodeType:
		t.Log("Checking equality of loop conditions")
		NodeEquality(t, n1.(*LoopNode).condition, n2.(*LoopNode).condition)
		t.Log("Checking equality of do loop statement(s)")
		NodeEquality(t, n1.(*LoopNode).do, n2.(*LoopNode).do)

	case AssignNodeType:
		if n1.(*AssignNode).name != n2.(*AssignNode).name {
			t.Errorf("Assigned value name is not the same (%s and %s)", n1.(*AssignNode).name, n2.(*AssignNode).name)
		} else {
			t.Logf("Assigned value name matches (%s)", n1.(*AssignNode).name)
		}

		if n1.(*AssignNode).declare != n2.(*AssignNode).declare {
			t.Errorf("Not same type of assigning (1: %v; 2: %v)", n1.(*AssignNode).declare, n2.(*AssignNode).declare)
		}

		t.Logf("Checking equality of assignment values")
		NodeEquality(t, n1.(*AssignNode).value, n2.(*AssignNode).value)

	case CallNodeType:
		n := n1.(*CallNode)
		m := n2.(*CallNode)

		NodeEquality(t, n.source, m.source)

		if n.keep == m.keep {
			t.Logf("Call node keep modifier doesn't match (%v and %v)", n.keep, m.keep)
		}

		if len(n.args) != len(m.args) {
			t.Fatalf("Call node arguments count does not match (%d and %d)", len(n.args), m.args)
		}

		for i, arg := range m.args {
			NodeEquality(t, n1.(*CallNode).args[i], arg)
		}

	case FunctionNodeType:
		n := n1.(*FunctionNode)
		m := n2.(*FunctionNode)

		if n.name != m.name {
			t.Errorf("Function node names don't match (%s and %s)", n.name, m.name)
		} else {
			t.Logf("Function node names match (%s)", n.name)
		}

		if len(n.params) != len(m.params) {
			t.Fatalf("Function node parameters count does not match (%d and %d)", len(n.params), len(m.params))
		} else {
			t.Logf("Function node parameters count is equal (%d) ", len(n.params))
		}

		for i, p := range m.params {
			if n.params[i] != p {
				t.Errorf("Function node parameter %d does not match: %s and %s", i, p, m.params)
			} else {
				t.Logf("Function node parameter %d matches (%s)", i, p)
			}
		}

		NodeEquality(t, n.logic, m.logic)

	case ReturnNodeType:
		NodeEquality(t, n1.(*ReturnNode).value, n2.(*ReturnNode).value)
	default:
		panic("unimplemented node equality")
	}
}

func TestParser_Parse(t *testing.T) {
	t.Logf("Getting test data")
	tokenData := GetTokenTestData()

	for name, data := range tokenData {
		if name != "empty_block" && name != "lambda" {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Logf("Initializing parser")
			p := NewParser(data.tokens)

			t.Logf("Parsing main")
			tree, err := p.Parse()

			if err != nil {
				t.Fatalf("Unexpected error(s): %s", err.(*ParsingError).Format([]rune{}))
			}

			t.Logf("Checking parsed tree")
			NodeEquality(t, tree, data.tree)
		})
	}
}

func BenchmarkParser_Parse(b *testing.B) {
	tokenData := GetTokenTestData()

	for name, data := range tokenData {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				p := NewParser(data.tokens)

				_, _ = p.Parse()
			}
		})
	}
}
