package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vinted/sbomsftw/pkg/dtrack"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const subCommandHelpMsg = `
Output collected SBOMs to stdout/file or Dependency Track for further analysis.

To Collect SBOMs from private GitHub repositories a valid set of credentials must be provided.
This must be done via environment variables. For example:
export SAC_GITHUB_USERNAME=Shelly 
export SAC_GITHUB_TOKEN=personal-access-token-with-read-scope

To upload SBOMs to Dependency Track a valid API Token and base URL must be provided.
This must be done via environment variables. For example:
export SAC_DEPENDENCY_TRACK_TOKEN=dependency-track-access-token-with-write-scope
export SAC_DEPENDENCY_TRACK_URL=https://dependency-track.evilcorp.com/`

// Root & Persistent CLI flags.
const (
	logLevelFlag  = "log-level"
	logFormatFlag = "log-format"

	tagsFlag           = "tags"
	outputFlag         = "output"
	classifierFlag     = "classifier"
	uploadToDTrackFlag = "upload-to-dependency-track"
	useMiddlewareFlag  = "middleware"
	purgeCacheFlag     = "purge-cache"
	softExitFlag       = "soft-exit"
	orgFlag            = "organization"
	excludeReposFlag   = "exclude-repos"
	pageCountFlag      = "page-count"
	pageIndexFlag      = "page-index"
)

// ENV keys.
const (
	envKeyGithubUsername = "GITHUB_USERNAME"
	envKeyGithubToken    = "GITHUB_TOKEN" //nolint:gosec
	envKeyDTrackURL      = "DEPENDENCY_TRACK_URL"
	envKeyDTrackToken    = "DEPENDENCY_TRACK_TOKEN"
	envKeyMiddleware     = "MIDDLEWARE_URL"
	envKeyMiddlewareUser = "MIDDLEWARE_USER"
	envKeyMiddlewarePass = "MIDDLEWARE_PASS"
)

const envPrefix = "SAC" // Software Asset Collector.

// Log formats.
const (
	logFormatSimple = "simple"
	logFormatFancy  = "fancy"
	logFormatJSON   = "json"
)

var (
	logLevel  string
	logFormat string
)

var rootCmd = &cobra.Command{
	Use:   "subcommand [repo/org/fs] [vcs-url/filesystem-path] [flags]",
	Short: "Collects CycloneDX SBOMs from Github repositories and filesystems",
	Example: `sa-collector repo https://github.com/ReactiveX/RxJava                  collect SBOMs from RxJava repository & output them to stdout
sa-collector repo https://github.com/ffuf/ffuf --output sboms.json     collect SBOMs from ffuf repository & write results to sboms.json

sa-collector org https://api.github.com/orgs/evil-corp/repos           collect SBOMs from evil-corp organization & output them to stdout

sa-collector fs /usr/local/bin --upload-to-dependency-track            collect SBOMs recursively from /usr/local/bin directory & upload them to Dependency Track
sa-collector fs / --exclusions './root'                                collect SBOMs recursively from root directory while excluding /root directory`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func setupLogrus(logLevel, logFormat string) error {
	// Error messages.
	const (
		invalidLogFormatError = "invalid log format - must be one of: simple/fancy/json"
		invalidLogLevelError  = "invalid log level - must be one of: debug/info/warn/error/fatal/panic"
	)

	logrus.SetOutput(os.Stdout)
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return errors.New(invalidLogLevelError)
	}
	logrus.SetLevel(lvl)

	switch logFormat {
	case logFormatSimple:
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	case logFormatFancy:
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	case logFormatJSON:
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		return errors.New(invalidLogFormatError)
	}

	return nil
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return setupLogrus(logLevel, logFormat)
	}

	// Usages.
	const (
		logLevelUsage  = "log level: debug/info/warn/error/fatal/panic"
		logFormatUsage = "log format: simple/fancy/json"

		outputUsage                  = "where to output SBOM results: (defaults to stdout when unspecified)"
		uploadToDependencyTrackUsage = "whether to upload collected SBOMs to Dependency Track (default: false)"
		tagsUsage                    = "tags to use when SBOMs are uploaded to Dependency Track (optional)"
		purgeCacheUsage              = "whether to purge gradle and go caches after a successful run (default: false)"
		orgFlagUsage                 = "used when using organization github app"
		excludeReposFlagUsage        = "used to exclude repos from gathering on org mode"
		softExitUsage                = "used on cleanup to exit soft without crashing"
		useMiddlewareUsage           = "used to change the dependency-track url to your own supplied API for SBOM consumption"
		pageCountFlagUsage           = "used with pagination per org to specify slice of pages"
		pageIndexFlagUsage           = "used with pagination per org to specify index of how many slices"
	)

	const classifierUsageTemplate = "classifier to use when uploading to Dependency Track. Valid values are: %s"
	classifierUsage := fmt.Sprintf(classifierUsageTemplate, dtrack.GetValidClassifiersString())

	rootCmd.PersistentFlags().StringVarP(&logFormat, logFormatFlag, "f", logFormatSimple, logFormatUsage)
	rootCmd.PersistentFlags().StringVarP(&logLevel, logLevelFlag, "l", logrus.InfoLevel.String(), logLevelUsage)

	rootCmd.PersistentFlags().StringP(outputFlag, "o", "", outputUsage)
	rootCmd.PersistentFlags().StringSliceP(tagsFlag, "t", nil, tagsUsage)

	rootCmd.PersistentFlags().StringP(classifierFlag, "c", dtrack.ValidClassifiers[0], classifierUsage)
	rootCmd.PersistentFlags().BoolP(uploadToDTrackFlag, "u", false, uploadToDependencyTrackUsage)

	rootCmd.PersistentFlags().BoolP(useMiddlewareFlag, "m", false, useMiddlewareUsage)

	rootCmd.PersistentFlags().BoolP(purgeCacheFlag, "p", false, purgeCacheUsage)
	rootCmd.PersistentFlags().BoolP(softExitFlag, "s", false, softExitUsage)

	rootCmd.PersistentFlags().StringP(orgFlag, "g", "", orgFlagUsage)
	rootCmd.PersistentFlags().StringSlice(excludeReposFlag, nil, excludeReposFlagUsage)

	rootCmd.PersistentFlags().IntP(pageCountFlag, "r", 0, pageCountFlagUsage)
	rootCmd.PersistentFlags().IntP(pageIndexFlag, "y", 0, pageIndexFlagUsage)
}

func initConfig() {
	viper.SetEnvPrefix(strings.ToLower(envPrefix))
	viper.AutomaticEnv() // read in environment variables that match.
}
