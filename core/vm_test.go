package core

import (
	"fmt"
	"testing"
)

func CompareChunks(t *testing.T, got *Chunk, want *Chunk) {
	if len(got.Constants) != len(want.Constants) {
		t.Errorf("constant count does not match; got %v, expected %v", len(got.Constants), len(want.Constants))
	}

	for i, v := range got.Constants {
		if v != want.Constants[i] {
			t.Errorf("constant %d does not match (%s and %s)", i, v.String(), want.Constants[i].String())
		}
	}

	if len(got.Bytecode) != len(want.Bytecode) {
		t.Errorf("bytecode size does not match; got %v, expected %v", len(got.Bytecode), len(want.Bytecode))
	}

	t.Log("instruction \t\tchunk got \t\tchunk want")

	i := 0

	for ; i < len(got.Bytecode); i++ {
		v := got.Bytecode[i]
		if i < len(want.Bytecode) {
			if v != want.Bytecode[i] {
				t.Errorf("i=%d mismatch \t%d (%s) \t\t%d (%s)", i, v, v.String(), want.Bytecode[i], want.Bytecode[i].String())
			} else {
				t.Logf("i=%d match    \t%d (%s) \t\t%d (%s)", i, v, v.String(), want.Bytecode[i], want.Bytecode[i].String())
			}
		} else {
			t.Errorf("i=%d mismatch \t%d (%s) \t\t- (None)", i, v, v.String())
		}
	}

	// if want bytecode is greater than got bytecode
	for ; i < len(want.Bytecode); i++ {
		v := want.Bytecode[i]
		t.Errorf("i=%d mismatch \t- (None) \t\t%d (%s)", i, v, v.String())
	}
}

func TestNewVM(t *testing.T) {
	// constants
	chunk := NewChunk([]Bytecode{
		InstructionConstant, 0,
	}, []Value{
		NumberValue(0),
	})
	stackSize := Pos(256)
	callstackSize := Pos(256)

	vm := NewVM(chunk, stackSize, callstackSize)

	// should start at first instruction
	if vm.ip != 0 {
		t.Errorf("vm.ip = %d, want 0", vm.ip)
	}

	// should have given instructions
	for i, v := range chunk.Bytecode {
		if v != vm.chunk.Bytecode[i] {
			t.Errorf("vm.Bytecode[%d] = %d, want %d", i, vm.chunk.Bytecode[i], v)
		}
	}

	// should have given constants
	for i, v := range chunk.Constants {
		if v != vm.chunk.Constants[i] {
			t.Errorf("vm.Constants[%d] = %d, want %d", i, vm.chunk.Constants[i], v)
		}
	}

	// should have given stack size
	if vm.stack.Size != stackSize {
		t.Errorf("vm.stack.Size = %d, want %d", vm.stack.Size, stackSize)
	}

	// should have given call stack size
	if vm.call.Size != callstackSize {
		t.Errorf("vm.call.Size = %d, want %d", vm.call.Size, callstackSize)
	}
}

func BenchmarkNewVM(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewVM(nil, 256, 256)
	}
}

