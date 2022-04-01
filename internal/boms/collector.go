package boms

/*
boms package provides implementation of various package managers that
are able to generate SBOMs from a give file system path
*/

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
)

type BOMCollectionFailed string

func (e BOMCollectionFailed) Error() string {
	return string(e)
}

const errUnsupportedRepo = BOMCollectionFailed("BOM collection failed for every root. Unsupported repository")

type BOMCollector interface {
	fmt.Stringer
	matchPredicate(bool, string) bool
	bootstrap(bomRoots []string) []string
	generateBOM(bomRoot string) (string, error)
}

func Collect(collector BOMCollector, repoPath string) (*cdx.BOM, error) {
	rootsFound, err := repoToRoots(repoPath, collector.matchPredicate)
	if err != nil {
		return nil, fmt.Errorf("can't to Collect BOMs for %s with %s: %w", repoPath, collector, err)
	}
	var generatedBOMs []string
	for _, r := range collector.bootstrap(rootsFound) {
		bom, err := collector.generateBOM(r)
		if err != nil {
			fmt.Printf("BOM generation failed: %s\n", err)
			continue
		}
		generatedBOMs = append(generatedBOMs, bom)
	}
	if len(generatedBOMs) == 0 {
		return nil, errUnsupportedRepo
	}
	mergedBom, err := Merge(JSON, generatedBOMs...)
	//filter
	if err != nil {
		return nil, err
	}
	//todo FilterOutByScope()
	//todo js/ruby tests became more like tests for this component - extract them!
	return attachCPEs(mergedBom), nil
}
