package cmd

import (
	"errors"
	"math/rand"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const subCommandHelpMsg = `
Output collect SBOMs to stdout or Dependency track for further analysis.

To Collect SBOMs from private GitHub repositories a valid set of credentials must be provided.
This must be done via environment variables. For example:
export SAC_GITHUB_USERNAME=Shelly 
export SAC_GITHUB_TOKEN=personal-access-token-with-read-scope

To upload SBOMs to Dependency Track a valid API Token and URL must be provided.
This must be done via environment variables. For example:
export SAC_DTRACK_TOKEN=dependency-track-access-token-with-write-scope
export SAC_DTRACK_URL=https://dependency-track.evilcorp.com/`

var rootCmd = &cobra.Command{
	Use:   "software-assets [repo/org] [vcs-url] [flags]",
	Short: "Collects CycloneDX SBOMs from Github repositories",
	Example: `sa-collector repo https://github.com/ReactiveX/RxJava               collect SBOMs from RxJava repository & output them to stdout
sa-collector repo https://github.com/ffuf/ffuf --output=dtrack      collect SBOMs from ffuf repository & upload them to Dependency Track

sa-collector org evil-corp                                          collect SBOMs from evil-corp organization & output them to stdout
sa-collector org evil-corp --output=dtrack                          collect SBOMs from evil-corp organization & upload them to Dependency Track`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var logLevel string

const cantBindFlagTemplate = "can't bind %s flag to viper: %v"

func init() {
	rand.Seed(time.Now().UnixNano())
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		//Setup logrus
		logrus.SetOutput(os.Stdout)
		lvl, err := logrus.ParseLevel(logLevel)
		if err != nil {
			return errors.New("invalid verbosity level - must be one of: debug/info/warn/error/fatal/panic")
		}
		logrus.SetLevel(lvl)
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		return nil
	}

	const outputFlag = "output"
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", logrus.InfoLevel.String(), "Log level: debug/info/warn/error/fatal/panic")
	rootCmd.PersistentFlags().StringP(outputFlag, "o", "stdout", "where to output SBOM results: stdout/dtrack/file")
	if err := viper.BindPFlag(outputFlag, rootCmd.PersistentFlags().Lookup(outputFlag)); err != nil {
		logrus.Fatalf(cantBindFlagTemplate, outputFlag, err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("sac")
	viper.AutomaticEnv() // read in environment variables that match
}
