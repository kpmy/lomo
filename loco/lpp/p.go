package lpp

import (
	"github.com/kpmy/lomo/ir"
	"github.com/kpmy/lomo/loco/lss"
)

type ForeignResolver func(name string) ir.ForeignType

type UnitParser interface {
	Unit() (*ir.Unit, error)
}

var ConnectToUnit func(lss.Scanner, ForeignResolver) UnitParser

var Std map[string]ir.ForeignType

func init() {
	Std = make(map[string]ir.ForeignType)
}
