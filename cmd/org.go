package cmd

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:   "org [GitHub Organization URL]",
	Short: "Collect SBOMs from every repository inside the given organization",
	Example: `sa-collector org https://api.github.com/orgs/evil-corp/repos
sa-collector org https://api.github.com/orgs/evil-corp/repos --upload-to-dependency-track --log-level=warn`,
	Long: "Collects SBOMs from every repository inside the given organization." + subCommandHelpMsg,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a valid GitHub organization URL is required")
		}

		if _, err := url.Parse(args[0]); err != nil { // Validate supplied URL early on
			return fmt.Errorf("invalid organization URL supplied: %v", err)
		}

		return cobra.MaximumNArgs(1)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := createAppFromCLI(cmd)
		if err != nil {
			logrus.Fatal(err)
		}

		app.SBOMsFromOrganization(args[0])
	},
}

func init() {
	rootCmd.AddCommand(orgCmd)
}
