package collectors

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/internal/boms"
	"os"
	fp "path/filepath"
	"strings"
)

//Bootstrap and BOM generation commands
const (
	jsBootstrapCmd = "pnpm install || npm install || yarn install"
	jsCDXGenCmd    = "export FETCH_LICENSE=true && cdxgen --type javascript"
)

var supportedFiles = []string{"yarn.lock", "bower.json", "package.json", "pnpm-lock.yaml", "package-lock.json"}

type JS struct {
	executor CLIExecutor
}

func NewJSCollector() JS {
	return JS{executor: defaultCLIExecutor{}}
}

//todo document this
func (j JS) matchPredicate(isDir bool, filepath string) bool {
	for _, p := range strings.Split(fp.Dir(filepath), string(os.PathSeparator)) {
		if p == "node_modules" { //Ignore files in node_modules directory
			return false
		}
	}
	for _, f := range supportedFiles {
		if fp.Base(filepath) == f {
			return true
		}
	}
	//top level node_modules as a special case. In rare cases there will be no lockfiles but node_modules dir will be present
	return fp.Base(filepath) == "node_modules" && isDir
}

func (j JS) String() string {
	return "JS/TS-JS"
}

//CollectBOM Implements BOMCollector interface
func (j JS) CollectBOM(repoPath string) (*cdx.BOM, error) {
	rootsFound, err := boms.RepoToRoots(repoPath, j.matchPredicate)
	if err != nil {
		return nil, fmt.Errorf("can't to collect BOMs for %s with %s: %w", repoPath, j, err)
	}

	var generatedBOMs []string
	for _, b := range j.bootstrap(rootsFound) {
		bom, err := j.executor.executeCDXGen(b, jsCDXGenCmd)
		if err != nil {
			fmt.Printf("BOM generation failed: %s\n", err)
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

func (j JS) bootstrap(bomRoots []string) []string {
	var bootstrappedRoots []string
	var dirsToFiles = make(map[string][]string)
	for _, r := range bomRoots {
		dir := fp.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], fp.Base(r))
	}
	for dir, files := range dirsToFiles {
		shouldBootstrap := len(files) == 1 && files[0] == "package.json"
		if shouldBootstrap {
			if err := j.executor.bootstrap(dir, jsBootstrapCmd); err != nil {
				fmt.Println(fmt.Errorf(bootstrapFailedErr, dir, err))
				continue
			}
		}
		bootstrappedRoots = append(bootstrappedRoots, dir)
	}
	return bootstrappedRoots
}
