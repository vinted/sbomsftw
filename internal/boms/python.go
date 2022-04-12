package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	fp "path/filepath"
)

var supportedPythonFiles = []string{"setup.py", "requirements.txt", "Pipfile.lock", "poetry.lock"}

type Python struct{ executor BOMBridge }

func NewPythonCollector() Python {
	return Python{executor: defaultBOMBridge{}}
}

//matchPredicate implements BOMCollector interface
func (g Python) matchPredicate(isDir bool, filepath string) bool {
	if isDir {
		return false
	}
	for _, f := range supportedPythonFiles {
		if fp.Base(filepath) == f {
			return true
		}
	}
	return false
}

func (g Python) generateBOM(bomRoot string) (*cdx.BOM, error) {
	return g.executor.bomFromCdxgen(bomRoot, python)
}

//bootstrap implements BOMCollector interface
func (g Python) bootstrap(bomRoots []string) []string {
	return squashRoots(bomRoots)
}

//String implements BOMCollector interface
func (g Python) String() string {
	return "Python collector"
}
