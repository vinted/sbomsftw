package cmd

import (
	"fmt"

	"github.com/vinted/sbomsftw/internal/app"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func createAppFromCLI(cmd *cobra.Command, verbose bool) (*app.App, error) {
	const (
		errTemplate  = "can't parse %s flag - exiting"
		warnTemplate = "env variable %s is not set. Private GitHub repositories won't be cloned"
	)

	githubUsername := viper.GetString(envKeyGithubUsername)
	githubToken := viper.GetString(envKeyGithubToken)

	if githubUsername == "" && verbose {
		log.Warnf(warnTemplate, fmt.Sprintf("%s_%s", envPrefix, envKeyGithubUsername))
	}
	if githubToken == "" && verbose {
		log.Warnf(warnTemplate, fmt.Sprintf("%s_%s", envPrefix, envKeyGithubToken))
	}

	var options []app.Option

	if githubUsername != "" && githubToken != "" {
		options = append(options, app.WithGitHubCredentials(githubUsername, githubToken))
	}

	uploadToDependencyTrack, err := cmd.Flags().GetBool(uploadToDTrackFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, uploadToDTrackFlag)
	}

	purgeCaches, err := cmd.Flags().GetBool(purgeCacheFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, purgeCacheFlag)
	}

	if purgeCaches {
		options = append(options, app.WithCachePurge())
	}

	softExit, err := cmd.Flags().GetBool(softExitFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, softExitFlag)
	}
	if softExit {
		options = append(options, app.WithSoftExit())
	}

	if uploadToDependencyTrack {
		classifier, err := cmd.Flags().GetString(classifierFlag)
		if err != nil {
			return nil, fmt.Errorf(errTemplate, classifierFlag)
		}

		baseURL := viper.GetString(envKeyDTrackURL)
		apiToken := viper.GetString(envKeyDTrackToken)
		options = append(options, app.WithDependencyTrack(baseURL, apiToken, classifier))
	}

	tags, err := cmd.Flags().GetStringSlice(tagsFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, tagsFlag)
	}

	options = append(options, app.WithTags(tags))

	orgName, err := cmd.Flags().GetString(orgFlag)
	if err != nil {
		log.Warn("github app org won't be used as no org set")
	}
	if orgName != "" {
		options = append(options, app.WithOrganization(orgName))
	}

	outputFile, err := cmd.Flags().GetString(outputFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, outputFlag)
	}

	return app.New(outputFile, options...)
}
