package cmd

import (
	"time"

	"strings"

	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/spf13/cobra"
)

// newAnnotateCmd adds a note to a task
func newAnnotateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "annotate TASK NOTE...",
		Short:             "Add an annotation to a task",
		Long:              `Annotations are just little text notes, with a date of when they were entered`,
		Aliases:           []string{"comment"},
		Args:              cobra.MinimumNArgs(2),
		ValidArgsFunction: completeActive,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := mustPoet()
			task, err := p.Store.GetByPrefix(args[0], taskpoet.StatusPending, taskpoet.StatusRecurring)
			if err != nil {
				return err
			}
			_, err = p.Store.Update(task.UUID, func(t *taskpoet.Task) error {
				return t.AddAnnotation(strings.Join(args[1:], " "), time.Now().UTC().Truncate(time.Second))
			})
			return err
		},
	}
	return cmd
}
