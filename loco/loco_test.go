package main

import (
	"bufio"
	"github.com/kpmy/lomo/ir"
	"github.com/kpmy/lomo/ir/target"
	_ "github.com/kpmy/lomo/ir/target/st/z"
	"github.com/kpmy/lomo/loco/lpp"
	_ "github.com/kpmy/lomo/loco/lpp/p"
	"github.com/kpmy/lomo/loco/lss"
	"github.com/kpmy/lomo/loom"
	"os"
	"runtime"
	"testing"
)

func TestBasics(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
					cache := make(map[string]*ir.Unit)
					ld := func(name string) (ret *ir.Unit) {
						if ret = cache[name]; ret == nil {
							if f, err := os.Open(name + ".ui"); err == nil {
								ret = target.Impl.OldCode(f)
								cache[name] = ret
							}
							f.Close()
						}
						return
					}
					var old loom.Cluster
					for i := 0; i < 100; i++ {
						mm := loom.Init("Top", ld)
						loom.Do(mm, nil, old).Wait()
						loom.Close(mm).Wait()
						old = mm
					}
				}
			}
		}
	}
}
