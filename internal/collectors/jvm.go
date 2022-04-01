package collectors

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/internal/boms"
	"os"
	fp "path/filepath"
)

const (
	tmpBomStorage = "/tmp/tmp.s6z8dLPp40Eetmb1aX6C2ZyGwHA3Bv.json"
	jvmCDXGenCmd  = "export FETCH_LICENSE=true && cdxgen -t java -o " + tmpBomStorage
)

type JVM struct {
	executor CLIExecutor
}

func NewJVMCollector() JVM {
	return JVM{executor: defaultCLIExecutor{}}
}

//matchPredicate Implements BOMCollector interface
func (j JVM) matchPredicate(isDir bool, filepath string) bool {
	if isDir {
		return false
	}
	for _, f := range []string{"pom.xml", "build.gradle", "build.gradle.kts", "sbt", "build.sbt"} {
		if fp.Base(filepath) == f {
			return true
		}
	}
	return false
}

func (j JVM) String() string {
	return "JVM - (Java/Kotlin/Scala/Groovy)"
}

//CollectBOM Implements BOMCollector interface
func (j JVM) CollectBOM(repoPath string) (*cdx.BOM, error) {
	rootsFound, err := boms.RepoToRoots(repoPath, j.matchPredicate)
	if err != nil {
		return nil, fmt.Errorf("can't to collect BOMs for %s with %s: %w", repoPath, j, err)
	}

	var generatedBOMs []string
	for _, b := range j.bootstrap(rootsFound) {
		_, err := j.executor.executeCDXGen(b, jvmCDXGenCmd)
		if err != nil {
			continue
		}
		bom, err := os.ReadFile(tmpBomStorage)
		if err != nil {
			continue
		}
		_ = os.Remove(tmpBomStorage)
		generatedBOMs = append(generatedBOMs, string(bom))
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

func (j JVM) bootstrap(bomRoots []string) []string {
	//Squash dirs
	var dirsToFiles = make(map[string][]string)
	for _, r := range bomRoots {
		dir := fp.Dir(r)
		dirsToFiles[dir] = append(dirsToFiles[dir], fp.Base(r))
	}
	squashed := make([]string, 0, len(dirsToFiles))
	for dir, _ := range dirsToFiles {
		squashed = append(squashed, dir)
	}
	return squashed
}
