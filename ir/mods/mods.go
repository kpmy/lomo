package mods

import (
	"lomo/loco/lss"
)

type Modifier int

const (
	NONE Modifier = iota
	IN
	OUT
	REG
	wrong
)

var ModMap map[string]Modifier
var ModSym map[Modifier]lss.Symbol
var SymMod map[lss.Symbol]Modifier

func (m Modifier) String() string {
	return ModSym[m].String()
}

func init() {
	ModSym = map[Modifier]lss.Symbol{IN: lss.Minus, OUT: lss.Plus, REG: lss.Reg, NONE: lss.None}
	SymMod = make(map[lss.Symbol]Modifier)
	for k, v := range ModSym {
		SymMod[v] = k
	}
	ModMap = make(map[string]Modifier)
	for i := int(NONE); i < int(wrong); i++ {
		m := Modifier(i)
		ModMap[m.String()] = m
	}
}
