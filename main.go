package main

import (
	"fmt"
	"github.com/vinted/software-assets/github"
	"github.com/vinted/software-assets/sboms"
	"github.com/vinted/software-assets/sboms/generators"
	"os"
	"path/filepath"
	"time"
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

func generateBOM(repoPath string) {
	bundler := generators.Bundler{}
	fmt.Printf("Attempting to generate bom with %s for %s\n", bundler, repoPath)

	relativeRoots, err := sboms.FindRoots(os.DirFS(repoPath), bundler.MatchPredicate)
	if len(relativeRoots) == 0 {
		fmt.Printf("Repository at %s unsupported for %s. Skipping\n", repoPath, bundler)
		return
	}

	absoluteRoots := sboms.RelativeToAbsoluteRoots(relativeRoots, repoPath)
	bom, err := bundler.GenerateBOM(absoluteRoots)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s is unable to generate BOM for repository at %s\n", bundler, repoPath)
		return
	}
	fmt.Println(bom)
}

func main() {
	setup()
	defer cleanup()

	orgURL := "https://api.github.com/orgs/vinted/repos"
	backoff := []time.Duration{5 * time.Second, 7 * time.Second, 10 * time.Second}
	repositories, err := github.GetRepositories(orgURL, 10*time.Second, backoff...)
	if err != nil {
		panic(err)
	}

	for _, repo := range repositories {
		err := repo.Clone()
		if err != nil {
			errMsg := fmt.Sprintf("Unable to clone %s, reason: %s", repo.Name, err.Error())
			_, _ = fmt.Fprintf(os.Stderr, errMsg)
			continue
		}
		generateBOM(repo.FsPath())
	}
}
