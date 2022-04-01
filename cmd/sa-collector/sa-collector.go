package main

import (
	"fmt"
	"github.com/vinted/software-assets/internal/boms"
	"github.com/vinted/software-assets/internal/requests"
	"github.com/vinted/software-assets/internal/vcs"
	"os"
	"path/filepath"
	"sync"
)

//Parameters used for making HTTP requests to GitHub and Dependency track
const (
	GithubUsername = ""
	GithubAPIToken = ""
	GithubReposURL = "https://api.github.com/orgs/vinted/repos"

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

func processRepoInternal(wg *sync.WaitGroup, collector boms.BOMCollector, repoPath string, results chan<- collectionResult) {
	defer wg.Done()
	fmt.Printf("attempting to generate bom entries with %s for %s\n", collector, repoPath)
	bom, err := boms.Collect(collector, repoPath)
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

func processRepoPath(repoPath string) error {
	availableCollectors := [...]boms.BOMCollector{boms.NewGolangCollector()}

	var wg sync.WaitGroup
	wg.Add(len(availableCollectors))

	results := make(chan collectionResult, len(availableCollectors))
	for _, c := range availableCollectors {
		go processRepoInternal(&wg, c, repoPath, results)
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
		return fmt.Errorf("can't merge BOM for %s: %w", repoPath, err)
	}
	bomString, err := boms.CdxToBOMString(boms.JSON, finalBOM)
	if err != nil {
		return fmt.Errorf("can't convert BOM for %s: %w", repoPath, err)
	}
	fmt.Println(bomString)
	fmt.Println("bom generated successfully")
	return nil
	//fmt.Printf("uploading %s SBOM to DT\n", repoPath)
	//reqConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, repoPath, bomString)
	//if _, err := requests.UploadBOM(reqConfig); err != nil {
	//	return fmt.Errorf("can't upload %s BOM to DT: %w", repoPath, err)
	//}
	//return nil
}

func processRepo(repository vcs.Repository) error {
	if repository.Archived {
		return nil
	}

	if repository.Language != "Go" {
		return nil
	}

	fmt.Println("Processing: " + repository.Name)

	err := repository.Clone(GithubUsername, GithubAPIToken)
	if err != nil {
		return fmt.Errorf("can't clone %s: %w", repository.Name, err)
	}

	collector := boms.NewGolangCollector()

	fmt.Printf("attempting to generate bom entries with %s for %s\n", collector, repository.FsPath())
	bom, err := boms.Collect(collector, repository.FsPath())
	if err != nil {
		fmt.Println(err)
		return nil
	}
	bomsFound := len(*bom.Components)
	fmt.Printf("%d BOMs collected for %s\n", bomsFound, repository.Name)
	return nil

	//fmt.Printf("uploading %s SBOM to DT\n", repository.Name)
	//reqConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, repository.Name, bomString)
	//if _, err := requests.UploadBOM(reqConfig); err != nil {
	//	return fmt.Errorf("can't upload %s BOM to DT: %w", repository.Name, err)
	//}
	//fmt.Println("Removing " + repository.FsPath())
	//_ = os.RemoveAll(repository.FsPath())
	//return nil
}

func main() {
	cleanup()
	setup()
	defer cleanup()

	//if err := processRepoPath("/tmp/checkouts/vitess"); err != nil {
	//	panic(err)
	//}
	reqConfig := requests.NewGetRepositoriesConfig(GithubReposURL, GithubUsername, GithubAPIToken)
	err := requests.WalkRepositories(reqConfig, func(repos []vcs.Repository) {
		for _, r := range repos {
			if err := processRepo(r); err != nil {
				fmt.Printf("can't collect BOMs for repository at %s: %s", r.FsPath(), err)
			}
		}
	})
	if err != nil {
		panic(err)
	}
}
