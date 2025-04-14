package collectors

import (
	"bytes"
	"context"
	"fmt"
	"github.com/anchore/syft/syft/format/cyclonedxjson"
	log "github.com/sirupsen/logrus"
	"runtime"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/anchore/syft/syft"
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
	bomConfig.Licenses.Coverage = 0
	bomConfig.Parallelism = 10
	LogMemoryUsage("before")
	syftSbom, err := syft.CreateSBOM(context.Background(), src, bomConfig)
	LogMemoryUsage("after")
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

func LogMemoryUsage(timing string) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Convert bytes to MB for easier reading
	allocatedMemory := memStats.Alloc / 1024 / 1024
	totalAllocatedMemory := memStats.TotalAlloc / 1024 / 1024
	systemMemory := memStats.Sys / 1024 / 1024
	numGC := memStats.NumGC

	fmt.Printf("Memory Usage:\n")
	fmt.Printf("Allocated %s: %v MB\n", timing, allocatedMemory)
	fmt.Printf("Total Allocated %s: %v MB\n", timing, totalAllocatedMemory)
	fmt.Printf("System %s: %v MB\n", timing, systemMemory)
	fmt.Printf("Number of GCs %s: %v\n", timing, numGC)
}
