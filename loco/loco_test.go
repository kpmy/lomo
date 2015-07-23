package main

import (
	"bufio"
	"lomo/ir"
	"lomo/ir/target"
	_ "lomo/ir/target/st"
	"lomo/loco/lpp"
	_ "lomo/loco/lpp/p"
	"lomo/loco/lss"
	"lomo/loom"
	"os"
	"sync"
	"testing"
	"time"
)

func TestBasics(t *testing.T) {
	if f, err := os.Open("Basic.lomo"); err == nil {
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
						if f, err := os.Open(name + ".ui"); err == nil {
							defer f.Close()
							ret = target.Impl.OldCode(f)
						}
						return
					}
					m := loom.New(ld)
					m.Init("Top")
					wg := &sync.WaitGroup{}
					m.Start(wg)
					wg.Wait()
					m.Start(wg)
					wg.Wait()
					time.Sleep(100 * time.Millisecond)
					m.Stop()
				}
			}
		}
	}
}
