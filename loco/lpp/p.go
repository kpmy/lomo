package lpp

import (
	"lomo/ir"
	"lomo/loco/lss"
)

type ForeignResolver func(name string) *ir.ForeignType

type UnitParser interface {
	Unit() (*ir.Unit, error)
}

var ConnectToUnit func(lss.Scanner, ForeignResolver) UnitParser
