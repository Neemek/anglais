package core

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type ValueType int

const (
	NilValueType ValueType = iota
	BoolValueType
	NumberValueType
	StringValueType
	ListValueType
	ObjectValueType
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
	case ObjectValueType:
		return "object"
	case NumberValueType:
		return "number"
	case StringValueType:
		return "string"
	case ListValueType:
		return "list"
	case FunctionValueType:
		return "function"
	case BuiltinFunctionValueType:
		return "builtin function"
	case VariableValueType:
		return "variable"
	}

	return "undefined"
}

// GoToValue convert go values to anglais VM-values. Works for some values (nil, bool, float64, string, slices, maps)
func GoToValue(gov interface{}) Value {
	switch v := gov.(type) {
	case nil:
		return NilValue{}
	case bool:
		return BoolValue(v)
	case float64:
		return NumberValue(v)
	case string:
		return StringValue(v)
	case []interface{}:
		values := make([]Value, len(v))
		for i, value := range v {
			values[i] = GoToValue(value)
		}

		return ListValue{
			values,
		}
	case map[string]interface{}:
		values := map[string]Value{}
		for key, value := range v {
			values[key] = GoToValue(value)
		}

		return ObjectValue{
			values,
		}
	}

	panic(fmt.Sprintf("unsupported automatic type conversion: %v (%s)", gov, reflect.TypeOf(gov).Name()))
}

type Value interface {
	// Type get the type of the value (a ValueType)
	Type() ValueType

	// String Convert this value to a string fit for human consumption
	String() string

	// DebugString get a debug string of this value. Used in lists.
	DebugString() string

	// Equals Check if two values are equal
	Equals(Value) bool

	// Get a member from the value. An error is returned if the member does not exist
	Get(string) (Value, error)
}

type NilValue struct{}

func (v NilValue) Type() ValueType {
	return NilValueType
}

func (v NilValue) String() string {
	return "nil"
}

func (v NilValue) DebugString() string {
	return v.String()
}

func (v NilValue) Equals(other Value) bool {
	return other.Type() == NilValueType
}

