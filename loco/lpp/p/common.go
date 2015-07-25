package p

import (
	"fmt"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"lomo/ir"
	"lomo/ir/ops"
	"lomo/ir/types"
	"lomo/loco/lss"
	"strconv"
)

type mark struct {
	rd        int
	line, col int
	marker    Marker
}

func (m *mark) Mark(msg ...interface{}) {
	m.marker.(*common).m = m
	m.marker.Mark(msg...)
}

func (m *mark) FutureMark() Marker { halt.As(100); return nil }

type common struct {
	sc    lss.Scanner
	sym   lss.Sym
	done  bool
	debug bool
	m     *mark
}

func (p *common) Mark(msg ...interface{}) {
	p.mark(msg...)
}

func (p *common) FutureMark() Marker {
	rd := p.sc.Read()
	str, pos := p.sc.Pos()
	m := &mark{marker: p, rd: rd, line: str, col: pos}
	return m
}

func (p *common) mark(msg ...interface{}) {
	rd := p.sc.Read()
	str, pos := p.sc.Pos()
	if len(msg) == 0 {
		p.m = &mark{rd: rd, line: str, col: pos}
	} else if p.m != nil {
		rd, str, pos = p.m.rd, p.m.line, p.m.col
		p.m = nil
	}
	if p.m == nil {
		panic(lss.Err("parser", rd, str, pos, msg...))
	}
}

func (p *common) next() lss.Sym {
	p.done = true
	if p.sym.Code != lss.None {
		//		fmt.Print("this ")
		//		fmt.Print("`" + fmt.Sprint(p.sym) + "`")
	}
	p.sym = p.sc.Get()
	//fmt.Print(" next ")
	if p.debug {
		fmt.Println("`" + fmt.Sprint(p.sym) + "`")
	}
	return p.sym
}

//expect is the most powerful step forward runner, breaks the compilation if unexpected sym found
func (p *common) expect(sym lss.Symbol, msg string, skip ...lss.Symbol) {
	assert.For(p.done, 20)
	if !p.await(sym, skip...) {
		p.mark(msg)
	}
	p.done = false
}

//await runs for the sym through skip list, but may not find the sym
func (p *common) await(sym lss.Symbol, skip ...lss.Symbol) bool {
	assert.For(p.done, 20)
	skipped := func() (ret bool) {
		for _, v := range skip {
			if v == p.sym.Code {
				ret = true
			}
		}
		return
	}

	for sym != p.sym.Code && skipped() && p.sc.Error() == nil {
		p.next()
	}
	p.done = p.sym.Code != sym
	return p.sym.Code == sym
}

//pass runs through skip list
func (p *common) pass(skip ...lss.Symbol) {
	skipped := func() (ret bool) {
		for _, v := range skip {
			if v == p.sym.Code {
				ret = true
			}
		}
		return
	}
	for skipped() && p.sc.Error() == nil {
		p.next()
	}
}

//run runs to the first sym through any other sym
func (p *common) run(sym lss.Symbol) {
	if p.sym.Code != sym {
		for p.sc.Error() == nil && p.next().Code != sym {
			if p.sc.Error() != nil {
				p.mark("not found")
				break
			}
		}
	}
}

func (p *common) ident() string {
	assert.For(p.sym.Code == lss.Ident, 20, "identifier expected")
	//добавить валидацию идентификаторов
	return p.sym.Str
}

func (p *common) is(sym lss.Symbol) bool {
	return p.sym.Code == sym
}

func (p *common) number() (t types.Type, v interface{}) {
	assert.For(p.is(lss.Number), 20, "number expected here")
	switch p.sym.NumberOpts.Modifier {
	case "":
		if p.sym.NumberOpts.Period {
			t, v = types.REAL, p.sym.Str
		} else {
			//x, err := strconv.Atoi(p.sym.Str)
			//assert.For(err == nil, 40)
			t, v = types.INTEGER, p.sym.Str
		}
	case "U":
		if p.sym.NumberOpts.Period {
			p.mark("hex integer value expected")
		}
		//fmt.Println(p.sym)
		if r, err := strconv.ParseUint(p.sym.Str, 16, 64); err == nil {
			t = types.CHAR
			v = rune(r)
		} else {
			p.mark("error while reading integer")
		}
	default:
		p.mark("unknown number format `", p.sym.NumberOpts.Modifier, "`")
	}
	return
}

