package main

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	requests "github.com/vinted/software-assets/internal"
	"github.com/vinted/software-assets/pkg/bomtools"
	"github.com/vinted/software-assets/pkg/repository"
	"log"
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

func uploadToDependencyTrack(repositoryName string, bom *cdx.BOM) error {
	bomString, err := bomtools.CDXToString(bom)
	if err != nil {
		return fmt.Errorf("can't convert cdx.BOM to string: %v\n", err)
	}

	uploadConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, repositoryName, bomString)
	if _, err = requests.UploadBOM(uploadConfig); err != nil {
		return fmt.Errorf("can't upload BOM to Dependecny track: %v\n", err)
	}
	return nil
}

func bomsFromRepository(vcsURL string) error {
	deleteRepository := func(repositoryPath string) {
		if err := os.RemoveAll(repositoryPath); err != nil {
			fmt.Fprintf(os.Stderr, "can't remove repository directory: %s\n", err)
		}
	}

	repo, err := repository.New(vcsURL, repository.Credentials{
		Username:    GithubUsername,
		AccessToken: GithubAPIToken,
	})
	if err != nil {
		return fmt.Errorf("can't clone %s: %v\n", vcsURL, err)
	}
	defer deleteRepository(repo.FSPath)
	bom, err := repo.ExtractBOMs(true)
	if err != nil {
		return fmt.Errorf("can't collect BOMs from %s: %v\n", repo, err)
	}
	if err = uploadToDependencyTrack(repo.Name, bom); err != nil {
		return fmt.Errorf("can't upload BOMs to Dependency Track: %v\n", err)
	}
	return nil
}

func bomsFromRepositories() {
	reqConfig := requests.NewGetRepositoriesConfig(GithubReposURL, GithubUsername, GithubAPIToken)
	err := requests.WalkRepositories(reqConfig, func(repositoryURLs []string) {
		for _, repositoryURL := range repositoryURLs {
			if err := bomsFromRepository(repositoryURL); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				continue
			}
		}
	})
	if err != nil {
		log.Fatalf("BOM collection from repositories failed: %s\n", err)
	}
}

func main() {
	cleanup()
	setup()
	defer cleanup()
	if err := bomsFromRepository("https://github.com/vinted/lighthouse-ci-action"); err != nil {
		log.Fatalln(err)
	}
}
