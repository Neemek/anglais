package core

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"strings"
)

type Pos int
type Bytecode byte

const (
	// InstructionReturn return to previous call pointer
	InstructionReturn Bytecode = iota
	// InstructionPop pop and delete the first item on the stack
	InstructionPop

	// InstructionAdd pop two and add them
	InstructionAdd
	// InstructionSub pop two and subtract the second from the first
	InstructionSub
	// InstructionMul pop two and multiply them
	InstructionMul
	// InstructionDiv pop two and divide the second by the first
	InstructionDiv
	// InstructionEquals whether the two top values on the stack are equal
	InstructionEquals
	// InstructionNotEqual whether the two top values on the stack are not equal
	InstructionNotEqual
	// InstructionNot inverts boolean (true => false, false => true)
	InstructionNot
	// InstructionLess pops two from stack, pushes whether the lowest is less than the highest
	InstructionLess
	// InstructionLessOrEqual pops two from stack, pushes whether the lowest is less or equal than the highest
	InstructionLessOrEqual
	// InstructionGreater pops two from stack, pushes whether the lowest is greater than the highest
	InstructionGreater
	// InstructionGreaterOrEqual pops two from stack, pushes whether the lowest is greater or equal than the highest
	InstructionGreaterOrEqual

	// InstructionAccessProperty gets a property from a value, and pops it onto the stack
	InstructionAccessProperty
	// InstructionCall pops a function object from the stack and begins execution of the chunk
	InstructionCall

	// InstructionDescend increase the scope depth
	InstructionDescend
	// InstructionAscend decrease the scope depth, and remove all variables on the stack which belong in a higher scope
	InstructionAscend

	// InstructionJump jump forwards by the value of the next two bytes as a u16
	InstructionJump
	// InstructionJumpFalse jump by the value of the two next bytes as an unsigned integer if the first value (popped) from the stack is false
	InstructionJumpFalse
	// InstructionLoop jump by the value of the two next bytes as an unsigned integer backwards if the first value (popped) from the stack is true
	InstructionLoop

	// InstructionGetLocal Push a constant to the stack (2 bytes, second = constant index)
	InstructionGetLocal
	// InstructionSetLocal Set a local variable
	InstructionSetLocal
	// InstructionDeclareLocal Declare a new local variable in the uppermost scope
	InstructionDeclareLocal
	// InstructionGetGlobal Set a global variable (the next byte is the index of the constant with the name of the variable
	InstructionGetGlobal
	// InstructionSetGlobal Push a constant to the stack (2 bytes, second = constant index)
	InstructionSetGlobal

	// InstructionStringConversion Take the top value on the stack and convert it to a string
	InstructionStringConversion
	// InstructionStringConcatenation Add two strings together, with the second value on the stack as left and the top as right
	InstructionStringConcatenation

	// InstructionSwap swap the two top items on the stack (1, 2 -> 2, 1)
	InstructionSwap

	// InstructionAnd pop two booleans and push true if both are true
	InstructionAnd
	// InstructionOr pop two booleans and push true if either are true
	InstructionOr

	// InstructionConstant Push a constant to the stack (2 bytes, second = constant index)
	InstructionConstant
	// InstructionTrue Push a true literal to the stack
	InstructionTrue
	// InstructionFalse Push a false literal to the stack
	InstructionFalse
	// InstructionNil Push a nil literal to the stack
	InstructionNil

	// InstructionNewList Push a new (empty) list to the stack
	InstructionNewList
	// InstructionAppend Append to a list. stack: (... > list > item) => (... > list)
	InstructionAppend
	// InstructionFormList Form items on the stack into a list. The 2 bytes after the instructions are the amount of
	// items to include minus one. (value of 0 => 1 item, value of 1 => 2 items, etc.) The order is reversed compared
	// to on the stack; the top value on the stack is the last in the list.
	InstructionFormList

	// InstructionBreakpoint for debugging purposes
	InstructionBreakpoint
)

