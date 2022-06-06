package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vinted/software-assets/internal"
)

func createAppFromCLI(cmd *cobra.Command) (*internal.App, error) {
	const (
		errTemplate  = "can't parse %s flag - exiting"
		warnTemplate = "env variable %s is not set. Private GitHub repositories won't be cloned"
	)

	githubUsername := viper.GetString(envKeyGithubUsername)
	githubToken := viper.GetString(envKeyGithubToken)

	if githubUsername == "" {
		log.Warnf(warnTemplate, fmt.Sprintf("%s_%s", envPrefix, envKeyGithubUsername))
	}
	if githubToken == "" {
		log.Warnf(warnTemplate, fmt.Sprintf("%s_%s", envPrefix, envKeyGithubToken))
	}

	var options []internal.Option

	if githubUsername != "" && githubToken != "" {
		options = append(options, internal.WithGitHubCredentials(githubUsername, githubToken))
	}

	uploadToDependencyTrack, err := cmd.Flags().GetBool(uploadToDTrackFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, uploadToDTrackFlag)
	}

	if uploadToDependencyTrack {
		baseURL := viper.GetString(envKeyDTrackURL)
		apiToken := viper.GetString(envKeyDTrackToken)
		options = append(options, internal.WithDependencyTrack(baseURL, apiToken))
	}

	tags, err := cmd.Flags().GetStringSlice(tagsFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, tagsFlag)
	}

	options = append(options, internal.WithTags(tags))

	outputFile, err := cmd.Flags().GetString(outputFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, outputFlag)
	}

	return internal.New(outputFile, options...)
}
