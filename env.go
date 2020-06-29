package main

import (
	"os"
	"regexp"
)

var DefaultArgsEnvKey = "WW_DEFAULT_ARGS"
var WordSplit = regexp.MustCompile("\"(.*)\"|([^\\s]+)")

// GetArgsFromEnvironment reads WW_DEFAULT_ARGS and produces a string slice containing those args.
func GetArgsFromEnvironment() []string {
	v := os.Getenv(DefaultArgsEnvKey)
	result := []string{}
	for _, match := range WordSplit.FindAllStringSubmatch(v, -1) {
		if match[1] != "" {
			result = append(result, match[1])

		} else if match[2] != "" {
			result = append(result, match[2])
		}
	}

	return result
}
