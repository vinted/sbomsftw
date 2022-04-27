package cmd

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/vinted/software-assets/internal"
)

var orgCmd = &cobra.Command{
	Use:   "org [GitHub Organization name]",
	Short: "Collect SBOMs from every repository inside the given organization",
	Example: `sa-collector org evil-corp
sa-collector org evil-corp --output=dtrack --log-level=warn`,
	Long: "Collects SBOMs from every repository inside the given organization." + subCommandHelpMsg,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || !regexp.MustCompile(`^[\w.-]*$`).MatchString(args[0]) {
			return errors.New("please provide a valid GitHub organization name")
		}
		return cobra.MaximumNArgs(1)(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		template := "https://api.github.com/orgs/%s/repos"
		internal.SBOMsFromOrganization(fmt.Sprintf(template, args[0]))
	},
}

func init() {
	const delayFlag = "delay"
	orgCmd.Flags().BoolP(delayFlag, "d", false, "whether to add a random delay when cloning repos")
	if err := viper.BindPFlag(delayFlag, orgCmd.Flags().Lookup(delayFlag)); err != nil {
		logrus.Fatalf(cantBindFlagTemplate, delayFlag, err)
		os.Exit(1)
	}
	rootCmd.AddCommand(orgCmd)
}
