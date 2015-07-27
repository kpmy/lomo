package loom

import (
	"bufio"
	"bytes"
	"github.com/kpmy/ypk/halt"
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

func stdUnit(f ir.ForeignType) *Unit {
	fake := ir.NewUnit(f.Name())
	fake.Variables = f.Variables()
	fake.Infix = f.Infix()
	switch f.Name() {
	case "RND":
		fake.Rules["res"] = &stdRnd{}
	default:
		halt.As(100, "unknown standard unit ", f.Name())
	}
	return &Unit{code: fake}
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
