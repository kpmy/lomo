package loom

import (
	"bufio"
	"bytes"
	"github.com/kpmy/ypk/assert"
	"github.com/kpmy/ypk/halt"
	"leaf/leaf"
	"log"
	"lomo/ir"
	"lomo/ir/types"
	"lomo/loco/lpp"
	_ "lomo/loco/lpp/p"
	"lomo/loco/lss"
	"math/rand"
	"time"
)

const STD = `
(* some standard units served by runtime *)
UNIT RND
	VAR res+, n- INTEGER
	INFIX res n
END RND

UNIT LD
	VAR res+ UNIT; name- STRING
	INFIX res name
END LD

UNIT LEAF
	VAR out+ ANY; mod- STRING; in- ANY
	INFIX out mod in
END LEAF
`

type stdRule interface {
	ir.Rule
	do(*Unit, object)
}

type stdRnd struct {
}

func (r *stdRnd) Show() string { return "std rnd" }

func (r *stdRnd) do(this *Unit, o object) {
	n := get(this.objects["n"]).toInt()
	n.Rand(rand.New(rand.NewSource(time.Now().UnixNano())), n)
	set(o, &value{typ: types.INTEGER, val: ThisInt(n)})
}

type stdLd struct{}

func (r *stdLd) Show() string { return "std ld" }

func (r *stdLd) do(this *Unit, o object) {
	n := get(this.objects["name"]).toStr()
	if u := this.loader(n); u != nil {
		set(o, &value{typ: types.UNIT, val: NewRef(u)})
	} else {
		halt.As(100, "unresolved unit ", n)
	}
}

type stdLeaf struct{}

func (r *stdLeaf) Show() string { return "std leaf" }

func (r *stdLeaf) do(this *Unit, o object) {
	mod := get(this.objects["mod"]).toStr()
	if loader, err := leaf.Prepare(mod, true); loader != nil {
		m, starter := loader()
		ch := m.Input()
		log.Println("start", mod)
		go func() { starter(); m.Stop() }()
		log.Println("started", mod)
		ch <- map[interface{}]interface{}{"type": "sig", "sig": "lomo", "data": "ping"}
		log.Println("send")

		set(o, &value{typ: types.UNIT, val: &Any{}})
	} else {
		halt.As(100, "leaf machine not started ", err)
	}
}

func stdUnit(f ir.ForeignType, ld Loader) *Unit {
	fake := ir.NewUnit(f.Name())
	fake.Variables = f.Variables()
	fake.Infix = f.Infix()
	switch f.Name() {
	case "RND":
		fake.Rules["res"] = &stdRnd{}
	case "LD":
		assert.For(ld != nil, 20)
		fake.Rules["res"] = &stdLd{}
	case "LEAF":
		assert.For(ld != nil, 20)
		fake.Rules["out"] = &stdLeaf{}
	default:
		halt.As(100, "unknown standard unit ", f.Name())
	}
	return &Unit{code: fake, loader: ld}
}

func precompile() {
	buf := bytes.NewBufferString(STD)
	sc := lss.ConnectTo(bufio.NewReader(buf))
	var err error
	for err == nil {
		p := lpp.ConnectToUnit(sc, func(string) ir.ForeignType { panic(0) })
		var u *ir.Unit
		if u, err = p.Unit(); u != nil && err == nil {
			lpp.Std[u.Name] = ir.NewForeign(u)
		}
	}

}

func init() {
	precompile()
}
