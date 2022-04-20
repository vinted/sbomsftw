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
