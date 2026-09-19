package cmd

import (
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/spf13/cobra"
)

// newLogCmd records something that is already done
func newLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "log COMPLETED_TASK_DESCRIPTION",
		Short:   "Log a completed task",
		Aliases: []string{"l"},
		Args:    cobra.MinimumNArgs(1),
		Long: `Log a completed task. Useful for when you do something that was not in your
actual TODO list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := taskpoet.NewTask(strings.Join(args, " "),
				taskpoet.WithCompleted(time.Now().UTC().Truncate(time.Second)),
			)
			if err != nil {
				return err
			}
			if err := mustPoet().Add(task); err != nil {
				return err
			}
			log.Info("logged task", "description", task.Description, "id", task.ShortID())
			return nil
		},
	}
	return cmd
}
