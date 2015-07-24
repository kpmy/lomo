package loom

import (
	"container/list"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
)

type object interface {
	init(*ir.Variable, chan bool, ...interface{})
	set(*value)
	get() *value
	schema() *ir.Variable
	control() chan bool
}

type exprStack struct {
	vl *list.List
}

func (s *exprStack) init() {
	s.vl = list.New()
}

func (s *exprStack) push(v *value) {
	assert.For(v != nil, 20)
	_, fake := v.val.(*value)
	assert.For(!fake, 21)
	s.vl.PushFront(v)
}

func (s *exprStack) pop() (ret *value) {
	if s.vl.Len() > 0 {
		el := s.vl.Front()
		ret = s.vl.Remove(el).(*value)
	} else {
		halt.As(100, "pop on empty stack")
	}
	return
}
