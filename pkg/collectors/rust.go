package collectors

import (
	"context"
	cdx "github.com/CycloneDX/cyclonedx-go"
	fp "path/filepath"
)

type Rust struct {
	executor shellExecutor
}

func NewRustCollector(ctx context.Context) Rust {
	return Rust{
		executor: newDefaultShellExecutor(ctx),
	}
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
	return SquashToDirs(bomRoots)
}

//GenerateBOM implements LanguageCollector interface
func (g Rust) GenerateBOM(bomRoot string) (*cdx.BOM, error) {
	const language = "rust"
	return g.executor.bomFromCdxgen(bomRoot, language, false)
}

//String implements LanguageCollector interface
func (g Rust) String() string {
	return "rust collector"
}
