package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "vince"
	app.Description = descriptionText
	app.Usage = "Modern reverse proxy for modern traffick"
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

const descriptionText = `
vince is a modern reverse proxy for http,tcp,udp,unix socket .. etc protocols with
modern tools for building and managing high performance , highly available services.
`
