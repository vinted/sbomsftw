package internal

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/vinted/software-assets/pkg/dtrack"

	"github.com/vinted/software-assets/pkg/collectors"

	cdx "github.com/CycloneDX/cyclonedx-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vinted/software-assets/pkg/bomtools"
	"github.com/vinted/software-assets/pkg/repository"
)

/*
SBOMsFromRepository given a VCS URL, collect SBOMs from a single repository.
Collected SBOMs will be outputted based on the --output CLI switch.
*/
func SBOMsFromRepository(vcsURL string) {
	setup()

	defer cleanup()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigs
		cancel()
	}()
	sbomsFromRepositoryInternal(ctx, vcsURL)
}

/*
SBOMsFromOrganization given a GitHub organization URL, collect SBOMs from every single repository.
Each collected SBOM will be outputted based on the --output CLI switch.
*/
func SBOMsFromOrganization(organizationURL string) {
	setup()

	defer cleanup()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigs
		cancel()
	}()

	githubUsername := viper.GetString(EnvKeyGithubUsername)
	githubAPIToken := viper.GetString(EnvKeyGithubToken)

	reqConfig := NewGetRepositoriesConfig(ctx, organizationURL, githubUsername, githubAPIToken)
	err := WalkRepositories(reqConfig, func(repositoryURLs []string) {
		for _, u := range repositoryURLs {
			select {
			case <-ctx.Done():
				return
			default:
				sbomsFromRepositoryInternal(ctx, u)
			}
		}
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		log.WithError(err).Fatal("Collection failed! Can't recover - exiting")
	}
}

/*
SBOMsFromFilesystem given a filesystem path, collect SBOMs from every subdirectory recursively.
In order not to recurse into certain subdirectories, pass them via --exclude switch.
Each collected SBOM will be outputted based on the --output CLI switch.
*/
func SBOMsFromFilesystem(fsPath string) {
	const errMsg = "File-system SBOM collection failed"

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigs
		cancel()
	}()

	exclusions := viper.GetStringSlice(CLIKeyExclusions)
	log.WithField("exclusions", exclusions).Infof("Extracting SBOMs from %s", fsPath)
	sboms, err := collectors.Syft{Exclusions: exclusions}.GenerateBOM(ctx, fsPath)

	if errors.Is(err, context.Canceled) {
		return // User cancelled - return
	}

	if err != nil {
		log.WithError(err).Fatal(errMsg)
	}

	if sboms == nil || sboms.Components == nil || len(*sboms.Components) == 0 {
		log.Warnf("no SBOMs were collected from %s", fsPath)
		return
	}

	sboms, err = bomtools.MergeBoms(sboms)

	if err != nil {
		log.WithError(err).Fatal(errMsg)
	}

	sboms = bomtools.FilterOutByScope(sboms, cdx.ScopeOptional)

	log.Infof("Collected %d SBOM components from %s", len(*sboms.Components), fsPath)
	outputSBOMs(ctx, sboms, fsPath)
}

// sbomsFromRepositoryInternal collect SBOMs from a single repository, given the VCS URL of the repository.
func sbomsFromRepositoryInternal(ctx context.Context, vcsURL string) {
	deleteRepository := func(repositoryPath string) {
		if err := os.RemoveAll(repositoryPath); err != nil {
			log.WithError(err).Errorf("can't remove repository at: %s", repositoryPath)
		}
	}

	repo, err := repository.New(ctx, vcsURL, repository.Credentials{
		Username:    viper.GetString(EnvKeyGithubUsername),
		AccessToken: viper.GetString(EnvKeyGithubToken),
	})
	if errors.Is(err, context.Canceled) {
		return
	}

	if err != nil {
		log.WithError(err).Errorf("can't clone %s", vcsURL)
		return
	}

	defer deleteRepository(repo.FSPath)
	sboms, err := repo.ExtractSBOMs(ctx, true)

	if errors.Is(err, context.Canceled) {
		return
	}

	if err != nil {
		log.WithError(err).Errorf("can't collect SBOMs from %s", repo.Name)
		return
	}

	if sboms == nil || sboms.Components == nil || len(*sboms.Components) == 0 {
		log.Warnf("no SBOMs were collected from %s", repo.Name)
		return
	}

	log.Infof("Collected %d SBOM components from %s", len(*sboms.Components), repo.Name)
	outputSBOMs(ctx, sboms, repo.Name)
}

/*
outputSBOMs Output SBOMs depending on the specified CLI flags.
When uploading to Dependency Track - dtrack-project-name CLI switch takes
precedence over the projectName argument
*/
func outputSBOMs(ctx context.Context, sboms *cdx.BOM, projectName string) {
	outputLocation := viper.GetString(CLIKeyOutput)
	if viper.GetString(CLIKeyDTrackProjectName) != "" { // CLI switch takes precedence
		projectName = viper.GetString(CLIKeyDTrackProjectName)
	}

	switch outputLocation {
	case OutputValueDtrack:
		uploadSBOMToDependencyTrack(ctx, projectName, sboms)
	case OutputValueStdout:
		printSBOMToStdout(sboms)
	default:
		writeSBOMToFile(sboms, outputLocation)
	}
}

