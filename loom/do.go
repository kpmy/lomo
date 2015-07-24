package loom

import (
	"github.com/kpmy/ypk/assert"
	"lomo/ir"
	"sync"
)

type Loader func(string) *ir.Unit

func imp(v *ir.Variable) string {
	assert.For(!v.Type.Basic, 20)
	return v.Unit.Name + ":" + v.Name
}

var _wg *sync.WaitGroup

func init() {
	_wg = &sync.WaitGroup{}
}

func Exit() {
	_wg.Wait()
}
