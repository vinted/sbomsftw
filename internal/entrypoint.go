package internal

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/vinted/software-assets/pkg/bomtools"
	"github.com/vinted/software-assets/pkg/repository"
)

func cleanup() {
	log.Info("cleaning up")
	if _, err := os.Stat("/tmp/checkouts"); !os.IsNotExist(err) {
		if err := os.RemoveAll("/tmp/checkouts"); err != nil {
			log.WithField("error", err).Error("can't remove /tmp/checkouts")
		}
	}
	if _, err := os.Stat("/path/to/whatever"); !os.IsNotExist(err) {
		gradleCache := filepath.Join(os.Getenv("HOME"), ".gradle")
		if err := os.RemoveAll(gradleCache); err != nil {
			log.WithField("error", err).Errorf("can't remove %s", gradleCache)
		}
	}
	os.Exit(0) //TODO Fix this so that we exit with a proper status code
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
	err = os.Setenv("ANDROID_HOME", "/Users/ugnius/Library/Android/sdk")
	if err != nil {
		panic("Unable to set ANDROID_HOME variable")
	}
}
//todo filter by scope early

func uploadToDependencyTrack(repositoryName string, bom *cdx.BOM) error {
	bomString, err := bomtools.CDXToString(bom)
	if err != nil {
		return fmt.Errorf("can't convert cdx.BOM to string: %v", err)
	}

	endpoint := viper.GetString("DTRACK_URL")
	apiToken := viper.GetString("DTRACK_TOKEN")
	//todo validate this part
	uploadConfig := NewUploadBOMConfig(endpoint, apiToken, repositoryName, bomString)
	if _, err = UploadBOM(uploadConfig); err != nil {
		log.WithField("error", err).Error("can't upload BOMs to Dependency Track")
		fmt.Printf("Here is your bom: \n%s\n", bomString)
		return err
	}
	return nil
}

func sbomsFromRepositoryInternal(vcsURL string) {
	deleteRepository := func(repositoryPath string) {
		if err := os.RemoveAll(repositoryPath); err != nil {
			log.Errorf("can't remove repository directory: %s\n", err)
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
		log.WithField("error", err).Errorf("can't collect BOMs from %s", repo.Name)
		return
	}
	log.Infof("Collected %d components from %s ⭐ ", len(*bom.Components), repo.Name)
	if viper.GetString("output") == "dtrack" { //TODO Move to enum
		if err = uploadToDependencyTrack(repo.Name, bom); err != nil {
			log.WithField("error", err).Errorf("can't upload BOMs from %s to dependency track", repo.Name)
		}
		return
	}
	// bomString, err := bomtools.CDXToString(bom)
	// if err != nil {
	// 	log.WithField("error", err).Error("can't convert cdx.BOM to string")
	// 	return
	// }
	// fmt.Println(bomString)
}

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
		}
	})
	if err != nil {
		log.WithField("error", err).Fatal("Collection failed! - Exiting")
	}
}