// uploadSBOMToDependencyTrack SBOM Output function: Dependency track.
func uploadSBOMToDependencyTrack(ctx context.Context, projectName string, sboms *cdx.BOM) {
	const errMsg = "can't upload SBOMs to Dependency Track"

	endpoint := viper.GetString(EnvKeyDTrackURL)
	apiToken := viper.GetString(EnvKeyDTrackToken)

	// Validate dependency track environment variables
	if endpoint == "" {
		reason := fmt.Sprintf("%s_%s env variable is missing", EnvPrefix, EnvKeyDTrackURL)
		log.WithField("reason", reason).Error(errMsg)
		return
	}

	if _, err := url.ParseRequestURI(endpoint); err != nil {
		reason := fmt.Sprintf("%s_%s env variable is not a valid URL", EnvPrefix, EnvKeyDTrackURL)
		log.WithField("reason", reason).Error(errMsg)
		return
	}

	if apiToken == "" {
		reason := fmt.Sprintf("%s_%s env variable is missing", EnvPrefix, EnvKeyDTrackToken)
		log.WithField("reason", reason).Error(errMsg)
		return
	}

	// TODO Creating new instance on every upload - not very efficient. Remove after requests.go is refactored
	client, err := dtrack.NewClient(endpoint, apiToken)
	if err != nil {
		log.WithField("reason", err).Error(errMsg)
		return
	}

	projectUUID, err := client.CreateProject(ctx, dtrack.CreateProjectPayload{
		Name:       projectName,
		Tags:       []string{"vinted"}, // Hardcoded for now - use tags from repo on next commit
		CodeOwners: "",                 // Add code owners later on
	})

	if errors.Is(err, context.Canceled) {
		return
	}

	if err != nil {
		log.WithField("reason", err).Error(errMsg)
		return
	}

	err = client.UploadSBOMs(ctx, dtrack.UploadSBOMsPayload{
		Sboms:       sboms,
		ProjectUUID: projectUUID,
	})

	if errors.Is(err, context.Canceled) {
		return
	}

	if err != nil {
		log.WithField("reason", err).Error(errMsg)
		return
	}

	log.Infof("SBOMS from %s were successfully uploaded to Dependency Track", projectName)
}

// printSBOMToStdout SBOM Output function: Stdout.
func printSBOMToStdout(bom *cdx.BOM) {
	bomString, err := bomtools.CDXToString(bom)
	if err != nil {
		log.WithError(err).Error("can't print SBOMs to stdout")

		return
	}
	fmt.Println(bomString)
}

// writeSBOMToFile SBOM Output function: File.
func writeSBOMToFile(bom *cdx.BOM, resultsFile string) {
	bomString, err := bomtools.CDXToString(bom)
	if err != nil {
		log.WithError(err).Error("can't write SBOMs to file")
		return
	}
	if err = os.WriteFile(resultsFile, []byte(bomString), 0o644); err != nil {
		log.WithError(err).Errorf("can't write SBOMs to %s", resultsFile)
	}
}

// Setup & cleanup functions.

func cleanup() {
	log.Debug("cleaning up - bye!")

	if _, err := os.Stat(repository.CheckoutsPath); !os.IsNotExist(err) {
		if err = os.RemoveAll(repository.CheckoutsPath); err != nil {
			log.WithError(err).Errorf("can't remove %s", repository.CheckoutsPath)
		}
	}
	gradleCache := filepath.Join(os.Getenv("HOME"), ".gradle")

	if _, err := os.Stat(gradleCache); !os.IsNotExist(err) {
		if err = os.RemoveAll(gradleCache); err != nil {
			log.WithError(err).Errorf("can't remove %s", gradleCache)
		}
	}
	os.Exit(0) // TODO Fix this so that we exit with a proper status code
}

func setup() {
	if err := os.Setenv("GEM_HOME", filepath.Join(os.Getenv("HOME"), ".gem")); err != nil {
		log.Fatal("Can't set GEM_HOME env variable. Exiting")
		os.Exit(1)
	}

	if err := os.Setenv("PATH", os.Getenv("PATH")+":"+"/usr/local/bin"); err != nil {
		log.Fatal("Can't append /usr/local/bin to PATH . Exiting")
		os.Exit(1)
	}

	if _, err := os.Stat(repository.CheckoutsPath); !os.IsNotExist(err) {
		if err = os.RemoveAll(repository.CheckoutsPath); err != nil {
			log.WithError(err).Errorf("can't remove %s", repository.CheckoutsPath)
		}
	}

	const warnTemplate = "env variable %s is not set. Private GitHub repositories won't be cloned"
	if viper.GetString(EnvKeyGithubUsername) == "" {
		log.Warnf(warnTemplate, fmt.Sprintf("%s_%s", EnvPrefix, EnvKeyGithubUsername))
		return
	}

	if viper.GetString(EnvKeyGithubToken) == "" {
		log.Warnf(warnTemplate, fmt.Sprintf("%s_%s", EnvPrefix, EnvKeyGithubToken))
		return
	}
}
