package core

import (
	"fmt"
	"testing"
)

func CompareStacks[T Value](t *testing.T, expected []T, actual *Stack[T]) {
	if actual.Current != Pos(len(expected)) {
		t.Errorf("Unexpected stack size. Expected %d, got %d", len(expected), actual.Current)
	}

	for i, v := range expected {
		t.Logf("Comparing value at index %d", i)
		CompareValues(t, actual.items[i], v)
	}

	for i := len(expected); i < int(actual.Current); i++ {
		t.Errorf("Unexpected item %d: %s", i, actual.items[i])
	}
}

func TestNewStack(t *testing.T) {
	size := 256

	s := NewStack[any](Pos(size))

	if s.Capacity != Pos(size) {
		t.Errorf("Stack size (%d) does not match expected size (%d)", s.Capacity, size)
	} else {
		t.Logf("Stack size is expected size (%d)", s.Capacity)
	}

	if s.Current != 0 {
		t.Errorf("Current pos (%d) not initialized to 0", s.Current)
	} else {
		t.Log("Current pos initialized to 0 as expected")
	}
}

func BenchmarkNewStack(b *testing.B) {
	for n := 0; n <= 512; n += 256 {
		b.Run(fmt.Sprintf("size=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = NewStack[any](Pos(n))
			}
		})
	}
}

func TestStackUnderflow(t *testing.T) {
	s := NewStack[any](64)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("popping an empty array did not panic (stack underflow)")
		}
	}()

	s.Pop()
}

func TestStackUnderflowByPeek(t *testing.T) {
	s := NewStack[any](64)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("peeking beyond an empty stack did not panic (stack underflow)")
		}
	}()

	s.Peek()
}

func GetExampleStackTestCases() []any {
	return []any{
		"Hello world!",
		16,
		true,
		false,
		2008,
		"",
		"Lorem ipsum dolor sit amet",
	}
}

func TestStack(t *testing.T) {
	for _, c := range GetExampleStackTestCases() {
		s := NewStack[any](1)
		s.Push(c)
		out := s.Pop()

		if out != c {
			t.Errorf("inputted item does not match outputted item")
		}
	}
}

func TestStackOverflow(t *testing.T) {
	s := NewStack[any](1)
	s.Push(1)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("pushing beyond a stack did not panic (stack overflow)")
		}
	}()

	s.Push(2)
}

func BenchmarkStack(b *testing.B) {
	for n := 256; n <= 512; n += 256 {
		b.Run(fmt.Sprintf("size_%d", n), func(b *testing.B) {
			s := NewStack[any](Pos(n))
			for i := 0; i < b.N; i++ {

				s.Push(2)
				s.Pop()
			}
		})
	}
}
