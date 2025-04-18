package core

import (
	"fmt"
	"testing"
)

func TestNewCompiler(t *testing.T) {
	c := NewCompiler([]rune{})

	if c == nil {
		t.Fatal("NewCompiler returned nil")
	}

	if c.ip != 0 {
		t.Error("compiler ip doesn't start at zero")
	}

	if c.Chunk == nil {
		t.Error("compiler chunk initialized to nil")
	}
}

func BenchmarkNewCompiler(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewCompiler([]rune{})
	}
}

type CompileTestData struct {
	program       *Program
	expectedStack []Value
}

func GetCompileTestData() map[string]CompileTestData {
	return map[string]CompileTestData{
		"constant_string": {
			&Program{
				[]string{},
				&BlockNode{
					[]Node{
						&AssignNode{
							"a",
							&StringNode{
								"Hello world!",
								"\"Hello world!\"",
								0, 0,
							},
							true,
							0, 0,
						},
					},
					0, 0,
				},
				"",
			},
			[]Value{
				&VariableValue{
					"a",
					&StringValue{"Hello world!"},
					0,
				},
			},
		},
		"conditional_false": {
			&Program{
				[]string{},
				&BlockNode{
					[]Node{
						&AssignNode{
							"a",
							&NumberNode{
								0,
								0, 0,
							},
							true,
							0, 0,
						},
						&ConditionalNode{
							&BooleanNode{
								false,
								0, 0,
							},
							&BlockNode{
								[]Node{
									&AssignNode{
										"a",
										&NumberNode{
											1,
											0, 0,
										},
										false,
										0, 0,
									},
								},
								0, 0,
							},
							nil,
							0, 0,
						},
					},
					0, 0,
				},
				"",
			},
			[]Value{
				&VariableValue{
					"a",
					&NumberValue{0},
					0,
				},
			},
		},
		"conditional_true": {
			&Program{
				[]string{},
				&BlockNode{
					[]Node{
						&AssignNode{
							"a",
							&NumberNode{
								0,
								0, 0,
							},
							true,
							0, 0,
						},
						&ConditionalNode{
							&BooleanNode{
								true,
								0, 0,
							},
							&BlockNode{
								[]Node{
									&AssignNode{
										"a",
										&NumberNode{
											1,
											0, 0,
										},
										false,
										0, 0,
									},
								},
								0, 0,
							},
							nil,
							0, 0,
						},
					},
					0, 0,
				},
				"",
			},
			[]Value{
				&VariableValue{
					"a",
					&NumberValue{1},
					0,
				},
			},
		},
		"conditional_else_false": {
			&Program{
				[]string{},
				&BlockNode{
					[]Node{
						&AssignNode{
							"a",
							&NumberNode{
								0,
								0, 0,
							},
							true,
							0, 0,
						},
						&ConditionalNode{
							&BooleanNode{
								false,
								0, 0,
							},
							&BlockNode{
								[]Node{
									&AssignNode{
										"a",
										&NumberNode{
											1,
											0, 0,
										},
										false,
										0, 0,
									},
								},
								0, 0,
							},
							&BlockNode{
								[]Node{
									&AssignNode{
										"a",
										&NumberNode{
											2,
											0, 0,
										},
										false,
										0, 0,
									},
								},
								0, 0,
							},
							0, 0,
						},
					},
					0, 0,
				},
				"",
			},
			[]Value{
				&VariableValue{
					"a",
					&NumberValue{2},
					0,
				},
			},
		},
		"conditional_else_true": {
			&Program{
				[]string{},
				&BlockNode{
					[]Node{
						&AssignNode{
							"a",
							&NumberNode{
								0,
								0, 0,
							},
							true,
							0, 0,
						},
						&ConditionalNode{
							&BooleanNode{
								true,
								0, 0,
							},
							&BlockNode{
								[]Node{
									&AssignNode{
										"a",
										&NumberNode{
											1,
											0, 0,
										},
										false,
										0, 0,
									},
								},
								0, 0,
							},
							&BlockNode{
								[]Node{
									&AssignNode{
										"a",
										&NumberNode{
											2,
											0, 0,
										},
										false,
										0, 0,
									},
								},
								0, 0,
							},
							0, 0,
						},
					},
					0, 0,
				},
				"",
			},
			[]Value{
				&VariableValue{
					"a",
					&NumberValue{1},
					0,
				},
			},
		},
		"addition": {
			&Program{
				[]string{},
				&BlockNode{
					[]Node{
						&AssignNode{
							"a",
							&BinaryNode{
								BinaryAddition,
								&NumberNode{
									1,
									0, 0,
								},
								&NumberNode{
									2,
									0, 0,
								},
								0, 0,
							},
							true,
							0, 0,
						},
					},
					0, 0,
				},
				"",
			},
			[]Value{
				&VariableValue{
					"a",
					&NumberValue{3},
					0,
				},
			},
		},
		"sum_function": {
			&Program{
				[]string{},
				&BlockNode{
					[]Node{
						&AssignNode{
							"sum",
							&FunctionNode{
								"sum",
								[]FunctionParameter{
									{
										"a",
										&NumberSignature{},
									},
									{
										"b",
										&NumberSignature{},
									},
								},
								&NumberSignature{},
								&BlockNode{
									[]Node{
										&ReturnNode{
											&BinaryNode{
												BinaryAddition,
												&ReferenceNode{
													"a",
													0, 0,
												},
												&ReferenceNode{
													"b",
													0, 0,
												},
												0, 0,
											},
											0, 0,
										},
									},
									0, 0,
								},
								0, 0,
							},
							true,
							0, 0,
						},
					},
					0, 0,
				},
				"",
			},
			[]Value{
				&VariableValue{
					"sum",

					&FunctionValue{
						"sum",
						[]FunctionParameter{
							{
								"a",
								&NumberSignature{},
							},
							{
								"b",
								&NumberSignature{},
							},
						},
						&NumberSignature{},
						NewChunk(
							[]Bytecode{
								InstructionDescend,
								InstructionGetLocal, 0,
								InstructionGetLocal, 1,
								InstructionAdd,
								InstructionReturn,
								InstructionAscend,
							},
							[]Value{
								&StringValue{"a"}, &StringValue{"b"},
							},
						),
						nil,
					},
					0,
				},
			},
		},
		"remove_func_vars": {
			&Program{
				[]string{},
				&BlockNode{
					[]Node{
						&AssignNode{
							"a",
							&FunctionNode{
								"a",
								[]FunctionParameter{},
								&NumberSignature{},
								&BlockNode{
									[]Node{
										&AssignNode{
											"b",
											&NumberNode{
												1,
												0, 0,
											},
											true,
											0, 0,
										},
										&ReturnNode{
											&ReferenceNode{
												"b",
												0, 0,
											},
											0, 0,
										},
									},
									0, 0,
								},
								0, 0,
							},
							true,
							0, 0,
						},
						&CallNode{
							&ReferenceNode{
								"a",
								0, 0,
							},
							[]Node{},
							false,
							0, 0,
						},
					},
					0, 0,
				},
				"",
			},
			[]Value{
				&VariableValue{
					"a",
					&FunctionValue{
						"a",
						[]FunctionParameter{},
						&NumberSignature{},
						NewChunk(
							[]Bytecode{
								InstructionDescend,
								InstructionConstant, 0,
								InstructionDeclareLocal, 1,
								InstructionGetLocal, 1,
								InstructionReturn,
								InstructionAscend,
							},
							[]Value{
								&NumberValue{1}, &StringValue{"b"},
							},
						),
						nil,
					},
					0,
				},
			},
		},
		"two_lists": {
			program: &Program{
				[]string{},
				&BlockNode{
					statements: []Node{
						&AssignNode{
							name: "a",
							value: &ListNode{
								items: []Node{
									&NumberNode{value: 1},
									&NumberNode{value: 2},
								},
							},
							declare: true,
						},
						&AssignNode{
							name: "b",
							value: &ListNode{
								items: []Node{
									&StringNode{value: "Hello"},
									&StringNode{value: "world"},
								},
							},
							declare: true,
						},
					},
				},
				"",
			},
			expectedStack: []Value{
				&VariableValue{
					name: "a",
					value: &ListValue{
						Items: []Value{
							&NumberValue{1},
							&NumberValue{2},
						},
					},
					scope: 0,
				},
				&VariableValue{
					name: "b",
					value: &ListValue{
						Items: []Value{
							&StringValue{"Hello"},
							&StringValue{"world"},
						},
					},
					scope: 0,
				},
			},
		},
	}
}

