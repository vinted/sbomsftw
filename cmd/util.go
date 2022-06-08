package cmd

import (
	"fmt"

	"github.com/vinted/software-assets/internal/app"

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

	if uploadToDependencyTrack {
		baseURL := viper.GetString(envKeyDTrackURL)
		apiToken := viper.GetString(envKeyDTrackToken)
		options = append(options, app.WithDependencyTrack(baseURL, apiToken))
	}

	tags, err := cmd.Flags().GetStringSlice(tagsFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, tagsFlag)
	}

	options = append(options, app.WithTags(tags))

	outputFile, err := cmd.Flags().GetString(outputFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, outputFlag)
	}

	return app.New(outputFile, options...)
}
