package p

import (
	"errors"
	"github.com/kpmy/ypk/assert"
	"log"
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/loco/lpp"
	"lomo/loco/lss"
)

type Marker interface {
	Mark(...interface{})
	FutureMark() Marker
}

type pr struct {
	common
	target
}

func (p *pr) init() {
	p.debug = true
	p.target.marker = p
	p.next()
}

func (p *pr) constDecl() {
	assert.For(p.is(lss.Const), 20, "CONST block expected")
	p.next()
	var fwd []func()
	for {
		if p.await(lss.Ident, lss.Separator, lss.Delimiter) {
			id := p.ident()
			p.next()
			c := &ir.Const{Unit: p.target.unit, Name: id}
			if p.await(lss.Equal, lss.Separator) {
				p.next()
				p.pass(lss.Separator)
				expr := &exprBuilder{tgt: &p.target, marker: p}
				expr.fwd = append(expr.fwd, func() {
					for i, x := range expr.fwd {
						if i > 0 {
							x()
						}
					}
				})
				p.expression(expr)
				c.Expr = expr.final()
				fwd = append(fwd, expr.fwd[0])
			} else if p.is(lss.Delimiter) { //ATOM
				c.Expr = &ir.AtomExpr{Value: id}
				p.next()
			} else {
				p.mark("delimiter or = expected")
			}
			p.target.c(c)
		} else {
			break
		}
	}
	for _, f := range fwd {
		f()
	}
}

func (p *pr) varDecl() {
	assert.For(p.sym.Code == lss.Var || p.sym.Code == lss.Reg, 20, "VAR block expected")
	mod := mods.NONE
	if p.is(lss.Reg) {
		mod = mods.REG
	}
	p.next()
	for {
		if p.await(lss.Ident, lss.Delimiter, lss.Separator) {
			var vl []*ir.Variable
			for {
				id := p.ident()
				v := &ir.Variable{Name: id, Unit: p.target.unit}
				vl = append(vl, v)
				p.next()
				if mod == mods.NONE && p.await(lss.Minus) || p.is(lss.Plus) {
					v.Modifier = mods.SymMod[p.sym.Code]
					p.next()
				} else if mod != mods.NONE && p.is(lss.Minus) || p.is(lss.Plus) {
					p.mark("registers private only")
				} else if mod == mods.REG {
					v.Modifier = mods.REG
				}
				if p.await(lss.Comma, lss.Separator) {
					p.next()
					p.pass(lss.Separator)
				} else {
					break
				}
			}
			if p.await(lss.Ident, lss.Separator) {
				tb := &ir.Type{}
				p.typ(p.resolve, tb)
				for _, v := range vl {
					v.Type = *tb
					if !tb.Basic {
						p.target.foreign(v.Name, v)
					}
					if !tb.Basic && v.Modifier != mods.NONE {
						p.mark("only hidden foreigns allowed")
					}
					p.target.obj(v.Name, v)
				}
			} else {
				p.mark("type or identifier expected")
			}
		} else {
			break
		}
	}
}

func (p *pr) rulesDecl() {
	assert.For(p.sym.Code == lss.Process, 20, "PROCESS block expected")
	p.next()
	for stop := false; !stop; {
		p.pass(lss.Delimiter, lss.Separator)
		expr := &exprBuilder{tgt: &p.target, marker: p}
		p.expression(expr)
		p.expect(lss.ArrowRight, "assign expected", lss.Delimiter, lss.Separator)
		p.next()
		p.pass(lss.Delimiter, lss.Separator)
		id := p.ident()
		var fid string
		p.next()
		if p.is(lss.Period) {
			u := p.target.unit.Variables[id]
			if u.Type.Basic {
				p.mark("only foreign types are selectable")
			}
			p.next()
			p.expect(lss.Ident, "foreign variable expected")
			fid = p.ident()
			p.next()
		} else {
			fid = id
			id = p.target.unit.Name
		}
		assert.For(fid != "", 40)
		p.target.assign(id, fid, expr)
		p.pass(lss.Separator, lss.Delimiter)
		stop = p.is(lss.End)
	}
}

func (p *pr) Unit() (u *ir.Unit, err error) {
	if err = p.sc.Error(); err != nil {
		return nil, err
	}
	if !p.debug {
		defer func() {
			if x := recover(); x != nil {
				log.Println(x) // later errors from parser
			}
		}()
	}
	err = errors.New("compiler error")
	p.expect(lss.Unit, "UNIT expected", lss.Delimiter, lss.Separator)
	p.next()
	p.expect(lss.Ident, "unit name expected", lss.Separator)
	p.target.init(p.ident())
	p.next()
	for p.await(lss.Const, lss.Separator, lss.Delimiter) {
		p.constDecl()
	}
	for p.await(lss.Var, lss.Separator, lss.Delimiter) || p.is(lss.Reg) {
		p.varDecl()
	}
	for stop := false; !stop; {
		p.pass(lss.Delimiter, lss.Separator)
		switch p.sym.Code {
		case lss.Infix:
			p.next()
			for stop := false; !stop; {
				if p.await(lss.Ident, lss.Separator) {
					obj := p.target.unit.Variables[p.ident()]
					if obj == nil {
						p.mark("unknown identifier")
					}
					p.target.unit.Infix = append(p.target.unit.Infix, obj)
					p.next()
					if p.await(lss.Delimiter, lss.Separator) {
						p.next()
						stop = true
					}
				} else if p.is(lss.Delimiter) {
					stop = true
					p.next()
				} else {
					p.mark("identifier expected", p.sym.Code)
				}
			}
		case lss.Pre:
			p.next()
			expr := &exprBuilder{tgt: &p.target, marker: p}
			p.expression(expr)
			p.target.unit.Pre = append(p.target.unit.Pre, expr.final())
		case lss.Post:
			p.next()
			expr := &exprBuilder{tgt: &p.target, marker: p}
			p.expression(expr)
			p.target.unit.Post = append(p.target.unit.Post, expr.final())
		default:
			stop = true
		}
	}
	if p.await(lss.Process, lss.Delimiter, lss.Separator) {
		p.rulesDecl()
	}
	p.expect(lss.End, "END expected", lss.Delimiter, lss.Separator)
	p.next()
	p.expect(lss.Ident, "unit name expected", lss.Separator)

	err = nil
	u = p.target.unit
	return
}

func lppc(sc lss.Scanner, r lpp.ForeignResolver) lpp.UnitParser {
	ret := &pr{}
	sc.Init(lss.Unit, lss.End, lss.Var, lss.Process, lss.Reg, lss.Const, lss.True, lss.False, lss.Null, lss.Undef, lss.Infix, lss.Pre, lss.Post, lss.Is, lss.In)
	ret.sc = sc
	ret._resolve = r
	ret.init()
	return ret
}

func init() {
	lpp.ConnectToUnit = lppc
}
