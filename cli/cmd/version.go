package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	goVersion "go.hein.dev/go-version"
)

var (
	shortened  = false
	version    = "dev"
	commit     = "none"
	date       = "unknown"
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Version will output the current build information",
		Long:  ``,
		Run: func(_ *cobra.Command, _ []string) {
			var response string
			versionOutput := goVersion.New(version, commit, date)

			if shortened {
				response = versionOutput.ToShortened()
			} else {
				response = versionOutput.ToJSON()
			}
			fmt.Printf("%+v", response)
		},
	}
)

func init() {
	versionCmd.Flags().BoolVarP(&shortened, "short", "s", false, "Use shortened output for version information.")
	rootCmd.AddCommand(versionCmd)
}
