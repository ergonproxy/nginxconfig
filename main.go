package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

//go:generate statik -p templates  -f -src ./templates/files/

func main() {
	app := cli.NewApp()
	app.Name = "vince"
	app.Description = descriptionText
	app.Usage = "Modern reverse proxy for modern traffick"
	app.Flags = []cli.Flag{
		&configFlag,
	}
	app.Commands = []*cli.Command{
		formatCommand(),
	}
	app.Action = start
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

var configFlag = cli.StringFlag{
	Name:        "c",
	Usage:       "Configuration directory tree",
	EnvVars:     []string{"VINCE_CONFIG"},
	DefaultText: strings.Join(defaultConfigFiles(), " or "),
}

func defaultWorkDirectories() []string {
	return []string{"/usr/local/vince", " /etc/vince", "/usr/local/etc/vince"}
}

func defaultConfigFiles() []string {
	return []string{"/usr/local/vince/conf/vince.conf", " /etc/vince/conf/vince.conf", "/usr/local/etc/vince/conf/vince.conf"}
}
