package internal

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
	"github.com/vinted/software-assets/pkg/repository"
)

/*
SBOMsFromRepository given a VCS URL, collect SBOMs from a single repository.
Collected SBOMs will be outputted based on the --output CLI switch
*/
func SBOMsFromRepository(vcsURL string) {
	setup()
	defer cleanup()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cleanup()
	}()
	sbomsFromRepositoryInternal(vcsURL)
}

/*
SBOMsFromOrganization given a GitHub organization URL, collect SBOMs from every single repository.
Each collected SBOM will be outputted based on the --output CLI switch
*/
func SBOMsFromOrganization(organizationURL string) {
	setup()
	defer cleanup()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cleanup()
	}()

	githubUsername := viper.GetString("GITHUB_USERNAME")
	githubAPIToken := viper.GetString("GITHUB_TOKEN")

	reqConfig := NewGetRepositoriesConfig(organizationURL, githubUsername, githubAPIToken)
	err := WalkRepositories(reqConfig, func(repositoryURLs []string) {
		for _, u := range repositoryURLs {
			sbomsFromRepositoryInternal(u)
			if viper.GetBool("delay") {
				min := 10
				max := 20
				delay := rand.Intn(max-min) + min
				log.Infof("sleeping for %d seconds before cloning another repository", delay)
				time.Sleep(time.Second * time.Duration(delay))
			}
		}
	})
	if err != nil {
		log.WithField("error", err).Fatal("Collection failed! Can't recover - exiting")
	}
}

//sbomsFromRepositoryInternal collect SBOMs from a single repository, given the VCS URL of the repository
func sbomsFromRepositoryInternal(vcsURL string) {
	deleteRepository := func(repositoryPath string) {
		if err := os.RemoveAll(repositoryPath); err != nil {
			log.WithField("error", err).Errorf("can't remove repository at: %s", repositoryPath)
		}
	}

	repo, err := repository.New(vcsURL, repository.Credentials{
		Username:    viper.GetString("GITHUB_USERNAME"),
		AccessToken: viper.GetString("GITHUB_TOKEN"),
	})
	if err != nil {
		log.WithField("error", err).Errorf("can't clone %s", vcsURL)
		return
	}
	defer deleteRepository(repo.FSPath)
	bom, err := repo.ExtractBOMs(true)
	if err != nil {
		log.WithField("error", err).Errorf("can't collect SBOMs from %s", repo.Name)
		return
	}
	if bom == nil || bom.Components == nil || len(*bom.Components) == 0 {
		log.Warnf("no SBOMs were collected from %s", repo.Name)
		return
	}

	log.Infof("Wooho! 🎉 Collected %d SBOM components from %s", len(*bom.Components), repo.Name)
	outputLocation := viper.GetString("output")
	switch outputLocation {
	case "dtrack":
		uploadSBOMToDependencyTrack(repo.Name, bom)
	case "stdout":
		printSBOMToStdout(bom)
	default:
		writeSBOMToFile(bom, outputLocation)
	}
}

//uploadSBOMToDependencyTrack SBOM Output function: Dependency track
func uploadSBOMToDependencyTrack(repositoryName string, bom *cdx.BOM) {
	const errMsg = "can't upload SBOMs to Dependency Track"
	bomString, err := bomtools.CDXToString(bom)
	if err != nil {
		log.WithField("error", err).Error(errMsg)
		return
	}

	endpoint := viper.GetString("DEPENDENCY_TRACK_URL")
	apiToken := viper.GetString("DEPENDENCY_TRACK_TOKEN")

	//Validate dependency track environment variables
	if endpoint == "" {
		log.WithField("error", "SAC_DEPENDENCY_TRACK_URL env variable is missing").Error(errMsg)
		return
	}
	if _, err = url.ParseRequestURI(endpoint); err != nil {
		log.WithField("error", "SAC_DEPENDENCY_TRACK_URL env variable is not a valid URL").Error(errMsg)
		return
	}
	if apiToken == "" {
		log.WithField("error", "SAC_DEPENDENCY_TRACK_TOKEN env variable is missing").Error(errMsg)
		return
	}

	uploadConfig := NewUploadBOMConfig(endpoint, apiToken, repositoryName, bomString)
	if _, err = UploadBOM(uploadConfig); err != nil {
		log.WithField("error", err).Error(errMsg)
		return
	}
	log.Infof("%s bom was successfully uploaded to Dependency Track", repositoryName)
}

//printSBOMToStdout SBOM Output function: Stdout
func printSBOMToStdout(bom *cdx.BOM) {
	bomString, err := bomtools.CDXToString(bom)
	if err != nil {
		log.WithField("error", err).Error("can't print SBOMs to stdout")
		return
	}
	fmt.Println(bomString)
}

//writeSBOMToFile SBOM Output function: File
func writeSBOMToFile(bom *cdx.BOM, resultsFile string) {
	bomString, err := bomtools.CDXToString(bom)
	if err != nil {
		log.WithField("error", err).Error("can't write SBOMs to file")
		return
	}
	if err = os.WriteFile(resultsFile, []byte(bomString), 0644); err != nil {
		log.WithField("error", err).Errorf("can't write SBOMs to %s", resultsFile)
	}
}

//Setup & cleanup functions

func cleanup() {
	log.Debug("cleaning up")
	if _, err := os.Stat(repository.CheckoutsPath); !os.IsNotExist(err) {
		if err = os.RemoveAll(repository.CheckoutsPath); err != nil {
			log.WithField("error", err).Errorf("can't remove %s", repository.CheckoutsPath)
		}
	}
	gradleCache := filepath.Join(os.Getenv("HOME"), ".gradle")
	if _, err := os.Stat(gradleCache); !os.IsNotExist(err) {
		if err = os.RemoveAll(gradleCache); err != nil {
			log.WithField("error", err).Errorf("can't remove %s", gradleCache)
		}
	}
	os.Exit(0) //TODO Fix this so that we exit with a proper status code
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
			log.WithField("error", err).Errorf("can't remove %s", repository.CheckoutsPath)
		}
	}

	const warnTemplate = "env variable %s is not set. Private GitHub repositories won't be cloned"
	if viper.GetString("GITHUB_USERNAME") == "" {
		log.Warnf(warnTemplate, "SAC_GITHUB_USERNAME")
		return
	}
	if viper.GetString("GITHUB_TOKEN") == "" {
		log.Warnf(warnTemplate, "SAC_GITHUB_TOKEN")
		return
	}
}
