package ast

import "github.com/shopspring/decimal"

type Node interface {
	node()
}

type Expression interface {
	Node
	expression()
}

type BinaryExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

type UnaryExpression struct {
	Operator string
	Expr     Expression
}

type FunctionCall struct {
	Name string
	Args []Expression
}

type Identifier struct {
	Path []string
}

type Literal struct {
	Value decimal.Decimal
}

type BoolLiteral struct {
	Value bool
}

func (BinaryExpression) node()       {}
func (BinaryExpression) expression() {}

func (UnaryExpression) node()       {}
func (UnaryExpression) expression() {}

func (FunctionCall) node()       {}
func (FunctionCall) expression() {}

func (Identifier) node()       {}
func (Identifier) expression() {}

func (Literal) node()       {}
func (Literal) expression() {}

func (BoolLiteral) node()       {}
func (BoolLiteral) expression() {}
