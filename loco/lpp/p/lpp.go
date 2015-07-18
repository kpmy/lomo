package p

import (
	"errors"
	"github.com/kpmy/ypk/assert"
	"log"
	"lomo/ir"
	"lomo/ir/mods"
	"lomo/ir/types"
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
	resolve lpp.ForeignResolver
}

func (p *pr) init() {
	p.debug = true
	p.target.marker = p
	p.next()
}

func (p *pr) typ(t *ir.Type) {
	assert.For(p.sym.Code == lss.Ident, 20, "type identifier expected here but found ", p.sym.Code)
	id := p.ident()
	if it := types.TypMap[id]; it != types.UNDEF {
		t.Basic = true
		t.Builtin = &ir.BuiltinType{Code: it}
	} else if ft := p.resolve(id); ft != nil { //append import resolver
		t.Basic = false
		t.Foreign = ft
	} else {
		p.mark("undefined type ", id)
	}
	p.next()
}

func (p *pr) varDecl() {
	assert.For(p.sym.Code == lss.Var, 20, "VAR block expected")
	p.next()
	for {
		if p.await(lss.Ident, lss.Delimiter, lss.Separator) {
			var vl []*ir.Variable
			for {
				id := p.ident()
				v := &ir.Variable{Name: id, Unit: p.target.unit}
				vl = append(vl, v)
				p.next()
				if p.await(lss.Minus) || p.is(lss.Plus) || p.is(lss.Times) {
					v.Modifier = mods.SymMod[p.sym.Code]
					p.next()
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
				p.typ(tb)
				for _, v := range vl {
					v.Type = tb
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
	for p.await(lss.Var, lss.Separator, lss.Delimiter) {
		p.varDecl()
	}
	p.expect(lss.Process, "PROCESS expected", lss.Delimiter, lss.Separator)
	p.rulesDecl()
	p.expect(lss.End, "END expected", lss.Delimiter, lss.Separator)
	p.next()
	p.expect(lss.Ident, "unit name expected", lss.Separator)

	err = nil
	u = p.target.unit
	return
}

func lppc(sc lss.Scanner, r lpp.ForeignResolver) lpp.UnitParser {
	ret := &pr{}
	sc.Init(lss.Unit, lss.End, lss.Var, lss.Process)
	ret.sc = sc
	ret.resolve = r
	ret.init()
	return ret
}

func init() {
	lpp.ConnectToUnit = lppc
}
