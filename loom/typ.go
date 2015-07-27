package loom

import (
	"fmt"
	"github.com/kpmy/trigo"
	"github.com/kpmy/ypk/assert"
	"lomo/ir/ops"
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

type Rat struct {
	big.Rat
}

func NewRat(x float64) (ret *Rat) {
	ret = &Rat{}
	ret.Rat = *big.NewRat(0, 1)
	return
}

func ThisRat(x *big.Rat) (ret *Rat) {
	ret = &Rat{}
	ret.Rat = *x
	return
}

type Cmp struct {
	re, im *big.Rat
}

func (c *Cmp) String() (ret string) {
	null := big.NewRat(0, 1)
	if c.re.Cmp(null) != 0 {
		ret = fmt.Sprint(c.re)
	}
	if eq := c.im.Cmp(null); eq > 0 {
		ret = fmt.Sprint(ret, "+i", c.im.Abs(c.im))
	} else if eq < 0 {
		ret = fmt.Sprint(ret, "-i", c.im.Abs(c.im))
	} else if ret == "" {
		ret = "0"
	}
	return
}

func NewCmp(re, im float64) (ret *Cmp) {
	ret = &Cmp{}
	ret.re = big.NewRat(0, 1).SetFloat64(re)
	ret.im = big.NewRat(0, 1).SetFloat64(im)
	return
}

func ThisCmp(c *Cmp) (ret *Cmp) {
	ret = &Cmp{}
	*ret = *c
	return
}

type Any struct {
	typ types.Type
	x   interface{}
}

func (a *Any) This() (types.Type, interface{}) {
	return a.typ, a.x
}

func (a *Any) String() string {
	return fmt.Sprint("^", a.x)
}

func (a *Any) Equal(b *Any) (ok bool) {
	ok = false
	if a.x != nil && b.x != nil {
		if a.typ == b.typ {
			v := calcDyadic(&value{typ: a.typ, val: a.x}, ops.Eq, &value{typ: b.typ, val: b.x})
			ok = v.toBool()
		}
	}
	return
}

func ThisAny(v *value) (ret *Any) {
	assert.For(v != nil, 20)
	if a, ok := v.val.(*Any); ok {
		ret = &Any{typ: a.typ, x: a.x}
	} else {
		ret = &Any{typ: v.typ, x: v.val}
	}
	return
}

func NewAny(typ types.Type, val interface{}) *Any {
	_, ok := val.(*Any)
	assert.For(!ok, 20)
	return &Any{typ: typ, x: val}
}

func compTypes(propose, expect types.Type) (ret bool) {
	switch {
	case propose == types.INTEGER && expect == types.REAL:
		ret = true
	case propose == types.BOOLEAN && expect == types.TRILEAN:
		ret = true
	case propose == types.ANY && expect == types.ATOM:
		ret = true
	case expect == types.ANY:
		ret = true
	case propose == expect:
		ret = true
	}
	return
}

func conv(v *value, target types.Type) (ret *value) {
	switch {
	case v.typ == types.INTEGER && target == types.REAL:
		i := v.toInt()
		x := big.NewRat(0, 1)
		ret = &value{typ: target, val: ThisRat(x.SetInt(i))}
	case v.typ == types.BOOLEAN && target == types.TRILEAN:
		b := v.toBool()
		x := tri.This(b)
		ret = &value{typ: target, val: x}
	case v.typ == types.ANY && target == types.ATOM:
		ret = &value{typ: target, val: Atom("")}
	case v.typ == target:
		ret = v
	}
	assert.For(ret != nil, 60, v.typ, target, v.val)
	return
}
