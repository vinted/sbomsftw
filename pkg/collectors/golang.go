package collectors

import (
	"context"
	"os"
	fp "path/filepath"
	"strings"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

type Golang struct {
	executor shellExecutor
}

func NewGolangCollector() Golang {
	return Golang{
		executor: defaultShellExecutor{},
	}
}

// MatchLanguageFiles implements LanguageCollector interface.
func (g Golang) MatchLanguageFiles(isDir bool, filepath string) bool {
	// Supported files by this collector
	const (
		goMod = "go.mod"
		goSum = "go.sum"
		goPkg = "Gopkg.lock"
	)

	for _, p := range strings.Split(fp.Dir(filepath), string(os.PathSeparator)) {
		if p == "vendor" {
			return false
		}
	}

	if isDir {
		return false
	}

	filename := fp.Base(filepath)

	return filename == goMod || filename == goSum || filename == goPkg
}

// GenerateBOM implements LanguageCollector interface
func (g Golang) GenerateBOM(ctx context.Context, bomRoot string) (*cdx.BOM, error) {
	const language = "golang"

	return g.executor.bomFromCdxgen(ctx, bomRoot, language, false)
}

// BootstrapLanguageFiles implements LanguageCollector interface.
func (g Golang) BootstrapLanguageFiles(_ context.Context, bomRoots []string) []string {
	return SquashToDirs(bomRoots)
}

func (g Golang) String() string {
	return "golang collector"
}
