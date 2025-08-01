package collectors

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/format/cyclonedxjson"
	"github.com/anchore/syft/syft/sbom"
	"github.com/anchore/syft/syft/source"
	log "github.com/sirupsen/logrus"
	"github.com/vinted/sbomsftw/pkg/bomtools"
)

type Syft struct {
	Exclusions []string
}

type sbomCollectionResult struct {
	sbom *cdx.BOM
	err  error
}

func init() {
	sql.Register("sqlite", &sqlite3.SQLiteDriver{})
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
	sbomBytes, err := formatSBOM(sbom)
	if err != nil {
		log.WithError(err).Errorf("can't format to bytes %s", err)
	}

	select {
	case <-ctx.Done():
		return
	default:
		finalSBOM, err := bomtools.StringToCDX(sbomBytes)
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
	bomConfig := syft.DefaultCreateSBOMConfig()
	syftSbom, err := syft.CreateSBOM(context.Background(), src, bomConfig)
	if err != nil {
		return nil, fmt.Errorf("can't create CycloneDX SBOM: %w", err)
	}
	artifacts := sbom.Artifacts{
		Packages:          syftSbom.Artifacts.Packages,
		LinuxDistribution: syftSbom.Artifacts.LinuxDistribution,
	}
	sbomFinal := &sbom.SBOM{
		Artifacts:     artifacts,
		Relationships: syftSbom.Relationships,
		Source:        src.Describe(),
	}
	return sbomFinal, err
}

func formatSBOM(s *sbom.SBOM) ([]byte, error) {
	formatEncoderConfig := cyclonedxjson.DefaultEncoderConfig()
	encoder, _ := cyclonedxjson.NewFormatEncoderWithConfig(formatEncoderConfig)
	var buffer bytes.Buffer
	err := encoder.Encode(&buffer, *s)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
