# ww - a better `watch`

![ww in action](demo.gif)

`watch` has been part of the furniture on Linux/UNIX for years. A classic example of a simple concept, implemented simply and elegantly.

But...  it could benefit from a few extra features to make it even better. `ww` began as a block in my shell config but I've now spit it out here to share with others.  `ww` brings you `watch` as you know, but with some extras:

* supports watching shell aliases, and even pipelines

* countdown number of seconds left before next execution

* watch for changes to files rather than running on an interval

* coloured bar along the top to clearly indicate success/failure

* highlight particular words in the output

* watch a command until it succeeds, then exit (handy for scripting, eg ensuring a host is responding to pings before SSHing to it)

## Installation

1. Clone this repository:

```
# change ~/ww to where you want to clone
git clone https://github.com/jtyers/ww.git ~/ww
```

2. Add this to `~/.zshrc`:

```
# replace path here with wherever you checked out this repo
source ~/ww/ww
```

## How to use

Run `cmd` every 3 seconds. If you omit `-n`, it defaults to `10`. `cmd` can be a shell alias or any pipeline, making it rather powerful if you need to run your own aliases over the top.

```
ww -n 3 cmd
```

Run `kubectl get pods | grep foo-bar` every 3 seconds until it succeeds, then quit.

```
# notice that the pipe character should be escaped
ww -n 3 --until -- kubectl get pods \| grep foo-bar

# or include it in quotes
ww -n 3 --until -- "kubectl get pods | grep foo-bar"

```

Run `npm run test`, and re-run if any files in the current directory are written to, renamed or deleted. This uses `inotify` under the hood currently, so only supports Linux, and may be slow in larger directory hierarchies.

```
ww -w npm run test
```

Run `tail log.txt`, and re-run any time files in the current directory are changed. Highlight instances of "error" and "fail" in the output. Repeat `-c <word>` to add more words. Highlighting is case-insensitive.

```
ww -w -c error -c fail tail log.txt
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

For maximum ease of use, I define these aliases to run `ww` in various forms quickly:
```
alias www='ww -w'
alias ww2='ww -n2'
alias ww5='ww -n5'
alias wwu='ww --until'
```

## Usage

```
ww - a better watch

USAGE
  ww [opts] [--] CMD

  --once, -1
    quit after CMD finishes successfully (exit code 0)

  --color, --colour, -c WORD
    highlight instances of WORD in output (can be repeated)

  --interval, -n N
    refresh every N seconds (ignored if -w is specified)

  --watch, -w
    refresh when files in the current directory are changed
    (requires inotifywatch to be installed)

  --watch-wait, -W SECONDS
    when --watch is used, ww will wait a short period after a
    change is detected to allow related I/O operations to complete
    (default: 0.25)

  --no-capture, -n
    allow underlying command to print straight to terminal rather
    than capturing output (used for slower commands, such as find,
    tail -f, etc; in this mode, --color has no effect)

  --until, -u
    wait until CMD has run successfully, then quit (this is just an
    alias for '--no-capture --once')

If WW_DEFAULT_ARGS is set, this can contain default arguments, processed before command line arguments on every invocation.

You can use any shell expansions or aliases in CMD, but remember to escape special characters (see EXAMPLES below).

Examples:
  ww df -h   # run df -h every 10 seconds

  # run 'go test' every 2 seconds, grepping for FAILED 
  # (note the escaped pipe character)
  ww -n 2 -- go test \| grep FAILED
  
  # run 'go test' every time files in the current directory
  # are changed
  ww -w -- go test
  
  # run 'ls ~/foo' continuously, if it fails, retry after 5 seconds, exit when it succeeds
  ww -u -n 5 -- ls ~/foo

```

`ww` works with `zsh`, tested on v5.7.1. Should work with any POSIX-compatible shell, though I haven't tested in other shells. Feedback and PRs welcome!


## Contributing

Contributions are very welcome, please raise a PR and state clearly what problem you're trying to solve. Keep in mind that `ww` is designed to be light and fast.
