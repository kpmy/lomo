package loom

import (
	"container/list"
	"fmt"
	"github.com/kpmy/lomo/ir"
	"github.com/kpmy/lomo/ir/types"
	"github.com/kpmy/trigo"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"math/big"
	"reflect"
)

type value struct {
	typ types.Type
	val interface{}
}

func (v *value) String() string {
	return fmt.Sprint(v.val)
}

func (v *value) toAtom() (ret Atom) {
	assert.For(v.typ == types.ATOM, 20)
	switch x := v.val.(type) {
	case Atom:
		ret = x
	case nil: //do nothing
	default:
		halt.As(100, "wrong atom ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toStr() (ret string) {
	assert.For(v.typ == types.STRING, 20)
	switch x := v.val.(type) {
	case string:
		ret = x
	default:
		halt.As(100, "wrong string ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toInt() (ret *big.Int) {
	assert.For(v.typ == types.INTEGER, 20)
	switch x := v.val.(type) {
	case int:
		ret = big.NewInt(int64(x))
	case *Int:
		ret = big.NewInt(0)
		ret.Add(ret, &x.Int)
	default:
		halt.As(100, "wrong integer ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toBool() (ret bool) {
	assert.For(v.typ == types.BOOLEAN, 20)
	switch x := v.val.(type) {
	case bool:
		ret = x
	default:
		halt.As(100, "wrong boolean ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toTril() (ret tri.Trit) {
	assert.For(v.typ == types.TRILEAN || v.typ == types.BOOLEAN, 20, v.typ)
	switch x := v.val.(type) {
	case tri.Trit:
		ret = x
	case bool:
		ret = tri.This(x)
	default:
		halt.As(100, "wrong trilean ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toReal() (ret *big.Rat) {
	assert.For(v.typ == types.REAL, 20)
	switch x := v.val.(type) {
	case *Rat:
		ret = big.NewRat(0, 1)
		ret.Add(ret, &x.Rat)
	default:
		halt.As(100, "wrong real ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toCmp() (ret *Cmp) {
	assert.For(v.typ == types.COMPLEX, 20)
	switch x := v.val.(type) {
	case *Cmp:
		ret = ThisCmp(x)
	default:
		halt.As(100, "wrong complex ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toRune() (ret rune) {
	assert.For(v.typ == types.CHAR, 20, v.typ)
	switch x := v.val.(type) {
	case rune:
		ret = x
	default:
		halt.As(100, "wrong rune ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toAny() (ret *Any) {
	assert.For(v.typ == types.ANY, 20)
	switch x := v.val.(type) {
	case *Any:
		ret = ThisAny(&value{typ: x.typ, val: x.x})
	default:
		halt.As(100, "wrong any ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toSet() (ret *Set) {
	assert.For(v.typ == types.SET, 20)
	switch x := v.val.(type) {
	case *Set:
		ret = ThisSet(x)
	default:
		halt.As(100, "wrong list ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toList() (ret *List) {
	assert.For(v.typ == types.LIST, 20)
	switch x := v.val.(type) {
	case *List:
		ret = ThisList(x)
	default:
		halt.As(100, "wrong list ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toMap() (ret *Map) {
	assert.For(v.typ == types.MAP, 20)
	switch x := v.val.(type) {
	case *Map:
		ret = ThisMap(x)
	default:
		halt.As(100, "wrong list ", reflect.TypeOf(x))
	}
	return
}

func (v *value) asList() (ret *List) {
	assert.For(v.typ == types.LIST, 20)
	switch x := v.val.(type) {
	case *List:
		ret = x
	default:
		halt.As(100, "wrong list ", reflect.TypeOf(x))
	}
	return
}

func (v *value) asMap() (ret *Map) {
	assert.For(v.typ == types.MAP, 20)
	switch x := v.val.(type) {
	case *Map:
		ret = x
	default:
		halt.As(100, "wrong list ", reflect.TypeOf(x))
	}
	return
}

func (v *value) toRef() (ret *Ref) {
	assert.For(v.typ == types.UNIT, 20)
	switch x := v.val.(type) {
	case *Ref:
		ret = ThisRef(x)
	default:
		halt.As(100, "wrong list ", reflect.TypeOf(x))
	}
	return
}

func cval(e *ir.ConstExpr) (ret *value) {
	t := e.Type
	switch t {
	case types.INTEGER:
		b := big.NewInt(0)
		if err := b.UnmarshalText([]byte(e.Value.(string))); err == nil {
			v := ThisInt(b)
			ret = &value{typ: t, val: v}
		} else {
			halt.As(100, "wrong integer")
		}
	case types.REAL:
		r := big.NewRat(0, 1)
		if err := r.UnmarshalText([]byte(e.Value.(string))); err == nil {
			v := ThisRat(r)
			ret = &value{typ: t, val: v}
		} else {
			halt.As(100, "wrong real")
		}
	case types.BOOLEAN:
		ret = &value{typ: t, val: e.Value.(bool)}
	case types.TRILEAN:
		ret = &value{typ: t, val: tri.NIL}
	case types.CHAR:
		var v rune
		switch x := e.Value.(type) {
		case int32:
			v = rune(x)
		case int:
			v = rune(x)
		default:
			halt.As(100, "unsupported rune coding")
		}
		ret = &value{typ: t, val: v}
	case types.STRING:
		v := e.Value.(string)
		ret = &value{typ: t, val: v}
	case types.ANY:
		ret = &value{typ: t, val: &Any{}}
	default:
		halt.As(100, "unknown type ", t, " for ", e)
	}
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
