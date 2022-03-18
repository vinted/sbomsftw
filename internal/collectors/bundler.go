package collectors

/*
collectors package provides implementation of various package managers that
are able to generate SBOMs from a give file system path
*/

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	gemfile     = "Gemfile"
	gemfileLock = "Gemfile.lock"
)

type Bundler struct{}

func (b Bundler) matchPredicate(filename string) bool {
	return filename == gemfile || filename == gemfileLock
}

func (b Bundler) String() string {
	return "Ruby-Bundler"
}

// CollectBOM TODO Add docs
func (b Bundler) CollectBOM(repoPath string) (string, error) {
	rootsFound, err := repoToRoots(repoPath, b.matchPredicate)
	if err != nil {
		return "", fmt.Errorf("can't to collect BOMs for %s with %s: %w", repoPath, b, err)
	}
	var generatedBOMs []string
	for _, root := range normalizeRoots(gemfileLock, rootsFound...) {
		bom, err := b.generateBOM(root)
		if err != nil {
			return "", err
		}
		generatedBOMs = append(generatedBOMs, bom)
	}
	//TODO We will need to return generated bom slice here because multiple handlers will run and result will be merged
	//to a single project before uploading to DT
	return Merge(JSON, generatedBOMs...)
}

func (b Bundler) generateBOM(bomRoot string) (string, error) {
	//Shell-out commands
	const (
		cdxGen         = "export FETCH_LICENSE=true && cdxgen --type ruby"                      //Generate BOM from ruby project
		bundlerInstall = "bundler install || bundler _1.9_ install || bundler _1.17.3_ install" //Creates Gemfile.lock from Gemfile if needed
	)
	//Error templates
	const (
		ErrBootstrap     = "can't boostrap %s: %w"        //Used whenever bundler install fails
		ErrCdxGenNonZero = "can't collect BOM for %s: %w" // Used when cdxgen exits with non 0 status code
		ErrCdxGenFailed  = "can't collect BOM for %s: %s" // Used when cdxgen exits with 0 status code but still errors out
	)
	if filepath.Base(bomRoot) == gemfile {
		/*
			Bootstrap by running bundler install. This runs two versions of bundler.
			Latest bundler and 1.17.3 bundler, this is needed for compatability reasons
			when working with old ruby projects.
		*/
		cmd := exec.Command("bash", "-c", bundlerInstall)
		cmd.Dir = filepath.Dir(bomRoot)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf(ErrBootstrap, bomRoot, err)
		}
	}

	cmd := exec.Command("bash", "-c", cdxGen)
	cmd.Dir = filepath.Dir(bomRoot)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(ErrCdxGenNonZero, bomRoot, err)
	}
	//cdxgen even on failure exits with 0 status code. This is kinda ugly but must compare stdout :/
	if strings.HasPrefix(string(out), "Unable to produce BOM") {
		errMsg := fmt.Sprintf(ErrCdxGenFailed, bomRoot, string(out))
		return "", BOMCollectionFailed(errMsg)
	}
	return string(out), nil
}

//todo implement function to get all repositories