func GetExecutionTestData() map[string]struct {
	chunk          *Chunk
	resultingStack []Value
} {
	return map[string]struct {
		chunk          *Chunk
		resultingStack []Value
	}{
		"two_plus_one": {
			NewChunk([]Bytecode{
				InstructionConstant, 0,
				InstructionConstant, 1,
				InstructionAdd,
			},
				[]Value{
					NumberValue(1), NumberValue(2),
				}),
			[]Value{
				NumberValue(3),
			},
		},
		"push_constant": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
				},
				[]Value{
					NumberValue(1),
				},
			),
			[]Value{
				NumberValue(1),
			},
		},
		"push_true": {
			NewChunk(
				[]Bytecode{
					InstructionTrue,
				},
				[]Value{},
			),
			[]Value{
				BoolValue(true),
			},
		},
		"push_false": {
			NewChunk(
				[]Bytecode{
					InstructionFalse,
				},
				[]Value{},
			),
			[]Value{
				BoolValue(false),
			},
		},
		"push_nil": {
			NewChunk(
				[]Bytecode{
					InstructionNil,
				},
				[]Value{},
			),
			[]Value{
				NilValue{},
			},
		},
		"empty": {
			NewChunk(
				[]Bytecode{},
				[]Value{},
			),
			[]Value{},
		},
		// (2 + 1) * 5 / (6 - 2)
		"full_arithmetic": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionConstant, 1,
					InstructionAdd,
					InstructionConstant, 2,
					InstructionMul,
					InstructionConstant, 3,
					InstructionConstant, 0,
					InstructionSub,
					InstructionDiv,
				},
				[]Value{
					NumberValue(2), NumberValue(1), NumberValue(5), NumberValue(6),
				},
			),
			[]Value{
				NumberValue((2.0 + 1.0) * 5.0 / (6.0 - 2.0)),
			},
		},
		"equality_true": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionConstant, 0,
					InstructionEquals,
				},
				[]Value{
					NumberValue(1),
				},
			),
			[]Value{
				BoolValue(true),
			},
		},
		"equality_false": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionConstant, 1,
					InstructionEquals,
				},
				[]Value{
					NumberValue(1), NumberValue(2),
				},
			),
			[]Value{
				BoolValue(false),
			},
		},
		"inequality_false": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionConstant, 0,
					InstructionNotEqual,
				},
				[]Value{
					NumberValue(1),
				},
			),
			[]Value{
				BoolValue(false),
			},
		},
		"inequality_true": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionConstant, 1,
					InstructionNotEqual,
				},
				[]Value{
					NumberValue(1), NumberValue(2),
				},
			),
			[]Value{
				BoolValue(true),
			},
		},
		"not_true": {
			NewChunk(
				[]Bytecode{
					InstructionTrue,
					InstructionNot,
				},
				[]Value{},
			),
			[]Value{
				BoolValue(false),
			},
		},
		"not_false": {
			NewChunk(
				[]Bytecode{
					InstructionFalse,
					InstructionNot,
				},
				[]Value{},
			),
			[]Value{
				BoolValue(true),
			},
		},
		"jump": {
			NewChunk(
				[]Bytecode{
					InstructionJump, 0, 2,
					InstructionConstant, 0, // should not execute
					InstructionConstant, 1, // should execute
				},
				[]Value{
					NumberValue(0), NumberValue(1),
				},
			),
			[]Value{
				NumberValue(1),
			},
		},
		"jump_false/false": {
			NewChunk(
				[]Bytecode{
					InstructionFalse,
					InstructionJumpFalse, 0, 2,
					InstructionConstant, 0, // should not execute
					InstructionConstant, 1, // should execute
				},
				[]Value{
					NumberValue(0), NumberValue(1),
				},
			),
			[]Value{
				NumberValue(1),
			},
		},
		"jump_false/true": {
			NewChunk(
				[]Bytecode{
					InstructionTrue,
					InstructionJumpFalse, 0, 2,
					InstructionConstant, 0, // should execute
					InstructionConstant, 1, // should execute
				},
				[]Value{
					NumberValue(0), NumberValue(1),
				},
			),
			[]Value{
				NumberValue(0), NumberValue(1),
			},
		},
		"declare_local": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionDeclareLocal, 1,
				},
				[]Value{
					NumberValue(0), StringValue("a"),
				},
			),
			[]Value{
				&VariableValue{
					"a",
					NumberValue(0),
					0,
				},
			},
		},
		"assign_local": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionDeclareLocal, 1,
					InstructionConstant, 2,
					InstructionSetLocal, 1, // reassign
				},
				[]Value{
					NumberValue(0), StringValue("a"), NumberValue(1),
				},
			),
			[]Value{
				&VariableValue{
					"a",
					NumberValue(1),
					0,
				},
			},
		},
		"get_local": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionDeclareLocal, 1,
					InstructionGetLocal, 1, // reassign
				},
				[]Value{
					NumberValue(0), StringValue("a"),
				},
			),
			[]Value{
				&VariableValue{
					"a",
					NumberValue(0),
					0,
				},
				NumberValue(0),
			},
		},
		"get_reassigned_local": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionDeclareLocal, 1,
					InstructionGetLocal, 1,
					InstructionConstant, 2,
					InstructionSetLocal, 1, // reassign
					InstructionGetLocal, 1,
				},
				[]Value{
					NumberValue(0), StringValue("a"), NumberValue(1),
				},
			),
			[]Value{
				&VariableValue{
					"a",
					NumberValue(1),
					0,
				},
				NumberValue(0),
				NumberValue(1),
			},
		},
		"variable_scope": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionDeclareLocal, 1,
					InstructionDescend,
					InstructionConstant, 2,
					InstructionDeclareLocal, 3,
					InstructionDescend,
					InstructionConstant, 4,
					InstructionDeclareLocal, 5,
					InstructionAscend,
					InstructionAscend,
				},
				[]Value{
					NumberValue(0), StringValue("a"),
					NumberValue(1), StringValue("b"),
					NumberValue(2), StringValue("c"),
				},
			),
			[]Value{
				&VariableValue{
					"a",
					NumberValue(0),
					0,
				},
			},
		},
		"function_call": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 0,
					InstructionConstant, 1,
					InstructionConstant, 2,
					InstructionCall,
				},
				[]Value{
					NumberValue(1),
					NumberValue(2),
					FunctionValue{
						Name:   "sum",
						Params: []string{"a", "b"},
						Chunk: NewChunk(
							[]Bytecode{
								InstructionGetLocal, 0,
								InstructionGetLocal, 1,
								InstructionAdd,
								InstructionReturn,
							},
							[]Value{
								StringValue("a"), StringValue("b"),
							},
						),
					},
				},
			),
			[]Value{
				NumberValue(3),
			},
		},
		"function_calling_function": {
			NewChunk(
				[]Bytecode{
					InstructionConstant, 3,
					InstructionDeclareLocal, 4,
					InstructionConstant, 0,
					InstructionConstant, 1,
					InstructionConstant, 2,
					InstructionCall,
				},
				[]Value{
					NumberValue(1),
					NumberValue(2),
					FunctionValue{
						Name:   "sum",
						Params: []string{"a", "b"},
						Chunk: NewChunk(
							[]Bytecode{
								InstructionGetLocal, 0,
								InstructionGetLocal, 2, InstructionCall, // square the number
								InstructionGetLocal, 1,
								InstructionGetLocal, 2, InstructionCall, // square the number
								InstructionAdd,
								InstructionReturn,
							},
							[]Value{
								StringValue("a"), StringValue("b"), StringValue("square"),
							},
						),
					},
					FunctionValue{
						Name:   "square",
						Params: []string{"n"},
						Chunk: NewChunk(
							[]Bytecode{
								InstructionGetLocal, 0,
								InstructionGetLocal, 0,
								InstructionMul,
								InstructionReturn,
							},
							[]Value{
								StringValue("n"),
							},
						),
					},
					StringValue("square"),
				},
			),
			[]Value{
				&VariableValue{
					"square",
					FunctionValue{
						Name:   "square",
						Params: []string{"n"},
						Chunk: NewChunk(
							[]Bytecode{
								InstructionGetLocal, 0,
								InstructionGetLocal, 0,
								InstructionMul,
								InstructionReturn,
							},
							[]Value{
								StringValue("n"),
							},
						),
					},
					0,
				},
				NumberValue(5),
			},
		},
	}
}