func (v NilValue) Get(key string) (Value, error) {
	return nil, errors.New("nil has no properties")
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

func (v BoolValue) DebugString() string {
	return v.String()
}

func (v BoolValue) Equals(other Value) bool {
	return other.Type() == BoolValueType && bool(other.(BoolValue)) == bool(v)
}

func (v BoolValue) Get(key string) (Value, error) {
	return nil, errors.New("booleans have no properties")
}

// ObjectValue An object with any number of members (key-value pairs)
type ObjectValue struct {
	members map[string]Value
}

func (v ObjectValue) Type() ValueType {
	return ObjectValueType
}

func (v ObjectValue) String() string {
	out := "{"
	for key, value := range v.members {
		if out != "{" {
			out += ", "
		}

		out += fmt.Sprintf("%q=%s", key, value.String())
	}
	out += "}"

	return out
}

func (v ObjectValue) DebugString() string {
	return v.String()
}

func (v ObjectValue) Equals(other Value) bool {
	// TODO implement object equality check
	return false
}

func (v ObjectValue) Get(key string) (Value, error) {
	if member := v.members[key]; member == nil {
		return nil, errors.New("no property found with name \"" + key + "\"")
	} else {
		return member, nil
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

func (v NumberValue) DebugString() string {
	return v.String()
}

func (v NumberValue) Equals(other Value) bool {
	return other.Type() == NumberValueType && float64(other.(NumberValue)) == float64(v)
}

func (v NumberValue) Get(key string) (Value, error) {
	// TODO maybe add standard functions for number values?
	return nil, errors.New("numbers have no properties")
}

type StringValue string

func (v StringValue) Type() ValueType {
	return StringValueType
}

func (v StringValue) String() string {
	return string(v)
}

func (v StringValue) DebugString() string {
	return "\"" + v.String() + "\""
}

func (v StringValue) Equals(other Value) bool {
	return other.Type() == StringValueType && string(other.(StringValue)) == string(v)
}

var StringPrototype = map[string]BuiltinFunctionValue{
	"split": {
		"split",
		[]string{"seperator"},
		func(vm *VM, this Value, m map[string]Value) (Value, error) {
			str := this.(StringValue).String()
			sep := m["seperator"].(StringValue).String()

			var out []string
			tmp := strings.Builder{}
			for i := 0; i < len(str)-len(sep); i++ {
				tmp.WriteRune([]rune(str)[i])

				if str[i:i+len(sep)] == sep {
					out = append(out, tmp.String())
					tmp.Reset()
				}
			}

			return GoToValue(out), nil
		},
		nil,
	},
}

func (v StringValue) Get(key string) (Value, error) {
	if prop, ok := StringPrototype[key]; ok {
		return prop, nil
	}

	return nil, errors.New(fmt.Sprintf("string has no property \"%s\"", key))
}

// ListValue a dynamic list of values
type ListValue struct {
	items []Value
}

func (v ListValue) Type() ValueType {
	return ListValueType
}

func (v ListValue) String() string {
	out := "["
	for i, item := range v.items {
		if i != 0 {
			out += ", "
		}
		out += item.DebugString()
	}
	out += "]"

	return out
}

func (v ListValue) DebugString() string {
	return v.String()
}

func (v ListValue) Equals(other Value) bool {
	return other.Type() == ListValueType &&
		slices.Equal(v.items, other.(ListValue).items)
}

var ListPrototype = map[string]BuiltinFunctionValue{
	"append": {
		"append",
		[]string{"item"},
		func(_ *VM, this Value, p map[string]Value) (Value, error) {
			return ListValue{
				append(this.(ListValue).items, p["item"]),
			}, nil
		},
		nil,
	},
	"at": {
		"at",
		[]string{"index"},
		func(_ *VM, this Value, p map[string]Value) (Value, error) {
			index := int(p["index"].(NumberValue))
			items := this.(ListValue).items

			if index >= len(items) {
				return nil, errors.New(fmt.Sprintf("list index %x out of range", index))
			}

			return items[index], nil
		},
		nil,
	},
	"length": {
		"length",
		[]string{},
		func(_ *VM, this Value, p map[string]Value) (Value, error) {
			return NumberValue(len(this.(ListValue).items)), nil
		},
		nil,
	},
	"map": {
		"map",
		[]string{"f"},
		func(vm *VM, value Value, m map[string]Value) (Value, error) {
			list := value.(ListValue)

			v := m["f"]
			var f Value
			f, ok := v.(FunctionValue)
			if !ok {
				f, ok = v.(BuiltinFunctionValue)

				if !ok {
					return nil, errors.New(fmt.Sprintf("not a function to apply: %s", v))
				}
			}

			for i, item := range list.items {
				var err error
				list.items[i], err = vm.Call(f, []Value{
					item,
				})

				if err != nil {
					return nil, err
				}
			}

			return list, nil
		},
		nil,
	},
	"reduce": {
		"reduce",
		[]string{"f", "start"},
		func(vm *VM, value Value, m map[string]Value) (Value, error) {
			list := value.(ListValue)
			f := m["f"]
			sum := m["start"]

			for _, v := range list.items {
				result, err := vm.Call(f, []Value{sum, v})
				if err != nil {
					return nil, err
				}
				sum = result
			}

			return sum, nil
		},
		nil,
	},
}

func (v ListValue) Get(key string) (Value, error) {
	if prop, ok := ListPrototype[key]; ok {
		return prop, nil
	}

	return nil, errors.New(fmt.Sprintf("list has no property \"%s\"", key))
}

type FunctionValue struct {
	Name   string
	Params []string
	Chunk  *Chunk
	Parent Value
}

func (v FunctionValue) Type() ValueType {
	return FunctionValueType
}

func (v FunctionValue) String() string {
	return fmt.Sprintf("<function name=%s>", v.Name)
}

func (v FunctionValue) DebugString() string {
	return v.String()
}

func (v FunctionValue) Equals(other Value) bool {
	return other.Type() == FunctionValueType &&
		v.Name == other.(FunctionValue).Name &&
		v.Chunk == other.(FunctionValue).Chunk
}

func (v FunctionValue) Get(_ string) (Value, error) {
	return nil, errors.New("functions have no properties")
}

type BuiltinFunctionValue struct {
	Name       string
	Parameters []string
	F          func(*VM, Value, map[string]Value) (Value, error)
	Parent     Value
}

func (v BuiltinFunctionValue) Type() ValueType {
	return BuiltinFunctionValueType
}

func (v BuiltinFunctionValue) String() string {
	return fmt.Sprintf("<function name=%s builtin>", v.Name)
}

func (v BuiltinFunctionValue) DebugString() string {
	return v.String()
}

func (v BuiltinFunctionValue) Equals(other Value) bool {
	return other.Type() == BuiltinFunctionValueType &&
		v.Name == other.(BuiltinFunctionValue).Name
}

func (v BuiltinFunctionValue) Get(_ string) (Value, error) {
	return nil, errors.New("functions have no properties")
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

func (v VariableValue) DebugString() string {
	return v.String()
}

func (v VariableValue) Equals(other Value) bool {
	return other.Type() == VariableValueType &&
		v.name == other.(VariableValue).name &&
		v.value.Equals(other.(VariableValue).value)
}

func (v VariableValue) Get(_ string) (Value, error) {
	return nil, errors.New("variables have no properties")
}
