package dirwalk

import (
	"fmt"
	"os"

	"github.com/karrick/godirwalk"
)

// WalkDirectory walks a directory hierarchy and returns a full list of files and directories under that directory
// (including the directory itself). You can also specify to ignore particular names, or particular paths. Matching
// is done using filepath.Match, which supports shell-style globbing.
func WalkDirectory(dirname string, nameExcludes []string, pathExcludes []string, followSymlinks bool) ([]string, error) {
	result := []string{}
	err := godirwalk.Walk(dirname, &godirwalk.Options{

		// Callback is called for every node visited. The dirent passed contains a name, but not a full file Stat.
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			result = append(result, osPathname)
			return nil
		},

		Unsorted: true, // (optional) set true for faster yet non-deterministic enumeration (see godoc)

		FollowSymbolicLinks: followSymlinks,

		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			fmt.Fprintf(os.Stderr, "skipping %s: %v", osPathname, err)
			return godirwalk.SkipNode
		},
	})

	return result, err
}
