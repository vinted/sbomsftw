package collectors

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

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
	src, err := getSource(repositoryPath)
	if err != nil {
		log.WithError(err).Errorf("can't get source %s", err)
	}

	// catalog the given source and return a SBOM
	sbom, err := getSBOM(src)
	if err != nil {
		log.WithError(err).Errorf("can't get sbom %s", err)
	}

	// take the SBOM object and encode it into the syft-json representation
	bytes, err := formatSBOM(sbom)
	if err != nil {
		log.WithError(err).Errorf("can't format to bytes %s", err)
	}

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

func getSource(input string) (source.Source, error) {
	src, err := syft.GetSource(context.Background(), input, nil)
	if err != nil {
		return nil, fmt.Errorf("can't get source from input for sbom: %w", err)
	}

	return src, err
}

func getSBOM(src source.Source) (*sbom.SBOM, error) {
	sbom, err := syft.CreateSBOM(context.Background(), src, nil)
	if err != nil {
		return nil, fmt.Errorf("can't create CycloneDX SBOM: %w", err)
	}

	return sbom, err
}

func formatSBOM(s *sbom.SBOM) ([]byte, error) {
	bytes, err := format.Encode(*s, syftjson.NewFormatEncoder())
	if err != nil {
		return nil, fmt.Errorf("can't json to bytes: %w", err)
	}
	return bytes, nil
}
