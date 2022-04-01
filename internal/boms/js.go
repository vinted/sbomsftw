package boms

import (
	"fmt"
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

func (j JS) generateBOM(bomRoot string) (string, error) {
	return j.executor.executeCDXGen(bomRoot, jsCDXGenCmd)
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
			if _, err := j.executor.shellOut(dir, jsBootstrapCmd); err != nil {
				fmt.Println(fmt.Errorf(bootstrapFailedErr, dir, err))
				continue
			}
		}
		bootstrappedRoots = append(bootstrappedRoots, dir)
	}
	return bootstrappedRoots
}
