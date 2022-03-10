package sboms

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

const (
	traversalFailedTemplate     = "FS Traversal using %s failed! Reason: %v"
	bomGenerationFailedTemplate = "BOM Generation for %s failed! Reason: %v"
)

type Handler interface {
	String() string
	MatchFile(string) bool
	GenerateBOM(string) (string, error)
}

type Logger interface {
	LogError(error)
	LogMessage(string)
}

// Collect walks the given file-system path (preferably a git repository) and
// collects Software Bill Of Materials (SBOMs) using the provided Handler.
// Empty results might be returned if the Handler doesn't support the repository
// or if the repository walk fails. Which can happen in some rare cases. E.g.
// missing permissions.
//
// As a third argument, this function also takes a logger. So any errors that
// occur while collecting SBOMs are logged by this function.
func Collect(fileSystem fs.FS, handler Handler, logger Logger) []string {
	bomRoots, err := findBOMRoots(fileSystem, handler)
	if err != nil {
		logger.LogError(fmt.Errorf(traversalFailedTemplate, handler, err))
		return []string{}
	}
	for _, bomRoot := range bomRoots {
		if _, err := handler.GenerateBOM(bomRoot); err != nil {
			logger.LogError(fmt.Errorf(bomGenerationFailedTemplate, handler, err))
			continue
		}
	}
	return []string{}
}

func findBOMRoots(fileSystem fs.FS, handler Handler) ([]string, error) {
	var roots []string
	err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if basename := filepath.Base(path); handler.MatchFile(basename) && !d.IsDir() {
			roots = append(roots, path)
		}
		return nil
	})
	return roots, err
}