func printChunk(t *testing.T, name string, chunk *Chunk) {
	t.Logf("=v= %s =v=", name)
	for i, bc := range chunk.Bytecode {
		t.Logf("i=%d \t%d \t(%s)", i, bc, bc)
	}

	t.Logf("=-= constants =-=")

	for i, ct := range chunk.Constants {
		t.Logf("c=%d \t%s", i, ct.DebugString())

		f, ok := ct.(*FunctionValue)
		if ok {
			printChunk(t, f.Name, f.Chunk)
		}
	}

	t.Logf("=^= %s =^=", name)
}

func TestCompile(t *testing.T) {
	data := GetCompileTestData()

	for name, testCase := range data {
		t.Run(name, func(t *testing.T) {
			t.Log("Initializing compiler")
			c := NewCompiler([]rune(testCase.program.String()))

			t.Log("Compiling node tree")
			err := c.Compile(testCase.program)
			if err != nil {
				t.Fatalf("Compiling failed: %v", err)
			}

			t.Log("Initializing vm")
			vm := NewVM(c.Chunk, 256, 256)

			t.Log("Printing chunk for debug")
			printChunk(t, name, c.Chunk)

			t.Log("Executing bytecode")
			for vm.Next() {
			}
			t.Log("Executed bytecode")

			CompareStacks(t, testCase.expectedStack, vm.stack)
		})
	}
}

func BenchmarkCompile(b *testing.B) {
	data := GetCompileTestData()
	for name, testCase := range data {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				c := NewCompiler([]rune{})
				_ = c.Compile(testCase.program)
			}
		})
	}
}

func TestCompiler_AddU16(t *testing.T) {
	for i := 0; i <= 0xffff; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c := NewCompiler([]rune{})
			c.addU16(uint16(i))

			if c.Chunk.Bytecode[0] != Bytecode(i>>8) {
				t.Errorf("first 8 bits don't match (got %s, expected %b)", c.Chunk.Bytecode[0], byte(i>>8))
			}

			if c.Chunk.Bytecode[1] != Bytecode(i&0xff) {
				t.Errorf("last 8 bits don't match (got %s, expected %b)", c.Chunk.Bytecode[1], byte(i&0xff))
			}
		})
	}
}

func TestCompiler_CleanStack(t *testing.T) {
	cases := GetCompileTestData()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := NewCompiler([]rune(tc.program.String()))
			err := c.Compile(tc.program)
			if err != nil {
				t.Fatalf("Compiling failed: %v", err)
			}

			vm := NewVM(c.Chunk, 256, 256)
			for vm.Next() {
			}

			// make sure stack has only assigned values
			for i := 0; i < int(vm.stack.Current); i++ {
				v := vm.stack.items[i]

				if v == nil || v.Type() != VariableValueType {
					t.Errorf("Unclean stack! value %v at %d on the stack is intermediary", v.String(), i)
				}
			}
		})
	}
}
