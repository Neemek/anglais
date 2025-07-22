package core

import (
	"errors"
	"fmt"
	"strings"
)

type Compiler struct {
	Chunk *Chunk
	ip    Pos
	scope Pos

	imports   []string
	fileStack *Stack[string]
	resolver  ImportsResolver

	source   []rune
	Warnings []CompilerError

	stack *Stack[LocalVariable]
}

type ImportsResolver interface {
	Resolve(path string) (string, error)
	IsSame(a, b string) bool
}

type LocalVariable struct {
	name      string
	signature TypeSignature
	scope     int
}

type CompilerError struct {
	Description string
	Causer      Node
	Source      []rune
	Trace       []string
}

func (e CompilerError) Error() string {
	return e.Description
}

func (e CompilerError) Format() string {
	b := strings.Builder{}

	src := e.Source

	b.WriteString(e.Description)

	// highlight offending area
	start, end := e.Causer.Bounds()

	lineEnd := 0
	lineStart := 0
	line := 1
	pos := 0

	for i := Pos(0); i <= start; i++ {
		pos++

		if src[i] == '\n' {
			line++
			lineStart = int(i) + 1
			pos = 0
		}
	}

	for lineEnd < int(end) {
		b.WriteString("\n")
		lineEnd = lineStart
		for lineEnd < len(src) {
			if src[lineEnd] == '\n' {
				lineEnd++
				break
			}

			lineEnd++
		}

		begin := max(0, int(start)-lineStart)
		length := int(min(end, Pos(lineEnd)) - max(start, Pos(lineStart)))

		lineDescriptor := fmt.Sprintf("%d:%d~%d",
			line,
			begin,
			begin+length,
		)

		b.WriteString(lineDescriptor)
		b.WriteString(" | ")

		b.WriteString(string(src[lineStart : lineEnd-1]))
		b.WriteString("\n")

		b.WriteString(strings.Repeat(" ", len(lineDescriptor)))
		b.WriteString("   ")
		b.WriteString(strings.Repeat(" ", max(int(start)-lineStart, 0)))
		b.WriteString(strings.Repeat("^", length))

		lineStart = lineEnd
		line++
	}

	b.WriteString("\nsource trace:")
	// print import stack trace
	for i := len(e.Trace) - 1; i >= 0; i-- {
		p := e.Trace[i]
		b.WriteString(fmt.Sprintf("\n[%d] %s", i+1, p))
	}

	return b.String()
}

func NewCompiler(source []rune) *Compiler {
	c := &Compiler{
		NewChunk(make([]Bytecode, 0), make([]Value, 0)),
		0,
		0,
		make([]string, 0),
		NewStack[string](256),
		nil,
		source,
		[]CompilerError{},
		NewStack[LocalVariable](256),
	}

	return c
}

func (c *Compiler) add(instruction Bytecode) {
	for len(c.Chunk.Bytecode) <= int(c.ip) {
		c.Chunk.Bytecode = append(c.Chunk.Bytecode, 0)
	}

	c.Chunk.Bytecode[c.ip] = instruction

	c.advance(1)
}

func (c *Compiler) addConstant(value Value) {
	chunk := c.Chunk
	for i := 0; i < len(chunk.Constants); i++ {
		if chunk.Constants[i].Equals(value) {
			c.add(Bytecode(i))

			return
		}
	}

	chunk.Constants = append(chunk.Constants, value)

	c.add(Bytecode(len(chunk.Constants) - 1))
}

func (c *Compiler) Compile(p *Program) error {
	c.fileStack.Push(p.Path)

	for _, i := range p.Imports {
		if err := c.resolveImport(i); err != nil {
			return err
		}
	}

	for _, s := range p.Block.statements {
		if err := c.compile(s); err != nil {
			return err
		}
	}

	c.fileStack.Pop()

	return nil
}

