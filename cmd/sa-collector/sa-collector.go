package main

import (
	"fmt"
	requests "github.com/vinted/software-assets/internal"
	"github.com/vinted/software-assets/pkg/bomtools"
	"github.com/vinted/software-assets/pkg/repository"
	"os"
	"path/filepath"
)

//Parameters used for making HTTP requests to GitHub and Dependency track
const (
	GithubUsername = ""
	GithubAPIToken = ""
	GithubReposURL = "https://api.github.com/orgs/vinted/repos"

	DTAPIToken = "oh no no no"
	DTEndpoint = "http://192.168.10.10:8081/api/v1/bom"
)

func cleanup() {
	// Log this error
	_ = os.RemoveAll("/tmp/checkouts")
}

func setup() {
	err := os.Setenv("GEM_HOME", filepath.Join(os.Getenv("HOME"), ".gem"))
	if err != nil {
		panic("Unable to set GEM_HOME env variable")
	}
	err = os.Setenv("PATH", os.Getenv("PATH")+":"+"/usr/local/bin")
	if err != nil {
		panic("Unable to append /usr/local/bin to PATH")
	}
}

func main() {
	//setup()
	//if err := processRepository("vmip-boston-housing-trainer", "/tmp/py-repos/ml-pytorch"); err != nil {
	//	panic(err)
	//}
	//return
	cleanup()
	setup()
	defer cleanup()

	deleteRepository := func(repositoryPath string) {
		if err := os.RemoveAll(repositoryPath); err != nil {
			fmt.Fprintf(os.Stderr, "can't remove repository directory: %s\n", err)
		}
	}

	reqConfig := requests.NewGetRepositoriesConfig(GithubReposURL, GithubUsername, GithubAPIToken)
	err := requests.WalkRepositories(reqConfig, func(repositoryURLs []string) {
		for _, repositoryURL := range repositoryURLs {
			repository, err := repository.NewFromVCS(repositoryURL, repository.Credentials{
				Username:    GithubUsername,
				AccessToken: GithubAPIToken,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "can't clone %s: %s\n", repositoryURL, err)
				continue
			}
			bom, err := repository.ExtractBOMs(true)
			if err != nil {
				fmt.Fprintf(os.Stderr, "can't collect BOMs from %s: %s\n", repository, err)
				deleteRepository(repository.FSPath)
				continue
			}
			bomString, err := bomtools.CDXToString(bom)
			if err != nil {
				fmt.Fprintf(os.Stderr, "can't convert cdx.BOM to string: %s\n", err)
				deleteRepository(repository.FSPath)
				continue
			}

			uploadConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, repository.Name, bomString)
			if _, err = requests.UploadBOM(uploadConfig); err != nil {
				fmt.Fprintf(os.Stderr, "can't upload BOM to Dependecny track: %s", err)
			}
			deleteRepository(repository.FSPath)
		}
	})
	if err != nil {
		panic(err)
	}
}
