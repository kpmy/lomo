package main

import (
	"bufio"
	"flag"
	"log"
	"lomo/ir"
	"lomo/loco/lpp"
	_ "lomo/loco/lpp/p"
	"lomo/loco/lss"
	"os"
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
				_, err = p.Unit()
			}
		}
	}
}
