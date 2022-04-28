package collectors

import (
	"os"
	fp "path/filepath"
	"strings"

	cdx "github.com/CycloneDX/cyclonedx-go"
	log "github.com/sirupsen/logrus"
)

var supportedJSFiles = []string{"yarn.lock", "bower.json", "package.json", "pnpm-lock.yaml", "package-lock.json"}

type JS struct {
	executor ShellExecutor
}

func NewJSCollector() JS {
	return JS{executor: DefaultShellExecutor{}}
}

//MatchLanguageFiles implements LanguageCollector interface
func (j JS) MatchLanguageFiles(isDir bool, filepath string) bool {
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
	return "javascript collector"
}

//GenerateBOM implements LanguageCollector interface
func (j JS) GenerateBOM(bomRoot string) (*cdx.BOM, error) {
	const language = "javascript"
	return j.executor.bomFromCdxgen(bomRoot, language, false)
}

//BootstrapLanguageFiles implements LanguageCollector interface
func (j JS) BootstrapLanguageFiles(bomRoots []string) []string {
	const bootstrapCmd = "pnpm install || npm install || yarn install"
	var bootstrappedRoots []string
	for dir, files := range SplitPaths(bomRoots) {
		if len(files) == 1 && files[0] == "package.json" { //Create a lock file if none exist yet
			if err := j.executor.shellOut(dir, bootstrapCmd); err != nil {
				log.WithFields(log.Fields{
					"collector": j,
					"error":     err,
				}).Debugf("can't bootstrap language files in: %s", dir)
				continue
			}
		}
		bootstrappedRoots = append(bootstrappedRoots, dir)
	}
	return bootstrappedRoots
}
