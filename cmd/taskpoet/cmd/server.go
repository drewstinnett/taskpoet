package cmd

import (
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "server",
		Short:  "Run http server",
		Long:   `Run an http server capable of hosting an API as well as remote sync and stuff`,
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			debug, _ := cmd.PersistentFlags().GetBool("debug")
			c := &taskpoet.RouterConfig{
				Debug:       debug,
				LocalClient: poetC,
			}
			r := taskpoet.NewRouter(c)
			checkErr(r.Run())
		},
	}
	cmd.PersistentFlags().BoolP("debug", "d", false, "Run GIN server in Debug mode")
	return cmd
}
