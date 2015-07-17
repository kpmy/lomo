package p

import (
	"errors"
	"fmt"
	"github.com/kpmy/ypk/assert"
	"log"
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
	p.next()
}

func (p *pr) typ() {
	assert.For(p.sym.Code == lss.Ident, 20, "type identifier expected here but found ", p.sym.Code)
	id := p.ident()
	fmt.Println(id)
	p.next()
}

func (p *pr) varDecl() {
	assert.For(p.sym.Code == lss.Var, 20, "VAR block expected")
	p.next()
	for {
		if p.await(lss.Ident, lss.Delimiter, lss.Separator) {
			for {
				id := p.ident()
				fmt.Println(id)
				p.next()
				if p.await(lss.Minus) || p.is(lss.Plus) || p.is(lss.Times) {
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
				p.typ()
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

func (p *pr) Unit() (err error) {
	if err = p.sc.Error(); err != nil {
		return err
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
	p.do(p.ident())
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
	return
}

func lppc(sc lss.Scanner) lpp.UnitParser {
	ret := &pr{}
	sc.Init(lss.Unit, lss.End, lss.Var, lss.Process)
	ret.sc = sc
	ret.init()
	return ret
}

func init() {
	lpp.ConnectToUnit = lppc
}