func (p *common) inside(b *selectBuilder) {
	if p.await(lss.Lbrak, lss.Separator, lss.Delimiter) {
		p.next()
		up := &exprBuilder{tgt: b.tgt, marker: b.marker}
		p.expression(up)
		if p.await(lss.Rbrak, lss.Separator) { // single index
			p.done = true
			b.list([]ir.Expression{up.final()})
		} else if p.is(lss.UpTo) {
			p.next()
			if p.await(lss.Rbrak, lss.Delimiter, lss.Separator) {
				p.done = true
				b.upto(up.final())
			} else {
				to := &exprBuilder{tgt: b.tgt, marker: b.marker}
				p.expression(to)
				b.upto(up.final(), to.final())
			}
		} else if p.is(lss.Comma) {
			el := []ir.Expression{up.final()}
			for p.await(lss.Comma, lss.Separator, lss.Delimiter) {
				p.next()
				e := &exprBuilder{tgt: b.tgt, marker: b.marker}
				p.expression(e)
				el = append(el, e.final())
			}
			b.list(el)
		}
		p.expect(lss.Rbrak, "] expected", lss.Separator, lss.Delimiter)
		p.next()
	}
}

func (p *common) factor(b *exprBuilder) {
	switch p.sym.Code {
	case lss.String:
		val := &ir.ConstExpr{}
		if len(p.sym.Str) == 1 && p.sym.StringOpts.Apos { //do it symbol
			val.Type = types.CHAR
			val.Value = rune(p.sym.Str[0])
			b.push(val)
			p.next()
		} else { //do string later
			val.Type = types.STRING
			val.Value = p.sym.Str
			b.push(val)
			p.next()
		}
	case lss.Number:
		t, v := p.number()
		e := &ir.ConstExpr{Type: t, Value: v}
		b.push(e)
		p.next()
	case lss.Undef:
		val := &ir.ConstExpr{}
		val.Type = types.ANY
		b.push(val)
		p.next()
	case lss.Im:
		p.next()
		p.factor(b)
		p.pass(lss.Separator)
		b.push(&ir.Monadic{Op: ops.Im})
	case lss.Ident:
		id := p.ident()
		var fid string
		var s *ir.SelectExpr
		sb := &selectBuilder{tgt: b.tgt, marker: p}
		p.next()
		if p.is(lss.Period) {
			if u := b.tgt.unit.Variables[id]; u != nil {
				if u.Type.Basic {
					p.mark("only foreign types are selectable")
				}
				p.next()
				p.expect(lss.Ident, "foreign variable expected")
				fid = p.ident()
				p.next()
				s = sb.foreign(id, fid)
			} else {
				p.mark("variable not found")
			}
		} else {
			fid = id
			id = b.tgt.unit.Name
			m := b.marker.FutureMark()
			if c := b.tgt.unit.Const[fid]; c != nil {
				s = &ir.SelectExpr{Const: c}
			} else if v := b.tgt.unit.Variables[fid]; v != nil {
				s = &ir.SelectExpr{Var: v}
			} else if c == nil && b.forward(func() { //forward constant case
				if c := b.tgt.unit.Const[fid]; c != nil {
					s.Const = c
				} else {
					m.Mark("identifier `" + fid + "` not found")
				}
			}) {
				s = &ir.SelectExpr{}
			} else {
				p.mark("identifier `" + fid + "` not found")
			}
		}
		assert.For(s != nil, 60)
		p.inside(sb)
		b.push(sb.merge(s))
	case lss.True, lss.False:
		val := &ir.ConstExpr{}
		val.Type = types.BOOLEAN
		val.Value = (p.sym.Code == lss.True)
		b.push(val)
		p.next()
	case lss.Null:
		val := &ir.ConstExpr{}
		val.Type = types.TRILEAN
		b.push(val)
		p.next()
	case lss.Not:
		p.next()
		p.factor(b)
		p.pass(lss.Separator)
		b.push(&ir.Monadic{Op: ops.Not})
	case lss.Lparen:
		p.next()
		expr := &exprBuilder{tgt: b.tgt, marker: b.marker}
		expr.fwd = append(expr.fwd, func() {
			for i, x := range expr.fwd {
				if i > 0 {
					x()
				}
			}
		})
		p.expression(expr)
		for _, f := range expr.fwd {
			f()
		}
		p.expect(lss.Rparen, ") expected", lss.Separator)
		p.next()
		b.push(expr)
	case lss.Colon:
		//skip for the parents
	default:
		p.mark(p.sym, " not an expression")
	}
}

