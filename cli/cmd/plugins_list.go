package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listPluginsCmd represents the listPlugins command
var listPluginsCmd = &cobra.Command{
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

func init() {
	pluginsCmd.AddCommand(listPluginsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listPluginsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listPluginsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
