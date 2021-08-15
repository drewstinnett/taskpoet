package cmd

import (
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run http server",
	Long:  `Run an http server capable of hosting an API as well as remote sync and stuff`,
	Run: func(cmd *cobra.Command, args []string) {
		debug, _ := cmd.PersistentFlags().GetBool("debug")
		c := &taskpoet.RouterConfig{
			Debug:       debug,
			LocalClient: localClient,
		}
		r := taskpoet.NewRouter(c)
		r.Run()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	serverCmd.PersistentFlags().BoolP("debug", "d", false, "Run GIN server in Debug mode")
}
