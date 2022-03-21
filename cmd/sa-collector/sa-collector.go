package main

import (
	"fmt"
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

func processRepo(repository vcs.Repository) error {
	if repository.Archived {
		return nil
	}
	collector := collectors.Bundler{} //Only bundler for now
	err := repository.Clone(GithubUsername, GithubAPIToken)
	if err != nil {
		return fmt.Errorf("can't clone %s: %w", repository.Name, err)
	}
	defer os.RemoveAll(repository.FsPath())
	fmt.Printf("attempting to generate bom entries with %s for %s\n", collector, repository.FsPath())
	bom, err := collector.CollectBOM(repository.FsPath())
	if err != nil {
		return fmt.Errorf("failure - %s: %w", collector, err)
	}
	fmt.Printf("uploading %s SBOM to DT\n", repository.Name)
	reqConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, repository.Name, bom)
	if _, err := requests.UploadBOM(reqConfig); err != nil {
		return fmt.Errorf("can't upload %s BOM to DT: %w", repository.Name, err)
	}
	return nil
}

func worker(wg *sync.WaitGroup, repositories <-chan vcs.Repository) {
	defer wg.Done()
	for r := range repositories {
		if err := processRepo(r); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
	}
}

func main() {
	setup()
	defer cleanup()

	const numberOfWorkers = 1 // Play around later on
	reposToProcess := make(chan vcs.Repository)

	var wg sync.WaitGroup
	for i := 0; i < numberOfWorkers; i++ {
		wg.Add(1)
		go worker(&wg, reposToProcess)
	}

	go func() {
		reqConfig := requests.NewGetRepositoriesConfig(GithubReposURL, GithubUsername, GithubAPIToken)
		err := requests.WalkRepositories(reqConfig, func(repos []vcs.Repository) {
			for _, r := range repos {
				reposToProcess <- r
			}
		})
		close(reposToProcess)
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()
}
