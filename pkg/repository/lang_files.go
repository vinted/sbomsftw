package repository

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

type languageFilesMatcher func(bool, string) bool

type noLanguageFilesFoundError string

func (e noLanguageFilesFoundError) Error() string {
	return string(e)
}

var ignoredDirs = regexp.MustCompile(`^(.git|test|tests)$`)

/*
languageFilesByPredicate walks the given file-system path and collects language files using the
provided languageFilesMatcher function. Empty results might be returned if the languageFilesMatcher returns
false for every file passed in or if the repository walk fails. Which can happen in some rare cases. E.g.
missing permissions.
*/
func languageFilesByPredicate(fileSystem fs.FS, predicate languageFilesMatcher) ([]string, error) {
	var languageFiles []string

	err := fs.WalkDir(fileSystem, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && ignoredDirs.MatchString(filepath.Base(path)) {
			return fs.SkipDir // Don't even traverse to .git/test/tests directories
		}
		if predicate(entry.IsDir(), path) {
			languageFiles = append(languageFiles, path)
		}

		return nil
	})

	return languageFiles, err
}

/*
relativeToAbsolutePaths takes in a slice of relative paths and a parent directory.
Returns absolute paths prefixed with parent dir. E.g. given relative paths:

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
func relativeToAbsolutePaths(parentDir string, relativeRoots ...string) (absoluteRoots []string) {
	for _, relativeRoot := range relativeRoots {
		absoluteRoots = append(absoluteRoots, filepath.Join(parentDir, relativeRoot))
	}

	return
}

func findLanguageFiles(repoPath string, predicate languageFilesMatcher) ([]string, error) {
	languageFiles, err := languageFilesByPredicate(os.DirFS(repoPath), predicate)
	if err != nil {
		return nil, fmt.Errorf("findLanguageFiles: can't walk %s: %v", repoPath, err)
	}

	languageFiles = relativeToAbsolutePaths(repoPath, languageFiles...)

	if len(languageFiles) == 0 {
		return nil, noLanguageFilesFoundError("no language files found for the supplied predicate")
	}

	return languageFiles, nil
}
