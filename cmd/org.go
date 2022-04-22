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
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	orgCmd.Flags().BoolP("delay", "d", false, "whether to add a random delay when cloning repos (default \"false\")")
	viper.BindPFlag("delay", orgCmd.Flags().Lookup("delay"))
	rootCmd.AddCommand(orgCmd)
}
