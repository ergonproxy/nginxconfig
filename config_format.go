package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli/v2"
)

type formatOption struct {
	write      bool
	json       bool
	jsonPretty bool
}

func format(file string, opts formatOption) error {
	cfg := defaultParseOpts()
	cfg.comments = true
	p := parse(file, cfg)
	if p.Errors != nil {
		return p.Errors[0]
	}
	if opts.json {
		b, err := json.Marshal(p.Config[0])
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(b)
		return err
	}
	if opts.jsonPretty {
		b, err := json.MarshalIndent(p.Config[0], "", "  ")
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(b)
		return err
	}
	out := build(p.Config[0].Parsed, 2, false)
	if !opts.write {
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
			&cli.BoolFlag{
				Name:  "json",
				Usage: "writes output as json ast",
			},
			&cli.BoolFlag{
				Name:  "json-pretty",
				Usage: "writes output as indented json ast",
			},
		},
		Action: func(ctx *cli.Context) error {
			a := ctx.Args().First()
			if a == "" {
				return errors.New("missing file")
			}
			return format(a, formatOption{
				write:      ctx.Bool("w"),
				json:       ctx.Bool("json"),
				jsonPretty: ctx.Bool("json-pretty"),
			})
		},
	}
}
