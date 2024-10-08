package main

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

	if c.chunk == nil {
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
				StringValue("Hello world!"),
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
					NumberValue(0),
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
					NumberValue(1),
					0,
				},
			},
		},
		"conditional_welse_false": {
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
					NumberValue(2),
					0,
				},
			},
		},
		"conditional_welse_true": {
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
					NumberValue(1),
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
				NumberValue(3),
			},
		},
		"sum_function": {
			&AssignNode{
				"sum",
				&FunctionNode{
					"sum",
					[]string{"a", "b"},
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
			[]Value{
				&VariableValue{
					"sum",
					FunctionValue{
						"sum",
						[]string{"a", "b"},
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
								StringValue("a"), StringValue("b"),
							},
						),
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

		f, ok := ct.(FunctionValue)
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
			c.Compile(testCase.tree)

			t.Log("Initializing vm")
			vm := NewVM(c.chunk, 256, 256)

			t.Log("Printing chunk for debug")
			printChunk(t, name, c.chunk)

			t.Log("Executing bytecode")
			for vm.HasNext() && vm.Next() {
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
				c.Compile(testCase.tree)
			}
		})
	}
}

func TestCompiler_AddU16(t *testing.T) {
	for i := 0; i <= 0xffff; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c := NewCompiler()
			c.addU16(uint16(i))

			if c.chunk.Bytecode[0] != Bytecode(i>>8) {
				t.Errorf("first 8 bits don't match (got %s, expected %b)", c.chunk.Bytecode[0], byte(i>>8))
			}

			if c.chunk.Bytecode[1] != Bytecode(i&0xff) {
				t.Errorf("last 8 bits don't match (got %s, expected %b)", c.chunk.Bytecode[1], byte(i&0xff))
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
		case BlockNodeType:
		case ConditionalNodeType:
		case LoopNodeType:
		case AssignNodeType:
		case FunctionNodeType:
		}

		t.Run(name, func(t *testing.T) {
			c := NewCompiler()
			c.Compile(tc.tree)

			vm := NewVM(c.chunk, 256, 256)
			for vm.HasNext() && vm.Next() {
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
