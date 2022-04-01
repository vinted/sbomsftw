package boms

import (
	"fmt"
	fp "path/filepath"
)

const (
	//Supported lockfiles
	gemfile     = "Gemfile"
	gemfileLock = "Gemfile.lock"
	//BOM gen & bootstrapping commands
	rubyCDXGenCmd    = "export FETCH_LICENSE=true && cdxgen --type ruby"
	rubyBootstrapCmd = "bundler install || bundler _1.9_ install || bundler _1.17.3_ install"
)

type Ruby struct{ executor CLIExecutor }

func NewRubyCollector() Ruby {
	return Ruby{executor: defaultCLIExecutor{}}
}

//matchPredicate implements BOMCollector interface
func (r Ruby) matchPredicate(isDir bool, filepath string) bool {
	if isDir { //Return false immediately - bundler only supports Gemfile & Gemfile.lock files
		return false
	}
	filename := fp.Base(filepath)
	return filename == gemfile || filename == gemfileLock
}

//String implements BOMCollector interface
func (r Ruby) String() string {
	return "Ruby-Bundler"
}

func (r Ruby) bootstrap(bomRoots []string) []string {
	var bootstrappedRoots []string
	var dirsToFiles = make(map[string][]string)
	for _, r := range bomRoots {
		dir := fp.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], fp.Base(r))
	}
	for dir, files := range dirsToFiles {
		shouldBootstrap := len(files) == 1 && files[0] == gemfile
		if shouldBootstrap {
			/*
				Bootstrap by running bundler install. This runs two versions of bundler.
				Latest bundler and 1.17.3 bundler, this is needed for compatability reasons
				when working with old ruby projects.
			*/

			if _, err := r.executor.shellOut(dir, rubyBootstrapCmd); err != nil {
				fmt.Println(fmt.Errorf(bootstrapFailedErr, dir, err))
				continue
			}
		}
		bootstrappedRoots = append(bootstrappedRoots, dir)
	}
	return bootstrappedRoots
}

func (r Ruby) generateBOM(bomRoot string) (string, error) {
	return r.executor.executeCDXGen(bomRoot, rubyCDXGenCmd)
}