func (b Bytecode) String() string {
	switch b {
	case InstructionReturn:
		return "RETURN"
	case InstructionPop:
		return "POP"
	case InstructionAdd:
		return "ADD"
	case InstructionSub:
		return "SUB"
	case InstructionMul:
		return "MUL"
	case InstructionDiv:
		return "DIV"
	case InstructionEquals:
		return "EQUALS"
	case InstructionNotEqual:
		return "NOT_EQUALS"
	case InstructionNot:
		return "NOT"
	case InstructionLess:
		return "LESS"
	case InstructionLessOrEqual:
		return "LESS_OR_EQUAL"
	case InstructionGreater:
		return "GREATER_OR_EQUAL"
	case InstructionGreaterOrEqual:
		return "GREATER_OR_EQUAL"
	case InstructionJump:
		return "JUMP"
	case InstructionJumpFalse:
		return "JUMP_FALSE"
	case InstructionLoop:
		return "LOOP"
	case InstructionConstant:
		return "CONSTANT"
	case InstructionTrue:
		return "TRUE"
	case InstructionFalse:
		return "FALSE"
	case InstructionNil:
		return "NIL"
	case InstructionGetLocal:
		return "GET_LOCAL"
	case InstructionDeclareLocal:
		return "DECLARE_LOCAL"
	case InstructionSetLocal:
		return "SET_LOCAL"
	case InstructionGetGlobal:
		return "GET_GLOBAL"
	case InstructionSetGlobal:
		return "SET_GLOBAL"
	case InstructionCall:
		return "CALL"
	case InstructionDescend:
		return "DESCEND"
	case InstructionAscend:
		return "ASCEND"
	case InstructionStringConversion:
		return "STRING_CONVERSION"
	case InstructionStringConcatenation:
		return "STRING_CONCATENATION"
	case InstructionSwap:
		return "SWAP"
	case InstructionAnd:
		return "AND"
	case InstructionOr:
		return "OR"
	case InstructionFormList:
		return "FORM_LIST"
	case InstructionBreakpoint:
		return "BREAKPOINT"
	case InstructionNewList:
		return "NEW_LIST"
	case InstructionAppend:
		return "APPEND"
	case InstructionAccessProperty:
		return "ACCESS_PROPERTY"
	}
	return "UNDEFINED"
}

type Chunk struct {
	Bytecode  []Bytecode
	Constants []Value
}

func (c Chunk) String() string {
	b := strings.Builder{}

	b.WriteString("=v= chunk =v=\n")
	for i, bc := range c.Bytecode {
		b.WriteString(fmt.Sprintf("i=%d \t%d \t(%s)\n", i, bc, bc))
	}

	b.WriteString("=-= constants =-=\n")

	for i, ct := range c.Constants {
		b.WriteString(fmt.Sprintf("c=%d \t%s\n", i, ct))

		f, ok := ct.(*FunctionValue)
		if ok {
			b.WriteString(f.Chunk.String())
		}
	}

	b.WriteString("=^= chunk =^=\n")

	return b.String()
}

func NewChunk(bytecode []Bytecode, constants []Value) *Chunk {
	return &Chunk{bytecode, constants}
}

func RegisterGOBTypes() {
	gob.Register(&StringValue{""})
	gob.Register(&BoolValue{false})
	gob.Register(&NumberValue{0})
	gob.Register(&FunctionValue{
		Name:   "",
		Params: nil,
		Chunk:  nil,
	})
}

func (c Chunk) Serialize() []byte {
	b := bytes.Buffer{}

	e := gob.NewEncoder(&b)

	err := e.Encode(c)
	if err != nil {
		log.Fatal(err)
	}

	return b.Bytes()
}

func DeserializeChunk(b []byte) *Chunk {
	m := Chunk{}

	buf := bytes.Buffer{}
	buf.Write(b)

	d := gob.NewDecoder(&buf)

	err := d.Decode(&m)

	if err != nil {
		log.Fatal(err)
	}

	return &m
}

type VM struct {
	// Replace with chunk of bytecode
	chunk *Chunk

	// instruction pointer
	ip    Pos
	scope Pos

	// global variable storage
	globals     map[string]Value
	variableEnd Pos

	stack *Stack[Value]
	call  *Stack[Call]
}

type Call struct {
	chunk       *Chunk
	ip          Pos
	stackEnd    Pos
	variableEnd Pos
	scope       Pos
}

