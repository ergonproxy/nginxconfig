package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli/v2"
)

func format(file string, write bool) error {
	cfg := defaultParseOpts()
	cfg.comments = true
	p := parse(file, cfg)
	if p.Errors != nil {
		return p.Errors[0]
	}
	out := build(p.Config[0].Parsed, 2, false)
	if !write {
		_, err := fmt.Fprint(os.Stdout, out)
		return err
	}
	stat, err := os.Stat(file)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, []byte(out), stat.Mode())
}

func formatCommand() *cli.Command {
	return &cli.Command{
		Name:    "format",
		Aliases: []string{"fmt"},
		Usage:   "formats vince configuration file",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "w",
				Usage: "overwrites the formated output to original file",
			},
		},
		Action: func(ctx *cli.Context) error {
			a := ctx.Args().First()
			if a == "" {
				return errors.New("missing file")
			}
			return format(a, ctx.Bool("w"))
		},
	}
}
