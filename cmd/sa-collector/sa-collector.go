package main

import (
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

func uploadBOM(bom, projectName string) {
	//Hardcoded for now, remove to config file and parse it later on
	fmt.Printf("Uploading %s SBOM to DT\n", projectName)
	reqConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, projectName, bom)
	if _, err := requests.UploadBOM(reqConfig); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
	}
}

func processRepos(repos []vcs.Repository) {
	bundler := collectors.Bundler{}
	for _, r := range repos {
		err := r.Clone(GithubUsername, GithubAPIToken)
		if err != nil {
			errMsg := fmt.Sprintf("Unable to clone %s, reason: %s", r.Name, err.Error())
			_, _ = fmt.Fprintf(os.Stderr, errMsg)
			continue
		}
		fmt.Printf("Attempting to generate bom entries with %s for %s\n", bundler, r.FsPath())
		bom, err := bundler.CollectBOM(r.FsPath())
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			continue
		}
		uploadBOM(bom, r.Name)
	}
}

func main() {
	setup()
	defer cleanup()

	reqConfig := requests.NewGetRepositoriesConfig(GithubReposURL, GithubUsername, GithubAPIToken)
	err := requests.WalkRepositories(reqConfig, processRepos)
	if err != nil {
		panic(err)
	}
}
