package core

import (
	"fmt"
	"strconv"
	"strings"
)

type NodeType int

type Node interface {
	Type() NodeType
	String() string

	Bounds() (Pos, Pos)
}

const (
	StringNodeType NodeType = iota
	NumberNodeType
	ReferenceNodeType
	BooleanNodeType
	NilNodeType
	ListNodeType
	BinaryNodeType
	UnaryNodeType
	BlockNodeType
	ConditionalNodeType
	LoopNodeType
	AssignNodeType
	CallNodeType
	FunctionNodeType
	ReturnNodeType
	AccessNodeType
	BreakpointNodeType
)

func (n NodeType) String() string {
	switch n {
	case StringNodeType:
		return "String"
	case NumberNodeType:
		return "Number"
	case ReferenceNodeType:
		return "Reference"
	case BinaryNodeType:
		return "Binary"
	case BooleanNodeType:
		return "Boolean"
	case NilNodeType:
		return "Nil"
	case BlockNodeType:
		return "Block"
	case ConditionalNodeType:
		return "Conditional"
	case LoopNodeType:
		return "Loop"
	case AssignNodeType:
		return "Assign"
	case CallNodeType:
		return "Call"
	case FunctionNodeType:
		return "Function"
	case ReturnNodeType:
		return "Return"
	case ListNodeType:
		return "List"
	case AccessNodeType:
		return "Access"
	case BreakpointNodeType:
		return "Breakpoint"
	case UnaryNodeType:
		return "Unary"
	}
	return "Invalid Node Type"
}

// ReferenceNode a reference to a variable on the stack
type ReferenceNode struct {
	name string

	start Pos
	end   Pos
}

func (n ReferenceNode) Type() NodeType {
	return ReferenceNodeType
}

func (n ReferenceNode) String() string {
	return n.name
}

