package cmd

import (
	"fmt"

	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/spf13/cobra"
)

// newListCmd lists the pending tasks
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [FILTER]",
		Short:   "List pending tasks, most urgent first",
		Aliases: []string{"ls", "active", "get", "g"},
		Long: `List the tasks that still need doing, most urgent first. Tasks that are
waiting for a later date are hidden, unless --waiting is given.

Any arguments are a case insensitive regex, matched against the description.`,
		Example: `List everything:
$ taskpoet list

Only tasks about the kitchen, in the home project:
$ taskpoet list --project home kitchen`,
		ValidArgsFunction: noComplete,
		RunE: func(cmd *cobra.Command, args []string) error {
			tableOpts, err := tableOptsWithCmd(cmd, args)
			if err != nil {
				return err
			}
			spawnRecurring()
			tableOpts.Statuses = []taskpoet.Status{taskpoet.StatusPending}
			tableOpts.Columns = []string{"ID", "Age", "Due", "Project", "Priority", "Description", "Urgency", "Tags"}
			tableOpts.SortBy = taskpoet.ByUrgency{}
			if !mustGetCmd[bool](cmd, "waiting") {
				tableOpts.Filters = append(tableOpts.Filters, taskpoet.FilterHidden)
			}

			table, err := mustPoet().TaskTable(*tableOpts)
			if err != nil {
				return err
			}
			fmt.Print(table)
			return nil
		},
	}
	bindTableOpts(cmd)
	cmd.Flags().Bool("waiting", false, "Include tasks that are waiting for a later date")
	return cmd
}
