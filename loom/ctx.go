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
	x *value
	c chan *value
	v *ir.Variable
}

func (i *innie) init(v *ir.Variable) {
	i.c = make(chan *value)
	i.v = v
}

func (i *innie) String() string {
	return i.v.Unit.Name + "." + i.v.Name
}

func (o *innie) get() *value { return <-o.c }
func (o *innie) set(x *value) {
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
	v *ir.Variable
}

func (d *direct) String() string {
	return d.v.Unit.Name + "." + d.v.Name
}

func (d *direct) init(v *ir.Variable) {
	d.c = make(chan *value)
	d.v = v
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
func (ctx *context) init(m *mach) (ret map[string]func()) {
	ctx.owner = m
	ctx.objects = make(map[string]object)
	ret = make(map[string]func())
	for k, v := range m.base.Variables {
		if v.Type.Basic {
			ctx.objects[k] = obj(v)
		} else {
			f := m.loader(v.Type.Foreign.Name())
			v.Type.Foreign = ir.NewForeign(f) //определения модулей не прогружаются при загрузке единичного модуля
			ret[v.Name] = m.prepare(v)
		}
	}
	return
}

func (ctx *context) set0(o object, v *value) {
	fmt.Println("set", o)
	o.set(v)
}

func (ctx *context) get0(o object) *value {
	defer fmt.Println("get", o)
	return o.get()
}

func (ctx *context) foreign(schema *ir.Variable, id string) (ret object) {
	assert.For(schema != nil, 20)
	fmt.Println(ctx.owner.base.Name, "foreign", schema.Unit.Name, schema.Name, id, imp(schema))
	f := ctx.owner.imps[imp(schema)].ctx
	ret = f.objects[id]
	assert.For(ret != nil, 60)
	fmt.Println("found")
	return
}

func (ctx *context) local(schema *ir.Variable) (ret object) {
	assert.For(schema != nil, 20)
	fmt.Println(schema.Unit.Name, schema.Name)
	ret = ctx.objects[schema.Name]
	return
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
			stack.push(ctx.get0(ctx.local(e.Var)))
		default:
			halt.As(100, reflect.TypeOf(e))
		}
	}
	expr(e)
	return stack.pop()
}

func (ctx *context) rule(o object, _r ir.Rule) {
	switch r := _r.(type) {
	case *ir.ConditionalRule:
		done := false
		for _, _ = range r.Blocks {
			panic(0)
		}
		if !done {
			ctx.set0(o, ctx.expr(r.Default))
		}
	default:
		halt.As(100, reflect.TypeOf(r))
	}
}

func (ctx *context) process(deps map[string]func()) (stop bool) {
	rg := &sync.WaitGroup{}
	starter := make(chan bool)
	count := 0
	stop = true
	for v, r := range ctx.owner.base.Rules {
		count++
		go func(v *ir.Variable, _r ir.Rule) {
			<-starter
			ctx.rule(ctx.local(v), _r)
			rg.Done()
		}(ctx.owner.base.Variables[v], r)
	}
	for v, r := range ctx.owner.base.ForeignRules {
		o := ctx.owner.base.Variables[v]
		for _, v := range o.Type.Foreign.Variables() {
			if rr := r[v.Name]; rr != nil {
				if pre := deps[o.Name]; pre != nil {
					pre()
				}
				count++
				go func(f, v *ir.Variable, _r ir.Rule) {
					<-starter
					ctx.rule(ctx.foreign(f, v.Name), _r)
					rg.Done()
				}(o, v, rr)
			}
		}
	}
	rg.Add(count)
	for i := 0; i < count; i++ {
		starter <- true
	}
	rg.Wait()
	return
}
