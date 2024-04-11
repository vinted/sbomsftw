package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vinted/sbomsftw/pkg"

	"github.com/vinted/sbomsftw/internal"

	cdx "github.com/CycloneDX/cyclonedx-go"
	log "github.com/sirupsen/logrus"
	"github.com/vinted/sbomsftw/pkg/bomtools"
	"github.com/vinted/sbomsftw/pkg/collectors"
	"github.com/vinted/sbomsftw/pkg/dtrack"
	"github.com/vinted/sbomsftw/pkg/repository"
)

type App struct {
	outputFile                     string
	tags                           []string
	githubUsername, githubAPIToken string // TODO Move later on to a separate GitHub client
	dependencyTrackClient          *dtrack.DependencyTrackClient
	purgeCache                     bool
}

type SBOMsFromFilesystemConfig struct {
	ProjectName, FilesystemPath string
	Exclusions, CodeOwners      []string
	StripCPEs                   bool
}

type options struct {
	tags                           []string
	githubUsername, githubAPIToken string // TODO Move later on to a separate GitHub client
	dependencyTrackClient          *dtrack.DependencyTrackClient
	purgeCache                     bool
}

type Option func(options *options) error

func WithDependencyTrack(baseURL, apiToken, classifier string) Option {
	return func(options *options) error {
		if baseURL == "" {
			return errors.New("dependency track base URL can't be empty")
		}

		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return errors.New("dependency track base URL must be a valid URL")
		}

		if apiToken == "" {
			return errors.New("dependency track API token can't be empty")
		}

		client, err := dtrack.NewClient(baseURL, apiToken, dtrack.WithClassifier(classifier))
		if err != nil {
			return fmt.Errorf("can't create dependency track client: %v", err)
		}

		options.dependencyTrackClient = client

		return nil
	}
}

func WithGitHubCredentials(username, apiToken string) Option {
	return func(options *options) error {
		if username == "" {
			return errors.New("GitHub username can't be empty")
		}

		if apiToken == "" {
			return errors.New("GitHub APIToken can't be empty")
		}

		options.githubUsername = username
		options.githubAPIToken = apiToken

		return nil
	}
}

func WithCachePurge() Option {
	return func(options *options) error {
		options.purgeCache = true
		return nil
	}
}

func WithTags(tags []string) Option {
	return func(options *options) error {
		options.tags = tags

		return nil
	}
}

func New(outputFile string, opts ...Option) (*App, error) {
	var options options
	for _, opt := range opts {
		err := opt(&options)
		if err != nil {
			return nil, err
		}
	}

	app := new(App)

	app.outputFile = outputFile

	app.githubUsername = options.githubUsername
	app.githubAPIToken = options.githubAPIToken

	app.tags = options.tags

	app.purgeCache = options.purgeCache
	app.dependencyTrackClient = options.dependencyTrackClient

	return app, nil
}

/*
SBOMsFromRepository given a VCS URL, collect SBOMs from a single repository.
Collected SBOMs will be outputted based on the --output CLI switch.
*/
func (a App) SBOMsFromRepository(repositoryURL string) {
	a.setup()

	defer a.cleanup()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigs
		cancel()
	}()
	a.sbomsFromRepositoryInternal(ctx, repositoryURL)
}

/*
SBOMsFromOrganization given a GitHub organization URL, collect SBOMs from every single repository.
Each collected SBOM will be outputted based on the --output CLI switch.
*/
func (a App) SBOMsFromOrganization(organizationURL string, delayAmount uint16) {
	a.setup()

	defer a.cleanup()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigs
		cancel()
	}()

	/*
		Collect SBOMs From repositories using a timer. Where we wait for [delayAmount uint16]secs to pass before
		processing next repository.
	*/

	collectSBOMsFromRepositories := func(repositoryURLs []string) {
		for idx, repositoryURL := range repositoryURLs {
			if idx == 0 {
				a.sbomsFromRepositoryInternal(ctx, repositoryURL)
				continue
			}

			delay := time.NewTimer(time.Duration(delayAmount) * time.Second)

			select {
			case <-ctx.Done():
				delay.Stop()
				return
			case <-delay.C:
				a.sbomsFromRepositoryInternal(ctx, repositoryURL)
			}
		}
	}

	c := internal.NewGetRepositoriesConfig(ctx, organizationURL, a.githubUsername, a.githubAPIToken)
	err := internal.WalkRepositories(c, collectSBOMsFromRepositories)

	if err != nil && !errors.Is(err, context.Canceled) {
		log.WithError(err).Fatal("Collection failed! Can't recover - exiting")
	}
}

