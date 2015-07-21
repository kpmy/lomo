package ir

import (
	"lomo/ir/mods"
	"lomo/ir/types"
)

type ForeignType interface {
	Name() string
	Variables() map[string]*Variable
}

type foreignType struct {
	name      string
	variables map[string]*Variable
}

func (f *foreignType) Name() string { return f.name }

func (f *foreignType) Variables() map[string]*Variable { return f.variables }

func NewForeign(u *Unit) ForeignType {
	ret := &foreignType{name: u.Name}
	ret.variables = make(map[string]*Variable)
	for k, v := range u.Variables {
		if v.Modifier == mods.IN || v.Modifier == mods.OUT {
			ret.variables[k] = v
		}
	}
	return ret
}

type BuiltinType struct {
	Code types.Type
}

type Type struct {
	Basic   bool
	Foreign ForeignType
	Builtin *BuiltinType
}

type Variable struct {
	Unit     *Unit
	Name     string
	Type     Type
	Modifier mods.Modifier
}

type Unit struct {
	Name         string
	Variables    map[string]*Variable
	Rules        map[string]Rule
	ForeignRules map[string]map[string]Rule
}

func NewUnit(name string) *Unit {
	u := &Unit{Name: name}
	u.Variables = make(map[string]*Variable)
	u.Rules = make(map[string]Rule)
	u.ForeignRules = make(map[string]map[string]Rule)
	return u
}
