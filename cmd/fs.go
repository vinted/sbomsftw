package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/vinted/sbomsftw/internal/app"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// filesystem subcommand
const (
	exclusionsFlag        = "exclude"
	codeOwnersFlag        = "code-owners"
	dTrackProjectNameFlag = "dtrack-project-name"
	stripCPEsFlag         = "strip-cpes"
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
		app, err := createAppFromCLI(cmd, false)
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

	stripCPEs, err := cmd.Flags().GetBool(stripCPEsFlag)
	if err != nil {
		return nil, fmt.Errorf(errTemplate, stripCPEsFlag, err)
	}

	return &app.SBOMsFromFilesystemConfig{
		FilesystemPath: args[0],
		CodeOwners:     codeOwners,
		Exclusions:     exclusions,
		ProjectName:    projectName,
		StripCPEs:      stripCPEs,
	}, nil
}

func init() {
	// Usages
	const (
		excludeUsage           = "exclude paths from being scanned using a glob expression (optional)"
		ownersUsage            = "owners of the filesystem - this will be reflected inside Dependency Track (optional)"
		dTrackProjectNameUsage = "project name to use when uploading to dependency-track (optional)"
		stripCPEUsage          = "whether to strip CPEs from the final SBOM. Useful to reduce false positives in DT"
	)

	fsCmd.Flags().StringSlice(codeOwnersFlag, nil, ownersUsage)
	fsCmd.Flags().String(dTrackProjectNameFlag, "", dTrackProjectNameUsage)

	fsCmd.Flags().StringArray(exclusionsFlag, nil, excludeUsage)

	fsCmd.Flags().BoolP(stripCPEsFlag, "x", false, stripCPEUsage)

	rootCmd.AddCommand(fsCmd)
}
