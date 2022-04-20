package bomtools

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
			return err
		}
		if d.IsDir() && filepath.Base(path) == ".git" { //todo test this case
			return fs.SkipDir // Don't even traverse to .git directories
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
		return nil, fmt.Errorf("repoToRoots: can't to convert repository : %v", err)
	}
	absoluteRoots := relativeToAbsoluteRoots(repoPath, relativeRoots...)
	if len(absoluteRoots) == 0 {
		return nil, NoRootsFoundError("No BOM roots found for supplied predicate")
	}
	return absoluteRoots, nil
}

func SquashRoots(bomRoots []string) []string {
	var dirsToFiles = make(map[string][]string)
	for _, r := range bomRoots {
		dir := filepath.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(r))
	}
	squashed := make([]string, 0, len(dirsToFiles))
	for dir := range dirsToFiles {
		squashed = append(squashed, dir)
	}
	return squashed
}

//TODO Add docs
func DirsToFiles(bomRoots []string) map[string][]string {
	var dirsToFiles = make(map[string][]string)
	for _, r := range bomRoots {
		dir := filepath.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(r))
	}
	return dirsToFiles
}
