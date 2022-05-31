package cmd

import (
	"errors"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/vinted/software-assets/internal"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const subCommandHelpMsg = `
Output collect SBOMs to stdout/file or Dependency track for further analysis.

To Collect SBOMs from private GitHub repositories a valid set of credentials must be provided.
This must be done via environment variables. For example:
export SAC_GITHUB_USERNAME=Shelly 
export SAC_GITHUB_TOKEN=personal-access-token-with-read-scope

To upload SBOMs to Dependency Track a valid API Token and URL must be provided.
This must be done via environment variables. For example:
export SAC_DEPENDENCY_TRACK_TOKEN=dependency-track-access-token-with-write-scope
export SAC_DEPENDENCY_TRACK_URL=https://dependency-track.evilcorp.com/`

// Usages for CLI switches
const (
	logFormatUsage   = "log format: simple/fancy/json"
	logLevelUsage    = "log level: debug/info/warn/error/fatal/panic"
	outputUsage      = "where to output SBOM results: stdout/dtrack/file"
	projectNameUsage = "project name to use when uploading to dependency-track (optional)"
)

// Error messages
const (
	cantBindFlagTemplate  = "can't bind %s flag to viper: %v"
	invalidLogFormatError = "invalid log format - must be one of: simple/fancy/json"
	invalidLogLevelError  = "invalid log level - must be one of: debug/info/warn/error/fatal/panic"
)

// Log formats
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
	Use:   "software-assets [repo/org] [vcs-url] [flags]",
	Short: "Collects CycloneDX SBOMs from Github repositories",
	Example: `sa-collector repo https://github.com/ReactiveX/RxJava               collect SBOMs from RxJava repository & output them to stdout
sa-collector repo https://github.com/ffuf/ffuf --output=dtrack      collect SBOMs from ffuf repository & upload them to Dependency Track

sa-collector org evil-corp                                          collect SBOMs from evil-corp organization & output them to stdout
sa-collector org evil-corp --output=dtrack                          collect SBOMs from evil-corp organization & upload them to Dependency Track

sa-collector fs /usr/local/bin                                      collect SBOMs recursively from /usr/local/bin directory
sa-collector fs / --exclude './root'                                collect SBOMs recursively from root directory while excluding /root directory`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func setupLogrus(logLevel, logFormat string) error {
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
	rand.Seed(time.Now().UnixNano())
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return setupLogrus(logLevel, logFormat)
	}

	rootCmd.PersistentFlags().StringVarP(&logFormat, "log-format", "f", logFormatSimple, logFormatUsage)
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", logrus.InfoLevel.String(), logLevelUsage)

	// Flags that will be later bound to viper
	const (
		output      = "output"
		projectName = "dtrack-project-name"
	)
	rootCmd.PersistentFlags().StringP(output, "o", internal.OutputValueStdout, outputUsage)
	rootCmd.PersistentFlags().String(projectName, "", projectNameUsage)

	if err := viper.BindPFlag(internal.CLIKeyOutput, rootCmd.PersistentFlags().Lookup(output)); err != nil {
		logrus.Fatalf(cantBindFlagTemplate, output, err)
	}

	if err := viper.BindPFlag(internal.CLIKeyDTrackProjectName, rootCmd.PersistentFlags().Lookup(projectName)); err != nil {
		logrus.Fatalf(cantBindFlagTemplate, projectName, err)
	}
}

func initConfig() {
	viper.SetEnvPrefix(strings.ToLower(internal.EnvPrefix))
	viper.AutomaticEnv() // read in environment variables that match
}
