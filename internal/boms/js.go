package boms

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"os"
	fp "path/filepath"
	"strings"
)

var supportedJSFiles = []string{"yarn.lock", "bower.json", "package.json", "pnpm-lock.yaml", "package-lock.json"}

type JS struct {
	executor BOMBridge
}

func NewJSCollector() JS {
	return JS{executor: defaultBOMBridge{}}
}

func (j JS) matchPredicate(isDir bool, filepath string) bool {
	for _, p := range strings.Split(fp.Dir(filepath), string(os.PathSeparator)) {
		if p == "node_modules" { //Ignore files in node_modules directory
			return false
		}
	}
	filename := fp.Base(filepath)
	for _, f := range supportedJSFiles {
		if filename == f {
			return true
		}
	}
	/*
		Top level node_modules as a special case. In rare cases there will be no lockfiles
		but node_modules dir will be present
	*/
	return filename == "node_modules" && isDir
}

func (j JS) String() string {
	return "JS/TS-JS"
}

func (j JS) generateBOM(bomRoot string) (*cdx.BOM, error) {
	return j.executor.bomFromCdxgen(bomRoot, javascript)
}

func (j JS) bootstrap(bomRoots []string) []string {
	const bootstrapCmd = "pnpm install || npm install || yarn install"
	var bootstrappedRoots []string
	for dir, files := range dirsToFiles(bomRoots) {
		shouldBootstrap := len(files) == 1 && files[0] == "package.json"
		if shouldBootstrap {
			if _, err := j.executor.shellOut(dir, bootstrapCmd); err != nil {
				fmt.Println(fmt.Errorf(bootstrapFailedErr, dir, err))
				continue
			}
		}
		bootstrappedRoots = append(bootstrappedRoots, dir)
	}
	return bootstrappedRoots
}
