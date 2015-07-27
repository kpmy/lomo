package loom

import (
	"github.com/kpmy/trigo"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"log"
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/ir/ops"
	"lomo/ir/types"
	"lomo/loco/lpp"
	"math"
	"math/big"
	"reflect"
	"sync"
)

type Cluster map[string]*Unit

type Unit struct {
	code    *ir.Unit
	objects map[string]object
	imps    map[string]*Unit
	loader  Loader
}

func (u *Unit) init(old *Unit) {
	log.Println("init", u.code.Name)
	u.objects = make(map[string]object)
	for k, v := range u.code.Variables {
		if v.Type.Basic {
			ctrl := make(chan bool)
			var o object
			if v.Modifier == mods.REG && old != nil {
				old := old.objects[v.Name]
				if om := old.(*mem); om != nil {
					o = obj(v, ctrl, om.f)
				}
			} else {
				o = obj(v, ctrl)
			}
			assert.For(o != nil, 60)
			u.objects[k] = o
			<-ctrl
		}
	}
}

func set(o object, v *value) {
	log.Println(o, "set", v)
	t := o.schema().Type.Builtin.Code
	assert.For(compTypes(v.typ, t), 60)
	o.write(conv(v, t))
}

func get(o object) *value {
	log.Println(o, "get")
	return o.read()
}

