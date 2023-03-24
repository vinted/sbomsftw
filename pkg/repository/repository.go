package repository

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/vinted/sbomsftw/pkg/collectors"

	"github.com/go-git/go-git/v5/plumbing/object"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	log "github.com/sirupsen/logrus"
	"github.com/vinted/sbomsftw/pkg"
	"github.com/vinted/sbomsftw/pkg/bomtools"
)

const CheckoutsPath = "/tmp/checkouts/"

type Credentials struct {
	Username    string
	AccessToken string
}

type Repository struct {
	Name               string
	FSPath             string
	CodeOwners         []string
	genericCollectors  []pkg.Collector
	languageCollectors []pkg.LanguageCollector
}

type BadVCSURLError struct {
	URL string
}

func (b BadVCSURLError) Error() string {
	return fmt.Sprintf("invalid VCS URL supplied %s\n", b.URL)
}

/*
New clones the repository supplied in the vcsURL parameter and returns a new Repository instance.
If repository is private credentials must be supplied.
*/
func New(ctx context.Context, vcsURL string, credentials Credentials) (*Repository, error) {
	urlPaths := strings.Split(vcsURL, "/")
	if len(urlPaths) == 0 {
		return nil, BadVCSURLError{URL: vcsURL}
	}

	name := strings.TrimSuffix(urlPaths[len(urlPaths)-1], ".git")
	fsPath := filepath.Join(CheckoutsPath, name)

	const cloneDepth = 100 // Clone only 100 most recent commits, this saves bandwidth & disk-space

	cloneOptions := &git.CloneOptions{
		URL:          vcsURL,
		SingleBranch: true,
		Tags:         git.NoTags,
		Depth:        cloneDepth,
	}

	log.WithField("VCS URL", vcsURL).Infof("cloning %s into %s", name, fsPath)
	clonedRepository, err := git.PlainCloneContext(ctx, fsPath, false, cloneOptions)
	if err != nil {
		// Retry to clone the repo with credentials if failed
		cloneOptions.Auth = &http.BasicAuth{Username: credentials.Username, Password: credentials.AccessToken}
		clonedRepository, err = git.PlainCloneContext(ctx, fsPath, false, cloneOptions)
		if err != nil {
			return nil, err
		}
	}

	return &Repository{
		Name:       name,
		FSPath:     fsPath,
		CodeOwners: parseCodeOwners(name, clonedRepository),
		genericCollectors: []pkg.Collector{
			collectors.Syft{}, collectors.CDXGen{}, collectors.RetireJS{},
		},
		languageCollectors: []pkg.LanguageCollector{
			collectors.NewPythonCollector(), collectors.NewRustCollector(), collectors.NewJVMCollector(),
			collectors.NewGolangCollector(), collectors.NewJSCollector(), collectors.NewRubyCollector(),
		},
	}, nil
}

func parseCodeOwners(repositoryName string, repository *git.Repository) []string {
	const errMsgTemplate = "can't parse code owners from %s"

	commitIterator, err := repository.Log(&git.LogOptions{All: true})
	if err != nil {
		log.WithError(err).Errorf(errMsgTemplate, repositoryName) // Not a critical error - log & forget

		return nil
	}

	// Map contributor email to its commit count
	contributorsToCommitCount := make(map[string]int)

	err = commitIterator.ForEach(func(c *object.Commit) error {
		contributorsToCommitCount[c.Author.Email] = contributorsToCommitCount[c.Author.Email] + 1
		return nil
	})

	if err != nil {
		log.WithError(err).Errorf(errMsgTemplate, repositoryName) // Not a critical error - log & forget

		return nil
	}

	contributorEmails := make([]string, 0, len(contributorsToCommitCount))
	for email := range contributorsToCommitCount {
		contributorEmails = append(contributorEmails, email)
	}

	// Sort contributors by their commit count in descending order
	sort.Slice(contributorEmails, func(i, j int) bool {
		return contributorsToCommitCount[contributorEmails[i]] > contributorsToCommitCount[contributorEmails[j]]
	})

	return contributorEmails
}

