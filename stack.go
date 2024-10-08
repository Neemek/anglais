package main

type Stack[T any] struct {
	Current Pos
	Size    Pos

	items []T
}

func NewStack[T any](size Pos) *Stack[T] {
	return &Stack[T]{
		items:   make([]T, size),
		Size:    size,
		Current: 0,
	}
}

func (s *Stack[T]) Push(items ...T) {
	for _, item := range items {
		if s.Current >= s.Size {
			panic("stack overflow")
		}

		s.items[s.Current] = item
		s.Current++
	}
}

func (s *Stack[T]) Pop() T {
	if s.Current <= 0 {
		panic("stack underflow")
	}

	s.Current--
	return s.items[s.Current]
}

func (s *Stack[T]) Peek() T {
	if s.Current <= 0 {
		panic("stack underflow")
	}

	return s.items[s.Current-1]
}

// check whether the stack is invalid (stack over-/underflow)
func (s *Stack[T]) check() {
	if s.Current >= s.Size {
		panic("stack underflow")
	}

	if s.Current < 0 {
		panic("stack underflow")
	}
}
