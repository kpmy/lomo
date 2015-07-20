package p

import (
	"lomo/ir"
	"lomo/ir/mods"
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

func (t *target) assign(name string, expr *exprBuilder) {
	if v, ok := t.unit.Variables[name]; ok {
		if v.Modifier == mods.IN {
			t.marker.Mark("variable is read-only")
		}
		if _, ok := t.unit.Rules[name]; !ok {
			r := &ir.ConditionalRule{}
			r.Default = expr.final()
			t.unit.Rules[name] = r
		} else {
			t.marker.Mark("already assigned")
		}
	} else {
		t.marker.Mark("identifier `", name, "` not found")
	}
}
