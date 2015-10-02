package ir

import (
	"github.com/kpmy/lomo/ir/mods"
	"github.com/kpmy/lomo/ir/ops"
	"github.com/kpmy/lomo/ir/types"
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

type WrappedExpression interface {
	Expression
	Process() Expression
}

type TypeTest struct {
	Typ     Type
	Operand Expression
}

func (t *TypeTest) Print() string { return "type test expr" }

type SetExpr struct {
	Expr []Expression
}

func (e *SetExpr) Print() string { return "set expr" }

type ListExpr struct {
	Expr []Expression
}

func (e *ListExpr) Print() string { return "list expr" }

type MapExpr struct {
	Key, Value []Expression
}

func (e *MapExpr) Print() string { return "map expr" }
