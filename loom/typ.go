package loom

import (
	"github.com/kpmy/trigo"
	"github.com/kpmy/ypk/assert"
	"lomo/ir/types"
	"math/big"
)

type Atom string

type Int struct {
	big.Int
}

func NewInt(x int64) (ret *Int) {
	ret = &Int{}
	ret.Int = *big.NewInt(x)
	return
}

func ThisInt(x *big.Int) (ret *Int) {
	ret = &Int{}
	ret.Int = *x
	return
}

func (i *Int) String() string {
	x, _ := i.Int.MarshalText()
	return string(x)
}

func compTypes(propose, expect types.Type) (ret bool) {
	switch {
	case propose == types.BOOLEAN && expect == types.TRILEAN:
		ret = true
	case propose == expect:
		ret = true
	}
	return
}

func conv(v *value, target types.Type) (ret *value) {
	switch {
	case v.typ == types.BOOLEAN && target == types.TRILEAN:
		b := v.toBool()
		x := tri.This(b)
		ret = &value{typ: target, val: x}
	case v.typ == target:
		ret = v
	}
	assert.For(ret != nil, 60, v.typ, target, v.val)
	return
}
