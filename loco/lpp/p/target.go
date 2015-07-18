package p

import (
	"lomo/ir"
)

type variable struct {
	modifier int
}

type target struct {
	name   string
	unit   *ir.Unit
	marker Marker
}

func (t *target) init(mod string) {
	t.name = mod
	t.unit = ir.NewUnit(mod)
}

func (t *target) obj(name string, obj *ir.Variable) {
	if _, ok := t.unit.Variables[name]; !ok {
		t.unit.Variables[name] = obj
	} else {
		t.marker.Mark("identifier `", name, "`  already exists")
	}
}
