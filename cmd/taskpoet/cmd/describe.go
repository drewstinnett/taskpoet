package cmd

import (
	"github.com/spf13/cobra"
)

// describeCmd represents the describe command
func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "describe",
		Short:   "Describe a task",
		Long:    `Describe a task...showing details, all that cool stuff`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"desc", "d"},
		Run: func(cmd *cobra.Command, args []string) {
			task, err := poetC.Task.GetWithPartialID(args[0], "", "")
			checkErr(err)
			err = poetC.Task.Describe(task)
			checkErr(err)
		},
	}
	return cmd
}
