package ir

import (
	"lomo/ir/mods"
	"lomo/ir/types"
)

type ForeignType struct {
	Name string
}

func NewForeign(u *Unit) *ForeignType {
	return &ForeignType{Name: u.Name}
}

type BuiltinType struct {
	Code types.Type
}

type Type struct {
	Basic   bool
	Foreign *ForeignType
	Builtin *BuiltinType
}

type Variable struct {
	Unit     *Unit
	Name     string
	Type     *Type
	Modifier mods.Modifier
}

type Unit struct {
	Name      string
	Variables map[string]*Variable
}

func NewUnit(name string) *Unit {
	u := &Unit{Name: name}
	u.Variables = make(map[string]*Variable)
	return u
}
