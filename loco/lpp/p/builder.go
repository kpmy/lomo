package p

import (
	"container/list"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
	"lomo/ir/mods"
	"reflect"
)

type exprBuilder struct {
	tgt    *target
	marker Marker
	stack  *list.List
}

func (b *exprBuilder) init() {
	if b.stack == nil {
		b.stack = list.New()
	}
}

func (b *exprBuilder) push(_e ir.Expression) {
	b.init()
	switch e := _e.(type) {
	case *ir.ConstExpr, *ir.SelectExpr:
		b.stack.PushFront(e)
	case *ir.Dyadic:
		e.Right = b.pop()
		e.Left = b.pop()
		b.stack.PushFront(e)
	case *ir.Ternary:
		e.Else = b.pop()
		e.Then = b.pop()
		e.If = b.pop()
		b.stack.PushFront(e)
	default:
		halt.As(100, reflect.TypeOf(e))
	}
}

func (b *exprBuilder) pop() (ret ir.Expression) {
	b.init()
	if b.stack.Len() > 0 {
		ret = b.stack.Remove(b.stack.Front()).(ir.Expression)
	} else {
		halt.As(100, "pop on empty stack")
	}
	return
}

func (b *exprBuilder) final() (ret ir.Expression) {
	b.init()
	ret = b.pop()
	assert.For(ret != nil && b.stack.Len() == 0, 60)
	return
}

type selectBuilder struct {
	tgt    *target
	marker Marker
}

func (b *selectBuilder) foreign(unit, id string) (sel *ir.SelectExpr) {
	if u := b.tgt.resolve(b.tgt.unit.Variables[unit].Type.Foreign.Name()); u != nil {
		if v, ok := u.Variables()[id]; ok {
			if v.Modifier != mods.OUT {
				b.marker.Mark("not an OUT var")
			}
			sel = &ir.SelectExpr{Var: b.tgt.unit.Variables[unit], Foreign: v}
		} else {
			b.marker.Mark("foreign ", unit, ".", id, " not found")
		}
	} else {
		b.marker.Mark("foreign `", unit, "` not resolved")
	}
	return
}
