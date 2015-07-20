package p

import (
	"container/list"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
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
	case *ir.ConstExpr:
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