func (c *Compiler) compile(tree Node) error {
	if tree == nil {
		panic("compile called with nil value")
	}

	switch tree.Type() {
	case StringNodeType:
		c.add(InstructionConstant)
		c.addConstant(&StringValue{
			tree.(*StringNode).value,
		})

	case NumberNodeType:
		c.add(InstructionConstant)
		c.addConstant(&NumberValue{tree.(*NumberNode).value})

	case ListNodeType:
		l := tree.(*ListNode)

		if len(l.items) == 0 {
			c.add(InstructionNewList)
		} else if c.isTreeConstant(l) {
			v, err := c.compute(l)
			if err != nil {
				panic(err) // this shouldn't happen
			}

			c.add(InstructionConstant)
			c.addConstant(v)
		} else {
			for _, n := range l.items {
				err := c.compile(n)
				if err != nil {
					return err
				}
			}
			c.add(InstructionFormList)
			c.addU16(uint16(len(l.items)))
		}

	case ReferenceNodeType:
		c.addGetVar(tree.(*ReferenceNode).name)

	case BinaryNodeType:
		err := c.compileBinary(tree.(*BinaryNode))
		if err != nil {
			return err
		}

	case UnaryNodeType:
		if c.isTreeConstant(tree.(*UnaryNode).value) {
			v, err := c.compute(tree)
			if err != nil {
				return err
			}

			c.add(InstructionConstant)
			c.addConstant(v)
		} else {
			err := c.compile(tree.(*UnaryNode).value)
			if err != nil {
				return err
			}

			switch tree.(*UnaryNode).UnaryOperation {
			case UnaryNegate:
				c.add(InstructionNegate)
			case UnaryNot:
				c.add(InstructionNot)
			}
		}

	case BooleanNodeType:
		if tree.(*BooleanNode).value {
			c.add(InstructionTrue)
		} else {
			c.add(InstructionFalse)
		}

	case NilNodeType:
		c.add(InstructionNil)

	case BlockNodeType:
		c.addDescend()
		for _, n := range tree.(*BlockNode).statements {
			err := c.compile(n)
			if err != nil {
				return err
			}
		}
		c.addAscend()

	case ConditionalNodeType:
		n := tree.(*ConditionalNode)

		// make sure condition is boolean
		sig, err := c.deduceSignature(n.condition)
		if err != nil {
			return err
		}
		if sig.Type() != TypeBoolean {
			return c.error(fmt.Sprintf("conditional requires boolean; cannot use non-boolean type %s", sig), n.condition)
		}

		// the stack should have whether the condition was truthful
		err = c.compile(n.condition)
		if err != nil {
			return err
		}

		// if the condition equated to true, we should jump over the body
		c.add(InstructionJumpFalse)
		// we save where uint16 jump by value is stored, and update it when
		// we know the size of this condition (in bytecode)
		jumpByPos := c.ip
		c.advance(2)

		// this part would be executed if the value was true
		err = c.compile(n.do)
		if err != nil {
			return err
		}

		// we store the position of the jump over the else code here
		var jumpOverElse Pos
		if n.otherwise != nil {
			// this would jump over the else/otherwise block in the code
			c.add(InstructionJump)
			jumpOverElse = c.ip
			c.advance(2)
		}

		// put the u16 of where to jump if the condition was false
		c.putU16(jumpByPos, uint16(c.ip-jumpByPos-2))

		if n.otherwise != nil {
			err := c.compile(n.otherwise)
			if err != nil {
				return err
			}
			c.putU16(jumpOverElse, uint16(c.ip-jumpOverElse-2))
		}

	case LoopNodeType:
		n := tree.(*LoopNode)

		// make sure condition is boolean
		sig, err := c.deduceSignature(n.condition)
		if err != nil {
			return err
		}
		if sig.Type() != TypeBoolean {
			return c.error(fmt.Sprintf("cannot loop over value of type %s; requires boolean", sig), n.condition)
		}

		conditionPos := c.ip
		err = c.compile(n.condition)
		if err != nil {
			return err
		}

		c.add(InstructionJumpFalse)
		jumpValuePos := c.ip
		c.advance(2)

		err = c.compile(n.do)
		if err != nil {
			return err
		}

		c.add(InstructionLoop)
		// condition pos < ip
		c.addU16(uint16(c.ip - conditionPos + 2))

		c.putU16(jumpValuePos, uint16(c.ip-jumpValuePos-2))

	case AssignNodeType:
		n := tree.(*AssignNode)

		if n.name == "_" {
			// allow non-ish statements
			err := c.compile(n.value)
			if err != nil {
				return err
			}
			c.add(InstructionPop)
		} else {
			if n.declare && c.isVarDeclaredHere(n.name) {
				return c.error(fmt.Sprintf("%s is already declared in this scope", n.name), n)
			}

			err := c.addSetVar(n.name, n.value, n.declare)
			if err != nil {
				return err
			}
		}

	case CallNodeType:
		n := tree.(*CallNode)

		s, err := c.deduceSignature(n.source)
		if err != nil {
			return err
		}

		f, ok := s.(*FunctionSignature)
		if !ok {
			return c.error(fmt.Sprintf("cannot call non-function value of type %s", s), n)
		}

		if !n.keep && f.Out.Type() != TypeNil {
			c.warn(fmt.Sprintf("shouldn't void result of function call (output is non-nil %s)", f.Out), n)
		}

		if len(n.args) != len(f.In) {
			return c.error(fmt.Sprintf("wrong argument count: function of signature %s got %d, requires %d", f, len(n.args), len(f.In)), n)
		}

		for i, arg := range n.args {
			sig, err := c.deduceSignature(arg)
			if err != nil {
				return err
			}

			// check that arg type is as required
			if !f.In[i].Matches(sig) {
				return c.error(fmt.Sprintf("argument #%d does not have expected type signature: got %s, requires %s", i, sig, f.In[i]), arg)
			}

			if c.isTreeConstant(arg) {
				v, err := c.compute(arg)
				if err != nil {
					return err
				}
				c.add(InstructionConstant)
				c.addConstant(v)
			} else {
				err = c.compile(arg)
				if err != nil {
					return err
				}
			}

		}

		err = c.compile(n.source)
		if err != nil {
			return err
		}

		c.add(InstructionCall)

		if !n.keep {
			c.add(InstructionPop)
		}

	case FunctionNodeType:
		n := tree.(*FunctionNode)

		fi := len(c.Chunk.Constants)
		c.Chunk.Constants = append(c.Chunk.Constants, nil)

		c.add(InstructionConstant)
		c.add(Bytecode(fi))

		// allow self-referencing
		sig, err := c.deduceSignature(n)
		if err != nil {
			return err
		}
		c.registerVar(n.name, sig)

		// keep track of main chunk
		mc := c.Chunk
		// and ip
		mip := c.ip

		// assign a new empty chunk
		c.Chunk = NewChunk(make([]Bytecode, 0), make([]Value, 0))
		// reset instruction pointer (ip)
		c.ip = 0

		c.descend()
		for _, p := range n.parameters {
			c.registerVar(p.Name, p.Signature)
		}

		err = c.affirmReturnSignature(n.logic, n.yield)
		if err != nil {
			return err
		}

		err = c.compile(n.logic)
		if err != nil {
			return err
		}
		c.ascend()

		mc.Constants[fi] = &FunctionValue{
			n.name,
			n.parameters,
			n.yield,
			c.Chunk,
			nil,
		}

		// restore old chunk and ip
		c.Chunk = mc
		c.ip = mip

	case AccessNodeType:
		n := tree.(*AccessNode)
		err := c.compile(n.source)
		if err != nil {
			return err
		}
		c.add(InstructionAccessProperty)
		c.addConstant(&StringValue{
			n.property,
		})

	case ReturnNodeType:
		err := c.compile(tree.(*ReturnNode).value)
		if err != nil {
			return err
		}
		c.add(InstructionReturn)

	case BreakpointNodeType:
		c.add(InstructionBreakpoint)

	default:
		panic(fmt.Sprintf("unimplemented compiling of %s", tree.Type()))
	}

	return nil
}

