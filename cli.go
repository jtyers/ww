package main

//go:generate slice -dir slice -type string -package slice

import (
	"fmt"
	"os"
	"time"

	"github.com/jtyers/ww/slice"
	"github.com/mkideal/cli"
)

type argT struct {
	cli.Helper

	Interval int `cli:"interval,n" dft:"2" usage:"run command every X seconds (ignored when --watch is specified)"`

	Shell bool `cli:"shell,s" usage:"run command inside a shell"`

	Highlights []string `cli:"color,colour,c" usage:"highlight specified text in output"`
}

func parseArgs() WWConfig {
	config := WWConfig{}

	// Prepend $WW_DEFAULT_ARGS to our args list then process them all as normal
	args := slice.NewStringSlice([]string{os.Args[0]}).
		Concat(GetArgsFromEnvironment()).
		Concat(os.Args[1:]).
		Value()

	res := cli.RunWithArgs(new(argT), args, func(ctx *cli.Context) error {
		a, _ := ctx.Argv().(*argT)

		//ctx.JSONln(ctx.Argv())
		//ctx.JSONln(ctx.Args())

		if len(ctx.Args()) == 0 {
			return fmt.Errorf("command required")
		}

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

		highlights := map[string]string{}
		if len(a.Highlights) > 0 {
			for _, highlight := range a.Highlights {
				highlights[highlight] = "[red]"
			}
		}

		config.Highlighter = NewHighlighter(highlights) // always set this so it is not nil

		return nil
	})

	if res != 0 {
		os.Exit(res)
	}

	fmt.Printf("config: %#v\n", config)
	return config
}
