package main

import (
	"errors"
	"fmt"
	"github.com/vinted/software-assets/internal/collectors"
	"github.com/vinted/software-assets/internal/requests"
	"github.com/vinted/software-assets/internal/vcs"
	"os"
	"path/filepath"
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

func collectBOM(repository vcs.Repository) {
	bundler := collectors.Bundler{}
	fmt.Printf("Attempting to generate bom entries with %s for %s\n", bundler, repository.FsPath())
	bom, err := bundler.CollectBOM(repository.FsPath())
	if err == nil {
		uploadBOM(bom, repository.Name)
		return
	}
	var e collectors.NoRootsFoundError
	if errors.As(err, &e) {
		//Log that handler haven't found anything
		return
	} else {
		fmt.Fprint(os.Stderr, err)
		return
	}
}

func uploadBOM(bom, projectName string) {
	//Hardcoded for now, remove to config file and parse it later on
	fmt.Printf("Uploading %s SBOM to DT\n", projectName)
	reqConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, projectName, bom)
	if _, err := requests.UploadBOM(reqConfig); err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error())
	}
}

func main() {
	setup()
	defer cleanup()

	reqConfig := requests.NewGetRepositoriesConfig(GithubReposURL, GithubUsername, GithubAPIToken)
	repositories, err := requests.GetRepositories(reqConfig)
	if err != nil {
		panic(err)
	}

	for _, repo := range repositories {
		err := repo.Clone(GithubUsername, GithubAPIToken)
		if err != nil {
			errMsg := fmt.Sprintf("Unable to clone %s, reason: %s", repo.Name, err.Error())
			_, _ = fmt.Fprintf(os.Stderr, errMsg)
			continue
		}
		collectBOM(repo)
	}
}
