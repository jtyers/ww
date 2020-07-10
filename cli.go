package main

//go:generate slice -dir slice -type string -package slice

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/integrii/flaggy"
	"github.com/jtyers/ww/slice"
	"github.com/jtyers/ww/trigger/fsnotify"
	"github.com/jtyers/ww/trigger/interval"
)

func parseArgs() (WWConfig, WWDisplay) {
	config := WWConfig{}

	flWatch := false
	flInterval := 2
	flFullscreen := false
	flShell := false
	flHighlights := []string{}
	flWatchExcludes := []string{".git"} // default value

	flaggy.Bool(&flFullscreen, "f", "fullscreen", "Run command in an ncurses-like full screen view")
	flaggy.Int(&flInterval, "n", "interval", "Run command every X seconds")
	flaggy.Bool(&flShell, "s", "shell", "Run command inside a shell (auto-detected via $SHELL)")
	flaggy.StringSlice(&flHighlights, "c", "color", "Colour (highlight) the given string in output (can be specified multiple times, case-insensitive)")
	flaggy.Bool(&flWatch, "w", "watch", "Watch current directory for changes")
	flaggy.StringSlice(&flWatchExcludes, "x", "exclude", "Exclude files/directories with the given name")

	flaggy.DefaultParser.ShowVersionWithVersionFlag = true
	flaggy.DefaultParser.ShowHelpOnUnexpected = true
	flaggy.DefaultParser.ShowHelpWithHFlag = true

	// Prepend $WW_DEFAULT_ARGS to our args list then process them all as normal
	args := slice.NewStringSlice(GetArgsFromEnvironment()).
		Concat(os.Args[1:]).
		Value()
	flaggy.ParseArgs(args)

	if len(flaggy.TrailingArguments) == 0 {
		flaggy.ShowHelpAndExit("command required")
	}

	config.Command = flaggy.TrailingArguments[0]
	config.Args = flaggy.TrailingArguments[1:]

	if flWatch {
		wd, err := os.Getwd()
		if err != nil {
			die(nil, "getwd: %v", err)
		}

		fsnotifyTrigger, err := fsnotify.NewFsNotifyTrigger(wd, flWatchExcludes)
		if err != nil {
			die(nil, "error creating fsnotify trigger: %v", err)
		}

		config.Trigger = fsnotifyTrigger

	} else if flInterval > 0 {
		i, err := time.ParseDuration(fmt.Sprintf("%ds", flInterval))
		if err != nil {
			die(nil, "invalid --interval: %v", err)
		}

		config.Trigger = &interval.IntervalWWTrigger{Interval: i}
	}

	if flShell {
		currentShell, ok := os.LookupEnv("SHELL")
		if !ok {
			currentShell = "/bin/sh"
		}

		newArgs := []string{
			"-c",
			strings.Join(
				slice.ConcatString([]string{config.Command}, slice.MapString(config.Args, func(s string, n int) string {
					// for every arg, if it contains whitespace, enclose it in quotes
					if strings.Contains(s, " ") || strings.Contains(s, "\t") {
						return "\"" + s + "\""
					}

					return s
				})),
				" ",
			),
		}

		config.Command = currentShell
		config.Args = newArgs
	}

	highlights := map[string]string{}
	if len(flHighlights) > 0 {
		for _, highlight := range flHighlights {
			highlights[highlight] = "[red]"
		}
	}

	config.Highlighter = NewHighlighter(highlights) // always set this so it is not nil

	var display WWDisplay
	if flFullscreen {
		display = &TviewDisplay{config: config}
	} else {
		display = &UiLiveDisplay{config: config}
	}

	return config, display
}