var DefaultGlobals = map[string]Value{
	"write": &BuiltinFunctionValue{
		"write", // always remember where you come from...
		[]string{"value"},
		func(_ *VM, this Value, v map[string]Value) (Value, error) {
			println(v["value"].String())
			return nil, nil
		},
		nil,
	},
	"print": &BuiltinFunctionValue{
		"print",
		[]string{"value"},
		func(_ *VM, this Value, v map[string]Value) (Value, error) {
			print(v["value"].String())
			return nil, nil
		},
		nil,
	},
	"format": &BuiltinFunctionValue{
		"format",
		[]string{"format_string", "values"},
		func(vm *VM, value Value, m map[string]Value) (Value, error) {
			valuies := m["values"].(*ListValue).items

			return GoToValue(fmt.Sprintf(m["format_string"].String(), valuies)), nil
		},
		nil,
	},
	"assertEq": &BuiltinFunctionValue{
		"assertEq",
		[]string{"a", "b"},
		func(vm *VM, this Value, params map[string]Value) (Value, error) {
			a := params["a"]
			b := params["b"]

			if !a.Equals(b) {
				return nil, errors.New(fmt.Sprintf("assertion failed: %s does not equal %s", a, b))
			}

			return &NilValue{}, nil
		},
		nil,
	},
	"assertNotEq": &BuiltinFunctionValue{
		"assertNotEq",
		[]string{"a", "b"},
		func(vm *VM, this Value, params map[string]Value) (Value, error) {
			a := params["a"]
			b := params["b"]

			if a.Equals(b) {
				return nil, errors.New(fmt.Sprintf("assertion failed: %s does not equal %s", a, b))
			}

			return &NilValue{}, nil
		},
		nil,
	},
}

func NewVM(chunk *Chunk, stackSize Pos, callstackSize Pos) *VM {
	vm := &VM{
		chunk: chunk,
		stack: NewStack[Value](stackSize),
		call:  NewStack[Call](callstackSize),

		globals: DefaultGlobals,
	}

	return vm
}

