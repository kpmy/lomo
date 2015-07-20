package main

import (
	"bufio"
	"flag"
	"log"
	"lomo/ir"
	"lomo/ir/target"
	_ "lomo/ir/target/st"
	"lomo/loco/lpp"
	_ "lomo/loco/lpp/p"
	"lomo/loco/lss"
	"lomo/loom"
	"os"
	"time"
)

var name string

func init() {
	flag.StringVar(&name, "source", "Simple", "-source=name")
}

func resolve(name string) (ret *ir.ForeignType) {
	if fname := name + ".ud"; name != "" {
		if f, err := os.Open(fname); err == nil {
			f.Close()
		}
		ret = &ir.ForeignType{Name: name}
	}
	return
}

func main() {
	log.Println("Lomo compiler, pk, 20150716")
	flag.Parse()
	if sname := name + ".lomo"; name != "" {
		if f, err := os.Open(sname); err == nil {
			sc := lss.ConnectTo(bufio.NewReader(f))
			for err == nil {
				p := lpp.ConnectToUnit(sc, resolve)
				var u *ir.Unit
				if u, err = p.Unit(); err == nil {
					if f, err := os.Create(u.Name + ".ui"); err == nil {
						target.Impl.NewCode(u, f)
						f.Close()
					}
					if f, err := os.Create(u.Name + ".ud"); err == nil {
						target.Impl.NewDef(ir.NewForeign(u), f)
						f.Close()
					}
					if u.Name == "Top" {
						ld := func(name string) (ret *ir.Unit) {
							if f, err := os.Open(u.Name + ".ui"); err == nil {
								defer f.Close()
								ret = target.Impl.OldCode(f)
							}
							return
						}
						m := loom.New(ld)
						m.Start("Top")
						time.Sleep(time.Second)
						m.Stop()
					}
				}
			}
		}
	}
	loom.Exit()
}
