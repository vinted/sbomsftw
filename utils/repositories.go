package utils

import (
	"io/fs"
	"path/filepath"
)

// GatherSBOMRoots walks the given repository path and calls predicate
// function for each file. Predicate functions gives clients a chance to
// tell which particular Package Manager files denote SBOM collection root.
// If a predicate function return true for a give file, the directory of that
// file gets included the return slice.
func GatherSBOMRoots(repoPath string, predicate func(string) bool) ([]string, error) {
	var collectionPaths = make(map[string]bool)
	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if baseName := filepath.Base(path); predicate(baseName) && !d.IsDir() {
			collectionPaths[filepath.Dir(path)] = true
		}
		return nil
	})
	return Keys(collectionPaths), err
}
