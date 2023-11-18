package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// commentCmd represents the comment command
func newCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "comment",
		Short:             "Add a comment to a task",
		Long:              `Comments are just little text notes, with a date of when they were entered`,
		Args:              cobra.MinimumNArgs(2),
		ValidArgsFunction: completeActive,
		Run: func(cmd *cobra.Command, args []string) {
			task, err := poetC.Task.GetWithPartialID(args[0], "", "")
			checkErr(err)
			checkErr(task.AddComment(strings.Join(args[1:], " ")))
			_, err = poetC.Task.Edit(task)
			checkErr(err)
		},
	}
	return cmd
}
