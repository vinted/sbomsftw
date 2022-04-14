package collectors

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	fp "path/filepath"
)

type Rust struct{ executor ShellExecutor }

func NewRustCollector() Rust {
	return Rust{executor: DefaultShellExecutor{}}
}

//MatchLanguageFiles implements LanguageCollector interface
func (g Rust) MatchLanguageFiles(isDir bool, filepath string) bool {
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

//BootstrapLanguageFiles implements LanguageCollector interface
func (g Rust) BootstrapLanguageFiles(bomRoots []string) []string {
	return bomRoots
}

func (g Rust) GenerateBOM(bomRoot string) (*cdx.BOM, error) {
	const language = "rust"
	return g.executor.bomFromCdxgen(bomRoot, language)
}

//String implements LanguageCollector interface
func (g Rust) String() string {
	return "rust collector"
}