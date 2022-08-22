package collectors

import "path/filepath"

/*
SquashToDirs - Squash file-paths to directories.
E.g. given the following input:

	[]string{
		"/tmp/test/go.mod",
		"/tmp/test/go.sum",
		"/tmp/inner-dir/go.mod",
		"/tmp/inner-dir/go.sum",
		"/tmp/inner-dir/deepest-dir/go.mod",
	}

this function will return a slice of:

	[]string{
		"/tmp/test",
		"/tmp/inner-dir",
		"/tmp/inner-dir/deepest-dir",
	}
*/
func SquashToDirs(pathsToSquash []string) []string {
	dirsToFiles := make(map[string][]string)

	for _, r := range pathsToSquash {
		dir := filepath.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(r))
	}
	squashed := make([]string, 0, len(dirsToFiles))

	for dir := range dirsToFiles {
		squashed = append(squashed, dir)
	}

	return squashed
}

/*
SplitPaths - Split filesystem paths to directory -> file mappings.

	E.g. given the following input: []string{
			"/tmp/test/go.mod",
			"/tmp/test/go.sum",
			"/tmp/inner-dir/go.mod",
			"/tmp/inner-dir/go.sum",
			"/tmp/inner-dir/deepest-dir/go.mod",
		}

this function will return a map of:

	[
		"/tmp/test" => ["go.mod", "go.sum"],
		"/tmp/inner-dir" => ["go.mod", "go.sum"],
		"/tmp/inner-dir/deepest-dir" => ["go.mod"],
	]
*/
func SplitPaths(bomRoots []string) map[string][]string {
	dirsToFiles := make(map[string][]string)

	for _, r := range bomRoots {
		dir := filepath.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(r))
	}

	return dirsToFiles
}
