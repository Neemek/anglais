package core

import "testing"

func CompareValues(t *testing.T, got Value, want Value) {
	if got == nil || want == nil {
		t.Fatalf("a value is nil: got %v; want %v", got, want)
	}

	if got.Type() != want.Type() {
		t.Fatalf("type mismatch: got %v want %v", got.Type(), want.Type())
	}

	switch got.Type() {
	case NilValueType:
		t.Logf("Both are nil")
		return
	case BoolValueType:
		if got.(*BoolValue).Boolean != want.(*BoolValue).Boolean {
			t.Errorf("bool value mismatch: got %v, want %v", got.(*BoolValue), want.(*BoolValue))
		} else {
			t.Logf("Both are same boolean (%s)", want.(*BoolValue).String())
		}
	case NumberValueType:
		if got.(*NumberValue).Number != want.(*NumberValue).Number {
			t.Errorf("number value mismatch: got %v, want %v", got.(*NumberValue), want.(*NumberValue))
		} else {
			t.Logf("Both are same number (%s)", got.(*NumberValue).String())
		}
	case StringValueType:
		if got.(*StringValue).Text != want.(*StringValue).Text {
			t.Errorf("string value mismatch: got %v, want %v", got.(*StringValue), want.(*StringValue))
		} else {
			t.Logf("Both are same string (%s)", got.(*StringValue).String())
		}
	case FunctionValueType:
		n := got.(*FunctionValue)
		m := want.(*FunctionValue)

		if n.Name != m.Name {
			t.Errorf("function name mismatch: got %v, want %v", n.Name, m.Name)
		}

		if len(n.Params) != len(m.Params) {
			t.Errorf("function params length mismatch: got %v, want %v", len(m.Params), len(n.Params))
		}

		for i, v := range n.Params {
			if v != m.Params[i] {
				t.Errorf("function params mismatch: got %v, want %v", v, m.Params[i])
			}
		}

		CompareChunks(t, n.Chunk, m.Chunk)
	case BuiltinFunctionValueType:
		n := got.(*BuiltinFunctionValue)
		m := want.(*BuiltinFunctionValue)

		if n.Name != m.Name {
			t.Errorf("builtin function name mismatch: got %v, want %v", n.Name, m.Name)
		}

		if !n.Signature.Matches(m.Signature) {
			t.Errorf("builtin function parameter count mismatch: got %v, want %v", n, m)
		}

	case VariableValueType:
		n := got.(*VariableValue)
		m := want.(*VariableValue)

		if n.name != m.name {
			t.Errorf("variable name mismatch: got %v, want %v", n.name, m.name)
		}

		if n.scope != m.scope {
			t.Errorf("variable scope mismatch: got %v, want %v", n.scope, m.scope)
		}

		CompareValues(t, n.value, m.value)

	case ListValueType:
		n := got.(*ListValue)
		m := want.(*ListValue)

		if len(n.Items) != len(m.Items) {
			t.Fatalf("list items length mismatch: got %d, want %d", len(n.Items), len(m.Items))
		}

		for i, v := range n.Items {
			t.Logf("comparing list items #%d: got %s, want %s", i, v, m.Items[i])
			CompareValues(t, v, m.Items[i])
		}

	case ObjectValueType:
		n := got.(*ObjectValue)
		m := want.(*ObjectValue)

		if len(n.Members) != len(m.Members) {
			t.Fatalf("object members count mismatch: got %d, want %d", len(n.Members), len(m.Members))
		}

		for k, v := range n.Members {
			t.Logf("comparing object member %s: got %s, want %s", k, v, m.Members[k])
			CompareValues(t, v, m.Members[k])
		}

	default:
		panic("unimplemented comparison")
	}
}
