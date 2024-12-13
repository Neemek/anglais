package core

import (
	"fmt"
	"strconv"
)

type ValueType int

const (
	NilValueType ValueType = iota
	BoolValueType
	NumberValueType
	StringValueType
	FunctionValueType
	BuiltinFunctionValueType
	VariableValueType
)

func (v ValueType) String() string {
	switch v {
	case NilValueType:
		return "nil"
	case BoolValueType:
		return "bool"
	case NumberValueType:
		return "number"
	case StringValueType:
		return "string"
	case FunctionValueType:
		return "function"
	case BuiltinFunctionValueType:
		return "builtin function"
	case VariableValueType:
		return "variable"
	}

	return "undefined"
}

func GetType(v string) ValueType {
	switch v {
	case "nil":
		return NilValueType
	case "bool":
		return BoolValueType
	case "number":
		return NumberValueType
	case "string":
		return StringValueType
	case "function":
		return FunctionValueType
	case "builtin":
		return BuiltinFunctionValueType
	case "variable":
		return VariableValueType
	}

	return 0
}

type Value interface {
	Type() ValueType
	String() string
}

type NilValue struct{}

func (v NilValue) Type() ValueType {
	return NilValueType
}

func (v NilValue) String() string {
	return "nil"
}

type BoolValue bool

func (v BoolValue) Type() ValueType {
	return BoolValueType
}

func (v BoolValue) String() string {
	if v {
		return "true"
	} else {
		return "false"
	}
}

// NumberValue Integer or floating-point values
type NumberValue float64

const NumberSize int = 64

func (v NumberValue) Type() ValueType {
	return NumberValueType
}

func (v NumberValue) String() string {
	return strconv.FormatFloat(float64(v), 'g', -1, NumberSize)
}

type StringValue string

func (v StringValue) Type() ValueType {
	return StringValueType
}

func (v StringValue) String() string {
	return string(v)
}

type FunctionValue struct {
	Name   string
	Params []string
	Chunk  *Chunk
}

func (v FunctionValue) Type() ValueType {
	return FunctionValueType
}

func (v FunctionValue) String() string {
	return fmt.Sprintf("<function name=%s>", v.Name)
}

type BuiltinFunctionValue struct {
	Name       string
	Parameters []string
	F          func(map[string]Value) Value
}

func (v BuiltinFunctionValue) Type() ValueType {
	return BuiltinFunctionValueType
}

func (v BuiltinFunctionValue) String() string {
	return fmt.Sprintf("<function name=%s builtin>", v.Name)
}

// VariableValue a value wrapper for variables kept on the stack
type VariableValue struct {
	name  string
	value Value
	scope Pos
}

func (v VariableValue) Type() ValueType {
	return VariableValueType
}

func (v VariableValue) String() string {
	return fmt.Sprintf("<variable name=%s value=%s scope=%d>", v.name, v.value, v.scope)

	// variables should not be accessed on the stack; normal values should be pushed and popped predictably
	//panic("tried getting string value of a unreachable value")
}

func (v VariableValue) equals(other VariableValue) bool {
	return v.name == other.name && v.value == other.value
}
