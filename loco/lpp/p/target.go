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
	name     string
	unit     *ir.Unit
	_resolve lpp.ForeignResolver
	marker   Marker
	cache    map[string]ir.ForeignType
}

func (t *target) resolve(name string) (ret ir.ForeignType) {
	var cyclic func(ir.ForeignType) bool
	cyclic = func(ii ir.ForeignType) (ok bool) {
		if ok = ii.Name() == t.unit.Name; !ok {
			for i := 0; i < len(ii.Imports()) && !ok; i++ {
				ok = ii.Imports()[i] == t.unit.Name
			}
		}
		return
	}
	if ret = t.cache[name]; ret == nil {
		if imp := t._resolve(name); imp != nil {
			if !cyclic(imp) {
				ret = imp
				t.cache[name] = ret
			} else {
				t.marker.Mark("cyclic import")
			}
		}

	}
	return
}

func (t *target) init(mod string) {
	t.name = mod
	t.unit = ir.NewUnit(mod)
	t.cache = make(map[string]ir.ForeignType)
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
			r := &ir.AssignRule{}
			r.Expr = expr.final()
			t.unit.Rules[name] = r
		} else {
			t.marker.Mark("already assigned")
		}
	}
	foreign := func(v *ir.Variable) {
		if _, ok := t.unit.ForeignRules[unit]; ok {
			if _, ok := t.unit.ForeignRules[unit][name]; !ok {
				r := &ir.AssignRule{}
				r.Expr = expr.final()
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
