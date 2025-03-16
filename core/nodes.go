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
}

const (
	StringNodeType NodeType = iota
	NumberNodeType
	ReferenceNodeType
	BooleanNodeType
	NilNodeType
	ListNodeType
	BinaryNodeType
	BlockNodeType
	ConditionalNodeType
	LoopNodeType
	AssignNodeType
	CallNodeType
	FunctionNodeType
	ReturnNodeType
	AccessNodeType
	ImportNodeType
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
	case ImportNodeType:
		return "Import"
	}
	return "Invalid Node Type"
}

// ReferenceNode a reference to a variable on the stack
type ReferenceNode struct {
	name string
}

func (n ReferenceNode) Type() NodeType {
	return ReferenceNodeType
}

func (n ReferenceNode) String() string {
	return n.name
}

// StringNode string/text values
type StringNode struct {
	value  string
	quoted string
}

func (n StringNode) Type() NodeType {
	return StringNodeType
}

func (n StringNode) String() string {
	return n.quoted
}

type NumberNode struct {
	value float64
}

func (n NumberNode) Type() NodeType {
	return NumberNodeType
}

func (n NumberNode) String() string {
	return strconv.FormatFloat(n.value, 'g', -1, NumberSize)
}

// ListNode a list or sequence of values (items)
type ListNode struct {
	items []Node
}

func (n ListNode) Type() NodeType {
	return ListNodeType
}

func (n ListNode) String() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, item := range n.items {
		sb.WriteString(item.String())
		if i > 0 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString("]")
	return sb.String()
}

type AccessNode struct {
	source   Node
	property string
}

func (n AccessNode) Type() NodeType {
	return AccessNodeType
}

func (n AccessNode) String() string {
	return fmt.Sprintf("(%s from %s)", n.property, n.source)
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

// BinaryNode All operations which take 2 variables
type BinaryNode struct {
	BinaryOperation
	Left  Node
	Right Node
}

func (n BinaryNode) Type() NodeType {
	return BinaryNodeType
}

func (n BinaryNode) String() string {
	return fmt.Sprintf("%s %s %s", n.Left.String(), n.BinaryOperation.String(), n.Right.String())
}

// BooleanNode boolean value
type BooleanNode struct {
	value bool
}

func (n BooleanNode) Type() NodeType {
	return BooleanNodeType
}

func (n BooleanNode) String() string {
	return strconv.FormatBool(n.value)
}

// NilNode nil value
type NilNode struct{}

func (n NilNode) Type() NodeType {
	return NilNodeType
}

func (n NilNode) String() string {
	return "nil"
}

// BlockNode block node with statements
type BlockNode struct {
	statements []Node
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

type ImportNode struct {
	path string
}

func (n ImportNode) Type() NodeType {
	return ImportNodeType
}

func (n ImportNode) String() string {
	return fmt.Sprintf("import %s", n.path)
}

// ConditionalNode conditionals (if statements)
type ConditionalNode struct {
	condition Node
	do        Node
	otherwise Node
}

func (n ConditionalNode) Type() NodeType {
	return ConditionalNodeType
}

func (n ConditionalNode) String() string {
	return fmt.Sprintf("if %s then %s otheriwise %s", n.condition.String(), n.do.String(), n.otherwise.String())
}

// LoopNode Loops (for/while)
type LoopNode struct {
	condition Node
	do        Node
}

func (n LoopNode) Type() NodeType {
	return LoopNodeType
}

func (n LoopNode) String() string {
	return fmt.Sprintf("while %s loop %s", n.condition.String(), n.do.String())
}

// AssignNode assignment
type AssignNode struct {
	name    string
	value   Node
	declare bool
}

func (n AssignNode) Type() NodeType {
	return AssignNodeType
}

func (n AssignNode) String() string {
	return fmt.Sprintf("set %s to %s", n.name, n.value)
}

// CallNode function call
type CallNode struct {
	source Node
	args   []Node
	keep   bool
}

func (n CallNode) Type() NodeType {
	return CallNodeType
}

func (n CallNode) String() string {
	return fmt.Sprintf("call %s with args (%s)", n.source.String(), n.args)
}

// FunctionNode definition of function
type FunctionNode struct {
	name   string
	params []string
	logic  Node
}

func (n FunctionNode) Type() NodeType {
	return FunctionNodeType
}

func (n FunctionNode) String() string {
	return fmt.Sprintf("definition of %s, do %s", n.name, n.logic.String())
}

// ReturnNode return a value out of this context
type ReturnNode struct {
	value Node
}

func (n ReturnNode) Type() NodeType {
	return ReturnNodeType
}

func (n ReturnNode) String() string {
	return fmt.Sprintf("return %s", n.value)
}

type BreakpointNode struct{}

func (n BreakpointNode) Type() NodeType {
	return BreakpointNodeType
}

func (n BreakpointNode) String() string {
	return "breakpoint"
}
