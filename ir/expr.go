package ir

import (
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
	Var *Variable
}

func (s *SelectExpr) Print() string { return "select expr" }
