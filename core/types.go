package core

import (
	"fmt"
	"strings"
)

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
	TypeComposite
	TypeInner
)

func (t Type) String() string {
	switch t {
	case TypeString:
		return "string"
	case TypeNumber:
		return "number"
	case TypeBoolean:
		return "boolean"
	case TypeNil:
		return "nil"
	case TypeList:
		return "list"
	case TypeObject:
		return "object"
	case TypeFunction:
		return "func"
	case TypeAny:
		return "any"
	case TypeComposite:
		return "composite"
	case TypeInner:
		return "inner"
	}

	panic(fmt.Sprintf("unsupported string conversion for type %v", int(t)))
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
		// try to deduce contents type
		var contains TypeSignature

		for _, p := range t.Items {
			sig := SignatureOf(p)
			if contains == nil {
				contains = sig
			} else if !contains.Matches(sig) {
				contains = &AnySignature{}
				break
			}
		}

		return &ListSignature{
			contains,
		}
	case *ObjectValue:
		return &ObjectSignature{}
	case *FunctionValue:
		params := make([]TypeSignature, len(t.Params))

		for i, p := range t.Params {
			params[i] = p.Signature
		}

		return &FunctionSignature{
			params,
			t.Yield,
		}
	case *BuiltinFunctionValue:
		return t.Signature
	}

	panic(fmt.Sprintf("unknown value; cannot get signature of %s", v))
}

type TypeSignature interface {
	Type() Type

	// Matches check if this type signature matches another.
	Matches(TypeSignature) bool

	// String create a human-readable string version of the value type.
	String() string
}

type NilSignature struct{}

func (*NilSignature) Type() Type {
	return TypeNil
}

func (s *NilSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeComposite {
		return other.Matches(s)
	}

	return other.Type() == TypeAny || other.Type() == TypeNil
}

func (*NilSignature) String() string {
	return "nil"
}

type StringSignature struct{}

func (*StringSignature) Type() Type {
	return TypeString
}

func (s *StringSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeComposite {
		return other.Matches(s)
	}

	return other.Type() == TypeAny || other.Type() == TypeString
}

func (*StringSignature) String() string {
	return "string"
}

type NumberSignature struct{}

func (*NumberSignature) Type() Type {
	return TypeNumber
}

func (s *NumberSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeComposite {
		return other.Matches(s)
	}

	return other.Type() == TypeAny || other.Type() == TypeNumber
}

func (*NumberSignature) String() string {
	return "number"
}

type BooleanSignature struct{}

func (*BooleanSignature) Type() Type {
	return TypeBoolean
}

func (s *BooleanSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeComposite {
		return other.Matches(s)
	}

	return other.Type() == TypeAny || other.Type() == TypeBoolean
}

func (*BooleanSignature) String() string {
	return "boolean"
}

type ListSignature struct {
	Contents TypeSignature
}

func (*ListSignature) Type() Type {
	return TypeList
}

func (s *ListSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeComposite {
		return other.Matches(s)
	}

	return other.Type() == TypeAny || (other.Type() == TypeList && other.(*ListSignature).Contents.Matches(s.Contents))
}

func (s *ListSignature) String() string {
	return fmt.Sprintf("list[%s]", s.Contents)
}

type ObjectSignature struct {
	Members map[string]TypeSignature
}

func (*ObjectSignature) Type() Type {
	return TypeObject
}

func (s *ObjectSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeComposite {
		return other.Matches(s)
	}

	if other.Type() == TypeAny {
		return true
	}

	if other.Type() != TypeObject {
		return false
	}

	o := other.(*ObjectSignature)

	if len(o.Members) != len(s.Members) {
		return false
	}

	for name, member := range s.Members {
		v, ok := o.Members[name]

		if !ok {
			return false
		}

		if !v.Matches(member) {
			return false
		}
	}

	return true
}

func (s *ObjectSignature) String() string {
	panic("unimplemented")
}

type FunctionSignature struct {
	In  []TypeSignature
	Out TypeSignature
}

func (*FunctionSignature) Type() Type {
	return TypeFunction
}

func (s *FunctionSignature) Matches(other TypeSignature) bool {
	if other.Type() == TypeComposite {
		return other.Matches(s)
	}

	if other.Type() == TypeAny {
		return true
	}

	if other.Type() != TypeFunction {
		return false
	}

	f := other.(*FunctionSignature)

	if !s.Out.Matches(f.Out) {
		return false
	}

	if len(f.In) != len(s.In) {
		return false
	}

	for i, p := range s.In {
		v := f.In[i]
		if !p.Matches(v) {
			return false
		}
	}

	return true
}

func (s *FunctionSignature) String() string {
	b := strings.Builder{}

	b.WriteString("func(")

	for i, t := range s.In {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(t.String())
	}

	b.WriteString(")")
	if s.Out.Type() != TypeNil {
		b.WriteString(" ")
		b.WriteString(s.Out.String())
	}

	return b.String()
}

type AnySignature struct{}

func (*AnySignature) Type() Type {
	return TypeAny
}

func (*AnySignature) Matches(_ TypeSignature) bool {
	return true
}

func (*AnySignature) String() string {
	return "any"
}

type CompositeSignature struct {
	A TypeSignature
	B TypeSignature
}

func (*CompositeSignature) Type() Type {
	return TypeComposite
}

func (s *CompositeSignature) Matches(other TypeSignature) bool {
	return s.A.Matches(other) || s.B.Matches(other)
}

func (s *CompositeSignature) String() string {
	return fmt.Sprintf("%s|%s", s.A, s.B)
}

type InnerSignature struct{}

func (*InnerSignature) Type() Type {
	return TypeInner
}

func (*InnerSignature) Matches(_ TypeSignature) bool {
	return false
}

func (*InnerSignature) String() string {
	return "inner"
}
