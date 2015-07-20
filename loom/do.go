package loom

import (
	"github.com/kpmy/ypk/assert"
	"lomo/ir"
	"sync"
)

type Msg map[interface{}]interface{}

type Loader func(string) *ir.Unit

type Machine interface {
	Start(name string)
	Stop()
}

type mach struct {
	loader Loader
	ctrl   chan Msg
	base   *ir.Unit
	ctx    *context
}

func typeOf(m Msg) (ret string) {
	if x, ok := m["type"]; ok {
		ret, ok = x.(string)
	}
	return
}

func (m *mach) process() func() func() bool {
	var pr func() bool
	pr = func() bool {
		//regular
		return m.ctx.process()
	}

	return func() func() bool {
		//init
		m.ctx = &context{}
		m.ctx.init(m)
		return pr
	}
}

func (m *mach) Start(u string) {
	if m.ctrl == nil {
		m.base = m.loader(u)
		assert.For(m.base != nil, 20)
		m.ctrl = make(chan Msg)
		wg.Add(1)
		go func(owner *mach) {
			init := owner.process()
			p := init()
			for stop := false; p != nil && !stop; {
				stop = p()
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
			wg.Done()
		}(m)
	}
}

func (m *mach) Stop() {
	if m.ctrl != nil {
		m.ctrl <- Msg{"type": "machine", "action": "stop"}
	}
}

func New(ld Loader) Machine {
	return &mach{loader: ld}
}

var wg *sync.WaitGroup

func init() {
	wg = &sync.WaitGroup{}
}

func Exit() {
	wg.Wait()
}
