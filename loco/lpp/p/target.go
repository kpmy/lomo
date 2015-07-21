package p

import (
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/loco/lpp"
)

type variable struct {
	modifier int
}

type target struct {
	name    string
	unit    *ir.Unit
	resolve lpp.ForeignResolver
	marker  Marker
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

func (t *target) foreign(name string, obj *ir.Variable) {
	r := make(map[string]ir.Rule)
	t.unit.ForeignRules[name] = r
}

func (t *target) assign(unit, name string, expr *exprBuilder) {
	local := func(v *ir.Variable) {
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
	}
	foreign := func(v *ir.Variable) {
		if _, ok := t.unit.ForeignRules[unit]; ok {
			if _, ok := t.unit.ForeignRules[unit][name]; !ok {
				r := &ir.ConditionalRule{}
				r.Default = expr.final()
				t.unit.ForeignRules[unit][name] = r
			} else {
				t.marker.Mark("already assigned")
			}
		} else {
			t.marker.Mark("wrong foreign")
		}
	}
	if unit == t.unit.Name {
		if v, ok := t.unit.Variables[name]; ok {
			local(v)
		} else {
			t.marker.Mark("identifier `", name, "` not found")
		}
	} else if u := t.resolve(t.unit.Variables[unit].Type.Foreign.Name()); u != nil {
		if v, ok := u.Variables()[name]; ok {
			foreign(v)
		} else {
			t.marker.Mark("identifier `", unit, ".", name, "` not found")
		}
	} else {
		t.marker.Mark("foreign `", unit, "` not resolved")
	}

}