func (c *Compiler) compileBinary(binary *BinaryNode) error {
	if c.isTreeConstant(binary) {
		v, err := c.compute(binary)
		if err != nil {
			return err
		}

		c.add(InstructionConstant)
		c.addConstant(v)
		return nil
	}

	err := c.compile(binary.Left)
	if err != nil {
		return err
	}
	err = c.compile(binary.Right)
	if err != nil {
		return err
	}

	switch binary.BinaryOperation {
	case BinaryAddition:
		res, err := c.deduceSignature(binary)
		if err != nil {
			return err
		}

		if res.Type() == TypeString {
			c.add(InstructionStringConcatenation)
		} else if res.Type() == TypeList {
			c.add(InstructionConcatLists)
		} else {
			c.add(InstructionAdd)
		}
	case BinarySubtraction:
		c.add(InstructionSub)
	case BinaryMultiplication:
		c.add(InstructionMul)
	case BinaryDivision:
		c.add(InstructionDiv)
	case BinaryEquality:
		c.add(InstructionEquals)
	case BinaryInequality:
		c.add(InstructionNotEqual)
	case BinaryLess:
		c.add(InstructionLess)
	case BinaryGreater:
		c.add(InstructionGreater)
	case BinaryLessEqual:
		c.add(InstructionLessOrEqual)
	case BinaryGreaterEqual:
		c.add(InstructionGreaterOrEqual)
	case BinaryAnd:
		c.add(InstructionAnd)
	case BinaryOr:
		c.add(InstructionOr)
	}

	return nil
}

