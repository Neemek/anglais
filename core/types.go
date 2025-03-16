package core

import "fmt"

type Type int

const (
	TypeString Type = iota
	TypeNumber
	TypeBoolean
	TypeNil
	TypeList
	TypeObject
	TypeFunction
	TypeAny
)

func (t Type) String() string {
	switch t {
	case TypeString:
		return "String"
	case TypeNumber:
		return "Number"
	case TypeBoolean:
		return "Boolean"
	case TypeNil:
		return "Nil"
	case TypeList:
		return "List"
	case TypeObject:
		return "Object"
	case TypeFunction:
		return "Function"
	}

	panic(fmt.Sprintf("unsupported string conversion for type %v", int(t)))
}

func TypeOf(v Value) Type {
	switch v.(type) {
	case *StringValue:
		return TypeString
	case *NumberValue:
		return TypeNumber
	case *BoolValue:
		return TypeBoolean
	case *ListValue:
		return TypeList
	case *ObjectValue:
		return TypeObject
	case *FunctionValue:
		return TypeFunction
	case *BuiltinFunctionValue:
		return TypeFunction
	}

	panic(fmt.Sprintf("unsupported value (of type %T)", v))
}

func SignatureOf(v Value) TypeSignature {
	switch t := v.(type) {
	case *StringValue:
		return &StringSignature{}
	case *NumberValue:
		return &NumberSignature{}
	case *BoolValue:
		return &BooleanSignature{}
	case *ListValue:
		return &ListSignature{}
	case *ObjectValue:
		return &ObjectSignature{}
	case *FunctionValue:
		return &FunctionSignature{}
	case *BuiltinFunctionValue:
		return t.Signature
	}

	panic(fmt.Sprintf("unknown value; cannot get signature of %s", v))
}

type TypeSignature interface {
	Type() Type

	// Matches check if this type signature matches another.
	Matches(TypeSignature) bool
}

type NilSignature struct{}

func (*NilSignature) Type() Type {
	return TypeNil
}

func (*NilSignature) Matches(other TypeSignature) bool {
	return other.Type() == TypeAny || other.Type() == TypeNil
}

type StringSignature struct{}

func (*StringSignature) Type() Type {
	return TypeString
}

func (*StringSignature) Matches(other TypeSignature) bool {
	return other.Type() == TypeAny || other.Type() == TypeString
}

type NumberSignature struct{}

func (*NumberSignature) Type() Type {
	return TypeNumber
}

func (*NumberSignature) Matches(other TypeSignature) bool {
	return other.Type() == TypeAny || other.Type() == TypeNumber
}

type BooleanSignature struct{}

func (*BooleanSignature) Type() Type {
	return TypeBoolean
}

func (*BooleanSignature) Matches(other TypeSignature) bool {
	return other.Type() == TypeAny || other.Type() == TypeBoolean
}

type ListSignature struct {
	contents TypeSignature
}

func (*ListSignature) Type() Type {
	return TypeList
}

func (s *ListSignature) Matches(other TypeSignature) bool {
	return other.Type() == TypeAny || other.Type() == TypeList && other.(*ListSignature).contents.Matches(s.contents)
}

type ObjectSignature struct {
	members map[string]TypeSignature
}

func (*ObjectSignature) Type() Type {
	return TypeObject
}

func (s *ObjectSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeAny {
		return true
	}

	if other.Type() != TypeObject {
		return false
	}

	o := other.(*ObjectSignature)

	if len(o.members) != len(s.members) {
		return false
	}

	for name, member := range s.members {
		v, ok := o.members[name]

		if !ok {
			return false
		}

		if !v.Matches(member) {
			return false
		}
	}

	return true
}

type FunctionSignature struct {
	in  []TypeSignature
	out TypeSignature
}

func (*FunctionSignature) Type() Type {
	return TypeFunction
}

func (s *FunctionSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeAny {
		return true
	}

	if other.Type() != TypeFunction {
		return false
	}

	f := other.(*FunctionSignature)

	if !s.out.Matches(f.out) {
		return false
	}

	if len(f.in) != len(s.in) {
		return false
	}

	for i, p := range s.in {
		v := f.in[i]
		if !p.Matches(v) {
			return false
		}
	}

	return true
}

type AnySignature struct{}

func (AnySignature) Type() Type {
	return TypeAny
}

func (AnySignature) Matches(_ TypeSignature) bool {
	return true
}
