package main

import "log"

type Compiler struct {
	chunk *Chunk
	ip    Pos
	scope Pos

	stack *Stack[LocalVariable]
}

type LocalVariable struct {
	name  string
	scope int
}

func NewCompiler() *Compiler {
	c := &Compiler{
		chunk: NewChunk(make([]Bytecode, 0), make([]Value, 0)),
		ip:    0,
		scope: 0,
		stack: NewStack[LocalVariable](256),
	}

	return c
}

func (c *Compiler) add(instruction Bytecode) {
	for len(c.chunk.Bytecode) <= int(c.ip) {
		c.chunk.Bytecode = append(c.chunk.Bytecode, 0)
	}

	c.chunk.Bytecode[c.ip] = instruction

	c.advance(1)
}

func (c *Compiler) addConstant(value Value) {
	chunk := c.chunk
	for i := 0; i < len(chunk.Constants); i++ {
		if chunk.Constants[i] == value {
			c.add(Bytecode(i))

			return
		}
	}

	chunk.Constants = append(chunk.Constants, value)

	c.add(Bytecode(len(chunk.Constants) - 1))
}

func (c *Compiler) Compile(tree Node) {
	if tree == nil {
		panic("nil value parse tree node")
	}

	switch tree.Type() {
	case StringNodeType:
		c.add(InstructionConstant)
		c.addConstant(StringValue(tree.(*StringNode).value))

	case NumberNodeType:
		c.add(InstructionConstant)
		c.addConstant(tree.(*NumberNode).value)

	case ReferenceNodeType:
		c.getVar(tree.(*ReferenceNode).name)

	case BinaryNodeType:
		c.compileBinary(tree.(*BinaryNode))

	case BooleanNodeType:
		if tree.(*BooleanNode).value {
			c.add(InstructionTrue)
		} else {
			c.add(InstructionFalse)
		}

	case NilNodeType:
		c.add(InstructionNil)

	case BlockNodeType:
		c.descend()
		for _, n := range tree.(*BlockNode).statements {
			c.Compile(n)
		}
		c.ascend()

	case ConditionalNodeType:
		n := tree.(*ConditionalNode)

		// the stack should have whether the condition was truthful
		c.Compile(n.condition)

		// if the condition equated to true, we should jump over the body
		c.add(InstructionJumpFalse)
		// we save where uint16 jump by value is stored, and update it when
		// we know the size of this condition (in bytecode)
		jumpByPos := c.ip
		c.advance(2)

		// this part would be executed if the value was true
		c.Compile(n.do)

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
			c.Compile(n.otherwise)
			c.putU16(jumpOverElse, uint16(c.ip-jumpOverElse-2))
		}

	case LoopNodeType:
		n := tree.(*LoopNode)

		conditionPos := c.ip
		c.Compile(n.condition)

		c.add(InstructionJumpFalse)
		jumpValuePos := c.ip
		c.advance(2)

		c.Compile(n.do)

		c.add(InstructionLoop)
		// condition pos < ip
		c.addU16(uint16(c.ip - conditionPos + 2))

		c.putU16(jumpValuePos, uint16(c.ip-jumpValuePos-2))

	case AssignNodeType:
		n := tree.(*AssignNode)

		if n.name == "_" {
			// allow non-ish statements
			c.Compile(n.value)
			c.add(InstructionPop)
		} else {
			c.setVar(n.name, n.value, n.declare)
		}

	case CallNodeType:
		n := tree.(*CallNode)

		c.descend()

		for _, arg := range n.args {
			c.Compile(arg)
		}

		c.getVar(n.name)

		c.add(InstructionCall)

		if !n.keep {
			c.add(InstructionPop)
		}

		// Only descend and remove variables that are no longer within scope when the stack is clean
		c.ascend()

	case FunctionNodeType:
		n := tree.(*FunctionNode)

		fi := len(c.chunk.Constants)
		c.chunk.Constants = append(c.chunk.Constants, nil)

		c.add(InstructionConstant)
		c.add(Bytecode(fi))

		// keep track of main chunk
		mc := c.chunk
		// and ip
		mip := c.ip

		// assign a new empty chunk
		c.chunk = NewChunk(make([]Bytecode, 0), make([]Value, 0))
		// reset instruction pointer (ip)
		c.ip = 0

		c.descend()

		for _, p := range n.params {
			c.registerVar(p)
		}
		c.Compile(n.logic)
		if n.logic.Type() != BlockNodeType {
			c.stack.Pop()
		}

		mc.Constants[fi] = FunctionValue{
			n.name,
			n.params,
			c.chunk,
		}

		c.ascend()

		// restore old chunk and ip
		c.chunk = mc
		c.ip = mip

	case ReturnNodeType:
		c.Compile(tree.(*ReturnNode).value)
		c.add(InstructionReturn)
	}
}

func (c *Compiler) compileBinary(binary *BinaryNode) {
	c.Compile(binary.Left)
	c.Compile(binary.Right)

	switch binary.BinaryOperation {
	case BinaryAddition:
		c.add(InstructionAdd)
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
	}
}

func (c *Compiler) getVar(name string) {
	if c.isLocal(name) {
		c.add(InstructionGetLocal)
		c.addConstant(StringValue(name))
	} else if c.isGlobal(name) {
		c.add(InstructionGetGlobal)
		c.addConstant(StringValue(name))
	} else {
		log.Fatalf("compiling: undefined variable %s", name)
	}
}

func (c *Compiler) setVar(name string, value Node, declare bool) {
	c.Compile(value)

	if declare {
		c.add(InstructionDeclareLocal)
		c.registerVar(name)
	} else {
		c.add(InstructionSetLocal)
	}

	c.addConstant(StringValue(name))
}

// keep track that a variable is declared but doesn't necessarily have a deducible type
func (c *Compiler) registerVar(name string) {
	c.stack.Push(LocalVariable{
		name,
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

// isGlobal whether a variable is defined in the standard global environment
func (c *Compiler) isGlobal(name string) bool {
	return DefaultGlobals[name] != nil
}

func (c *Compiler) ascend() {
	c.scope--

	for ; c.stack.Current > 0 && c.stack.Peek().scope > int(c.scope); c.stack.Pop() {
	}

	if c.scope != 0 {
		c.add(InstructionAscend)
	}
}

func (c *Compiler) descend() {
	c.scope++
	if c.scope != 1 {
		c.add(InstructionDescend)
	}
}

func (c *Compiler) advance(amount Pos) {
	c.ip += amount
}

func (c *Compiler) addU16(v uint16) {
	c.add(Bytecode(v >> 8))   // first 8 bits
	c.add(Bytecode(v & 0xff)) // last 8 bits
}

// putU16 put a unsigned 16-bit value at an arbitrary position.
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
