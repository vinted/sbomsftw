package generators

/*
generators package provides implementation of various package managers that
are able to generate SBOMs from a give file system path
*/

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

const (
	gemfile     = "Gemfile"
	gemfileLock = "Gemfile.lock"
)

type Bundler struct{}

func (b Bundler) MatchPredicate(filename string) bool {
	return filename == gemfile || filename == gemfileLock
}

func (b Bundler) String() string {
	return "Ruby-Bundler"
}

// GenerateBOM TODO Add docs
func (b Bundler) GenerateBOM(bomRoots []string) ([]string, error) {
	const bomGenTemplate = "cyclonedx-ruby -p %s --output /dev/stdout 2>/dev/null | sed '$d'"
	var results []string
	for _, bomRoot := range b.normalizeRoots(bomRoots) {
		if filepath.Base(bomRoot) == gemfile {
			cmd := exec.Command("bash", "-c", "bundler install || bundler _1.17.3_ install")
			cmd.Dir = filepath.Dir(bomRoot)
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("unable to boostrap %s. Reason: %w", bomRoot, err)
			}
		}
		cmd := fmt.Sprintf(bomGenTemplate, filepath.Dir(bomRoot)) //TODO Use a sanitizer here to prevent cmd injection
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			return nil, fmt.Errorf("unable to generate bom for %s. Reason: %w", bomRoot, err)
		}
		results = append(results, string(out))
	}
	return results, nil
}

// normalizeRoots TODO Add docs
func (b Bundler) normalizeRoots(rawRoots []string) (result []string) {
	var dirsToFiles = make(map[string][]string)
	for _, bomRoot := range rawRoots {
		dir := filepath.Dir(bomRoot)
		dirsToFiles[dir] = append(dirsToFiles[dir], filepath.Base(bomRoot))
	}
	for dir, files := range dirsToFiles {
		if len(files) == 1 {
			result = append(result, filepath.Join(dir, files[0]))
			continue
		}
		result = append(result, filepath.Join(dir, gemfileLock))
	}
	return
}