func (n ReferenceNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// StringNode string/text values
type StringNode struct {
	value  string
	quoted string

	start Pos
	end   Pos
}

func (n StringNode) Type() NodeType {
	return StringNodeType
}

func (n StringNode) String() string {
	return n.quoted
}

func (n StringNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

type NumberNode struct {
	value float64

	start Pos
	end   Pos
}

func (n NumberNode) Type() NodeType {
	return NumberNodeType
}

func (n NumberNode) String() string {
	return strconv.FormatFloat(n.value, 'g', -1, NumberSize)
}

func (n NumberNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// ListNode a list or sequence of values (items)
type ListNode struct {
	items   []Node
	content TypeSignature

	start Pos
	end   Pos
}

func (n ListNode) Type() NodeType {
	return ListNodeType
}

func (n ListNode) String() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, item := range n.items {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(item.String())
	}
	sb.WriteString("]")
	return sb.String()
}

func (n ListNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

type AccessNode struct {
	source   Node
	property string

	start Pos
	end   Pos
}

func (n AccessNode) Type() NodeType {
	return AccessNodeType
}

func (n AccessNode) String() string {
	return fmt.Sprintf("(%s from %s)", n.property, n.source)
}

func (n AccessNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

type BinaryOperation uint

func (n BinaryOperation) String() string {
	switch n {
	case BinaryAddition:
		return "add"
	case BinarySubtraction:
		return "subtract"
	case BinaryMultiplication:
		return "multiply"
	case BinaryDivision:
		return "divide"
	case BinaryEquality:
		return "equality"
	case BinaryInequality:
		return "inequality"
	case BinaryLess:
		return "less"
	case BinaryGreater:
		return "greater"
	case BinaryLessEqual:
		return "less or equal"
	case BinaryGreaterEqual:
		return "greater or equal"
	case BinaryAnd:
		return "and"
	case BinaryOr:
		return "or"
	}

	return "undefined arithmetic operation"
}

const (
	BinaryAddition BinaryOperation = iota
	BinarySubtraction
	BinaryMultiplication
	BinaryDivision

	BinaryAnd
	BinaryOr

	// Comparison
	BinaryEquality
	BinaryInequality
	BinaryLess
	BinaryGreater
	BinaryLessEqual
	BinaryGreaterEqual
)

func (n BinaryOperation) Symbol() string {
	switch n {
	case BinaryAddition:
		return "+"
	case BinarySubtraction:
		return "-"
	case BinaryMultiplication:
		return "*"
	case BinaryDivision:
		return "/"
	case BinaryEquality:
		return "=="
	case BinaryInequality:
		return "!="
	case BinaryLess:
		return "<"
	case BinaryGreater:
		return ">"
	case BinaryAnd:
		return "&&"
	case BinaryOr:
		return "||"
	case BinaryLessEqual:
		return "<="
	case BinaryGreaterEqual:
		return ">="
	}

	panic("unsupported binary operation to symbol conversion for " + n.String())
}

// BinaryNode All operations which take 2 variables
type BinaryNode struct {
	BinaryOperation
	Left  Node
	Right Node

	start Pos
	end   Pos
}

func (n BinaryNode) Type() NodeType {
	return BinaryNodeType
}

func (n BinaryNode) String() string {
	return fmt.Sprintf("%s %s %s", n.Left.String(), n.BinaryOperation.String(), n.Right.String())
}

func (n BinaryNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

type UnaryOperation int

const (
	UnaryNegate UnaryOperation = iota
	UnaryNot
)

func (op UnaryOperation) String() string {
	switch op {
	case UnaryNegate:
		return "negate"
	case UnaryNot:
		return "not"
	}

	panic("unimplemented unary operation to string conversion")
}

func (op UnaryOperation) Symbol() string {
	switch op {
	case UnaryNegate:
		return "-"
	case UnaryNot:
		return "!"
	}

	panic("unimplemented unary operation to symbol conversion")
}

type UnaryNode struct {
	UnaryOperation
	value Node

	start Pos
	end   Pos
}

func (n UnaryNode) Type() NodeType {
	return UnaryNodeType
}

func (n UnaryNode) String() string {
	return fmt.Sprintf("%s %s", n.UnaryOperation.String(), n.value.String())
}

func (n UnaryNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// BooleanNode boolean value
type BooleanNode struct {
	value bool

	start Pos
	end   Pos
}

func (n BooleanNode) Type() NodeType {
	return BooleanNodeType
}

func (n BooleanNode) String() string {
	return strconv.FormatBool(n.value)
}

func (n BooleanNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// NilNode nil value
type NilNode struct {
	start Pos
	end   Pos
}

func (n NilNode) Type() NodeType {
	return NilNodeType
}

func (n NilNode) String() string {
	return "nil"
}

func (n NilNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// BlockNode block node with statements
type BlockNode struct {
	statements []Node

	start Pos
	end   Pos
}

func (n BlockNode) Type() NodeType {
	return BlockNodeType
}

func (n BlockNode) String() string {
	builder := strings.Builder{}

	for _, stmt := range n.statements {
		builder.WriteString(stmt.String())
		builder.WriteString("\n")
	}

	return builder.String()
}

func (n BlockNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// ConditionalNode conditionals (if statements)
type ConditionalNode struct {
	condition Node
	do        Node
	otherwise Node

	start Pos
	end   Pos
}

func (n ConditionalNode) Type() NodeType {
	return ConditionalNodeType
}

func (n ConditionalNode) String() string {
	if n.otherwise == nil {
		return fmt.Sprintf("if %s then %s", n.condition.String(), n.do.String())
	}

	return fmt.Sprintf("if %s then %s otheriwise %s", n.condition.String(), n.do.String(), n.otherwise.String())
}

func (n ConditionalNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// LoopNode Loops (for/while)
type LoopNode struct {
	condition Node
	do        Node

	start Pos
	end   Pos
}

func (n LoopNode) Type() NodeType {
	return LoopNodeType
}

func (n LoopNode) String() string {
	return fmt.Sprintf("while %s loop %s", n.condition.String(), n.do.String())
}

func (n LoopNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// AssignNode assignment
type AssignNode struct {
	name    string
	value   Node
	declare bool

	start Pos
	end   Pos
}

func (n AssignNode) Type() NodeType {
	return AssignNodeType
}

func (n AssignNode) String() string {
	return fmt.Sprintf("set %s to %s", n.name, n.value)
}

func (n AssignNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// CallNode function call
type CallNode struct {
	source Node
	args   []Node
	keep   bool

	start Pos
	end   Pos
}

func (n CallNode) Type() NodeType {
	return CallNodeType
}

func (n CallNode) String() string {
	return fmt.Sprintf("call %s with args (%s)", n.source.String(), n.args)
}

func (n CallNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// FunctionNode definition of function
type FunctionNode struct {
	name       string
	parameters []FunctionParameter
	yield      TypeSignature
	logic      Node

	start Pos
	end   Pos
}

type FunctionParameter struct {
	Name      string
	Signature TypeSignature
}

func (n FunctionNode) Type() NodeType {
	return FunctionNodeType
}

func (n FunctionNode) String() string {
	return fmt.Sprintf("definition of %s, do %s", n.name, n.logic.String())
}

func (n FunctionNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

// ReturnNode return a value out of this context
type ReturnNode struct {
	value Node

	start Pos
	end   Pos
}

func (n ReturnNode) Type() NodeType {
	return ReturnNodeType
}

func (n ReturnNode) String() string {
	return fmt.Sprintf("return %s", n.value)
}

func (n ReturnNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}

type BreakpointNode struct {
	start Pos
	end   Pos
}

func (n BreakpointNode) Type() NodeType {
	return BreakpointNodeType
}

func (n BreakpointNode) String() string {
	return "breakpoint"
}

func (n BreakpointNode) Bounds() (Pos, Pos) {
	return n.start, n.end
}
