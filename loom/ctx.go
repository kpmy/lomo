package loom

import (
	"container/list"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"log"
	"lomo/ir"
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
	rd      *sync.Mutex
	wr      *sync.Mutex
}

func (ctx *context) init(m *mach) {
	//fmt.Println("init context", m.base.Name)
	ctx.owner = m
	ctx.objects = make(map[string]object)
	ctx.ctrl = make(chan bool)
	ctx.rd = new(sync.Mutex)
	ctx.wr = new(sync.Mutex)
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

func (ctx *context) refresh(full bool) {
	log.Println("refresh", ctx.owner.base.Name)
	ctx.wr.Lock()
	ctx.rd.Lock()
	for _, v := range ctx.owner.base.Variables {
		if v.Type.Basic {
			//fmt.Println("drop", v.Unit.Name, v.Name)
			select {
			case ctx.ctrl <- !full:
				//default:
			}
		}
	}
	log.Println("refreshed", ctx.owner.base.Name)
	ctx.rd.Unlock()
	ctx.wr.Unlock()
}

func (ctx *context) set0(o object, v *value) {
	ctx.wr.Lock()
	log.Println("set", o, v)
	t := o.schema().Type.Builtin.Code
	assert.For(compTypes(v.typ, t), 60)
	o.set(conv(v, t))
	ctx.wr.Unlock()
}

func (ctx *context) get0(o object) (v *value) {
	ctx.rd.Lock()
	defer log.Println("get", o)
	v = o.get()
	ctx.rd.Unlock()
	return
}

func (ctx *context) foreign(schema *ir.Variable, id string) (c *context, ret object) {
	assert.For(schema != nil, 20)
	//fmt.Println(schema.Unit.Name, schema.Name, "foreign", id, imp(schema))
	if pre := ctx.owner.imps[imp(schema)]; pre != nil {
		ret = pre.ctx.objects[id]
		c = pre.ctx
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
				rd, obj := ctx.foreign(e.Var, e.Foreign.Name)
				stack.push(rd.get0(obj))
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

func (ctx *context) rule(wr *context, o object, _r ir.Rule) {
	switch r := _r.(type) {
	case *ir.AssignRule:
		wr.set0(o, ctx.expr(r.Expr))
	default:
		halt.As(100, reflect.TypeOf(r))
	}
}

func (ctx *context) process() {
	log.Println("process", ctx.owner.base.Name)
	rg := &sync.WaitGroup{}
	starter := make(chan bool)
	count := 0
	for v, r := range ctx.owner.base.Rules {
		count++
		go func(v *ir.Variable, _r ir.Rule) {
			<-starter
			ctx.rule(ctx, ctx.local(v), _r)
			rg.Done()
		}(ctx.owner.base.Variables[v], r)
	}
	for v, r := range ctx.owner.base.ForeignRules {
		o := ctx.owner.base.Variables[v]
		for _, v := range o.Type.Foreign.Variables() {
			if rr := r[v.Name]; rr != nil {
				count++
				go func(f, v *ir.Variable, _r ir.Rule) {
					wr, pre := ctx.foreign(f, v.Name)
					<-starter
					ctx.rule(wr, pre, _r)
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