func (c *Compiler) deduceSignature(tree Node) (TypeSignature, error) {
	switch tree.Type() {
	case StringNodeType:
		return &StringSignature{}, nil
	case NumberNodeType:
		return &NumberSignature{}, nil
	case ReferenceNodeType:
		n := tree.(*ReferenceNode)
		sig, err := c.getVarSignature(n.name, n)
		if err != nil {
			return nil, err
		}

		return sig, nil

	case BooleanNodeType:
		return &BooleanSignature{}, nil
	case NilNodeType:
		return &NilSignature{}, nil
	case ListNodeType:
		n := tree.(*ListNode)

		contents := n.content
		// check for contents type
		for _, v := range n.items {
			sig, err := c.deduceSignature(v)
			if err != nil {
				return nil, err
			}

			if contents == nil {
				contents = sig
			} else if !contents.Matches(sig) {
				return nil, c.error(fmt.Sprintf("non-conforming item type"), v)
			}
		}

		if contents == nil {
			return nil, c.error("can't deduce content type", n)
		}

		return &ListSignature{
			contents,
		}, nil
	case BinaryNodeType:
		n := tree.(*BinaryNode)
		l, err := c.deduceSignature(n.Left)
		if err != nil {
			return nil, err
		}
		r, err := c.deduceSignature(n.Right)
		if err != nil {
			return nil, err
		}

		if !l.Matches(r) {
			return nil, c.error(fmt.Sprintf("cannot perform binary %s on different types: %s and %s", n.BinaryOperation, l, r), n)
		}

		switch n.BinaryOperation {
		case BinarySubtraction, BinaryMultiplication, BinaryDivision:
			if l.Type() != TypeNumber {
				return nil, c.error(fmt.Sprintf("cannot perform binary %s non-number type %s", n.BinaryOperation, l), n)
			}

			return &NumberSignature{}, nil
		case BinaryAddition:
			switch l.Type() {
			case TypeString:
				return &StringSignature{}, nil
			case TypeNumber:
				return &NumberSignature{}, nil
			case TypeList:
				return &ListSignature{
					l.(*ListSignature).Contents,
				}, nil
			default:
				return nil, c.error(fmt.Sprintf("cannot perform binary addition on type %s", l), n)
			}
		case BinaryAnd, BinaryOr:
			if l.Type() != TypeBoolean {
				return nil, c.error(fmt.Sprintf("cannot perform binary %s on non-boolean type %s", n.BinaryOperation, l), n)
			}

			return &BooleanSignature{}, nil
		case BinaryEquality, BinaryInequality:
			return &BooleanSignature{}, nil
		case BinaryLess, BinaryGreater, BinaryLessEqual, BinaryGreaterEqual:
			if l.Type() != TypeNumber {
				return nil, c.error(fmt.Sprintf("cannot perform number comparison (%s) on non-number type %s", n.BinaryOperation, l), n)
			}

			return &BooleanSignature{}, nil
		}

		return nil, c.error(fmt.Sprintf("cannot deduce result type of binary %s", n.BinaryOperation), n)
	case AccessNodeType:
		n := tree.(*AccessNode)
		sig, err := c.deduceSignature(n.source)
		if err != nil {
			return nil, err
		}

		switch sig.Type() {
		case TypeString:
			v := StringPrototype[n.property]
			if v == nil {
				return nil, c.error(fmt.Sprintf("string has no property %s", n.property), tree)
			}

			return SignatureOf(v), nil
		case TypeList:
			v := ListPrototype[n.property]
			if v == nil {
				return nil, c.error(fmt.Sprintf("list has no property %s", n.property), tree)
			}

			return SignatureOf(v), nil
		case TypeObject:
			if v, ok := ObjectPrototype[n.property]; ok {
				return SignatureOf(v), nil
			}

			v := sig.(*ObjectSignature).Members[n.property]
			if v == nil {
				return nil, c.error(fmt.Sprintf("object has no property %s", n.property), tree)
			}

			return v, nil

		default:
			return nil, c.error(fmt.Sprintf("cannot access property from value of type %s", sig), n)
		}

	case CallNodeType:
		n := tree.(*CallNode)
		sig, err := c.deduceSignature(n.source)
		if err != nil {
			return nil, err
		}

		if sig.Type() != TypeFunction {
			return nil, c.error(fmt.Sprintf("cannot call value of type %s", sig.Type()), n.source)
		}

		f := sig.(*FunctionSignature)

		if len(n.args) != len(f.In) {
			return nil, c.error(fmt.Sprintf("bad argument count (expected %v, got %v)", len(f.In), len(n.args)), n)
		}

		var innerType TypeSignature
		if n.source.Type() == AccessNodeType {
			member, err := c.deduceSignature(n.source.(*AccessNode).source)
			if err != nil {
				return nil, err
			}

			if member.Type() == TypeList {
				innerType = member.(*ListSignature).Contents
			}
		}

		// type check arguments
		for i, arg := range n.args {
			sig, err := c.deduceSignature(arg)
			if err != nil {
				return nil, err
			}

			t := f.In[i]
			if t.Type() == TypeInner {
				if innerType == nil {
					return nil, c.error(fmt.Sprintf("function source (%s) has no inner type", sig), n.source)
				}

				t = innerType
			}

			if !sig.Matches(t) {
				return nil, c.error(fmt.Sprintf("argument #%d has wrong type signature. requires %s, got %s", i, f.In[i], sig), arg)
			}
		}

		if f.Out.Type() == TypeInner {
			if innerType == nil {
				return nil, c.error(fmt.Sprintf("function source (%s) has no inner type", sig), n.source)
			}

			return innerType, nil
		}

		return f.Out, nil

	case FunctionNodeType:
		n := tree.(*FunctionNode)

		sigs := make([]TypeSignature, len(n.parameters))

		for i, p := range n.parameters {
			sigs[i] = p.Signature
		}

		return &FunctionSignature{
			sigs,
			n.yield,
		}, nil

	case UnaryNodeType:
		n := tree.(*UnaryNode)

		sig, err := c.deduceSignature(n.value)
		if err != nil {
			return nil, err
		}

		switch n.UnaryOperation {
		case UnaryNegate:
			if sig.Type() != TypeNumber {
				return nil, c.error(fmt.Sprintf("cannot perform negation on type %s (must be number)", n.UnaryOperation), n)
			}
			return &NumberSignature{}, nil
		case UnaryNot:
			if sig.Type() != TypeBoolean {
				return nil, c.error(fmt.Sprintf("cannot perform negation on type %s (must be boolean)", n.UnaryOperation), n)
			}
			return &BooleanSignature{}, nil
		}

		return nil, c.error(fmt.Sprintf("unimplemented result type deduction for unary %s", n.UnaryOperation), n)
	default:
		return nil, c.error(fmt.Sprintf("impossible to deduce signature of %s", tree.Type()), tree)
	}
}

