package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHighlight(t *testing.T) {
	sample := `The licenses for most software are designed to take away your
freedom to share and change it.  By contrast, the GNU General Public
License is intended to guarantee your freedom to share and change free
software--to make sure the software is free for all its users.`

	var tests = []struct {
		name       string
		highlights map[string]string
		input      string
		output     string
	}{
		{
			"should not change input when highlights is empty",
			map[string]string{},
			sample,
			sample,
		},
		{
			"should handle nil highlights",
			nil,
			sample,
			sample,
		},
		{
			"should change input for specified highlights",
			map[string]string{
				"word": "[red]",
				"this": "[green]",
			},
			"this word or this word",
			"[green]this[-:-:-] [red]word[-:-:-] or [green]this[-:-:-] [red]word[-:-:-]",
		},
		{
			"should change input for specified highlights ignoring case",
			map[string]string{
				"word": "[red]",
			},
			"this word or this WORD or this Word or this wORD",
			"this [red]word[-:-:-] or this [red]WORD[-:-:-] or this [red]Word[-:-:-] or this [red]wORD[-:-:-]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			highlighter := NewHighlighter(test.highlights)

			// when
			result := highlighter.Highlight(test.input)

			// then

			require.Equal(t, test.output, result)
		})
	}
}
