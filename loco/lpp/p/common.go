package p

import (
	"fmt"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"lomo/loco/lss"
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