func TestVM_Execution(t *testing.T) {
	data := GetExecutionTestData()

	for name, test := range data {
		t.Run(name, func(t *testing.T) {
			vm := NewVM(test.chunk, 256, 256)

			for vm.HasNext() && vm.Next() {
			}

			CompareStacks(t, test.resultingStack, vm.stack)
		})
	}
}

func BenchmarkVM_Execution(b *testing.B) {
	data := GetExecutionTestData()

	for name, test := range data {
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				vm := NewVM(test.chunk, 256, 256)
				for vm.HasNext() && vm.Next() {
				}
			}
		})
	}
}

func TestVM_NextByte(t *testing.T) {
	vm := NewVM(
		NewChunk(
			[]Bytecode{
				InstructionConstant, 0,
			},
			[]Value{
				NumberValue(0),
			},
		),
		16,
		16,
	)

	b, err := vm.TryNextByte()

	if err != nil {
		t.Fatal(err)
	}

	if b != InstructionConstant {
		t.Errorf("got %v; want %v", b, InstructionConstant)
	}

	b, err = vm.TryNextByte()

	if err != nil {
		t.Fatal(err)
	}

	if b != 0 {
		t.Errorf("got %v; want %v", b, 0)
	}

	b, err = vm.TryNextByte()

	if err == nil {
		t.Errorf("didn't get expected error")
	}

	if b != 0 {
		t.Errorf("got %v; want %v", b, nil)
	}
}

