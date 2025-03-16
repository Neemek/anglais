package core

import (
	"fmt"
)

type Compiler struct {
	Chunk *Chunk
	ip    Pos
	scope Pos

	imports  map[string]Node
	resolver ImportsResolver

	stack *Stack[LocalVariable]
}

type ImportsResolver interface {
	Resolve(path string) (Node, error)
}

type LocalVariable struct {
	name  string
	scope int
}

func NewCompiler() *Compiler {
	c := &Compiler{
		Chunk:   NewChunk(make([]Bytecode, 0), make([]Value, 0)),
		ip:      0,
		scope:   0,
		stack:   NewStack[LocalVariable](256),
		imports: make(map[string]Node),
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

func (c *Compiler) Compile(tree Node) error {
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
				err := c.Compile(n)
				if err != nil {
					return err
				}
			}
			c.add(InstructionFormList)
			c.addU16(uint16(len(l.items)))
		}

	case ReferenceNodeType:
		c.getVar(tree.(*ReferenceNode).name)

	case BinaryNodeType:
		err := c.compileBinary(tree.(*BinaryNode))
		if err != nil {
			return err
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
		c.descend()
		for _, n := range tree.(*BlockNode).statements {
			err := c.Compile(n)
			if err != nil {
				return err
			}
		}
		c.ascend()

	case ConditionalNodeType:
		n := tree.(*ConditionalNode)

		// the stack should have whether the condition was truthful
		err := c.Compile(n.condition)
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
		err = c.Compile(n.do)
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
			err := c.Compile(n.otherwise)
			if err != nil {
				return err
			}
			c.putU16(jumpOverElse, uint16(c.ip-jumpOverElse-2))
		}

	case LoopNodeType:
		n := tree.(*LoopNode)

		conditionPos := c.ip
		err := c.Compile(n.condition)
		if err != nil {
			return err
		}

		c.add(InstructionJumpFalse)
		jumpValuePos := c.ip
		c.advance(2)

		err = c.Compile(n.do)
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
			err := c.Compile(n.value)
			if err != nil {
				return err
			}
			c.add(InstructionPop)
		} else {
			err := c.setVar(n.name, n.value, n.declare)
			if err != nil {
				return err
			}
		}

	case CallNodeType:
		n := tree.(*CallNode)

		for _, arg := range n.args {
			err := c.Compile(arg)
			if err != nil {
				return err
			}
		}

		err := c.Compile(n.source)
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

		// keep track of main chunk
		mc := c.Chunk
		// and ip
		mip := c.ip

		// assign a new empty chunk
		c.Chunk = NewChunk(make([]Bytecode, 0), make([]Value, 0))
		// reset instruction pointer (ip)
		c.ip = 0

		for _, p := range n.params {
			c.registerVar(p)
		}

		err := c.Compile(n.logic)
		if err != nil {
			return err
		}

		if n.logic.Type() != BlockNodeType {
			c.stack.Pop()
		}

		mc.Constants[fi] = &FunctionValue{
			n.name,
			n.params,
			c.Chunk,
			nil,
		}

		// restore old chunk and ip
		c.Chunk = mc
		c.ip = mip

	case AccessNodeType:
		n := tree.(*AccessNode)
		err := c.Compile(n.source)
		if err != nil {
			return err
		}
		c.add(InstructionAccessProperty)
		c.addConstant(&StringValue{
			n.property,
		})

	case ImportNodeType:
		n := tree.(*ImportNode)

		t := c.resolveImport(n.path).(*BlockNode)

		for _, statement := range t.statements {
			err := c.Compile(statement)
			if err != nil {
				return err
			}
		}

	case ReturnNodeType:
		err := c.Compile(tree.(*ReturnNode).value)
		if err != nil {
			return err
		}
		c.add(InstructionReturn)

	case BreakpointNodeType:
		c.add(InstructionBreakpoint)
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

	err := c.Compile(binary.Left)
	if err != nil {
		return err
	}
	err = c.Compile(binary.Right)
	if err != nil {
		return err
	}

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
	case BinaryAnd:
		c.add(InstructionAnd)
	case BinaryOr:
		c.add(InstructionOr)
	}

	return nil
}

func (c *Compiler) getVar(name string) {
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

func (c *Compiler) setVar(name string, value Node, declare bool) error {
	err := c.Compile(value)
	if err != nil {
		return err
	}

	if declare {
		c.add(InstructionDeclareLocal)
		c.registerVar(name)
	} else {
		c.add(InstructionSetLocal)
	}

	c.addConstant(&StringValue{
		name,
	})

	return nil
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
	case BlockNodeType, ConditionalNodeType, LoopNodeType, AssignNodeType, CallNodeType, FunctionNodeType,
		ReturnNodeType, AccessNodeType, BreakpointNodeType, ImportNodeType, ReferenceNodeType:
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

	var v interface{}
	switch n.BinaryOperation {
	case BinaryAddition:
		v = l.(*NumberValue).float64 + r.(*NumberValue).float64
	case BinarySubtraction:
		v = l.(*NumberValue).float64 - r.(*NumberValue).float64
	case BinaryMultiplication:
		v = l.(*NumberValue).float64 * r.(*NumberValue).float64
	case BinaryDivision:
		v = l.(*NumberValue).float64 / r.(*NumberValue).float64
	case BinaryAnd:
		v = l.(*BoolValue).bool && r.(*BoolValue).bool
	case BinaryOr:
		v = l.(*BoolValue).bool && r.(*BoolValue).bool
	case BinaryEquality:
		v = l.Equals(r)
	case BinaryInequality:
		v = !l.Equals(r.(*BoolValue))
	case BinaryLess:
		v = l.(*NumberValue).float64 < r.(*NumberValue).float64
	case BinaryGreater:
		v = l.(*NumberValue).float64 > r.(*NumberValue).float64
	case BinaryLessEqual:
		v = l.(*NumberValue).float64 <= r.(*NumberValue).float64
	case BinaryGreaterEqual:
		v = l.(*NumberValue).float64 >= r.(*NumberValue).float64
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

func (c *Compiler) resolveImport(path string) Node {
	if chunk, ok := c.imports[path]; ok {
		return chunk
	}

	// find tree
	tree, err := c.resolver.Resolve(path)
	if err != nil {
		panic(err)
	}

	c.imports[path] = tree

	return tree
}

func (c *Compiler) SetImportsResolver(resolver ImportsResolver) {
	c.resolver = resolver
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
