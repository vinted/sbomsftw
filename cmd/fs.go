package cmd

import (
	"errors"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	"github.com/vinted/software-assets/internal"
)

var fsCmd = &cobra.Command{
	Use:   "fs [Filesystem path] [flags]",
	Short: "Collect SBOMs from a filesystem path",
	Example: `sa-collector fs /usr/local/bin
sa-collector fs / --exclude './root' --log-level=warn`,
	Long: "Collect SBOMs from a filesystem path." + subCommandHelpMsg,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("a valid filesystem path is required")
		}

		if _, err := os.Stat(args[0]); err != nil {
			return err
		}

		return cobra.MaximumNArgs(1)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		internal.SBOMsFromFilesystem(args[0])
	},
}

func init() {
	const exclude = "exclude"
	fsCmd.Flags().StringArrayP(exclude, "e", nil, "exclude paths from being scanned using a glob expression")

	if err := viper.BindPFlag(internal.CLIKeyExclusions, fsCmd.Flags().Lookup(exclude)); err != nil {
		logrus.Fatalf(cantBindFlagTemplate, exclude, err)
	}

	rootCmd.AddCommand(fsCmd)
}