func (c *Compiler) getVarSignature(name string, causer Node) (TypeSignature, error) {
	if c.isGlobal(name) {
		return SignatureOf(DefaultGlobals[name]), nil
	}

	for i := c.stack.Current - 1; i >= 0; i-- {
		v := c.stack.items[i]
		if v.name == name {
			return v.signature, nil
		}
	}

	return nil, c.error(fmt.Sprintf("variable %s not defined", name), causer)
}

func (c *Compiler) affirmReturnSignature(tree Node, sig TypeSignature) error {
	switch tree.Type() {
	case BlockNodeType:
		c.descend()
		for _, stmt := range tree.(*BlockNode).statements {
			if err := c.affirmReturnSignature(stmt, sig); err != nil {
				return err
			}
		}
		c.ascend()

	case ReturnNodeType:
		n := tree.(*ReturnNode)
		v, err := c.deduceSignature(n.value)
		if err != nil {
			return err
		}

		if !sig.Matches(v) {
			return c.error(fmt.Sprintf("function cannot return a value with type %s. defined to be %s", v, sig), n)
		}

	case ConditionalNodeType:
		n := tree.(*ConditionalNode)

		if err := c.affirmReturnSignature(n.do, sig); err != nil {
			return err
		}

		if n.otherwise != nil {
			if err := c.affirmReturnSignature(n.otherwise, sig); err != nil {
				return err
			}
		}

	case LoopNodeType:
		n := tree.(*LoopNode)

		if err := c.affirmReturnSignature(n.do, sig); err != nil {
			return err
		}

	case AssignNodeType:
		n := tree.(*AssignNode)

		if !n.declare {
			prev, err := c.getVarSignature(n.name, n)
			if err != nil {
				return err
			}

			sig, err := c.deduceSignature(n.value)
			if err != nil {
				return err
			}

			if !sig.Matches(prev) {
				return c.error(fmt.Sprintf("cannot assign value of type %s to variable %s of type %s", sig, n.name, prev), n.value)
			}

			return nil
		}

		sig, err := c.deduceSignature(n.value)
		if err != nil {
			return err
		}
		c.registerVar(n.name, sig)
	default:
	}

	return nil
}

