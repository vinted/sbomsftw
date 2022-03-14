package sboms

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

type BOMRootMatcher func(string) bool

/*
FindRoots walks the given file-system path (preferably a git repository) and
collects relative BOM roots using the provided BOMRootMatcher function.
A BOM root is considered a file that can be used by cyclonedx-tools to create SBOMs. E.g.
Gemfile.lock or go.mod. Empty results might be returned if the BOMRootMatcher doesn't support
the repository or if the repository walk fails. Which can happen in some rare cases. E.g.
missing permissions.
*/
func FindRoots(fileSystem fs.FS, predicate BOMRootMatcher) ([]string, error) {
	var roots []string
	err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("unable to walk file system path: %w", err)
		}
		if basename := filepath.Base(path); predicate(basename) && !d.IsDir() {
			roots = append(roots, path)
		}
		return nil
	})
	return roots, err
}

/*RelativeToAbsoluteRoots takes in a slice of relative BOM roots (as returned from FindRoots) function
and a parent directory. Returns absolute roots prefixed with parent dir. E.g. given relative roots:
[]string{
		"Packages",
		"Packages.lock",
		"inner-dir/Packages.lock",
		"inner-dir/deepest-dir/Packages.lock",
	}
and a parent directory of '/tmp/test-repository' returns
[]string{
		"/tmp/test-repository/Packages",
		"/tmp/test-repository/Packages.lock",
		"/tmp/test-repository/inner-dir/Packages.lock",
		"/tmp/test-repository/inner-dir/deepest-dir/Packages.lock",
	}
*/
func RelativeToAbsoluteRoots(relativeRoots []string, parentDir string) []string {
	var absoluteRoots []string
	for _, relativeRoot := range relativeRoots {
		absoluteRoots = append(absoluteRoots, filepath.Join(parentDir, relativeRoot))
	}
	return absoluteRoots
}
