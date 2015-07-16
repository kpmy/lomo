package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"lomo/loco/lss"
	"os"
)

var name string

func init() {
	flag.StringVar(&name, "source", "Simple", "-source=name")
}

func main() {
	log.Println("Lomo compiler, pk, 20150716")
	flag.Parse()
	if sname := name + ".lomo"; name != "" {
		if f, err := os.Open(sname); err == nil {
			sc := lss.ConnectTo(bufio.NewReader(f))
			sc.Init(lss.Unit, lss.End)
			for sc.Error() == nil {
				fmt.Println(sc.Get())
			}
		}
	}
}
