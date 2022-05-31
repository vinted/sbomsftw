package collectors

import (
	"context"
	"fmt"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/anchore/stereoscope/pkg/image"
	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/pkg/cataloger"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
	"github.com/vinted/software-assets/pkg/bomtools"
)

type Syft struct{}

type sbomCollectionResult struct {
	sbom *cdx.BOM
	err  error
}

func (s Syft) generateBOMInternal(ctx context.Context, repositoryPath string, result chan<- sbomCollectionResult) {
	const bomFormat = "cyclonedxjson"

	input := source.Input{
		Location:    repositoryPath,
		UserInput:   "dir:" + repositoryPath,
		Scheme:      source.DirectoryScheme,
		ImageSource: image.UnknownSource,
		Platform:    "",
	}
	src, _, err := source.New(input, nil, nil)
	if err != nil {
		err = fmt.Errorf("%s repository path is invalid: %v\n", repositoryPath, err)
		select {
		case <-ctx.Done():
			return
		default:
			result <- sbomCollectionResult{sbom: nil, err: err}
			return
		}
	}

	if ctx.Err() != nil {
		return // Return early & don't execute Syft if context is Done.
	}

	cfg := cataloger.DefaultConfig()
	cfg.Search.Scope = source.AllLayersScope
	pkgCatalog, relationships, osRelease, err := syft.CatalogPackages(src, cfg)
	if err != nil {
		err = fmt.Errorf("can't collect SBOMs for %s: %v", repositoryPath, err)
		select {
		case <-ctx.Done():
			return
		default:
			result <- sbomCollectionResult{sbom: nil, err: err}
			return
		}
	}

	sbom := sbom.SBOM{
		Artifacts: sbom.Artifacts{
			PackageCatalog:    pkgCatalog,
			LinuxDistribution: osRelease,
		},
		Relationships: relationships,
		Source:        src.Metadata,
	}
	cdxString, err := syft.Encode(sbom, syft.FormatByName(bomFormat))
	if err != nil {
		err = fmt.Errorf("can't encode SBOMs to %s format: %v\n", bomFormat, err)
		select {
		case <-ctx.Done():
			return
		default:
			result <- sbomCollectionResult{sbom: nil, err: err}
			return
		}
	}

	select {
	case <-ctx.Done():
		return
	default:
		finalSBOM, err := bomtools.StringToCDX(cdxString)
		result <- sbomCollectionResult{sbom: finalSBOM, err: err}
	}
}

// GenerateBOM implements Collector interface
func (s Syft) GenerateBOM(ctx context.Context, repositoryPath string) (*cdx.BOM, error) {
	worker := make(chan sbomCollectionResult, 1)
	go s.generateBOMInternal(ctx, repositoryPath, worker)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-worker:
		return result.sbom, result.err
	}
}

// String implements Collector interface
func (s Syft) String() string {
	return "generic syft collector"
}
