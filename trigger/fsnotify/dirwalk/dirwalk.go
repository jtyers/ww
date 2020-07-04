package dirwalk

import (
	"fmt"
	"os"
	"path/filepath"

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
			basename := filepath.Base(osPathname)
			for _, nameExclude := range nameExcludes {
				match, err := filepath.Match(nameExclude, basename)
				if err != nil {
					return fmt.Errorf("matching %q: %v", nameExclude, err)
				}

				if match {
					fmt.Printf("skipping %s\n", osPathname)
					return nil
				}
			}
			for _, pathExclude := range pathExcludes {
				match, err := filepath.Match(pathExclude, osPathname)
				if err != nil {
					return fmt.Errorf("matching %q: %v", pathExclude, err)
				}

				if match {
					fmt.Printf("skipping %s\n", osPathname)
					return nil
				}
			}

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