/*
SBOMsFromFilesystem given a filesystem path, collect SBOMs from every subdirectory recursively.
In order not to recurse into certain subdirectories, pass them via --exclude switch.
Each collected SBOM will be outputted based on the --output CLI switch.
*/
func (a App) SBOMsFromFilesystem(config *SBOMsFromFilesystemConfig) {
	const errMsg = "File-system SBOM collection failed"

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigs
		cancel()
	}()

	log.WithField("exclusions", config.Exclusions).Infof("Extracting SBOMs from %s", config.FilesystemPath)
	sboms, err := collectors.Syft{Exclusions: config.Exclusions}.GenerateBOM(ctx, config.FilesystemPath)

	if errors.Is(err, context.Canceled) {
		return // User cancelled - return
	} else if err != nil {
		log.WithError(err).Fatal(errMsg)
	}

	if sboms == nil || sboms.Components == nil || len(*sboms.Components) == 0 {
		log.Warnf("no SBOMs were collected from %s", config.FilesystemPath)

		return
	}

	// We only generate one sbom here
	mergedSBOMparam := bomtools.MergeSBOMParam{
		SBOMs:         []*cdx.BOM{sboms},
		OptionalParam: "device",
	}

	sboms, err = bomtools.MergeSBOMs(mergedSBOMparam)
	if err != nil {
		log.WithError(err).Fatal(errMsg)
	}

	sboms = bomtools.FilterOutComponentsWithoutAType(sboms)
	sboms = bomtools.FilterOutByScope(sboms, cdx.ScopeOptional)

	if config.StripCPEs {
		sboms = bomtools.StripCPEsFromComponents(sboms)
	}

	sboms = bomtools.SetCreatedAtProperty(sboms)

	log.Infof("Collected %d SBOM components from %s", len(*sboms.Components), config.FilesystemPath)

	if a.outputFile == "" {
		a.printSBOMsToStdout(sboms)
	} else {
		a.writeSBOMsToFile(sboms)
	}

	projectName := config.ProjectName
	if projectName == "" {
		projectName = config.FilesystemPath
	}

	a.uploadSBOMsToDependencyTrack(ctx, projectName, sboms, config.CodeOwners)
}

// sbomsFromRepositoryInternal collect SBOMs from a single repository, given the VCS URL of the repository.
func (a App) sbomsFromRepositoryInternal(ctx context.Context, repositoryURL string) {
	deleteRepository := func(repositoryPath string) {
		if err := os.RemoveAll(repositoryPath); err != nil {
			log.WithError(err).Errorf("can't remove repository at: %s", repositoryPath)
		}
	}

	repo, err := repository.New(ctx, repositoryURL, repository.Credentials{
		Username:    a.githubUsername,
		AccessToken: a.githubAPIToken,
	})
	if errors.Is(err, context.Canceled) {
		return
	} else if err != nil {
		log.WithError(err).Errorf("can't clone %s", repositoryURL)

		return
	}

	defer deleteRepository(repo.FSPath)
	sboms, err := repo.ExtractSBOMs(ctx, true)

	if errors.Is(err, context.Canceled) {
		return
	} else if err != nil {
		log.WithError(err).Errorf("can't collect SBOMs from %s", repo.Name)

		return
	}

	if sboms == nil || sboms.Components == nil || len(*sboms.Components) == 0 {
		log.Warnf("no SBOMs were collected from %s", repo.Name)

		return
	}

	log.Infof("Collected %d SBOM components from %s", len(*sboms.Components), repo.Name)

	if a.outputFile == "" {
		a.printSBOMsToStdout(sboms)
	} else {
		a.writeSBOMsToFile(sboms)
	}

	a.uploadSBOMsToDependencyTrack(ctx, repo.Name, sboms, repo.CodeOwners)
}

