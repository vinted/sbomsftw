package collectors

import (
	"context"
	fp "path/filepath"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

type Rust struct {
	executor shellExecutor
}

func NewRustCollector() Rust {
	return Rust{
		executor: defaultShellExecutor{},
	}
}

// MatchLanguageFiles implements LanguageCollector interface.
func (g Rust) MatchLanguageFiles(isDir bool, filepath string) bool {
	// Supported files by this collector
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

// BootstrapLanguageFiles implements LanguageCollector interface.
func (g Rust) BootstrapLanguageFiles(_ context.Context, bomRoots []string) []string {
	return SquashToDirs(bomRoots)
}

// GenerateBOM implements LanguageCollector interface.
func (g Rust) GenerateBOM(ctx context.Context, bomRoot string) (*cdx.BOM, error) {
	const language = "rust"
	return g.executor.bomFromCdxgen(ctx, bomRoot, language, false)
}

// String implements LanguageCollector interface.
func (g Rust) String() string {
	return "rust collector"
}
