package cmd

import (
	"strings"

	"github.com/charmbracelet/log"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// logCmd represents the log command
func newLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "log COMPLETED_TASK_DESCRIPTION",
		Short:   "Log a completed task",
		Aliases: []string{"l"},
		Long: `Log a completed task. Useful for when you do something that was not in your
actual TODO list`,
		Run: func(cmd *cobra.Command, args []string) {
			got, err := poetC.Task.Log(&taskpoet.Task{
				Description: strings.Join(args, " "),
			}, &taskpoet.Task{})
			checkErr(err)
			log.Print("logged task", "description", got.Description)
		},
	}
	return cmd
}
