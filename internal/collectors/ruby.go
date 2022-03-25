package collectors

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/internal/boms"
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

//CollectBOM implements BOMCollector interface
func (r Ruby) CollectBOM(repoPath string) (*cdx.BOM, error) {
	rootsFound, err := boms.RepoToRoots(repoPath, r.matchPredicate)
	if err != nil {
		return nil, fmt.Errorf("can't to collect BOMs for %s with %s: %w", repoPath, r, err)
	}
	var generatedBOMs []string
	for _, root := range boms.NormalizeRoots(gemfileLock, rootsFound...) {
		bom, err := r.generateBOM(root)
		if err != nil {
			fmt.Printf("BOM collection failed: %s\n", err)
			continue
		}
		generatedBOMs = append(generatedBOMs, bom)
	}
	if len(generatedBOMs) == 0 {
		return nil, errUnsupportedRepo
	}
	mergedBom, err := boms.Merge(boms.JSON, generatedBOMs...)
	if err != nil {
		return nil, err
	}
	return boms.AttachCPEs(mergedBom), nil
}

func (r Ruby) generateBOM(bomRoot string) (string, error) {
	if fp.Base(bomRoot) == gemfile { //Creates Gemfile.lock from Gemfile if needed
		/*
			Bootstrap by running bundler install. This runs two versions of bundler.
			Latest bundler and 1.17.3 bundler, this is needed for compatability reasons
			when working with old ruby projects.
		*/

		if err := r.executor.bootstrap(fp.Dir(bomRoot), rubyBootstrapCmd); err != nil {
			return "", fmt.Errorf(bootstrapFailedErr, bomRoot, err)
		}
	}
	return r.executor.executeCDXGen(fp.Dir(bomRoot), rubyCDXGenCmd)
}
