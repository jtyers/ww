# ww - a better `watch`

`watch` has been part of the furniture for us working on Linux/UNIX for years. A classic example of a simple concept, implemented simply and elegantly.

But...  it could benefit from a few extra features to make it even better. `ww` began as a block in my shell config but I've now spit it out here to share with others.

`ww` brings you `watch` as you know, but with some extras:

* runs as an alias, so you can use your shell aliases to watch

* countdown number of seconds left before next execution

* coloured bar along the top to clearly indicate success/failure

* watch for changes to files rather than running on an interval

* highlight particular words in the output

* use `-1` to run until the command succeeds

## Installation

Clone this repository, and then add this to `~/.zshrc`:

```
# replace path here with wherever you checked out this repo
source ~/ww/ww
```

## Quick start

Run `cmd` every 3 seconds. If you omit `-n`, it defaults to `10`. A bar is drawn across the top of the screen to show the outcome of each run. `cmd` can be a shell alias, making it rather powerful if you need to run your own aliases over the top.

```
ww -n 3 cmd
```

Run `cmd` every 3 seconds until it succeeds, then quit. `--once` can be used instead of `-1`.

```
ww -n 3 -1 cmd
```

Run `cmd`, and re-run if any files in the current directory are written to, renamed or deleted. This uses `inotifywait -r` under the hood currently, so only supports Linux, and may be slow in larger directory hierarchies.

```
ww -w cmd
```

Run `cmd`, and highlight and the words **error** and **fail** in the output. Repeat `-c` to add more words. Highlighting is case-insensitive.

```
ww -w -c error -c fail cmd
```

If you need to pass arguments to `cmd`, be sure to use `--` so that `ww` doesn't gobble them up. For example:

```
ww -n 5 -- df -h
```

Use `WW_DEFAULT_ARGS` to set default arguments. For example:
```
export WW_DEFAULT_ARGS="-c err -c fail"

# ww will now always highlight "err" and "fail" in output
ww my-command
```

## Usage

```
ww - a better watch

usage: ww [opts] CMD

  --once, -1
    quit after CMD finishes successfully (exit code 0)

  --color, --colour, -c WORD
    highlight instances of WORD in output (can be repeated)

  --interval, -n N
    refresh every N seconds (ignored if -w is specified)

  --watch, -w
    refresh when files in the current directory are changed
    (required inotifywatch to be installed)

If WW_DEFAULT_ARGS is set, this can contain default arguments, processed before command line arguments on every invocation.
```

`ww` works with `zsh`, tested on v5.7.1. I haven't tested in other shells, but would welcome feedback and PRs to enable `ww` for those too.


## Contributing

Contributions are very welcome, please raise a PR and state clearly what problem you're trying to solve. Keep in mind that `ww` is designed to be light.
