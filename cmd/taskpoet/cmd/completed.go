package cmd

import (
	"fmt"

	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/spf13/cobra"
)

// newCompletedCmd lists what has been done
func newCompletedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "completed [FILTER]",
		Short:             "List completed tasks",
		Aliases:           []string{"history", "hist"},
		ValidArgsFunction: noComplete,
		Long: `List completed tasks, most recent first.

Any arguments are a case insensitive regex, matched against the description.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			tableOpts, err := tableOptsWithCmd(cmd, args)
			if err != nil {
				return err
			}
			tableOpts.Statuses = []taskpoet.Status{taskpoet.StatusCompleted}
			tableOpts.Columns = []string{"ID", "Completed", "Project", "Description", "Tags"}
			tableOpts.SortBy = taskpoet.ByCompleted{}

			table, err := mustPoet().TaskTable(*tableOpts)
			if err != nil {
				return err
			}
			fmt.Print(table)
			return nil
		},
	}
	bindTableOpts(cmd)
	return cmd
}
