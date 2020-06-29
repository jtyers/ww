package main

import (
	"regexp"
)

// tview colour tag which resets foreground/background/flags.
var RESET = "[-:-:-]"

type highlightItem struct {
	Regexp   *regexp.Regexp
	ColorTag string
}

type Highlighter struct {
	highlightItems []highlightItem
}

// NewHighlighter creates a new Highlighter with the passed-in map of
// strings to highlight. The key is the value to highlight, and the value
// is the colour to change to. The value should be a full tcell tag (e.g. "[red]").
func NewHighlighter(highlights map[string]string) *Highlighter {
	highlightItems := []highlightItem{}

	for k, v := range highlights {
		highlightItems = append(highlightItems, highlightItem{
			// Prepend '(?i)' to our regex, which sets the case-insensitive flag.
			Regexp:   regexp.MustCompile("(?i)" + regexp.QuoteMeta(k)),
			ColorTag: v,
		})
	}

	return &Highlighter{highlightItems}
}

func (h *Highlighter) Highlight(input string) string {
	for _, item := range h.highlightItems {
		input = item.Regexp.ReplaceAllString(input, item.ColorTag+"${0}"+RESET)
	}

	return input
}
