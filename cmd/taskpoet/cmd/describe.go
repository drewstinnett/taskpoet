package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newDescribeCmd shows the details of a task
func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "describe TASK",
		Short:             "Describe a task",
		Long:              `Describe a task...showing details, all that cool stuff`,
		Args:              cobra.ExactArgs(1),
		Aliases:           []string{"desc", "d", "info"},
		ValidArgsFunction: completeAny,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := mustPoet()
			task, err := p.Store.GetByPrefix(args[0])
			if err != nil {
				return err
			}
			out, err := p.DescribeTask(*task)
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}
	return cmd
}
