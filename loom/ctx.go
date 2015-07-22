package loom

import (
	"container/list"
	"fmt"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/fn"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
	"lomo/ir/mods"
	"reflect"
	"sync"
)

type object interface {
	init(*ir.Variable, chan bool)
	set(*value)
	get() *value
}

type context struct {
	owner   *mach
	objects map[string]object
	ctrl    chan bool
}

type innie struct {
	x    *value
	c    chan *value
	v    *ir.Variable
	ctrl chan bool
}

func (i *innie) init(v *ir.Variable, ctrl chan bool) {
	i.c = make(chan *value)
	i.ctrl = ctrl
	i.v = v
}

func (i *innie) String() string {
	return i.v.Unit.Name + "." + i.v.Name
}

func (o *innie) get() *value { return <-o.c }
func (o *innie) set(x *value) {
	assert.For(x != nil, 20)
	if fn.IsNil(o.x) {
		o.x = x
		go func() {
			for stop := false; !stop; {
				select {
				case o.c <- o.x:
				case stop = <-o.ctrl:
					o.x = nil
					fmt.Println("dropped", o.v.Name)
				}
			}
		}()
	} else {
		halt.As(100, "already assigned")
	}
}

type outie struct {
	x    *value
	c    chan *value
	v    *ir.Variable
	ctrl chan bool
}

func (o *outie) init(v *ir.Variable, ctrl chan bool) {
	o.c = make(chan *value)
	o.ctrl = ctrl
	o.v = v
}

func (o *outie) get() *value { return <-o.c }
func (o *outie) set(x *value) {
	assert.For(x != nil, 20)
	if fn.IsNil(o.x) {
		o.x = x
		go func() {
			for stop := false; !stop; {
				select {
				case o.c <- o.x:
				case stop = <-o.ctrl:
					o.x = nil
					fmt.Println("dropped", o.v.Name)
				}
			}
		}()
	} else {
		halt.As(100, "already assigned")
	}
}

func (i *outie) String() string {
	return i.v.Unit.Name + "." + i.v.Name
}

type mem struct {
	s    chan *value
	c    chan *value
	ctrl chan bool
	v    *ir.Variable
}

func (m *mem) String() string {
	return m.v.Unit.Name + "." + m.v.Name
}

func (m *mem) init(v *ir.Variable, ctrl chan bool) {
	m.c = make(chan *value)
	m.s = make(chan *value)
	m.v = v
	m.ctrl = ctrl
	go func() {
		x := <-m.s
		for stop := false; !stop; {
			select {
			case m.c <- x:
			case x = <-m.s:
			case s := <-m.ctrl:
				if !s {
					stop = true
					fmt.Println("dropped", m.v.Name)
				}
			}
		}
	}()
}

func (o *mem) get() *value { return <-o.c }
func (o *mem) set(v *value) {
	assert.For(v != nil, 20)
	o.s <- v
}

type direct struct {
	x    *value
	c    chan *value
	v    *ir.Variable
	ctrl chan bool
}

func (d *direct) String() string {
	return d.v.Unit.Name + "." + d.v.Name
}

func (d *direct) init(v *ir.Variable, ctrl chan bool) {
	d.c = make(chan *value)
	d.v = v
	d.ctrl = ctrl
}

func (o *direct) get() *value {
	return <-o.c
}

func (o *direct) set(x *value) {
	assert.For(x != nil, 20)
	if fn.IsNil(o.x) {
		o.x = x
		go func() {
			for stop := false; !stop; {
				select {
				case o.c <- o.x:
				case stop = <-o.ctrl:
					o.x = nil
					fmt.Println("dropped", o.v.Name)
				}
			}
		}()
	} else {
		halt.As(100, "already assigned")
	}
}

func obj(v *ir.Variable, ctrl chan bool) (ret object) {
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
	ret.init(v, ctrl)
	return
}
func (ctx *context) init(m *mach) {
	fmt.Println("init context", m.base.Name)
	ctx.owner = m
	ctx.objects = make(map[string]object)
	ctx.ctrl = make(chan bool)
	for k, v := range m.base.Variables {
		if v.Type.Basic {
			ctx.objects[k] = obj(v, ctx.ctrl)
		} else {
			f := m.loader(v.Type.Foreign.Name())
			v.Type.Foreign = ir.NewForeign(f) //определения модулей не прогружаются при загрузке единичного модуля
			m.prepare(v)
		}
	}
	return
}

func (ctx *context) detach(full bool) {
	for _, v := range ctx.owner.base.Variables {
		if v.Type.Basic {
			fmt.Println("drop", v.Unit.Name, v.Name)
			select {
			case ctx.ctrl <- true:
			default:
			}
			if full {
				select {
				case ctx.ctrl <- false:
				default:
				}
			}
		} else {
			if pre := ctx.owner.imps[imp(v)]; pre != nil {
				if !full {
					pre.Reset()
				}
			}
		}
	}
}

func (ctx *context) set0(o object, v *value) {
	fmt.Println("set", o, v)
	o.set(v)
}

func (ctx *context) get0(o object) (v *value) {
	defer fmt.Println("get", o)
	v = o.get()
	return
}

func (ctx *context) foreign(schema *ir.Variable, id string) (ret object) {
	assert.For(schema != nil, 20)
	fmt.Println(schema.Unit.Name, schema.Name, "foreign", id, imp(schema))
	if pre := ctx.owner.imps[imp(schema)]; pre != nil {
		pre.Start()
		ret = pre.ctx.objects[id]
	}
	assert.For(ret != nil, 60)
	return
}

func (ctx *context) local(schema *ir.Variable) (ret object) {
	assert.For(schema != nil, 20)
	fmt.Println("local", schema.Unit.Name, schema.Name)
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
			if e.Foreign == nil {
				stack.push(ctx.get0(ctx.local(e.Var)))
			} else {
				stack.push(ctx.get0(ctx.foreign(e.Var, e.Foreign.Name)))
			}
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

func (ctx *context) process() {
	fmt.Println("process", ctx.owner.base.Name)
	rg := &sync.WaitGroup{}
	starter := make(chan bool)
	count := 0
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
}
