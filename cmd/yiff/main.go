package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/msyrus/yiff"
)

func main() {
	var validate bool
	flag.BoolVar(&validate, "validate", false, "only checks validation")
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		os.Exit(1)
	}
	if !validate && len(files) != 2 {
		os.Exit(2)
	}

	fls := []io.Reader{}
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Fatalf("can't open %s; error: %v", file, err)
		}
		fls = append(fls, f)
	}

	if validate {
		for i, fl := range fls {
			if _, err := yiff.Parse(fl); err != nil {
				log.Printf("%s is not valid; error: %v", files[i], err)
			}
		}
		return
	}

	if _, err := yiff.Diff(fls[0], fls[1]); err != nil {
		log.Fatal(err)
	}
}
