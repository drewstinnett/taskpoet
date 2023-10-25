package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// pluginsCmd represents the plugins command
func newPluginsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plugins",
		Short:   "Task Plugins",
		Aliases: []string{"plugin", "p"},
		Long:    `Task plugin operations`,
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("plugins called")
		},
	}
	cmd.AddCommand((newPluginsListCmd()))
	cmd.AddCommand((newSyncCmd()))
	return cmd
}
