package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"

	// Include all plugins
	_ "github.com/drewstinnett/taskpoet/plugins/task/all"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
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
	return cmd
}
