package lpp

import (
	"lomo/loco/lss"
)

type UnitParser interface {
	Unit() error
}

var ConnectToUnit func(s lss.Scanner) UnitParser