/*
	Output Functions
*/

// writeSBOMToFile SBOM Output function: File.
func (a App) writeSBOMsToFile(sboms *cdx.BOM) {
	bomString, err := bomtools.CDXToString(sboms)
	if err != nil {
		log.WithError(err).Error("can't write SBOMs to file")
		return
	}
	if err = os.WriteFile(a.outputFile, []byte(bomString), 0o644); err != nil {
		log.WithError(err).Errorf("can't write SBOMs to %s", a.outputFile)
	}
}

// printSBOMsToStdout SBOM Output function: Stdout.
func (a App) printSBOMsToStdout(sboms *cdx.BOM) {
	bomString, err := bomtools.CDXToString(sboms)
	if err != nil {
		log.WithError(err).Error("can't print SBOMs to stdout")

		return
	}
	fmt.Println(bomString)
}

// uploadSBOMsToDependencyTrack SBOM Output function: Dependency track.
func (a App) uploadSBOMsToDependencyTrack(ctx context.Context, projectName string, sboms *cdx.BOM, codeOwners []string) {
	if a.dependencyTrackClient == nil {
		return
	}

	uploadSBOMs := func() error {
		return a.dependencyTrackClient.UploadSBOMs(ctx, dtrack.UploadSBOMsPayload{
			Sboms:       sboms,
			ProjectName: projectName,
			Tags:        a.tags,
			CodeOwners:  codeOwners,
		})
	}

	const (
		successMessage = "SBOMS from %s were successfully uploaded to Dependency Track"
		errorMessage   = "can't upload SBOMs to Dependency Track"
	)

	err := uploadSBOMs()

	if err == nil {
		log.Infof(successMessage, projectName)
		return
	}

	if errors.Is(err, context.Canceled) {
		return
	}

	var e pkg.BadStatusError
	if ok := errors.As(err, &e); !ok {
		log.WithField("reason", err).Error(errorMessage)
		return
	}

	if e.Status < http.StatusInternalServerError {
		log.WithField("reason", err).Error(errorMessage)
		return
	}

	log.WithField("reason", err).Debugf("can't upload SBOMs due to server error - retrying")

	// We have a 500 error here - retry SBOM upload one more time before giving up
	err = uploadSBOMs()

	if errors.Is(err, context.Canceled) {
		return
	}

	if err != nil {
		log.WithField("reason", err).Error(errorMessage)
		return
	}

	log.Infof(successMessage, projectName)
}

// Setup & cleanup functions.

func (a App) cleanup() {
	exitCode := 0

	log.Debug("cleaning up - bye!")

	/*
		Remove a directory if it exists.
		If for some reason directory removal fails. Log the error and set the appropriate exit code
	*/
	removeDirectory := func(directoryPath string) {
		if _, err := os.Stat(directoryPath); !os.IsNotExist(err) {
			if err = os.RemoveAll(directoryPath); err != nil {
				exitCode = 2 // ENOENT

				log.WithError(err).Errorf("can't remove %s", directoryPath)
			}
		}
	}

	removeDirectory(repository.CheckoutsPath) // Always remove checkouts directory

	if a.purgeCache {
		// Build caches
		const (
			goCache     = "go"
			gradleCache = ".gradle"
		)

		// Purge go & .gradle caches if user desires
		removeDirectory(filepath.Join(os.Getenv("HOME"), goCache))
		removeDirectory(filepath.Join(os.Getenv("HOME"), gradleCache))
	}

	os.Exit(exitCode)
}

func (a App) setup() {
	if err := os.Setenv("GEM_HOME", filepath.Join(os.Getenv("HOME"), ".gem")); err != nil {
		log.Fatal("Can't set GEM_HOME env variable. Exiting")
	}

	if err := os.Setenv("PATH", os.Getenv("PATH")+":"+"/usr/local/bin"); err != nil {
		log.Fatal("Can't append /usr/local/bin to PATH . Exiting")
	}

	if _, err := os.Stat(repository.CheckoutsPath); !os.IsNotExist(err) {
		if err = os.RemoveAll(repository.CheckoutsPath); err != nil {
			log.WithError(err).Errorf("can't remove %s", repository.CheckoutsPath)
		}
	}
}