// isVarDeclaredHere check whether a variable is declared in the current scope
func (c *Compiler) isVarDeclaredHere(name string) bool {
	for i := c.stack.Current - 1; i >= 0 && c.stack.items[i].scope == int(c.scope); i-- {
		v := c.stack.items[i]
		if v.name == name {
			return true
		}
	}

	return false
}

func (c *Compiler) addGetVar(name string) {
	if c.isGlobal(name) {
		c.add(InstructionGetGlobal)
		c.addConstant(&StringValue{
			name,
		})
	} else {
		c.add(InstructionGetLocal)
		c.addConstant(&StringValue{
			name,
		})
	}
}

func (c *Compiler) addSetVar(name string, value Node, declare bool) error {
	err := c.compile(value)
	if err != nil {
		return err
	}

	t, err := c.deduceSignature(value)
	if err != nil {
		return err
	}

	if declare {
		c.add(InstructionDeclareLocal)
		c.registerVar(name, t)
	} else {
		vt, err := c.getVarSignature(name, value)
		if err != nil {
			return err
		}

		if !vt.Matches(t) {
			return c.error(fmt.Sprintf("cannot assign value of type %s to variable %s of type %s", t, name, vt), value)
		}

		c.add(InstructionSetLocal)
	}

	c.addConstant(&StringValue{
		name,
	})

	return nil
}

// keep track that a variable is declared but doesn't necessarily have a deducible type
func (c *Compiler) registerVar(name string, t TypeSignature) {
	c.stack.Push(LocalVariable{
		name,
		t,
		int(c.scope),
	})
}

// isLocal whether a variable of with the name provided is declared within the local scope
func (c *Compiler) isLocal(name string) bool {
	for i := c.stack.Current - 1; i >= 0; i-- {
		if c.stack.items[i].name == name {
			return true
		}
	}
	return false
}

