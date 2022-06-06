package cmd

import (
	"errors"
	"fmt"
	"github.com/vinted/software-assets/internal/app"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// filesystem subcommand
const (
	exclusionsFlag        = "exclude"
	codeOwnersFlag        = "code-owners"
	dTrackProjectNameFlag = "dtrack-project-name"
)

var fsCmd = &cobra.Command{
	Use:   "fs [Filesystem path] [flags]",
	Short: "Collect SBOMs from a filesystem path",
	Example: `sa-collector fs /usr/local/bin
sa-collector fs /usr/bin --dtrack-project-name 'eu-redis-node-1'
sa-collector fs / --exclusions './root' --log-level=warn --code-owners 'sre@evil-corp.com'`,
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
		app, err := createAppFromCLI(cmd)
		if err != nil {
			logrus.Fatal(err)
		}

		config, err := createSBOMsFromFilesystemConfig(cmd, args)
		if err != nil {
			logrus.Fatal(err)
		}

		app.SBOMsFromFilesystem(config)
	},
}

func createSBOMsFromFilesystemConfig(cmd *cobra.Command, args []string) (*app.SBOMsFromFilesystemConfig, error) {
	const errTemplate = "can't parse %s flag: %v"

	codeOwners, err := cmd.Flags().GetStringSlice(codeOwnersFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, codeOwnersFlag, err)
	}

	exclusions, err := cmd.Flags().GetStringArray(exclusionsFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, exclusionsFlag, err)
	}

	projectName, err := cmd.Flags().GetString(dTrackProjectNameFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, dTrackProjectNameFlag, err)
	}

	return &app.SBOMsFromFilesystemConfig{
		FilesystemPath: args[0],
		CodeOwners:     "CODE OWNERS: \n" + strings.Join(codeOwners, "\n"), // TODO move this formatting out later on
		Exclusions:     exclusions,
		ProjectName:    projectName,
	}, nil
}

func init() {
	// Usages
	const (
		excludeUsage           = "exclude paths from being scanned using a glob expression (optional)"
		ownersUsage            = "owners of the filesystem - this will be reflected inside Dependency Track (optional)"
		dTrackProjectNameUsage = "project name to use when uploading to dependency-track (optional)"
	)

	fsCmd.Flags().StringSlice(codeOwnersFlag, nil, ownersUsage)
	fsCmd.Flags().String(dTrackProjectNameFlag, "", dTrackProjectNameUsage)

	fsCmd.Flags().StringArray(exclusionsFlag, nil, excludeUsage)

	rootCmd.AddCommand(fsCmd)
}