func (u *Unit) expr(e ir.Expression) *value {
	stack := &exprStack{}
	stack.init()
	var expr func(ir.Expression)
	expr = func(_e ir.Expression) {
		switch e := _e.(type) {
		case ir.WrappedExpression:
			expr(e.Process())
		case *ir.ConstExpr:
			stack.push(cval(e))
		case *ir.AtomExpr:
			stack.push(&value{typ: types.ATOM, val: Atom(e.Value)})
		case *ir.SelectExpr:
			if e.Var != nil {
				e.Var = u.code.Variables[e.Var.Name]
				var o object
				if e.Foreign == nil {
					o = u.objects[e.Var.Name]
				} else {
					if imp := u.imps[imp(e.Var)]; imp != nil {
						o = imp.objects[e.Foreign.Name]
					}
				}
				stack.push(get(o))
			} else if e.Const != nil {
				if c := u.code.Const[e.Const.Name]; c != nil {
					expr(c.Expr)
				} else {
					halt.As(100, "wrong constant name", e.Const.Name)
				}
			} else {
				halt.As(100)
			}
			if e.Inner != mods.NONE {
				base := stack.pop()
				switch {
				case e.Inner == mods.LIST && len(e.ExprList) == 1: //single item
					expr(e.ExprList[0])
					_i := stack.pop()
					idx := int(_i.toInt().Int64())

					switch base.typ {
					case types.STRING:
						s := []rune(base.toStr())
						assert.For(idx >= 0 && idx < len(s), 40)
						stack.push(&value{typ: types.CHAR, val: s[idx]})
					default:
						halt.As(100, "not indexable", base.typ)
					}
				case e.Inner == mods.LIST && len(e.ExprList) > 1: //some items
					switch base.typ {
					case types.STRING:
						s := []rune(base.toStr())
						var ret []rune

						for _, _e := range e.ExprList {
							expr(_e)
							_i := stack.pop()
							i := int(_i.toInt().Int64())
							assert.For(i >= 0 && i < len(s), 40)
							ret = append(ret, s[i])
						}
						stack.push(&value{typ: types.STRING, val: string(ret)})
					default:
						halt.As(100, "not indexable")
					}
				case e.Inner == mods.RANGE && len(e.ExprList) == 2: //range min (from, to) .. max(from, to) with reverse
					expr(e.ExprList[0])
					_f := stack.pop()
					expr(e.ExprList[1])
					_t := stack.pop()
					from := _f.toInt().Int64()
					to := _t.toInt().Int64()

					if int64(math.Max(float64(from), float64(to))) == to { //forward
						switch base.typ {
						case types.STRING:
							s := []rune(base.toStr())
							var ret []rune
							for i := int(from); i <= int(to); i++ {
								assert.For(i >= 0 && i < len(s), 40)
								ret = append(ret, s[i])
							}
							stack.push(&value{typ: types.STRING, val: string(ret)})
						default:
							halt.As(100, "not indexable", base.typ)
						}
					} else {
						switch base.typ {
						case types.STRING:
							s := []rune(base.toStr())
							var ret []rune
							for i := int(to); i >= int(from); i-- {
								assert.For(i >= 0 && i < len(s), 40)
								ret = append(ret, s[i])
							}
							stack.push(&value{typ: types.STRING, val: string(ret)})
						default:
							halt.As(100, "not indexable", base.typ)
						}
					}
				case e.Inner == mods.RANGE && len(e.ExprList) == 1: //open range from `from` to the end of smth
					expr(e.ExprList[0])
					_i := stack.pop()
					idx := int(_i.toInt().Int64())

					switch base.typ {
					case types.STRING:
						s := []rune(base.toStr())
						var ret []rune
						for i := int(idx); i < len(s); i++ {
							assert.For(i >= 0 && i < len(s), 40)
							ret = append(ret, s[i])
						}
						stack.push(&value{typ: types.STRING, val: string(ret)})
					default:
						halt.As(100, "not indexable")
					}
				default:
					halt.As(100, "unexpected selector ", base)
				}
			}
		case *ir.Monadic:
			expr(e.Expr)
			v := stack.pop()
			switch e.Op {
			case ops.Neg:
				switch v.typ {
				case types.INTEGER:
					i := v.toInt()
					i = i.Neg(i)
					v = &value{typ: v.typ, val: ThisInt(i)}
				case types.REAL:
					i := v.toReal()
					i = i.Neg(i)
					stack.push(&value{typ: v.typ, val: ThisRat(i)})
				default:
					halt.As(100, v.typ)
				}
			case ops.Not:
				switch v.typ {
				case types.BOOLEAN:
					b := v.toBool()
					v = &value{typ: v.typ, val: !b}
				case types.TRILEAN:
					t := v.toTril()
					stack.push(&value{typ: v.typ, val: tri.Not(t)})
				/*case types.SET:
				s := v.toSet()
				ns := ThisSet(s)
				ns.inv = !ns.inv
				ctx.push(&value{typ: v.typ, val: ns})*/
				default:
					halt.As(100, "unexpected logical type")
				}
			case ops.Im:
				switch v.typ {
				case types.INTEGER:
					i := v.toInt()
					im := big.NewRat(0, 1)
					im.SetInt(i)
					re := big.NewRat(0, 1)
					c := &Cmp{}
					c.re = re
					c.im = im
					v = &value{typ: types.COMPLEX, val: c}
				case types.REAL:
					im := v.toReal()
					re := big.NewRat(0, 1)
					c := &Cmp{}
					c.re = re
					c.im = im
					v = &value{typ: types.COMPLEX, val: c}
				default:
					halt.As(100, "unexpected operand type ", v.typ)
				}
			default:
				halt.As(100, e.Op)
			}
			stack.push(v)
		case *ir.Dyadic:
			var l, r *value
			if !(e.Op == ops.Or || e.Op == ops.And) {
				expr(e.Left)
				l = stack.pop()
				expr(e.Right)
				r = stack.pop()
				v := calcDyadic(l, e.Op, r)
				stack.push(v)
			} else {
				expr(e.Left)
				l = stack.pop()
				switch e.Op {
				case ops.And:
					switch l.typ {
					case types.BOOLEAN:
						lb := l.toBool()
						if lb {
							expr(e.Right)
							r = stack.pop()
							rb := r.toBool()
							lb = lb && rb
						}
						stack.push(&value{typ: l.typ, val: lb})
					case types.TRILEAN:
						lt := l.toTril()
						if !tri.False(lt) {
							expr(e.Right)
							r = stack.pop()
							rt := r.toTril()
							lt = tri.And(lt, rt)
						}
						stack.push(&value{typ: l.typ, val: lt})
					default:
						halt.As(100, "unexpected logical type")
					}
				case ops.Or:
					switch l.typ {
					case types.BOOLEAN:
						lb := l.toBool()
						if !lb {
							expr(e.Right)
							r = stack.pop()
							rb := r.toBool()
							lb = lb || rb
						}
						stack.push(&value{typ: l.typ, val: lb})
					case types.TRILEAN:
						lt := l.toTril()
						if !tri.True(lt) {
							expr(e.Right)
							r = stack.pop()
							rt := r.toTril()
							lt = tri.Or(lt, rt)
						}
						stack.push(&value{typ: l.typ, val: lt})
					default:
						halt.As(100, "unexpected logical type")
					}
				default:
					halt.As(100, "unknown dyadic op ", e.Op)
				}
			}
		case *ir.Ternary:
			expr(e.If)
			c := stack.pop()
			if c.toBool() {
				expr(e.Then)
			} else {
				expr(e.Else)
			}
		case *ir.InfixExpr:
			var args []*value
			for _, a := range e.Args {
				expr(a)
				args = append(args, stack.pop())
			}
			cm := Init(e.Unit.Name(), u.loader)
			c := make(chan value, 1)
			Do(cm, func(cm Cluster) {
				inf := cm[e.Unit.Name()]
				assert.For(inf != nil, 40)
				for i := len(args) - 1; i >= 0; i-- {
					go func(i int, v *value) {
						o := inf.objects[inf.code.Infix[i].Name]
						set(o, v)
					}(i+1, args[i])
				}
				go func(c chan value) {
					o := inf.objects[inf.code.Infix[0].Name]
					v := get(o)
					log.Println("infix", v)
					c <- *v
				}(c)
			}).Wait()
			v := <-c
			Close(cm).Wait()
			stack.push(&v)
		default:
			halt.As(100, reflect.TypeOf(e))
		}
	}
	expr(e)
	return stack.pop()
}

