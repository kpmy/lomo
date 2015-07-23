package loom

import (
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
	"sync"
)

type Msg map[interface{}]interface{}

type Loader func(string) *ir.Unit

type Machine interface {
	Init(name string)
	Start(*sync.WaitGroup)
	Wait()
	Stop()
}

type mach struct {
	loader  Loader
	ctrl    chan Msg
	base    *ir.Unit
	ctx     *context
	imps    map[string]*mach
	started *sync.WaitGroup
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

func (m *mach) prepare(v *ir.Variable) {
	n := &mach{}
	n.init(m.loader)
	m.imps[imp(v)] = n
	n.Init(v.Type.Foreign.Name())
}

func (m *mach) handle(msg Msg) (stop bool) {
	//fmt.Println(m.base.Name, "handle", msg)
	switch t := typeOf(msg); t {
	case "machine":
		action, _ := msg["action"].(string)
		switch action {
		case "stop":
			m.ctx.refresh(true)
			stop = true
			m.ctrl <- nil
		case "do":
			sg := new(sync.WaitGroup)
			for _, n := range m.imps {
				n.ctx.refresh(false)
				n.Start(sg)
			}
			sg.Wait()
			m.started.Add(1)
			m.ctx.refresh(false)
			m.ctx.process()
			m.started.Done()
		}
	default:
		halt.As(100, t)
	}
	return
}

func (m *mach) Init(u string) {
	//fmt.Println("init", u)
	if m.ctrl == nil {
		m.base = m.loader(u)
		assert.For(m.base != nil, 20)
		ctrl := make(chan Msg)
		m.ctx = &context{}
		m.ctx.init(m)
		wg.Add(1)
		go func(owner *mach, ctrl chan Msg) {
			owner.ctrl = ctrl
			owner.ctrl <- nil
			for stop := false; !stop; {
				select {
				case msg := <-owner.ctrl:
					stop = m.handle(msg)
				}
			}
			owner.ctrl = nil
			for _, m := range owner.imps {
				m.Stop()
			}
			wg.Done()
		}(m, ctrl)
		<-ctrl
		//fmt.Println("ready", u)
	}
}

func (m *mach) Wait() {
	if m.ctrl != nil {
		m.started.Wait()
	}
}

func (m *mach) Start(wg *sync.WaitGroup) {
	if m.ctrl != nil {
		//fmt.Println("start", m.base.Name)
		m.started = wg
		m.ctrl <- Msg{"type": "machine", "action": "do"}
	} else {
		halt.As(100, "not initialized")
	}
}

func (m *mach) Stop() {
	if m.ctrl != nil {
		m.ctrl <- Msg{"type": "machine", "action": "stop"}
		<-m.ctrl
	} else {
		halt.As(100, "not initialized")
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
