package loom

import (
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/fn"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/ir/types"
)

type mem struct {
	s    chan interface{}
	c    chan interface{}
	ctrl chan bool
	v    *ir.Variable
	f    interface{}
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

func (m *mem) init(v *ir.Variable, ctrl chan bool, def ...interface{}) {
	m.c = make(chan interface{})
	m.s = make(chan interface{})
	m.v = v
	m.ctrl = ctrl
	go func() {
		x := <-m.s
		m.ctrl <- true
		for stop := false; !stop; {
			select {
			case n := <-m.s:
				m.f = n
			case m.c <- x:
			case s := <-m.ctrl:
				if s {
					stop = true
					//fmt.Println("dropped", m.v.Name)
				} else {
					if !fn.IsNil(m.f) {
						x = m.f
					}
				}
			}
		}
		m.ctrl <- true
	}()
	if len(def) != 0 && !fn.IsNil(def[0]) {
		m.s <- def[0]
	} else {
		m.s <- m.defaults(v).val
	}
}

func (o *mem) get() *value { return &value{typ: o.v.Type.Builtin.Code, val: <-o.c} }
func (o *mem) set(v *value) {
	assert.For(v != nil, 20)
	o.s <- v.val
}

func (o *mem) control() chan bool {
	return o.ctrl
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

func (d *direct) init(v *ir.Variable, ctrl chan bool, def ...interface{}) {
	d.c = make(chan interface{})
	d.s = make(chan interface{})
	d.v = v
	d.ctrl = ctrl
	go func() {
		d.ctrl <- true
		var x interface{}
		for stop := false; !stop; {
			if fn.IsNil(x) {
				select {
				case x = <-d.s:
				case s := <-d.ctrl:
					stop = s
				}
			} else {
				select {
				case d.c <- x:
				case s := <-d.ctrl:
					stop = s
					if !s {
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
		d.ctrl <- true
	}()
}

func (o *direct) get() *value {
	return &value{typ: o.v.Type.Builtin.Code, val: <-o.c}
}

func (o *direct) set(x *value) {
	assert.For(x != nil, 20)
	o.s <- x.val
}

func (o *direct) control() chan bool {
	return o.ctrl
}

func obj(v *ir.Variable, ctrl chan bool, userData ...interface{}) (ret object) {
	switch v.Modifier {
	case mods.REG:
		ret = &mem{}
	default: //var
		ret = &direct{}
	}
	ret.init(v, ctrl, userData...)
	return
}
