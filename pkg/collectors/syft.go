package collectors

import (
	"context"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/format"
	"github.com/anchore/syft/syft/format/syftjson"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
	"github.com/vinted/sbomsftw/pkg/bomtools"
)

type Syft struct {
	Exclusions []string
}

type sbomCollectionResult struct {
	sbom *cdx.BOM
	err  error
}

func (s Syft) generateBOMInternal(ctx context.Context, repositoryPath string, result chan<- sbomCollectionResult) {
	src := getSource(repositoryPath)

	// catalog the given source and return a SBOM
	sbom := getSBOM(src)

	// take the SBOM object and encode it into the syft-json representation
	bytes := formatSBOM(sbom)

	select {
	case <-ctx.Done():
		return
	default:
		finalSBOM, err := bomtools.StringToCDX(bytes)
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

func getSource(input string) source.Source {
	src, err := syft.GetSource(context.Background(), input, nil)
	if err != nil {
		panic(err)
	}

	return src
}

func getSBOM(src source.Source) sbom.SBOM {
	s, err := syft.CreateSBOM(context.Background(), src, nil)
	if err != nil {
		panic(err)
	}

	return *s
}

func formatSBOM(s sbom.SBOM) []byte {
	bytes, err := format.Encode(s, syftjson.NewFormatEncoder())
	if err != nil {
		panic(err)
	}
	return bytes
}