// Next execute instruction
// returns true if more instructions should be executed
func (vm *VM) Next() bool {
	if !vm.HasNext() {
		return false
	}

	switch vm.NextByte() {
	case InstructionReturn:
		if vm.call.Current == 0 {
			return false
		} else {
			v := vm.stack.Pop()
			c := vm.call.Pop()

			// reset stack current and variable end and scope
			vm.variableEnd = c.variableEnd
			vm.stack.Current = c.stackEnd
			vm.scope = c.scope

			// reset to calling position
			vm.ip = c.ip
			vm.chunk = c.chunk

			vm.purgeVars()

			vm.stack.Push(v)
		}

	case InstructionPop:
		vm.stack.Pop()

	case InstructionConstant:
		vm.stack.Push(vm.ReadConstant())

	case InstructionAdd:
		r := vm.stack.Pop().(*NumberValue).float64
		l := vm.stack.Pop().(*NumberValue).float64

		vm.stack.Push(&NumberValue{l + r})

	case InstructionSub:
		r := vm.stack.Pop().(*NumberValue).float64
		l := vm.stack.Pop().(*NumberValue).float64

		vm.stack.Push(&NumberValue{l - r})

	case InstructionMul:
		r := vm.stack.Pop().(*NumberValue).float64
		l := vm.stack.Pop().(*NumberValue).float64

		vm.stack.Push(&NumberValue{l * r})

	case InstructionDiv:
		r := vm.stack.Pop().(*NumberValue).float64
		l := vm.stack.Pop().(*NumberValue).float64

		vm.stack.Push(&NumberValue{l / r})

	case InstructionEquals:
		vm.stack.Push(
			&BoolValue{vm.stack.Pop().Equals(vm.stack.Pop())},
		)

	case InstructionNotEqual:
		vm.stack.Push(
			&BoolValue{!vm.stack.Pop().Equals(vm.stack.Pop())},
		)

	case InstructionNot:
		b := vm.stack.Pop().(*BoolValue).bool
		vm.stack.Push(&BoolValue{!b})

	case InstructionAnd:
		r := vm.stack.Pop().(*BoolValue).bool
		l := vm.stack.Pop().(*BoolValue).bool
		vm.stack.Push(&BoolValue{l && r})

	case InstructionOr:
		r := vm.stack.Pop().(*BoolValue).bool
		l := vm.stack.Pop().(*BoolValue).bool
		vm.stack.Push(&BoolValue{l || r})

	case InstructionLess:
		r := vm.stack.Pop().(*NumberValue).float64
		l := vm.stack.Pop().(*NumberValue).float64

		vm.stack.Push(&BoolValue{l < r})

	case InstructionLessOrEqual:
		r := vm.stack.Pop().(*NumberValue).float64
		l := vm.stack.Pop().(*NumberValue).float64

		vm.stack.Push(&BoolValue{l <= r})

	case InstructionGreater:
		r := vm.stack.Pop().(*NumberValue).float64
		l := vm.stack.Pop().(*NumberValue).float64

		vm.stack.Push(&BoolValue{l > r})

	case InstructionGreaterOrEqual:
		r := vm.stack.Pop().(*NumberValue).float64
		l := vm.stack.Pop().(*NumberValue).float64

		vm.stack.Push(&BoolValue{l >= r})

	case InstructionCall:
		v := vm.stack.Pop()
		switch f := v.(type) {
		case *FunctionValue:
			vm.call.Push(Call{
				chunk:       vm.chunk,
				ip:          vm.ip,
				stackEnd:    vm.stack.Current - Pos(len(f.Params)),
				variableEnd: vm.variableEnd,
				scope:       vm.scope,
			})

			for i := len(f.Params) - 1; i >= 0; i-- {
				p := vm.stack.Current - Pos(len(f.Params)) + Pos(i)
				vm.stack.items[p] = &VariableValue{
					f.Params[i],
					vm.stack.items[p],
					vm.scope,
				}
			}

			if f.Parent != nil {
				vm.addVar("this", f.Parent)
			}

			vm.variableEnd = vm.stack.Current

			vm.chunk = f.Chunk
			vm.ip = 0
		case *BuiltinFunctionValue:
			args := map[string]Value{}

			for i := len(f.Parameters) - 1; i >= 0; i-- {
				args[f.Parameters[i]] = vm.stack.Pop()
			}

			v, err := f.F(vm, f.Parent, args)
			if err != nil {
				vm.error(err.Error())
			}

			vm.stack.Push(v)
		default:
			vm.error(fmt.Sprintf("value called is not a function (%s, type %T)", v.DebugString(), v))
			return false
		}

	case InstructionJump:
		vm.ip += Pos(vm.NextU16())

	case InstructionLoop:
		vm.ip -= Pos(vm.NextU16())

	case InstructionJumpFalse:
		n := vm.NextU16()
		if !vm.stack.Pop().(*BoolValue).bool {
			vm.ip += Pos(n)
		}

	case InstructionGetLocal:
		name := vm.GetConstant(vm.NextByte()).(*StringValue).string
		v := vm.getVar(name)

		if v == nil {
			vm.error(fmt.Sprintf("cannot get local: undefined variable %s", name))
			return false
		}

		vm.stack.Push(v.value)

	case InstructionSetLocal:
		value := vm.stack.Pop().(Value)
		name := vm.GetConstant(vm.NextByte()).(*StringValue).string

		v := vm.getVar(name)

		if v == nil {
			vm.error(fmt.Sprintf("cannot set local: undefined variable %s", name))
		}

		v.value = value

	case InstructionDeclareLocal:
		vm.addVar(
			vm.GetConstant(vm.NextByte()).(*StringValue).string,
			vm.stack.Pop().(Value),
		)

	case InstructionGetGlobal:
		vm.stack.Push(vm.globals[vm.GetConstant(vm.NextByte()).(*StringValue).string])

	case InstructionSetGlobal:
		vm.globals[vm.GetConstant(vm.NextByte()).(*StringValue).string] = vm.stack.Pop()

	case InstructionTrue:
		vm.stack.Push(&BoolValue{true})

	case InstructionFalse:
		vm.stack.Push(&BoolValue{false})

	case InstructionNil:
		vm.stack.Push(&NilValue{})

	case InstructionFormList:
		n := int(vm.NextU16())

		items := make([]Value, n+1)
		for i := 0; i <= n; i++ {
			items[n-i] = vm.stack.Pop()
		}

	case InstructionNewList:
		vm.stack.Push(&ListValue{[]Value{}})

	case InstructionAppend:
		value := vm.stack.Pop()
		list := vm.stack.Pop().(*ListValue)
		list.items = append(list.items, value)
		vm.stack.Push(list)

	case InstructionDescend:
		vm.descend()

	case InstructionAscend:
		vm.ascend()

	case InstructionStringConversion:
		v := vm.stack.Pop()
		vm.stack.Push(&StringValue{v.String()})

	case InstructionStringConcatenation:
		r := vm.stack.Pop().(*StringValue).string
		l := vm.stack.Pop().(*StringValue).string

		vm.stack.Push(&StringValue{l + r})

	case InstructionSwap:
		r := vm.stack.Pop()
		l := vm.stack.Pop()

		vm.stack.Push(r, l)

	case InstructionAccessProperty:
		source := vm.stack.Pop()
		property := vm.ReadConstant()

		member, err := source.Get(property.(*StringValue).String())
		if err != nil {
			vm.error(err.Error())
		}

		// add parent if function
		if member.Type() == FunctionValueType {
			member.(*FunctionValue).Parent = source
		} else if member.Type() == BuiltinFunctionValueType {
			member.(*BuiltinFunctionValue).Parent = source
		}

		vm.stack.Push(member)

	case InstructionBreakpoint:

	default:
		panic("invalid byte code")
	}

	return true
}

