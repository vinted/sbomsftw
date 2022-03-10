package handlers

import (
	"fmt"
	"path/filepath"
)

const (
	gemfile          = "Gemfile"
	gemfileLock      = "Gemfile.lock"
	bootstrapCommand = "bundler install"
	bomGenTemplate   = "cyclonedx-ruby -p %s --output /dev/stdout 2>/dev/null | sed '$d'"
)

type Bundler struct {
	Executor CLIExecutor
}

func (b Bundler) MatchFile(filename string) bool {
	return filename == gemfile || filename == gemfileLock
}

func (b Bundler) GenerateBOM(bomRoot string) (string, error) {
	if filepath.Base(bomRoot) == gemfile {
		if err := b.Executor.Cd(filepath.Dir(bomRoot)); err != nil {
			return "", err
		}
		if _, err := b.Executor.ShellOut(bootstrapCommand); err != nil {
			return "", err
		}
	}
	return b.Executor.ShellOut(fmt.Sprintf(bomGenTemplate, filepath.Dir(bomRoot)))
}
