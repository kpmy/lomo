package loom

import (
	"github.com/kpmy/trigo"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/fn"
	"github.com/kpmy/ypk/halt"
	"log"
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/ir/types"
)

type object interface {
	init(*ir.Variable, chan bool, ...interface{})
	write(*value)
	read() *value
	schema() *ir.Variable
	control() chan bool
}

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
		ret = &value{typ: t, val: NewInt(0)}
	case types.BOOLEAN:
		ret = &value{typ: t, val: false}
	case types.TRILEAN:
		ret = &value{typ: t, val: tri.NIL}
	case types.REAL:
		ret = &value{typ: t, val: NewRat(0)}
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
			case stop = <-m.ctrl:
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

func (o *mem) read() *value { return &value{typ: o.v.Type.Builtin.Code, val: <-o.c} }
func (o *mem) write(v *value) {
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
					log.Println("rd", x)
				case stop = <-d.ctrl:
				}
			} else {
				select {
				case d.c <- x:
					log.Println("wr", x)
				case stop = <-d.ctrl:
				}
			}
		}
		d.ctrl <- true
	}()
}

func (o *direct) read() *value {
	return &value{typ: o.v.Type.Builtin.Code, val: <-o.c}
}

func (o *direct) write(x *value) {
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
