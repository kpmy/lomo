package loom

import (
	"container/list"
	"fmt"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
	"lomo/ir/mods"
	"reflect"
	"sync"
)

type object interface {
	init(*ir.Variable)
	set(*value)
	get() *value
}

type context struct {
	owner   *mach
	objects map[string]object
}

type innie struct {
}

func (i *innie) init(v *ir.Variable) {

}

func (o *innie) get() *value { panic(0) }
func (o *innie) set(*value)  { panic(0) }

type outie struct {
}

func (o *outie) init(v *ir.Variable) {

}
func (o *outie) get() *value { panic(0) }
func (o *outie) set(*value)  { panic(0) }

type mem struct {
}

func (m *mem) init(v *ir.Variable) {

}
func (o *mem) get() *value { panic(0) }
func (o *mem) set(*value)  { panic(0) }

type direct struct {
	x *value
	c chan *value
}

func (d *direct) init(v *ir.Variable) {
	d.c = make(chan *value)
}

func (o *direct) get() *value {
	return <-o.c
}

func (o *direct) set(x *value) {
	assert.For(x != nil, 20)
	if o.x == nil {
		o.x = x
		go func() {
			for {
				o.c <- o.x
			}
		}()
	} else {
		halt.As(100, "already assigned")
	}
}

func obj(v *ir.Variable) (ret object) {
	switch v.Modifier {
	case mods.IN:
		ret = &innie{}
	case mods.REG:
		ret = &mem{}
	case mods.OUT:
		ret = &outie{}
	default: //var
		ret = &direct{}
	}
	ret.init(v)
	return
}
func (ctx *context) init(m *mach) {
	ctx.owner = m
	ctx.objects = make(map[string]object)
	for k, v := range m.base.Variables {
		ctx.objects[k] = obj(v)
	}
}

func (ctx *context) set(schema *ir.Variable, v *value) {
	fmt.Print(schema.Unit.Name, ".", schema.Name, " <- ", v, "\r")
	ctx.objects[schema.Name].set(v)
}

func (ctx *context) get(schema *ir.Variable) *value {
	fmt.Print(schema.Unit.Name, ".", schema.Name, " -> ")
	v := ctx.objects[schema.Name].get()
	fmt.Println(v)
	return v
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

func (ctx *context) expr(e ir.Expression) *value {
	stack := &exprStack{}
	stack.init()
	var expr func(ir.Expression)
	expr = func(_e ir.Expression) {
		switch e := _e.(type) {
		case *ir.ConstExpr:
			stack.push(&value{typ: e.Type, val: e.Value})
		case *ir.SelectExpr:
			stack.push(ctx.get(e.Var))
		default:
			halt.As(100, reflect.TypeOf(e))
		}
	}
	expr(e)
	return stack.pop()
}

func (ctx *context) process() (stop bool) {
	rg := &sync.WaitGroup{}
	starter := make(chan bool)
	count := 0
	stop = true
	for v, r := range ctx.owner.base.Rules {
		count++
		go func(v *ir.Variable, _r ir.Rule) {
			<-starter
			switch r := _r.(type) {
			case *ir.ConditionalRule:
				done := false
				for _, _ = range r.Blocks {
					panic(0)
				}
				if !done {
					ctx.set(v, ctx.expr(r.Default))
				}
			default:
				halt.As(100, reflect.TypeOf(r))
			}
			rg.Done()
		}(ctx.owner.base.Variables[v], r)
	}
	rg.Add(count)
	for i := 0; i < count; i++ {
		starter <- true
	}
	rg.Wait()
	return
}