// isTreeConstant check if a node tree is constant (predictable)
func (c *Compiler) isTreeConstant(tree Node) bool {
	switch tree.Type() {
	case StringNodeType, NumberNodeType, BooleanNodeType, NilNodeType:
		return true
	case ListNodeType:
		for _, item := range tree.(*ListNode).items {
			if !c.isTreeConstant(item) {
				return false
			}
		}

		return true
	case BinaryNodeType:
		return c.isTreeConstant(tree.(*BinaryNode).Left) && c.isTreeConstant(tree.(*BinaryNode).Right)
	case CallNodeType:
		for _, arg := range tree.(*CallNode).args {
			if !c.isTreeConstant(arg) {
				return false
			}
		}
		return c.isTreeConstant(tree.(*CallNode).source)
	case BlockNodeType, ConditionalNodeType, LoopNodeType, AssignNodeType, FunctionNodeType,
		ReturnNodeType, AccessNodeType, BreakpointNodeType, ReferenceNodeType:
		return false
	default:
		panic(fmt.Sprintf("unexpected node %s", tree))
	}
}

func (c *Compiler) compute(tree Node) (Value, error) {
	switch n := tree.(type) {
	case *StringNode:
		return &StringValue{
			n.value,
		}, nil

	case *NumberNode:
		return &NumberValue{
			n.value,
		}, nil

	case *BooleanNode:
		return &BoolValue{
			n.value,
		}, nil

	case *NilNode:
		return &NilValue{}, nil

	case *ListNode:
		items := make([]Value, len(n.items))
		var err error
		for i, item := range n.items {
			items[i], err = c.compute(item)

			if err != nil {
				return nil, err
			}
		}
		return &ListValue{
			items,
		}, nil

	case *BinaryNode:
		return c.computeBinary(n)

	case *UnaryNode:
		v, err := c.compute(n.value)
		if err != nil {
			return nil, err
		}

		return &NumberValue{
			-v.(*NumberValue).Number,
		}, nil

	case *CallNode:
		source, err := c.compute(n.source)
		if err != nil {
			return nil, err
		}

		f, ok := source.(*BuiltinFunctionValue)
		if !ok {
			return nil, nil
		}

		if !f.Constant {
			return nil, nil
		}

		args := make([]Value, len(f.Signature.In))
		for i, arg := range n.args {
			args[i], err = c.compute(arg)
			if err != nil {
				return nil, err
			}
		}

		return f.F(nil, nil, args)

	default:
		panic(fmt.Sprintf("unexpected node %s, %T", tree.String(), tree))
	}
}

func (c *Compiler) computeBinary(n *BinaryNode) (Value, error) {
	l, err := c.compute(n.Left)
	if err != nil {
		return nil, err
	}
	r, err := c.compute(n.Right)
	if err != nil {
		return nil, err
	}

	if l.Type() != r.Type() {
		return nil, c.error(fmt.Sprintf("cannot perform binary %s on different types %s and %s", n.BinaryOperation, l.Type(), r.Type()), n)
	}

	// perform type check
	switch n.BinaryOperation {
	case BinaryAddition:
		if l.Type() != StringValueType && l.Type() != NumberValueType && l.Type() != ListValueType {
			return nil, c.error(fmt.Sprintf("cannot add values of type %s", l.Type()), n)
		}
	case BinarySubtraction, BinaryMultiplication, BinaryDivision, BinaryLess, BinaryGreater, BinaryLessEqual, BinaryGreaterEqual:
		if l.Type() != NumberValueType {
			return nil, c.error(fmt.Sprintf("cannot do binary %s on non-number type %s", n.BinaryOperation, l.Type()), n)
		}
	case BinaryAnd, BinaryOr:
		if l.Type() != BoolValueType {
			return nil, c.error(fmt.Sprintf("cannot do binary %s on non-boolean type %s", n.BinaryOperation, l.Type()), n)
		}
	case BinaryEquality, BinaryInequality:
		// can compare all types with themselves
	default:
	}

	var v interface{}
	switch n.BinaryOperation {
	case BinaryAddition:
		switch l.Type() {
		case NumberValueType:
			v = l.(*NumberValue).Number + r.(*NumberValue).Number
		case StringValueType:
			v = l.(*StringValue).Text + r.(*StringValue).Text
		case ListValueType:
			v = append(l.(*ListValue).Items, r.(*ListValue).Items...)
		default:
			return nil, c.error(fmt.Sprintf("cannot perform binary add on type %s", l.Type()), n)
		}
	case BinarySubtraction:
		v = l.(*NumberValue).Number - r.(*NumberValue).Number
	case BinaryMultiplication:
		v = l.(*NumberValue).Number * r.(*NumberValue).Number
	case BinaryDivision:
		v = l.(*NumberValue).Number / r.(*NumberValue).Number
	case BinaryAnd:
		v = l.(*BoolValue).Boolean && r.(*BoolValue).Boolean
	case BinaryOr:
		v = l.(*BoolValue).Boolean && r.(*BoolValue).Boolean
	case BinaryEquality:
		v = l.Equals(r)
	case BinaryInequality:
		v = !l.Equals(r)
	case BinaryLess:
		v = l.(*NumberValue).Number < r.(*NumberValue).Number
	case BinaryGreater:
		v = l.(*NumberValue).Number > r.(*NumberValue).Number
	case BinaryLessEqual:
		v = l.(*NumberValue).Number <= r.(*NumberValue).Number
	case BinaryGreaterEqual:
		v = l.(*NumberValue).Number >= r.(*NumberValue).Number
	}

	return GoToValue(v), nil
}

