package repository

import (
	"errors"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/vinted/software-assets/pkg"
	"github.com/vinted/software-assets/pkg/bomtools"
	"github.com/vinted/software-assets/pkg/collectors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Credentials struct {
	Username    string
	AccessToken string
}

type Repository struct {
	Name               string
	FSPath             string
	genericCollectors  []pkg.Collector
	languageCollectors []pkg.LanguageCollector
}

type BadVCSURLError struct {
	URL string
}

func (b BadVCSURLError) Error() string {
	return fmt.Sprintf("invalid VCS URL supplied %s\n", b.URL)
}

func NewFromVCS(vcsURL string, credentials Credentials) (*Repository, error) {
	const checkoutsPath = "/tmp/checkouts/"

	urlPaths := strings.Split(vcsURL, "/")
	if len(urlPaths) == 0 {
		return nil, BadVCSURLError{URL: vcsURL}
	}
	name := strings.TrimSuffix(urlPaths[len(urlPaths)-1], ".git")

	_, err := git.PlainClone(filepath.Join(checkoutsPath, name), false, &git.CloneOptions{
		URL:      vcsURL,
		Progress: os.Stdout,
		Auth:     &http.BasicAuth{Username: credentials.Username, Password: credentials.AccessToken},
	})
	if err != nil {
		return nil, err
	}
	return &Repository{
		Name:              name,
		FSPath:            filepath.Join(checkoutsPath, name),
		genericCollectors: []pkg.Collector{collectors.Syft{}, collectors.Trivy{}},
		languageCollectors: []pkg.LanguageCollector{
			collectors.NewPythonCollector(), collectors.NewRustCollector(), collectors.NewJVMCollector(),
			collectors.NewGolangCollector(), collectors.NewJSCollector(), collectors.NewRubyCollector()},
	}, nil
}

func NewFromFS(filesystemPath string) (*Repository, error) {
	_, err := os.Stat(filesystemPath)
	if err != nil {
		return nil, fmt.Errorf("can't create repository from file system: %w\n", err)
	}

	return &Repository{
		FSPath:            filesystemPath,
		Name:              filepath.Base(filesystemPath),
		genericCollectors: []pkg.Collector{collectors.Syft{}, collectors.Trivy{}},
		languageCollectors: []pkg.LanguageCollector{
			collectors.NewPythonCollector(), collectors.NewRustCollector(), collectors.NewJVMCollector(),
			collectors.NewGolangCollector(), collectors.NewJSCollector(), collectors.NewRubyCollector()},
	}, nil
}

func (r Repository) ExtractBOMs(includeGenericCollectors bool) (*cdx.BOM, error) {
	var collectedBOMs []*cdx.BOM
	if includeGenericCollectors { //Generate base BOM with generic collectors (syft & trivy)
		for _, c := range r.genericCollectors {
			bom, err := c.GenerateBOM(r.FSPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s failed for: %s - error: %s\n", c, r.FSPath, err)
				continue
			}
			collectedBOMs = append(collectedBOMs, bom)
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(r.languageCollectors))
	results := make(chan *cdx.BOM, len(r.languageCollectors))
	for _, c := range r.languageCollectors {
		go r.bomsFromCollector(&wg, c, results)
	}
	wg.Wait()
	close(results)
	for r := range results {
		collectedBOMs = append(collectedBOMs, r)
	}
	return bomtools.MergeBoms(collectedBOMs...)
}

func (r Repository) bomsFromCollector(wg *sync.WaitGroup, collector pkg.LanguageCollector, results chan<- *cdx.BOM) {
	defer wg.Done()
	rootsFound, err := bomtools.RepoToRoots(r.FSPath, collector.MatchLanguageFiles)
	if err != nil {
		var e bomtools.NoRootsFoundError
		if errors.As(err, &e) {
			fmt.Fprintf(os.Stderr, "%s: found no language files for %s - skipping\n", collector, r)
		} else {
			fmt.Fprintf(os.Stderr, "%s: can't convert repo to roots - %s\n", collector, err)
		}
		return
	}
	fmt.Fprintf(os.Stdout, "extracting BOMs from %s with %s\n", r, collector)
	var collectedBOMs []*cdx.BOM
	for _, root := range collector.BootstrapLanguageFiles(rootsFound) {
		bom, err := collector.GenerateBOM(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bomsFromCollector: %s cdxgen failed on %s\n", collector, root)
			continue
		}
		collectedBOMs = append(collectedBOMs, bom)
		os.Remove(root)
	}
	mergedBOM, err := bomtools.MergeBoms(collectedBOMs...)
	if err != nil {
		if errors.Is(err, bomtools.ErrNoBOMsToMerge) {
			fmt.Fprintf(os.Stderr, "bomsFromCollector: %s found no BOMs\n", collector)
		} else {
			fmt.Fprintf(os.Stderr, "bomsFromCollector: failed to merge BOMs from %s: %s\n", collector, err)
		}
		return
	}
	results <- mergedBOM
}

func (r Repository) String() string {
	return r.Name
}
