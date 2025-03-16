package core

import (
	"fmt"
	"testing"
)

func TestNewCompiler(t *testing.T) {
	c := NewCompiler()

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
		_ = NewCompiler()
	}
}

type CompileTestData struct {
	tree          Node
	expectedStack []Value
}

func GetCompileTestData() map[string]CompileTestData {
	return map[string]CompileTestData{
		"constant_string": {
			&StringNode{
				"Hello world!",
				"\"Hello world!\"",
			},
			[]Value{
				&StringValue{"Hello world!"},
			},
		},
		"conditional_false": {
			&BlockNode{
				[]Node{
					&AssignNode{
						"a",
						&NumberNode{
							0,
						},
						true,
					},
					&ConditionalNode{
						&BooleanNode{
							false,
						},
						&BlockNode{
							[]Node{
								&AssignNode{
									"a",
									&NumberNode{
										1,
									},
									false,
								},
							},
						},
						nil,
					},
				},
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
			&BlockNode{
				[]Node{
					&AssignNode{
						"a",
						&NumberNode{
							0,
						},
						true,
					},
					&ConditionalNode{
						&BooleanNode{
							true,
						},
						&BlockNode{
							[]Node{
								&AssignNode{
									"a",
									&NumberNode{
										1,
									},
									false,
								},
							},
						},
						nil,
					},
				},
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
			&BlockNode{
				[]Node{
					&AssignNode{
						"a",
						&NumberNode{
							0,
						},
						true,
					},
					&ConditionalNode{
						&BooleanNode{
							false,
						},
						&BlockNode{
							[]Node{
								&AssignNode{
									"a",
									&NumberNode{
										1,
									},
									false,
								},
							},
						},
						&BlockNode{
							[]Node{
								&AssignNode{
									"a",
									&NumberNode{
										2,
									},
									false,
								},
							},
						},
					},
				},
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
			&BlockNode{
				[]Node{
					&AssignNode{
						"a",
						&NumberNode{
							0,
						},
						true,
					},
					&ConditionalNode{
						&BooleanNode{
							true,
						},
						&BlockNode{
							[]Node{
								&AssignNode{
									"a",
									&NumberNode{
										1,
									},
									false,
								},
							},
						},
						&BlockNode{
							[]Node{
								&AssignNode{
									"a",
									&NumberNode{
										2,
									},
									false,
								},
							},
						},
					},
				},
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
			&BinaryNode{
				BinaryAddition,
				&NumberNode{
					1,
				},
				&NumberNode{
					2,
				},
			},
			[]Value{
				&NumberValue{3},
			},
		},
		"sum_function": {&BlockNode{
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
										&ReferenceNode{"a"},
										&ReferenceNode{"b"},
									},
								},
							},
						},
					},
					true,
				},
			},
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
			&BlockNode{
				[]Node{
					&AssignNode{
						"a",
						&FunctionNode{
							"a",
							[]FunctionParameter{},
							&NilSignature{},
							&BlockNode{
								[]Node{
									&AssignNode{
										"b",
										&NumberNode{1},
										true,
									},
									&ReturnNode{
										&ReferenceNode{"b"},
									},
								},
							},
						},
						true,
					},
					&CallNode{
						&ReferenceNode{
							"a",
						},
						[]Node{},
						false,
					},
				},
			},
			[]Value{
				&VariableValue{
					"a",
					&FunctionValue{
						"a",
						[]FunctionParameter{},
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
	}
}

func printChunk(t *testing.T, name string, chunk *Chunk) {
	t.Logf("=v= %s =v=", name)
	for i, bc := range chunk.Bytecode {
		t.Logf("i=%d \t%d \t(%s)", i, bc, bc)
	}

	t.Logf("=-= constants =-=")

	for i, ct := range chunk.Constants {
		t.Logf("c=%d \t%s", i, ct)

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
			c := NewCompiler()

			t.Log("Compiling node tree")
			err := c.Compile(testCase.tree)
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
				c := NewCompiler()
				_ = c.Compile(testCase.tree)
			}
		})
	}
}

func TestCompiler_AddU16(t *testing.T) {
	for i := 0; i <= 0xffff; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c := NewCompiler()
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
		switch tc.tree.Type() {
		// skip all expected unclean nodes
		case StringNodeType, NumberNodeType, ReferenceNodeType, BooleanNodeType, NilNodeType, BinaryNodeType, ReturnNodeType:
			continue

		case CallNodeType:
			if tc.tree.(*CallNode).keep {
				// if we know it should be unclean, skip it
				continue
			}

		// clean statements
		default:
		}

		t.Run(name, func(t *testing.T) {
			c := NewCompiler()
			err := c.Compile(tc.tree)
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