// isGlobal whether a variable is defined in the standard global environment
func (c *Compiler) isGlobal(name string) bool {
	return DefaultGlobals[name] != nil
}

func (c *Compiler) ascend() {
	c.scope--

	for ; c.stack.Current > 0 && c.stack.Peek().scope > int(c.scope); c.stack.Pop() {
	}
}

func (c *Compiler) addAscend() {
	c.ascend()
	c.add(InstructionAscend)
}

func (c *Compiler) descend() {
	c.scope++
}

func (c *Compiler) addDescend() {
	c.descend()
	c.add(InstructionDescend)
}

func (c *Compiler) error(msg string, causer Node) CompilerError {
	return CompilerError{
		msg,
		causer,
		c.source,
		c.fileStack.items[0:c.fileStack.Current],
	}
}

func (c *Compiler) warn(msg string, causer Node) {
	c.Warnings = append(c.Warnings, c.error(msg, causer))
}

func (c *Compiler) resolveImport(path string) error {
	// if already imported and available
	for _, i := range c.imports {
		if c.resolver.IsSame(path, i) {
			return nil
		}
	}

	// stop recursive imports
	for i := c.fileStack.Current - 1; i >= 0; i-- {
		if c.resolver.IsSame(path, c.fileStack.items[i]) {
			return errors.New("recursive imports")
		}
	}

	src, err := c.resolver.Resolve(path)
	if err != nil {
		return err
	}

	l := NewLexer(src)
	tokens, err := l.Tokenize()
	if err != nil {
		return err
	}

	parser := NewParser(src, tokens)
	p, err := parser.Parse(path)
	if err != nil {
		return err
	}

	oldSrc := c.source

	// update source for more descriptive errors
	c.source = []rune(src)
	if err := c.Compile(p); err != nil {
		return err
	}

	c.source = oldSrc

	return nil
}

func (c *Compiler) SetImportsResolver(resolver ImportsResolver) {
	c.resolver = resolver
}

func (c *Compiler) SetSource(src string) {
	c.source = []rune(src)
}

func (c *Compiler) advance(amount Pos) {
	c.ip += amount
}

func (c *Compiler) addU16(v uint16) {
	c.add(Bytecode(v >> 8))   // first 8 bits
	c.add(Bytecode(v & 0xff)) // last 8 bits
}

// putU16 put an unsigned 16-bit value at an arbitrary position.
// p is the position before the value
func (c *Compiler) putU16(p Pos, v uint16) {
	// save original position
	start := c.ip

	// move to position
	c.ip = p
	// set values of the next 2 bytes to the u16
	c.addU16(v)

	// restore position
	c.ip = start
}
