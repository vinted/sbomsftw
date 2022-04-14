package collectors

import (
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

func (s Syft) GenerateBOM(repositoryPath string) (*cdx.BOM, error) {
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
		return nil, fmt.Errorf("invalid repository path supplied for syft collection: %v\n", err)
	}
	cfg := cataloger.DefaultConfig()
	cfg.Search.Scope = source.AllLayersScope
	pkgCatalog, relationships, osRelease, err := syft.CatalogPackages(src, cfg)
	if err != nil {
		return nil, fmt.Errorf("syft collection for %s failed: %v\n", repositoryPath, err)
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
		return nil, fmt.Errorf("can't decode syft bom to %s: %v\n", bomFormat, err)
	}
	return bomtools.StringToCDX(cdxString)
}

//String implements BOMCollector interface
func (s Syft) String() string {
	return "syft collector"
}