func TestVM_NextU16_Empty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("didn't panic when not enough bytes")
		}
	}()

	vm := NewVM(
		NewChunk(
			[]Bytecode{},
			[]Value{},
		),
		16,
		16,
	)
	vm.NextU16()
}

func TestVM_NextU16_One(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("didn't panic when not enough bytes")
		}
	}()

	vm := NewVM(
		NewChunk(
			[]Bytecode{
				0,
			},
			[]Value{},
		),
		16,
		16,
	)

	vm.NextU16()
}

func TestVM_NextU16(t *testing.T) {
	for i := 0; i <= 0xFFFF; i++ {
		t.Run(fmt.Sprintf("value-%d", i), func(t *testing.T) {
			vm := NewVM(
				NewChunk(
					[]Bytecode{
						Bytecode((i >> 8) & 0xFF),
						Bytecode(i & 0xFF),
					},
					[]Value{},
				),
				16,
				16,
			)

			b := vm.NextU16()

			if uint16(i) != b {
				t.Errorf("got %v; want %v", b, i)
			}
		})
	}
}

func TestVM_Jump(t *testing.T) {
	vm := NewVM(
		NewChunk(
			[]Bytecode{
				InstructionJump, 0, 2,
				InstructionConstant, 0,
				InstructionConstant, 1,
				InstructionConstant, 2,
			},
			[]Value{
				NumberValue(0), NumberValue(1), NumberValue(2),
			},
		),
		16,
		16,
	)

	vm.Next()

	if vm.ip != 5 {
		t.Errorf("jumped got %v; want %v", vm.ip-3, 2)
	}
}

func TestVM_JumpFalse(t *testing.T) {
	vm := NewVM(
		NewChunk(
			[]Bytecode{
				InstructionFalse,
				InstructionJumpFalse, 0, 2,
				InstructionConstant, 0,
				InstructionConstant, 1,
				InstructionConstant, 2,
			},
			[]Value{
				NumberValue(0), NumberValue(1), NumberValue(2),
			},
		),
		16,
		16,
	)

	vm.Next()
	vm.Next()

	if vm.ip != 6 {
		t.Errorf("jumped got %v; want %v", vm.ip-4, 2)
	}
}

func TestVM_DontJumpFalse(t *testing.T) {
	vm := NewVM(
		NewChunk(
			[]Bytecode{
				InstructionTrue,
				InstructionJumpFalse, 0, 2,
				InstructionConstant, 0,
				InstructionConstant, 1,
				InstructionConstant, 2,
			},
			[]Value{
				NumberValue(0), NumberValue(1), NumberValue(2),
			},
		),
		16,
		16,
	)

	vm.Next()
	vm.Next()

	if vm.ip != 4 {
		t.Errorf("ip is %v; want %v", vm.ip, 4)
	}
}

func TestVM_GetGlobal(t *testing.T) {}
