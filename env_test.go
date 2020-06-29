package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetArgsFromEnvironment(t *testing.T) {
	var tests = []struct {
		name   string
		input  string
		output []string
	}{
		{
			"should split on spaces",
			"-c word -s --colour hello",
			[]string{"-c", "word", "-s", "--colour", "hello"},
		},
		{
			"should split on tabs",
			"-c word\t-s --colour\thello",
			[]string{"-c", "word", "-s", "--colour", "hello"},
		},
		{
			"should consider quoted args together",
			"-c word\t-s --colour \"hello world\" -c \"foo\" -c \"another.js\"",
			[]string{"-c", "word", "-s", "--colour", "hello world", "-c", "foo", "-c", "another.js"},
		},
		{
			"should convert empty quote marks to empty string",
			"-c \"\"",
			[]string{"-c", ""},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			os.Setenv(DefaultArgsEnvKey, test.input)

			// when
			result := GetArgsFromEnvironment()

			// then

			require.Equal(t, test.output, result)
		})
	}
}
