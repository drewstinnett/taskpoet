package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/drewstinnett/taskpoet/v2/taskpoet"

	"github.com/spf13/cobra"
)

// newDeleteCmd marks a task as deleted
func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete TASK",
		Short: "Delete a task",
		Long: `Mark a task as deleted. The task is kept in the database, but it won't show up
in any list. Deleting a recurrence template also stops it from spawning new tasks.`,
		Aliases: []string{"del", "rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := mustPoet()
			task, err := p.Store.GetByPrefix(args[0], taskpoet.StatusPending, taskpoet.StatusRecurring)
			if err != nil {
				return err
			}
			if _, err := p.Store.Delete(task.UUID); err != nil {
				return err
			}
			log.Info("Deleted task", "task", task.Description, "id", task.ShortID())
			return nil
		},
		ValidArgsFunction: completeActive,
	}
	return cmd
}
