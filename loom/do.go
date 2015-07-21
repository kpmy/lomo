package loom

import (
	"fmt"
	"github.com/kpmy/ypk/assert"
	"lomo/ir"
	"sync"
)

type Msg map[interface{}]interface{}

type Loader func(string) *ir.Unit

type Machine interface {
	Init(name string)
	Start()
	Stop()
}

type mach struct {
	loader Loader
	ctrl   chan Msg
	base   *ir.Unit
	ctx    *context
	imps   map[string]*mach
}

func (m *mach) init(ld Loader) {
	m.loader = ld
	m.imps = make(map[string]*mach)
}

func typeOf(m Msg) (ret string) {
	if x, ok := m["type"]; ok {
		ret, ok = x.(string)
	}
	return
}

func imp(v *ir.Variable) string {
	assert.For(!v.Type.Basic, 20)
	return v.Unit.Name + ":" + v.Name
}

func (m *mach) prepare(v *ir.Variable) func() {
	n := &mach{}
	n.init(m.loader)
	m.imps[imp(v)] = n
	n.Init(v.Type.Foreign.Name())
	return func() {
		n.Start()
	}
}

func (m *mach) process() func() (func(map[string]func()) bool, map[string]func()) {
	var pr func(map[string]func()) bool
	pr = func(d map[string]func()) bool {
		//regular
		return m.ctx.process(d)
	}

	return func() (func(map[string]func()) bool, map[string]func()) {
		//init
		m.ctx = &context{}
		deps := m.ctx.init(m)
		return pr, deps
	}
}

func (m *mach) Init(u string) {
	if m.ctrl == nil {
		m.base = m.loader(u)
		assert.For(m.base != nil, 20)
		m.ctrl = make(chan Msg)
		wg.Add(1)
		init := m.process()
		p, deps := init()
		go func(owner *mach) {
			if m := <-owner.ctrl; m != nil { //didn't even started
				wg.Done()
				return
			}
			for stop := false; p != nil && !stop; {
				stop = p(deps)
				select {
				case msg := <-owner.ctrl:
					switch typeOf(msg) {
					case "machine":
						action, _ := msg["action"].(string)
						stop = action == "stop"
					}
				default:
				}
			}
			owner.ctrl = nil
			for _, m := range owner.imps {
				m.Stop()
			}
			wg.Done()
		}(m)
	}
}

func (m *mach) Start() {
	if m.ctrl != nil {
		fmt.Println("start", m.base.Name)
		m.ctrl <- nil
	}
}

func (m *mach) Stop() {
	if m.ctrl != nil {
		m.ctrl <- Msg{"type": "machine", "action": "stop"}
	}
}

func New(ld Loader) Machine {
	m := &mach{}
	m.init(ld)
	return m
}

var wg *sync.WaitGroup

func init() {
	wg = &sync.WaitGroup{}
}

func Exit() {
	wg.Wait()
}