/*
ExtractSBOMs extracts SBOMs for every possible language from the repository.
If includeGenericCollectors is set to true then additional collectors such as:
syft & retirejs & cdxgen are executed against the repository as well. This tends to produce richer SBOM results
*/
func (r Repository) ExtractSBOMs(ctx context.Context, includeGenericCollectors bool) (*cdx.BOM, error) {
	var collectedSBOMs []*cdx.BOM
	// Generate base SBOM with generic collectors (syft/retirejs/cdxgen)
	if includeGenericCollectors {
		for _, c := range r.genericCollectors {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				log.WithField("repository", r.Name).Infof("extracting SBOMs with: %s", c)
				bom, err := c.GenerateBOM(ctx, r.FSPath)

				if err == nil {
					collectedSBOMs = append(collectedSBOMs, bom)
					continue
				}

				log.WithFields(log.Fields{"repository": r.Name, "error": err}).Debugf("%s failed to collect SBOMs", c)
			}
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err() // Return early if user cancelled
	}

	for res := range r.filterApplicableCollectors() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			collector := res.collector
			languageFiles := res.languageFiles
			log.WithField("repository", r.Name).Infof("extracting SBOMs with %s", collector)

			/*
				Generate SBOMs from every directory that contains language files
			*/
			var sbomsFromCollector []*cdx.BOM
			for _, collectionPath := range collector.BootstrapLanguageFiles(ctx, languageFiles) {
				b, err := collector.GenerateBOM(ctx, collectionPath)
				if err == nil {
					sbomsFromCollector = append(sbomsFromCollector, b)
					continue
				}
				logFields := log.Fields{"collection path": collectionPath, "error": err}
				log.WithFields(logFields).Debugf("%s failed for %s", collector, r)
			}
			/*
				Collector traversed the whole repository and generated SBOMs for every collection path.
				Time to merge those SBOMs into a single one
			*/

			mergedSBOM, err := bomtools.MergeSBOMs(sbomsFromCollector...)
			if err == nil {
				// Append merged SBOM from this collector & move on to the next one
				collectedSBOMs = append(collectedSBOMs, mergedSBOM)
				continue
			}
			if errors.Is(err, bomtools.ErrNoBOMsToMerge) {
				log.WithField("repository", r.Name).Debugf("%s found no SBOMs", collector)
				continue
			}
			logFields := log.Fields{"repository": r.Name, "error": err}
			log.WithFields(logFields).Debugf("%s failed to merge SBOMs", collector)
		}
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// All collectors are finished - merge collected SBOMs into a single one
		merged, err := bomtools.MergeSBOMs(collectedSBOMs...)
		if err != nil {
			return nil, fmt.Errorf("%s: ExtractSBOMs can't merge sboms - %s", r, err)
		}

		/*
			Filter optional SBOM components. Some libraries are only used for development purposes: E.g. junit/mockito.
			These libraries aren't included in release builds & we don't want to track them for vulnerabilities.
			These test libraries have the CycloneDX optional scope attached to them - so we filter out all optional
			components before returning the final SBOM.
		*/
		result := bomtools.FilterOutComponentsWithoutAType(merged)
		result = bomtools.FilterOutByScope(result, cdx.ScopeOptional)

		return result, nil
	}
}

type applicableCollector struct {
	collector     pkg.LanguageCollector
	languageFiles []string
}

/*
filterApplicableCollectors - walk the repository and identify which collectors are applicable. E.g.
given the following repository structure:

	/tmp/some-repo/Cargo.toml
	/tmp/some-repo/file1.rs
	/tmp/some-repo/file2.rs
	/tmp/some-repo/inner-dir/yarn.lock
	/tmp/some-repo/inner-dir/index.js

filterApplicableCollectors would return a closed channel with the following elements:

	applicableCollector struct {
		collector:     pkg.Rust
		languageFiles: ["/tmp/some-repo/Cargo.toml"]
	},
	applicableCollector struct {
		collector:     pkg.JS
		languageFiles: ["/tmp/some-repo/inner-dir/yarn.lock"]
	}
*/
func (r Repository) filterApplicableCollectors() <-chan applicableCollector {
	// walk this repository with a given collector - see if it can find any language files
	filter := func(wg *sync.WaitGroup, collector pkg.LanguageCollector, results chan<- applicableCollector) {
		defer wg.Done()
		languageFiles, err := findLanguageFiles(r.FSPath, collector.MatchLanguageFiles)
		if err == nil {
			results <- applicableCollector{collector: collector, languageFiles: languageFiles}
			return
		}
		var e noLanguageFilesFoundError
		if errors.As(err, &e) {
			log.WithField("repository", r.Name).Debugf("%s found no language files - skipping", collector)
			return
		}
		logFields := log.Fields{"repository": r.Name, "error": err}
		log.WithFields(logFields).Warnf("%s failed to walk repository for language files", collector)
	}

	var wg sync.WaitGroup
	wg.Add(len(r.languageCollectors))
	results := make(chan applicableCollector, len(r.languageCollectors))
	for _, c := range r.languageCollectors {
		go filter(&wg, c, results)
	}
	wg.Wait()
	close(results)

	return results
}

func (r Repository) String() string {
	return r.Name
}
