package collectors

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
	"os"
	fp "path/filepath"
)

//Supported files by this collector
const (
	gemfile     = "Gemfile"
	gemfileLock = "Gemfile.lock"
)

type Ruby struct{ executor ShellExecutor }

func NewRubyCollector() Ruby {
	return Ruby{executor: DefaultShellExecutor{}}
}

//MatchLanguageFiles implements LanguageCollector interface
func (r Ruby) MatchLanguageFiles(isDir bool, filepath string) bool {
	if isDir { //Return false immediately - bundler only supports Gemfile & Gemfile.lock files
		return false
	}
	filename := fp.Base(filepath)
	return filename == gemfile || filename == gemfileLock
}

//String implements LanguageCollector interface
func (r Ruby) String() string {
	return "ruby collector"
}

//BootstrapLanguageFiles implements LanguageCollector interface
func (r Ruby) BootstrapLanguageFiles(bomRoots []string) []string {
	const bootstrapCmd = "bundler install ||  bundler _1.9_ install || bundler _1.17.3_ install"
	for dir, files := range bomtools.DirsToFiles(bomRoots) {
		if len(files) == 1 && files[0] == gemfile {
			/*
				BootstrapLanguageFiles by running bundler install. This runs two versions of bundler.
				Latest bundler and 1.17.3 bundler, this is needed for compatability reasons
				when working with old ruby projects.
			*/

			if _, err := r.executor.shellOut(dir, bootstrapCmd); err != nil {
				fmt.Fprintf(os.Stderr, "%s: can't BootstrapLanguageFiles %s - %s\n", r, dir, err)
				continue
			}
		}
	}
	return bomRoots
}

//GenerateBOM implements LanguageCollector interface
func (r Ruby) GenerateBOM(bomRoot string) (*cdx.BOM, error) {
	const language = "ruby"
	return r.executor.bomFromCdxgen(bomRoot, language)
}
