package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listPluginsCmd represents the listPlugins command
func newPluginsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "list",
		Short:  "List Plugins",
		Long:   `List Plugins`,
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			ps, err := poetC.Task.GetPlugins()
			checkErr(err)
			for name, c := range ps {
				p := c()
				fmt.Printf("%+v - %+v\n", name, p.Description())
				// err := localClient.Task.SyncPlugin(p)
				// CheckErr(err)
			}
		},
	}
	return cmd
}
