package cmd

import (
	"fmt"
	"log"

	// Include all plugins
	_ "github.com/drewstinnett/taskpoet/plugins/task/all"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:    "sync",
	Short:  "Sync Tasks from Plugins",
	Long:   `Pull in tasks from external places like Gitlab, Github...ServiceNow maybe even?`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("sync called")
		ps, err := poetC.Task.GetPlugins()
		checkErr(err)
		for name, c := range ps {
			log.Printf("%+v %+v\n", name, c)
			p := c()
			err := poetC.Task.SyncPlugin(p)
			checkErr(err)
		}
	},
}

func init() {
	pluginsCmd.AddCommand(syncCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// syncCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// syncCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
