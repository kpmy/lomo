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
	fwd    []func()
}

func (b *exprBuilder) Print() string { return "expr builder" }

func (b *exprBuilder) init() {
	if b.stack == nil {
		b.stack = list.New()
	}
}

func (b *exprBuilder) push(_e ir.Expression) {
	b.init()
	switch e := _e.(type) {
	case *exprBuilder:
		b.stack.PushFront(e.final())
	case *ir.ConstExpr, *ir.SelectExpr, *ir.ListExpr, *ir.MapExpr, *ir.SetExpr:
		b.stack.PushFront(e)
	case *ir.Monadic:
		e.Expr = b.pop()
		b.stack.PushFront(e)
	case *ir.TypeTest:
		e.Operand = b.pop()
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
	case *ir.InfixExpr:
		for i := 1; i < len(e.Unit.Infix()); i++ {
			e.Args = append(e.Args, b.pop())
		}
		b.stack.PushFront(e)
	default:
		halt.As(100, reflect.TypeOf(e))
	}
	//fmt.Println("push", _e)
}

func (b *exprBuilder) pop() (ret ir.Expression) {
	b.init()
	if b.stack.Len() > 0 {
		ret = b.stack.Remove(b.stack.Front()).(ir.Expression)
	} else {
		halt.As(100, "pop on empty stack")
	}
	//fmt.Println("pop", ret)
	return
}

func (b *exprBuilder) forward(f func()) bool {
	if b.fwd != nil {
		b.fwd = append(b.fwd, f)
	}
	return b.fwd != nil
}

func (b *exprBuilder) final() (ret ir.Expression) {
	b.init()
	ret = b.pop()
	assert.For(ret != nil && b.stack.Len() == 0, 60, b.stack.Len())
	return
}

type selectBuilder struct {
	tgt    *target
	marker Marker
	inner  *ir.SelectExpr
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

func (b *selectBuilder) list(el []ir.Expression) {
	b.inner = &ir.SelectExpr{}
	b.inner.Inner = mods.LIST
	b.inner.ExprList = el
}

func (b *selectBuilder) upto(e ...ir.Expression) {
	b.inner = &ir.SelectExpr{}
	b.inner.Inner = mods.RANGE
	b.inner.ExprList = append(b.inner.ExprList, e...)
}

func (b *selectBuilder) deref() {
	b.inner = &ir.SelectExpr{Inner: mods.DEREF}
}

func (b *selectBuilder) merge(s *ir.SelectExpr) (sel *ir.SelectExpr) {
	sel = s
	if b.inner != nil {
		sel.Inner = b.inner.Inner
		sel.ExprList = b.inner.ExprList
	}
	return
}
