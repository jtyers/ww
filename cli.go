package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mkideal/cli"
)

type argT struct {
	cli.Helper

	Interval int `cli:"interval,n" dft:"2" usage:"run command every X seconds (ignored when --watch is specified)"`

	Shell bool `cli:"shell,s" usage:"run command inside a shell"`

	Highlights []string `cli:"highlight,H" usage:"highlight specified text in output"`
}

func parseArgs() WWConfig {
	config := WWConfig{}

	cli.Run(new(argT), func(ctx *cli.Context) error {
		a, _ := ctx.Argv().(*argT)

		//ctx.JSONln(ctx.Argv())
		//ctx.JSONln(ctx.Args())

		config.Command = ctx.Args()[0]
		config.Args = ctx.Args()[1:]

		if a.Interval > 0 {
			Interval, err := time.ParseDuration(fmt.Sprintf("%ds", a.Interval))
			if err != nil {
				die(nil, "invalid --interval: %v", err)
			}

			config.Trigger = &IntervalWWTrigger{Interval}
		}

		if a.Shell {
			currentShell, ok := os.LookupEnv("SHELL")
			if !ok {
				currentShell = "/bin/sh"
			}

			newArgs := []string{"-c", config.Command}

			for _, arg := range config.Args {
				newArgs = append(newArgs, arg)
			}

			config.Command = currentShell
			config.Args = newArgs
		}

		return nil
	})

	fmt.Printf("config: %#v\n", config)
	return config
}
