package boms

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type BOMRootMatcher func(bool, string) bool

type NoRootsFoundError string

func (e NoRootsFoundError) Error() string {
	return string(e)
}

/*
findRoots walks the given file-system path (preferably a git repository) and
collects relative BOM roots using the provided BOMRootMatcher function.
A BOM root is considered a file that can be used by cyclonedx-tools to create SBOMs. E.g.
Gemfile.lock or go.mod. Empty results might be returned if the BOMRootMatcher doesn't support
the repository or if the repository walk fails. Which can happen in some rare cases. E.g.
missing permissions.
*/
func findRoots(fileSystem fs.FS, predicate BOMRootMatcher) ([]string, error) {
	var roots []string
	err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("unable to walk file system path: %w", err)
		}
		if predicate(d.IsDir(), path) {
			roots = append(roots, path)
		}
		return nil
	})
	return roots, err
}

/*relativeToAbsoluteRoots takes in a slice of relative BOM roots (as returned from findRoots) function
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
func relativeToAbsoluteRoots(parentDir string, relativeRoots ...string) (absoluteRoots []string) {
	for _, relativeRoot := range relativeRoots {
		absoluteRoots = append(absoluteRoots, filepath.Join(parentDir, relativeRoot))
	}
	return
}

func RepoToRoots(repoPath string, predicate BOMRootMatcher) ([]string, error) {
	relativeRoots, err := findRoots(os.DirFS(repoPath), predicate)
	if err != nil {
		return nil, fmt.Errorf("can't to convert repo to roots: %w", err)
	}
	absoluteRoots := relativeToAbsoluteRoots(repoPath, relativeRoots...)
	if len(absoluteRoots) == 0 {
		return nil, NoRootsFoundError("No BOM roots found for supplied predicate")
	}
	return absoluteRoots, nil
}

// NormalizeRoots TODO Add docs -- this needs to be private or extracted from here preferrably
func NormalizeRoots(preferredFile string, rawRoots ...string) (result []string) {
	var dirsToFiles = make(map[string][]string)
	for _, r := range rawRoots {
		dir := filepath.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(r))
	}
	for dir, files := range dirsToFiles {
		if len(files) == 1 {
			//Just return the single file, doesn't matter if it's a lockfile or not
			result = append(result, filepath.Join(dir, files[0]))
			continue
		}
		//Use preferred file (lockfile) whenever multiple-lockfiles BOM roots are present
		result = append(result, filepath.Join(dir, preferredFile))
	}
	return
}
