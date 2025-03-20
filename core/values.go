package core

import (
	"errors"
	"fmt"
	"reflect"
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

// GoToValue convert go values to anglais VM-values. Works for some values (nil, bool, float64, int, string, slices, maps)
func GoToValue(gov interface{}) Value {
	switch v := gov.(type) {
	case nil:
		return &NilValue{}
	case bool:
		return &BoolValue{
			v,
		}
	case int:
		return &NumberValue{
			float64(v),
		}
	case float64:
		return &NumberValue{
			v,
		}
	case string:
		return &StringValue{
			v,
		}
	case []interface{}:
		values := make([]Value, len(v))
		for i, value := range v {
			values[i] = GoToValue(value)
		}

		return &ListValue{
			values,
		}
	case map[string]interface{}:
		values := map[string]Value{}
		for key, value := range v {
			values[key] = GoToValue(value)
		}

		return &ObjectValue{
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

	// Clone create a clone of the value. The returned value is a pointer to a new value of the same type as the value.
	Clone() Value
}

type NilValue struct{}

func (v *NilValue) Type() ValueType {
	return NilValueType
}

func (v *NilValue) String() string {
	return "nil"
}

func (v *NilValue) DebugString() string {
	return v.String()
}

func (v *NilValue) Equals(other Value) bool {
	return other.Type() == NilValueType
}

func (v *NilValue) Get(_ string) (Value, error) {
	return nil, errors.New("nil has no properties")
}

func (v *NilValue) Clone() Value {
	return &NilValue{}
}

type BoolValue struct {
	Boolean bool
}

func (v *BoolValue) Type() ValueType {
	return BoolValueType
}

func (v *BoolValue) String() string {
	if v.Boolean {
		return "true"
	} else {
		return "false"
	}
}

func (v *BoolValue) DebugString() string {
	return v.String()
}

func (v *BoolValue) Equals(other Value) bool {
	return other.Type() == BoolValueType && other.(*BoolValue).Boolean == v.Boolean
}

func (v *BoolValue) Get(_ string) (Value, error) {
	return nil, errors.New("booleans have no properties")
}

func (v *BoolValue) Clone() Value {
	return &BoolValue{
		v.Boolean,
	}
}

// ObjectValue An object with any number of members (key-value pairs)
type ObjectValue struct {
	Members map[string]Value
}

func (v *ObjectValue) Type() ValueType {
	return ObjectValueType
}

func (v *ObjectValue) String() string {
	out := "{"
	for key, value := range v.Members {
		if out != "{" {
			out += ", "
		}

		out += fmt.Sprintf("%q=%s", key, value.String())
	}
	out += "}"

	return out
}

func (v *ObjectValue) DebugString() string {
	return v.String()
}

func (v *ObjectValue) Equals(other Value) bool {
	object, ok := other.(*ObjectValue)
	if !ok {
		return false
	}

	for key, value := range v.Members {
		if !object.Members[key].Equals(value) {
			return false
		}
	}

	return true
}

var ObjectPrototype = map[string]Value{
	"set": &BuiltinFunctionValue{
		"set",
		&FunctionSignature{
			[]TypeSignature{&StringSignature{}, &ListSignature{}},
			&NilSignature{},
		},
		func(vm *VM, _this Value, params []Value) (Value, error) {
			this := _this.(*ObjectValue)

			p := params[1]
			v, ok := params[0].(*StringValue)
			if !ok {
				return nil, errors.New("property is not a string")
			}

			this.Members[v.Text] = p

			return &NilValue{}, nil
		},
		nil,
		false,
	},
}

func (v *ObjectValue) Get(key string) (Value, error) {
	if member, ok := v.Members[key]; ok {
		return member, nil
	} else if p, ok := ObjectPrototype[key]; ok {
		return p, nil
	} else {
		return nil, errors.New("no property found with name \"" + key + "\"")
	}
}

func (v *ObjectValue) Clone() Value {
	m := make(map[string]Value, len(v.Members))

	for name, mem := range v.Members {
		m[name] = mem.Clone()
	}

	return &ObjectValue{
		m,
	}
}

// NumberValue Integer or floating-point values
type NumberValue struct {
	Number float64
}

const NumberSize int = 64

func (v *NumberValue) Type() ValueType {
	return NumberValueType
}

func (v *NumberValue) String() string {
	return strconv.FormatFloat(v.Number, 'g', -1, NumberSize)
}

func (v *NumberValue) DebugString() string {
	return v.String()
}

func (v *NumberValue) Equals(other Value) bool {
	return other.Type() == NumberValueType && other.(*NumberValue).Number == v.Number
}

func (v *NumberValue) Get(_ string) (Value, error) {
	// TODO maybe add standard functions for number values?
	return nil, errors.New("numbers have no properties")
}

func (v *NumberValue) Clone() Value {
	return &NumberValue{
		v.Number,
	}
}

type StringValue struct {
	Text string
}

func (v *StringValue) Type() ValueType {
	return StringValueType
}

func (v *StringValue) String() string {
	return v.Text
}

func (v *StringValue) DebugString() string {
	return "\"" + v.String() + "\""
}

func (v *StringValue) Equals(other Value) bool {
	return other.Type() == StringValueType && other.(*StringValue).Text == v.Text
}

var StringPrototype = map[string]*BuiltinFunctionValue{
	"split": {
		"split",
		&FunctionSignature{
			[]TypeSignature{&StringSignature{}},
			&NilSignature{},
		},
		func(vm *VM, this Value, v []Value) (Value, error) {
			str := this.(*StringValue).String()
			sep := v[0].(*StringValue).String()

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
		true,
	},
	"length": {
		Name: "length",
		Signature: &FunctionSignature{
			[]TypeSignature{},
			&NumberSignature{},
		},
		F: func(vm *VM, this Value, _ []Value) (Value, error) {
			return GoToValue(len(this.(*StringValue).Text)), nil
		},
	},
}

func (v *StringValue) Get(key string) (Value, error) {
	if prop, ok := StringPrototype[key]; ok {
		return prop, nil
	}

	return nil, errors.New(fmt.Sprintf("string has no property \"%s\"", key))
}

func (v *StringValue) Clone() Value {
	return &StringValue{
		v.Text,
	}
}

// ListValue a dynamic list of values
type ListValue struct {
	Items []Value
}

func (v *ListValue) Type() ValueType {
	return ListValueType
}

func (v *ListValue) String() string {
	out := "["
	for i, item := range v.Items {
		if i != 0 {
			out += ", "
		}
		out += item.DebugString()
	}
	out += "]"

	return out
}

func (v *ListValue) DebugString() string {
	return v.String()
}

func (v *ListValue) Equals(other Value) bool {
	if other.Type() != ListValueType {
		return false
	}

	l := other.(*ListValue)

	if len(v.Items) != len(l.Items) {
		return false
	}

	for i, item := range v.Items {
		if !item.Equals(l.Items[i]) {
			return false
		}
	}

	return true
}

var ListPrototype = map[string]*BuiltinFunctionValue{
	"append": {
		"append",
		&FunctionSignature{
			[]TypeSignature{&AnySignature{}},
			&NilSignature{},
		},
		func(_ *VM, this Value, v []Value) (Value, error) {
			this.(*ListValue).Items = append(this.(*ListValue).Items, v[0])
			return &NilValue{}, nil
		},
		nil,
		false,
	},
	"at": {
		"at",
		&FunctionSignature{
			[]TypeSignature{
				&NumberSignature{},
			},
			&AnySignature{},
		},
		func(_ *VM, this Value, p []Value) (Value, error) {
			items := this.(*ListValue).Items
			index := int(p[0].(*NumberValue).Number)

			if index >= len(items) {
				return nil, errors.New(fmt.Sprintf("list index %x out of range", index))
			}

			return items[index], nil
		},
		nil,
		false,
	},
	"length": {
		"length",
		&FunctionSignature{
			[]TypeSignature{},
			&NumberSignature{},
		},
		func(_ *VM, this Value, _ []Value) (Value, error) {
			return GoToValue(len(this.(*ListValue).Items)), nil
		},
		nil,
		false,
	},
	"map": {
		"map",
		&FunctionSignature{
			[]TypeSignature{
				&FunctionSignature{
					[]TypeSignature{
						&AnySignature{},
					},
					&AnySignature{},
				},
			},
			&ListSignature{},
		},
		func(vm *VM, value Value, m []Value) (Value, error) {
			list := value.(*ListValue)

			v := m[0]
			var f Value
			switch a := v.(type) {
			case *FunctionValue, *BuiltinFunctionValue:
				f = a
			default:
				return nil, errors.New(fmt.Sprintf("not a function to apply: %s", v))
			}

			for i, item := range list.Items {
				var err error
				list.Items[i], err = vm.Call(f, []Value{
					item,
				})

				if err != nil {
					return nil, err
				}
			}

			return list, nil
		},
		nil,
		false,
	},
	"reduce": {
		"reduce",
		&FunctionSignature{
			[]TypeSignature{
				&FunctionSignature{
					[]TypeSignature{
						&AnySignature{},
						&AnySignature{},
					},
					&AnySignature{},
				},
				&AnySignature{},
			},
			&AnySignature{},
		},
		func(vm *VM, value Value, m []Value) (Value, error) {
			list := value.(*ListValue)
			f := m[0]
			sum := m[1]

			for _, v := range list.Items {
				result, err := vm.Call(f, []Value{sum, v})
				if err != nil {
					return nil, err
				}
				sum = result
			}

			return sum, nil
		},
		nil,
		false,
	},
}

func (v *ListValue) Get(key string) (Value, error) {
	if prop, ok := ListPrototype[key]; ok {
		return prop, nil
	}

	return nil, errors.New(fmt.Sprintf("list has no property \"%s\"", key))
}

func (v *ListValue) Clone() Value {
	n := make([]Value, len(v.Items))

	for i, item := range v.Items {
		n[i] = item.Clone()
	}

	return &ListValue{
		n,
	}
}

type FunctionValue struct {
	Name   string
	Params []FunctionParameter
	Yield  TypeSignature
	Chunk  *Chunk
	Parent Value
}

func (v *FunctionValue) Type() ValueType {
	return FunctionValueType
}

func (v *FunctionValue) String() string {
	return fmt.Sprintf("<function name=%s>", v.Name)
}

func (v *FunctionValue) DebugString() string {
	return v.String()
}

func (v *FunctionValue) Equals(other Value) bool {
	return other.Type() == FunctionValueType &&
		v.Name == other.(*FunctionValue).Name &&
		v.Chunk == other.(*FunctionValue).Chunk
}

func (v *FunctionValue) Get(_ string) (Value, error) {
	return nil, errors.New("functions have no properties")
}

func (v *FunctionValue) Clone() Value {
	return &FunctionValue{
		v.Name,
		v.Params,
		v.Yield,
		v.Chunk,
		v.Parent,
	}
}

type BuiltinFunctionValue struct {
	Name      string
	Signature *FunctionSignature
	F         func(*VM, Value, []Value) (Value, error)
	Parent    Value
	Constant  bool
}

func (v *BuiltinFunctionValue) Type() ValueType {
	return BuiltinFunctionValueType
}

func (v *BuiltinFunctionValue) String() string {
	return fmt.Sprintf("<function name=%s builtin>", v.Name)
}

func (v *BuiltinFunctionValue) DebugString() string {
	return v.String()
}

func (v *BuiltinFunctionValue) Equals(other Value) bool {
	return other.Type() == BuiltinFunctionValueType &&
		v.Name == other.(*BuiltinFunctionValue).Name
}

func (v *BuiltinFunctionValue) Get(_ string) (Value, error) {
	return nil, errors.New("functions have no properties")
}

func (v *BuiltinFunctionValue) Clone() Value {
	return &BuiltinFunctionValue{
		v.Name,
		v.Signature,
		v.F,
		v.Parent,
		v.Constant,
	}
}

// VariableValue a value wrapper for variables kept on the stack
type VariableValue struct {
	name  string
	value Value
	scope Pos
}

func (v *VariableValue) Type() ValueType {
	return VariableValueType
}

func (v *VariableValue) String() string {
	return fmt.Sprintf("<variable name=%s value=%s scope=%d>", v.name, v.value, v.scope)

	// variables should not be accessed on the stack; normal values should be pushed and popped predictably
	//panic("tried getting string value of a unreachable value")
}

func (v *VariableValue) DebugString() string {
	return v.String()
}

func (v *VariableValue) Equals(other Value) bool {
	return other.Type() == VariableValueType &&
		v.name == other.(*VariableValue).name &&
		v.value.Equals(other.(*VariableValue).value)
}

func (v *VariableValue) Get(_ string) (Value, error) {
	return nil, errors.New("variables have no properties")
}

func (v *VariableValue) Clone() Value {
	return &VariableValue{
		v.name,
		v.value.Clone(),
		v.scope,
	}
}
