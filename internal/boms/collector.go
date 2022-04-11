package boms

/*
boms package provides implementation of various package managers that
are able to generate SBOMs from a give file system path
*/

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"os"
	"sync"
)

type language int

const (
	ruby language = iota
	javascript
	jvm
	golang
	rust
)

func (l language) String() string {
	return [...]string{"ruby", "javascript", "jvm", "golang", "rust"}[l]
}

type BOMCollector interface {
	fmt.Stringer
	matchPredicate(bool, string) bool
	bootstrap(bomRoots []string) []string
	generateBOM(bomRoot string) (*cdx.BOM, error)
}

func CollectFromRepo(repoPath string, collectors ...BOMCollector) (*cdx.BOM, error) {
	var collectedBOMs []*cdx.BOM
	trivyBOM, err := bomFromTrivy(repoPath) //Quickly hit repo with trivy, see if it can generate BOM
	if err != nil {
		fmt.Fprintf(os.Stderr, "trivy failed for: %s - error: %s\n", repoPath, err)
	} else {
		collectedBOMs = append(collectedBOMs, trivyBOM)
	}
	var wg sync.WaitGroup
	wg.Add(len(collectors))
	results := make(chan *cdx.BOM, len(collectors))
	for _, c := range collectors {
		go collectFromRepoInternal(&wg, c, repoPath, results)
	}
	wg.Wait()
	close(results)
	for r := range results {
		collectedBOMs = append(collectedBOMs, r)
	}
	return Merge(collectedBOMs...)
}

func collectFromRepoInternal(wg *sync.WaitGroup, collector BOMCollector, repoPath string, results chan<- *cdx.BOM) {
	defer wg.Done()
	rootsFound, err := repoToRoots(repoPath, collector.matchPredicate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s can't convert %s to roots: %s\n", collector, repoPath, err)
		return
	}
	var generatedBOMs []*cdx.BOM
	for _, r := range collector.bootstrap(rootsFound) {
		bom, err := collector.generateBOM(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s BOM generation failed for %s - %s\n", collector, repoPath, err)
			continue
		}
		generatedBOMs = append(generatedBOMs, bom)
	}
	if len(generatedBOMs) == 0 {
		fmt.Fprintf(os.Stderr, "%s has collected no BOMs for %s\n", collector, repoPath)
		return
	}
	bom, err := Merge(generatedBOMs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't merge BOMs collected by %s - %s\n", collector, err)
		return
	}
	fmt.Printf("Found %d BOMs for %s has with %s\n", len(*bom.Components), repoPath, collector)
	results <- bom
}
