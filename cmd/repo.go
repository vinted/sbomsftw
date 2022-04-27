package cmd

import (
	"errors"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/vinted/software-assets/internal"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo [GitHub repository URL] [flags]",
	Short: "Collect SBOMs from a single repository",
	Example: `sa-collector repo https://github.com/ReactiveX/RxJava
sa-collector repo https://github.com/ffuf/ffuf --output=dtrack --log-level=warn`,
	Long: "Collect SBOMs from a single repository." + subCommandHelpMsg,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a valid repository URL is required")
		}
		if _, err := url.ParseRequestURI(args[0]); err != nil {
			return errors.New("please supply repository URL in a form of: https://github.com/org/repo-name")
		}
		return cobra.MaximumNArgs(1)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		internal.SBOMsFromRepository(args[0])
	},
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