func (vm *VM) Call(v Value, args []Value) (Value, error) {
	switch f := v.(type) {
	case *FunctionValue:
		vm.call.Push(Call{
			chunk:       vm.chunk,
			ip:          vm.ip,
			stackEnd:    vm.stack.Current,
			variableEnd: vm.variableEnd,
			scope:       vm.scope,
		})

		for i := 0; i < len(f.Params); i++ {
			vm.addVar(f.Params[i], args[i])
		}

		if f.Parent != nil {
			vm.addVar("this", f.Parent)
		}

		vm.variableEnd = vm.stack.Current

		vm.chunk = f.Chunk
		vm.ip = 0

		for vm.chunk.Bytecode[vm.ip] != InstructionReturn && vm.Next() {
		}

		if vm.HasNext() {
			vm.Next()
		}

		return vm.stack.Pop(), nil

	case *BuiltinFunctionValue:
		argies := map[string]Value{}

		for i, arg := range args {
			argies[f.Parameters[i]] = arg
		}

		return f.F(vm, f.Parent, argies)
	}

	return nil, errors.New(fmt.Sprintf("value is not a function (%s)", v.DebugString()))
}

func (vm *VM) TryNextByte() (Bytecode, error) {
	if !vm.HasNext() {
		return 0, errors.New("there are no more instructions")
	}

	v := vm.chunk.Bytecode[vm.ip]
	vm.ip++

	return v, nil
}

func (vm *VM) NextByte() Bytecode {
	b, err := vm.TryNextByte()

	if err != nil {
		panic(err)
	}

	return b
}

func (vm *VM) ascend() {
	vm.scope--

	if vm.scope < 0 {
		panic("invalid scope")
	}

	vm.purgeVars()
}

// purgeVars remove all variables not within scope
func (vm *VM) purgeVars() {
	for ; vm.variableEnd > 0 && vm.stack.items[vm.variableEnd-1].(*VariableValue).scope > vm.scope; vm.variableEnd-- {
		vm.stack.Pop()
	}
}

func (vm *VM) descend() {
	vm.scope++
}

func (vm *VM) addVar(name string, value Value) {
	vm.variableEnd++
	vm.stack.Push(&VariableValue{
		name,
		value,
		vm.scope,
	})
}

func (vm *VM) getVar(name string) *VariableValue {
	for i := vm.variableEnd - 1; i >= 0; i-- {
		v, ok := vm.stack.items[i].(*VariableValue)

		if !ok {
			continue
		}

		if v.name == name {
			return v
		}
	}

	return nil
}

func (vm *VM) HasNext() bool {
	return vm.ip < Pos(len(vm.chunk.Bytecode))
}

func (vm *VM) GetConstant(id Bytecode) Value {
	return vm.chunk.Constants[id]
}

func (vm *VM) ReadConstant() Value {
	return vm.GetConstant(vm.NextByte())
}

func (vm *VM) NextU16() uint16 {
	return (uint16(vm.NextByte()) << 8) | uint16(vm.NextByte())
}

func (vm *VM) error(error string) {
	log.Fatal(error)
}

func (vm *VM) SetGlobal(name string, value Value) {
	vm.globals[name] = value
}

func (vm *VM) GetGlobal(name string) Value {
	return vm.globals[name]
}
