package main

import (
	"fmt"
	"github.com/vinted/software-assets/internal/boms"
	"github.com/vinted/software-assets/internal/collectors"
	"github.com/vinted/software-assets/internal/requests"
	"github.com/vinted/software-assets/internal/vcs"
	"os"
	"path/filepath"
	"sync"
)

//Parameters used for making HTTP requests to GitHub and Dependency track
const (
	GithubUsername = "oh no no no"
	GithubAPIToken = "oh no no no"
	GithubReposURL = "oh no no no"

	DTAPIToken = "oh no no no"
	DTEndpoint = "oh no no no"
)

func cleanup() {
	_ = os.RemoveAll("/tmp/checkouts")
}

func setup() {
	err := os.Setenv("GEM_HOME", filepath.Join(os.Getenv("HOME"), ".gem"))
	if err != nil {
		panic("Unable to set GEM_HOME env variable")
	}
}

type collectionResult struct {
	bomContents string //in json
	err         error
}

func processRepoInternal(wg *sync.WaitGroup, collector collectors.BOMCollector, repoPath string, results chan<- collectionResult) {
	defer wg.Done()
	fmt.Printf("attempting to generate bom entries with %s for %s\n", collector, repoPath)
	bom, err := collector.CollectBOM(repoPath)
	if err != nil {
		fmt.Println(err)
		results <- collectionResult{err: err}
		return
	}
	bomString, err := boms.CdxToBOMString(boms.JSON, bom)
	if err != nil {
		fmt.Println(err)
		results <- collectionResult{err: err}
		return
	}
	fmt.Printf("BOM collection succeded for %s\n", collector)
	results <- collectionResult{bomContents: bomString, err: nil}
}

func processRepo(repository vcs.Repository) error {
	if repository.Archived {
		return nil
	}

	err := repository.Clone(GithubUsername, GithubAPIToken)
	if err != nil {
		return fmt.Errorf("can't clone %s: %w", repository.Name, err)
	}

	availableCollectors := [...]collectors.BOMCollector{collectors.NewRubyCollector(), collectors.NewJSCollector()}

	var wg sync.WaitGroup
	wg.Add(len(availableCollectors))

	results := make(chan collectionResult, len(availableCollectors))
	for _, c := range availableCollectors {
		go processRepoInternal(&wg, c, repository.FsPath(), results)
	}
	wg.Wait()
	close(results)

	var collectedBOMs []string
	for r := range results {
		if r.err == nil {
			collectedBOMs = append(collectedBOMs, r.bomContents)
		}
	}
	finalBOM, err := boms.Merge(boms.JSON, collectedBOMs...)
	if err != nil {
		return fmt.Errorf("can't merge BOM for %s: %w", repository.Name, err)
	}
	bomString, err := boms.CdxToBOMString(boms.JSON, finalBOM)
	if err != nil {
		return fmt.Errorf("can't convert BOM for %s: %w", repository.Name, err)
	}

	fmt.Printf("uploading %s SBOM to DT\n", repository.Name)
	reqConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, repository.Name, bomString)
	if _, err := requests.UploadBOM(reqConfig); err != nil {
		return fmt.Errorf("can't upload %s BOM to DT: %w", repository.Name, err)
	}
	return nil
}

func main() {
	cleanup()
	setup()
	defer cleanup()

	reqConfig := requests.NewGetRepositoriesConfig(GithubReposURL, GithubUsername, GithubAPIToken)
	err := requests.WalkRepositories(reqConfig, func(repos []vcs.Repository) {
		for _, r := range repos {
			if err := processRepo(r); err != nil {
				fmt.Printf("Can't to collect BOMs for repository at %s: %s", r.FsPath(), err)
			}
		}
	})
	if err != nil {
		panic(err)
	}
}
