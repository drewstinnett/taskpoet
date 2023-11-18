package cmd

import (
	"github.com/charmbracelet/log"

	"github.com/spf13/cobra"
)

// completeCmd represents the complete command
func newCompleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "done TASK",
		Short:   "Mark a task as done",
		Long:    `Mark a task as done`,
		Aliases: []string{"c", "complete", "finish"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			task, err := poetC.Task.GetWithPartialID(args[0], "", "/active")
			checkErr(err)
			checkErr(poetC.Task.Complete(task))
			log.Info("Completed task, nice work!", "task", task.Description, "id", task.ShortID())
		},
		ValidArgsFunction: completeActive,
	}
	cmd.PersistentFlags().IntP("limit", "l", 100, "Limit to N results")
	return cmd
}