func (u *Unit) rule(o object, _r ir.Rule) {
	switch r := _r.(type) {
	case *ir.AssignRule:
		v := u.expr(r.Expr)
		log.Println("for ", o.schema().Name)
		set(o, v)
	case stdRule:
		r.do(u, o)
	default:
		halt.As(100, reflect.TypeOf(r))
	}
}

func Init(_top string, ld Loader) (ret map[string]*Unit) {
	assert.For(ld != nil, 20, _top)
	ret = make(map[string]*Unit)
	var run func(*Unit)
	run = func(u *Unit) {
		u.imps = ret
		for _, v := range u.code.Variables {
			if !v.Type.Basic {
				if dep := ld(v.Type.Foreign.Name()); dep != nil {
					ret[imp(v)] = &Unit{code: dep, loader: ld}
					v.Type.Foreign = ir.NewForeign(dep)
					run(ret[imp(v)])
				}
			}
		}
	}
	if std := lpp.Std[_top]; std != nil {
		ret[_top] = stdUnit(std)
		run(ret[_top])
	} else if top := ld(_top); top != nil {
		ret[_top] = &Unit{code: top, loader: ld}
		run(ret[_top])
	}
	return
}

func Do(um Cluster, pre func(Cluster), old ...Cluster) (ret *sync.WaitGroup) {
	ret = &sync.WaitGroup{}
	for _, u := range um {
		var o *Unit
		if len(old) > 0 && old[0] != nil {
			o = old[0][u.code.Name]
		}
		u.init(o)
	}
	if pre != nil {
		pre(um)
	}
	for _, u := range um {
		_wg.Add(1)
		ret.Add(1)
		go func(this *Unit) {
			rg := &sync.WaitGroup{}
			for _, e := range this.code.Pre {
				rg.Add(1)
				go func(pre ir.Expression) {
					v := this.expr(pre)
					if !v.toBool() {
						panic("precondition violated")
					}
					rg.Done()
				}(e)
			}
			rg.Wait()
			for v, r := range this.code.Rules {
				rg.Add(1)
				go func(o object, r ir.Rule) {
					this.rule(o, r)
					rg.Done()
				}(this.objects[v], r)
			}
			for v, r := range this.code.ForeignRules {
				fv := this.code.Variables[v]
				for _, f := range fv.Type.Foreign.Variables() {
					if fr := r[f.Name]; fr != nil {
						imp := this.imps[imp(fv)]
						rg.Add(1)
						go func(o object, r ir.Rule) {
							this.rule(o, r)
							rg.Done()
						}(imp.objects[f.Name], fr)
					}
				}
			}
			rg.Wait()
			for _, e := range this.code.Post {
				rg.Add(1)
				go func(post ir.Expression) {
					v := this.expr(post)
					if !v.toBool() {
						panic("precondition violated")
					}
					rg.Done()
				}(e)
			}
			rg.Wait()
			_wg.Done()
			ret.Done()
		}(u)
	}
	return
}

func Close(um Cluster) (ret *sync.WaitGroup) {
	ret = &sync.WaitGroup{}
	for _, u := range um {
		ret.Add(1)
		_wg.Add(1)
		go func(u *Unit) {
			for _, o := range u.objects {
				o.control() <- true
				<-o.control()
			}
			_wg.Done()
			ret.Done()
		}(u)
	}
	return
}

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
