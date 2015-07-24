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
	"runtime"
)

var name string

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
						for i := 0; i < 20; i++ {
							mm := loom.Init("Top", ld)
							loom.Do(mm, old).Wait()
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
