package main

import (
	"bufio"
	"flag"
	"github.com/kpmy/lomo/ir"
	"github.com/kpmy/lomo/ir/target"
	_ "github.com/kpmy/lomo/ir/target/st/z"
	"github.com/kpmy/lomo/loco/lpp"
	_ "github.com/kpmy/lomo/loco/lpp/p"
	"github.com/kpmy/lomo/loco/lss"
	"github.com/kpmy/lomo/loom"
	"log"
	"os"
)

var name string

func init() {
	flag.StringVar(&name, "source", "Simple", "-source=name")
}

func resolve(name string) (ret ir.ForeignType) {
	if fname := name + ".ud"; name != "" {
		if f, err := os.Open(fname); err == nil {
			ret = target.Impl.OldDef(f)
			f.Close()
		}
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
						for i := 0; i < 1; i++ {
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
	loom.Exit()
}
