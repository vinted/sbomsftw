package boms

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	fp "path/filepath"
)

type Rust struct{ executor BOMBridge }

func NewRustCollector() Rust {
	return Rust{executor: defaultBOMBridge{}}
}

//matchPredicate implements BOMCollector interface
func (g Rust) matchPredicate(isDir bool, filepath string) bool {
	//Supported files by this collector
	const (
		cargoToml = "Cargo.toml"
		cargoLock = "Cargo.lock"
	)
	if isDir {
		return false
	}
	filename := fp.Base(filepath)
	return filename == cargoToml || filename == cargoLock
}

//bootstrap implements BOMCollector interface
func (g Rust) bootstrap(bomRoots []string) []string {
	return squashRoots(bomRoots)
}

func (g Rust) generateBOM(bomRoot string) (*cdx.BOM, error) {
	return g.executor.bomFromCdxgen(bomRoot, rust)
}

//String implements BOMCollector interface
func (g Rust) String() string {
	return "Rust collector"
}
