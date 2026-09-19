package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/drewstinnett/taskpoet/v2/taskpoet"

	"github.com/spf13/cobra"
)

// newCompleteCmd marks a task as done
func newCompleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "done TASK",
		Short:   "Mark a task as done",
		Long:    `Mark a task as done`,
		Aliases: []string{"c", "complete", "finish"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := mustPoet()
			task, err := p.Store.GetByPrefix(args[0], taskpoet.StatusPending)
			if err != nil {
				return err
			}
			if _, err := p.Store.Complete(task.UUID); err != nil {
				return err
			}
			log.Info("Completed task, nice work!", "task", task.Description, "id", task.ShortID())
			return nil
		},
		ValidArgsFunction: completeActive,
	}
	return cmd
}
