package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"os"
	fp "path/filepath"
	"strings"
)

type Golang struct{ executor BOMBridge }

func NewGolangCollector() Golang {
	return Golang{executor: defaultBOMBridge{}}
}

//matchPredicate implements BOMCollector interface
func (g Golang) matchPredicate(isDir bool, filepath string) bool {
	//Supported files by this collector
	const (
		goMod = "go.mod"
		goSum = "go.sum"
		goPkg = "Gopkg.lock"
	)
	for _, p := range strings.Split(fp.Dir(filepath), string(os.PathSeparator)) {
		if p == "vendor" { //Ignore files in vendor directory - add a test for this
			return false
		}
	}
	if isDir {
		return false
	}
	filename := fp.Base(filepath)
	return filename == goMod || filename == goSum || filename == goPkg
}

func (g Golang) generateBOM(bomRoot string) (*cdx.BOM, error) {
	return g.executor.bomFromCdxgen(bomRoot, golang)
}

//bootstrap implements BOMCollector interface
func (g Golang) bootstrap(bomRoots []string) []string {
	return squashRoots(bomRoots)
}

//String implements BOMCollector interface
func (g Golang) String() string {
	return "Golang collector"
}
