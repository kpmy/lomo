package loom

import (
	"container/list"
	"fmt"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/fn"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/ir/types"
	"reflect"
	"sync"
)

type object interface {
	init(*ir.Variable, chan bool)
	set(*value)
	get() *value
	schema() *ir.Variable
}

type context struct {
	owner   *mach
	objects map[string]object
	ctrl    chan bool
}

type mem struct {
	s    chan interface{}
	c    chan interface{}
	ctrl chan bool
	v    *ir.Variable
}

func (o *mem) schema() *ir.Variable { return o.v }
func (m *mem) String() string {
	return m.v.Unit.Name + "." + m.v.Name
}

func (m *mem) defaults(v *ir.Variable) (ret *value) {
	switch t := v.Type.Builtin.Code; t {
	case types.INTEGER:
		ret = &value{typ: types.INTEGER, val: NewInt(0)}
	default:
		halt.As(100, t)
	}
	return
}

func (m *mem) init(v *ir.Variable, ctrl chan bool) {
	m.c = make(chan interface{})
	m.s = make(chan interface{})
	m.v = v
	m.ctrl = ctrl
	go func() {
		x := <-m.s
		var f interface{} //future
		for stop := false; !stop; {
			select {
			case n := <-m.s:
				f = n
			case m.c <- x:
			case s := <-m.ctrl:
				if !s {
					stop = true
					//fmt.Println("dropped", m.v.Name)
				} else {
					if !fn.IsNil(f) {
						x = f
					}
				}
			}
		}
		fmt.Println(m, f)
	}()
	m.s <- m.defaults(v).val
}

func (o *mem) get() *value { return &value{typ: o.v.Type.Builtin.Code, val: <-o.c} }
func (o *mem) set(v *value) {
	assert.For(v != nil, 20)
	o.s <- v.val
}

type direct struct {
	s    chan interface{}
	c    chan interface{}
	v    *ir.Variable
	ctrl chan bool
}

func (o *direct) schema() *ir.Variable { return o.v }
func (d *direct) String() string {
	return d.v.Unit.Name + "." + d.v.Name
}

func (d *direct) init(v *ir.Variable, ctrl chan bool) {
	d.c = make(chan interface{})
	d.s = make(chan interface{})
	d.v = v
	d.ctrl = ctrl
	go func() {
		var x interface{}
		for stop := false; !stop; {
			if fn.IsNil(x) {
				select {
				case x = <-d.s:
				case s := <-d.ctrl:
					stop = !s
				}
			} else {
				select {
				case d.c <- x:
				case s := <-d.ctrl:
					stop = !s
					if s {
						x = nil
						for br := false; !br; {
							select {
							case _ = <-d.c:
							default:
								br = true
							}
						}
					}
				}
			}
		}
		fmt.Println(d, x)
	}()
}

func (o *direct) get() *value {
	return &value{typ: o.v.Type.Builtin.Code, val: <-o.c}
}

func (o *direct) set(x *value) {
	assert.For(x != nil, 20)
	o.s <- x.val
}

func obj(v *ir.Variable, ctrl chan bool) (ret object) {
	switch v.Modifier {
	case mods.REG:
		ret = &mem{}
	default: //var
		ret = &direct{}
	}
	ret.init(v, ctrl)
	return
}
func (ctx *context) init(m *mach) {
	//fmt.Println("init context", m.base.Name)
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
			//fmt.Println("drop", v.Unit.Name, v.Name)
			select {
			case ctx.ctrl <- !full:
				//default:
			}
		}
	}
}

func (ctx *context) set0(o object, v *value) {
	//fmt.Println("set", o, v)
	t := o.schema().Type.Builtin.Code
	assert.For(compTypes(v.typ, t), 60)
	o.set(conv(v, t))
}

func (ctx *context) get0(o object) (v *value) {
	//defer fmt.Println("get", o)
	v = o.get()
	return
}

func (ctx *context) foreign(schema *ir.Variable, id string) (ret object) {
	assert.For(schema != nil, 20)
	//fmt.Println(schema.Unit.Name, schema.Name, "foreign", id, imp(schema))
	if pre := ctx.owner.imps[imp(schema)]; pre != nil {
		pre.Start(ctx.owner.started)
		ret = pre.ctx.objects[id]
	}
	assert.For(ret != nil, 60)
	return
}

func (ctx *context) local(schema *ir.Variable) (ret object) {
	assert.For(schema != nil, 20)
	//fmt.Println("local", schema.Unit.Name, schema.Name)
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
			stack.push(cval(e))
		case *ir.SelectExpr:
			e.Var = ctx.owner.base.Variables[e.Var.Name]
			if e.Foreign == nil {
				stack.push(ctx.get0(ctx.local(e.Var)))
			} else {
				stack.push(ctx.get0(ctx.foreign(e.Var, e.Foreign.Name)))
			}
		case *ir.Dyadic:
			var l, r *value
			expr(e.Left)
			l = stack.pop()
			expr(e.Right)
			r = stack.pop()
			v := calcDyadic(l, e.Op, r)
			stack.push(v)
		case *ir.Ternary:
			expr(e.If)
			c := stack.pop()
			if c.toBool() {
				expr(e.Then)
			} else {
				expr(e.Else)
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
	case *ir.AssignRule:
		ctx.set0(o, ctx.expr(r.Expr))
	default:
		halt.As(100, reflect.TypeOf(r))
	}
}

func (ctx *context) process() {
	//fmt.Println("process", ctx.owner.base.Name)
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
					pre := ctx.foreign(f, v.Name)
					<-starter
					ctx.rule(pre, _r)
					rg.Done()
				}(o, v, rr)
			}
		}
	}
	rg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			starter <- true
		}()
	}
	rg.Wait()
}
