/*
Copyright © 2022 Infosec Team <infosec@vinted.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"errors"
	"os"

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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "software-assets [repo/org] [vcs-url] [flags]",
	Short: "Collects CycloneDX SBOMs from Github repositories",
	Example: `sa-collector repo https://github.com/ReactiveX/RxJava               collect SBOMs from RxJava repository & output them to stdout
sa-collector repo https://github.com/ffuf/ffuf --output=dtrack      collect SBOMs from ffuf repository & upload them to Dependency Track

sa-collector org evil-corp                                          collect SBOMs from evil-corp organization & output them to stdout
sa-collector org evil-corp --output=dtrack                          collect SBOMs from evil-corp organization & upload them to Dependency Track`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var logLevel string

func init() {
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

	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", logrus.InfoLevel.String(), "Log level: debug/info/warn/error/fatal/panic")
	rootCmd.PersistentFlags().StringP("output", "o", "stdout", "where to output SBOM results: stdout/dtrack/file")
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

// initConfig reads in ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix("sac")
	viper.AutomaticEnv() // read in environment variables that match
}
