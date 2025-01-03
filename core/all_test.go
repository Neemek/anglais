package core

import (
	"testing"
)

type AllTestCase struct {
	src           string
	expectedStack []Value
}

func GetAllTestCases() map[string]AllTestCase {
	return map[string]AllTestCase{
		"constant_number": {
			"a := 1",
			[]Value{
				&VariableValue{
					"a",
					NumberValue(1),
					0,
				},
			},
		},
	}
}

func TestAll(t *testing.T) {
	cases := GetAllTestCases()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Logf("Initializing lexer")
			l := NewLexer(tc.src)

			t.Logf("Lexing tokens")
			tokens, err := l.Tokenize()

			if err != nil {
				t.Fatalf("Unexpeced error tokenizing: %v", err)
			}

			t.Log("Initializing parser")
			p := NewParser(tokens)

			t.Log("Parsing tokens")
			tree, err := p.Parse()

			if err != nil {
				print(err.(*ParsingError).Format([]rune(tc.src)))
				t.Fatalf("parser had an error")
			}

			t.Log("Initializing compiler")
			c := NewCompiler()

			t.Log("Compiling parse tree")
			c.Compile(tree)

			printChunk(t, name, c.Chunk)

			t.Log("Initializing vm")
			vm := NewVM(c.Chunk, 256, 256)

			t.Log("Running bytecode")
			for vm.HasNext() && vm.Next() {
			}

			t.Log("Comparing stacks")
			CompareStacks(t, tc.expectedStack, vm.stack)
		})
	}
}

func BenchmarkAll(b *testing.B) {
	cases := GetAllTestCases()

	for name, tc := range cases {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				l := NewLexer(tc.src)
				tokens, _ := l.Tokenize()

				p := NewParser(tokens)
				tree, _ := p.Parse()

				c := NewCompiler()
				c.Compile(tree)

				vm := NewVM(c.Chunk, 256, 256)

				for vm.HasNext() && vm.Next() {
				}
			}
		})
	}
}
