package ir

import (
	"github.com/kpmy/lomo/ir/mods"
	"github.com/kpmy/lomo/ir/types"
)

type ForeignType interface {
	Name() string
	Variables() map[string]*Variable
	Imports() []string
	Infix() []*Variable
}

type foreignType struct {
	name      string
	variables map[string]*Variable
	imps      []string
	inf       []*Variable
}

func (f *foreignType) Name() string { return f.name }

func (f *foreignType) Variables() map[string]*Variable { return f.variables }

func (f *foreignType) Imports() []string { return f.imps }

func (f *foreignType) Infix() []*Variable { return f.inf }

func NewForeign(u *Unit) ForeignType {
	ret := &foreignType{name: u.Name}
	ret.variables = make(map[string]*Variable)
	imps := make(map[string]*Variable)
	for k, v := range u.Variables {
		if v.Modifier == mods.IN || v.Modifier == mods.OUT {
			ret.variables[k] = v
		}
		if !v.Type.Basic {
			imps[v.Type.Foreign.Name()] = v
		}
	}
	for k, _ := range imps {
		ret.imps = append(ret.imps, k)
	}
	for _, v := range u.Infix {
		ret.inf = append(ret.inf, v)
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

type Const struct {
	Unit     *Unit
	Name     string
	Expr     Expression
	Modifier mods.Modifier
}

type Unit struct {
	Name         string
	Variables    map[string]*Variable
	Const        map[string]*Const
	Rules        map[string]Rule
	ForeignRules map[string]map[string]Rule
	Infix        []*Variable
	Pre, Post    []Expression
}

func NewUnit(name string) *Unit {
	u := &Unit{Name: name}
	u.Variables = make(map[string]*Variable)
	u.Rules = make(map[string]Rule)
	u.ForeignRules = make(map[string]map[string]Rule)
	u.Const = make(map[string]*Const)
	return u
}
