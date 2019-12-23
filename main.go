package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ergongate/vince/engine"
)

func main() {
	flag.Parse()
	err := engine.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
