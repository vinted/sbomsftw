package cmd

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo [GitHub repository URL] [flags]",
	Short: "Collect SBOMs from a single repository",
	Example: `sa-collector repo https://github.com/ReactiveX/RxJava
sa-collector repo https://github.com/ffuf/ffuf --output sboms.json --log-level warn`,
	Long: "Collect SBOMs from a single repository." + subCommandHelpMsg,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a valid repository URL is required")
		}
		if _, err := url.Parse(args[0]); err != nil {
			return fmt.Errorf("invalid repository URL supplied: %v", err)
		}

		return cobra.MaximumNArgs(1)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := createAppFromCLI(cmd, true)
		if err != nil {
			logrus.Fatal(err)
		}

		app.SBOMsFromRepository(args[0])
	},
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