func (p *common) cpx(b *exprBuilder) {
	p.factor(b)
	p.pass(lss.Separator)
	switch op := p.sym.Code; op {
	case lss.Ncmp, lss.Pcmp:
		p.next()
		p.pass(lss.Separator)
		if p.sym.Code != lss.Im {
			p.factor(b)
		} else {
			p.mark("imaginary operator not expected")
		}
		b.push(&ir.Dyadic{Op: ops.Map(op)})

	}
}

func (p *common) power(b *exprBuilder) {
	p.cpx(b)
	for stop := false; !stop; {
		p.pass(lss.Separator)
		switch op := p.sym.Code; op {
		case lss.ArrowUp:
			p.next()
			p.pass(lss.Separator)
			p.cpx(b)
			b.push(&ir.Dyadic{Op: ops.Map(op)})
		default:
			stop = true
		}
	}
}

func (p *common) product(b *exprBuilder) {
	p.pass(lss.Separator)
	p.power(b)
	for stop := false; !stop; {
		p.pass(lss.Separator)
		switch op := p.sym.Code; op {
		case lss.Times, lss.Div, lss.Mod, lss.Divide, lss.And:
			p.next()
			p.pass(lss.Separator)
			p.power(b)
			b.push(&ir.Dyadic{Op: ops.Map(op)})
		default:
			stop = true
		}
	}
}

func (p *common) quantum(b *exprBuilder) {
	switch {
	case p.is(lss.Minus):
		p.next()
		p.pass(lss.Separator)
		p.product(b)
		b.push(&ir.Monadic{Op: ops.Neg})
	default:
		p.pass(lss.Separator)
		p.product(b)
	}
	for stop := false; !stop; {
		p.pass(lss.Separator)
		switch op := p.sym.Code; op {
		case lss.Plus, lss.Minus, lss.Or:
			p.next()
			p.pass(lss.Separator)
			p.product(b)
			b.push(&ir.Dyadic{Op: ops.Map(op)})
		default:
			stop = true
		}
	}
}

func (p *common) cmp(b *exprBuilder) {
	p.pass(lss.Separator)
	p.quantum(b)
	p.pass(lss.Separator)
	switch op := p.sym.Code; op {
	case lss.Equal, lss.Nequal, lss.Geq, lss.Leq, lss.Gtr, lss.Lss:
		p.next()
		p.pass(lss.Separator)
		p.quantum(b)
		b.push(&ir.Dyadic{Op: ops.Map(op)})
	}

}

func (p *common) expression(b *exprBuilder) {
	p.pass(lss.Separator)
	p.cmp(b)
	if p.await(lss.Quest, lss.Separator, lss.Delimiter) {
		p.next()
		p.pass(lss.Separator, lss.Delimiter)
		p.expression(b)
		p.expect(lss.Colon, "expected `:` symbol", lss.Separator, lss.Delimiter)
		p.next()
		p.pass(lss.Separator, lss.Delimiter)
		p.expression(b)
		b.push(&ir.Ternary{})
	}

}
