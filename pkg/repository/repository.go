package repository

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"strings"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/vinted/software-assets/pkg"
	"github.com/vinted/software-assets/pkg/bomtools"
	"github.com/vinted/software-assets/pkg/collectors"
)

const CheckoutsPath = "/tmp/checkouts/"

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

func New(vcsURL string, credentials Credentials) (*Repository, error) {
	urlPaths := strings.Split(vcsURL, "/")
	if len(urlPaths) == 0 {
		return nil, BadVCSURLError{URL: vcsURL}
	}
	name := strings.TrimSuffix(urlPaths[len(urlPaths)-1], ".git")

	fsPath := filepath.Join(CheckoutsPath, name)
	log.WithField("VCS URL", vcsURL).Infof("cloning %s into %s", name, fsPath)
	_, err := git.PlainClone(fsPath, false, &git.CloneOptions{URL: vcsURL})
	if err != nil {
		//Retry to clone the repo with credentials if failed
		_, err = git.PlainClone(fsPath, false, &git.CloneOptions{
			URL:  vcsURL,
			Auth: &http.BasicAuth{Username: credentials.Username, Password: credentials.AccessToken},
		})
		if err != nil {
			return nil, err
		}
	}
	return &Repository{
		Name:              name,
		FSPath:            filepath.Join(CheckoutsPath, name),
		genericCollectors: []pkg.Collector{collectors.Syft{}, collectors.Trivy{}, collectors.CDXGen{}},
		languageCollectors: []pkg.LanguageCollector{
			collectors.NewPythonCollector(), collectors.NewRustCollector(), collectors.NewJVMCollector(),
			collectors.NewGolangCollector(), collectors.NewJSCollector(), collectors.NewRubyCollector()},
	}, nil
}

func (r Repository) ExtractBOMs(includeGenericCollectors bool) (*cdx.BOM, error) {
	var collectedBOMs []*cdx.BOM
	if includeGenericCollectors { //Generate base BOM with generic collectors (syft & trivy)
		for _, c := range r.genericCollectors {
			log.WithField("repository", r).Infof("extracting SBOMs with: %s", c)
			bom, err := c.GenerateBOM(r.FSPath)
			if err != nil {
				log.WithFields(log.Fields{
					"repository": r,
					"error":      err,
				}).Debugf("%s failed to collect SBOMs", c)
				continue
			}
			collectedBOMs = append(collectedBOMs, bom)
		}
	}

	for _, c := range r.languageCollectors {
		bom := r.bomsFromCollector(c)
		if bom != nil {
			collectedBOMs = append(collectedBOMs, bom)
		}
	}
	merged, err := bomtools.MergeBoms(collectedBOMs...)
	if err != nil {
		return nil, fmt.Errorf("%s: ExtractBOMs can't merge sboms - %s", r, err)
	}
	return bomtools.FilterOutByScope(merged, cdx.ScopeOptional), nil
}

func (r Repository) bomsFromCollector(collector pkg.LanguageCollector) *cdx.BOM {
	languageFiles, err := findLanguageFiles(r.FSPath, collector.MatchLanguageFiles)
	if err != nil {
		var e noLanguageFilesFoundError
		if errors.As(err, &e) {
			log.WithField("repository", r).Debugf("%s found no language files - skipping ❎ ", collector)
		} else {
			log.WithFields(log.Fields{
				"repository": r,
				"error":      err,
			}).Warnf("%s can't convert repository to roots ❌ ", collector)
		}
		return nil
	}
	log.WithField("repository", r).Infof("extracting SBOMs with %s", collector)
	var collectedBOMs []*cdx.BOM
	for _, collectionPath := range collector.BootstrapLanguageFiles(languageFiles) {
		bom, err := collector.GenerateBOM(collectionPath)
		if err != nil {
			log.WithField("collection path", collectionPath).Debugf("%s failed for %s", collector, r)
			continue
		}
		collectedBOMs = append(collectedBOMs, bom)
	}
	mergedBOM, err := bomtools.MergeBoms(collectedBOMs...)
	if err != nil {
		if errors.Is(err, bomtools.ErrNoBOMsToMerge) {
			log.WithField("repository", r).Warnf("%s found no SBOMs", collector)
		} else {
			log.WithFields(log.Fields{
				"repository": r,
				"error":      err,
			}).Warnf("%s failed to merge SBOMs", collector)
		}
		return nil
	}
	return mergedBOM
}

func (r Repository) String() string {
	return r.Name
}
