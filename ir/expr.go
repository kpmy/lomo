package ir

import (
	"lomo/ir/mods"
	"lomo/ir/ops"
	"lomo/ir/types"
)

type Expression interface {
	Print() string
}

type ConstExpr struct {
	Type  types.Type
	Value interface{}
}

func (c *ConstExpr) Print() string { return "const expr" }

type SelectExpr struct {
	Const    *Const
	Var      *Variable
	Foreign  *Variable
	Inner    mods.Modifier
	ExprList []Expression
}

func (s *SelectExpr) Print() string { return "select expr" }

type Monadic struct {
	Op   ops.Operation
	Expr Expression
}

func (m *Monadic) Print() string { return "monadic op" }

type Dyadic struct {
	Op          ops.Operation
	Left, Right Expression
}

func (d *Dyadic) Print() string { return "dyadic op" }

type Ternary struct {
	If, Then, Else Expression
}

func (t *Ternary) Print() string { return "ternary op" }

type AtomExpr struct {
	Value string
}

func (a *AtomExpr) Print() string { return "atom expr" }

type InfixExpr struct {
	Unit ForeignType
	Args []Expression
}

func (i *InfixExpr) Print() string { return "infix expr" }
