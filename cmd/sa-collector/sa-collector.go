package main

import (
	"fmt"
	"github.com/vinted/software-assets/internal/boms"
	"github.com/vinted/software-assets/internal/requests"
	"github.com/vinted/software-assets/internal/vcs"
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

func processRepository(projectName, repoPath string) error {
	//Proceed
	availableCollectors := []boms.BOMCollector{
		boms.NewGolangCollector(),
		boms.NewJSCollector(),
		boms.NewJVMCollector(),
		boms.NewRubyCollector(),
		boms.NewRustCollector(),
	}
	bom, err := boms.CollectFromRepo(repoPath, availableCollectors...)
	if err != nil {
		return fmt.Errorf("BOM collection failed for %s - %w", projectName, err)
	}

	bomString, err := boms.CdxToBOMString(boms.JSON, bom) //TODO Make this agnostic
	if err != nil {
		return fmt.Errorf("can't convert BOM for %s: %w", repoPath, err)
	}
	fmt.Printf("uploading %s SBOM to DT\n", repoPath)
	reqConfig := requests.NewUploadBOMConfig(DTEndpoint, DTAPIToken, projectName, bomString)
	if _, err := requests.UploadBOM(reqConfig); err != nil {
		return fmt.Errorf("can't upload %s BOM to DT: %w", repoPath, err)
	}
	return nil
}

func main() {
	//setup()
	//if err := processRepository("vitess", "/tmp/checkouts/hubot-xmpp"); err != nil {
	//	panic(err)
	//}
	//return
	cleanup()
	setup()
	defer cleanup()
	reqConfig := requests.NewGetRepositoriesConfig(GithubReposURL, GithubUsername, GithubAPIToken)
	err := requests.WalkRepositories(reqConfig, func(repos []vcs.Repository) {
		for _, r := range repos {
			if r.Archived {
				continue
			}
			fmt.Printf("cloning %s\n", r.Name)
			err := r.Clone(GithubUsername, GithubAPIToken)
			if err != nil {
				fmt.Fprintf(os.Stderr, "can't clone %s: %s\n", r.Name, err)
				continue
			}
			if err := processRepository(r.Name, r.FsPath()); err != nil {
				fmt.Printf("processing repository at %s failed: %s\n", r.FsPath(), err)
			}
			_ = os.RemoveAll(r.FsPath())
		}
	})
	if err != nil {
		panic(err)
	}
}
