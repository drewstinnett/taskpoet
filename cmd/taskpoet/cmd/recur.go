package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// newRecurCmd creates the instances of recurring tasks that are due
func newRecurCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recur",
		Short: "Create the recurring tasks that are due",
		Long: `Recurring tasks are templates that spawn pending tasks as they come due. This
happens on its own whenever you list tasks (turn that off with the
recurrence.enabled setting), this command lets you do it by hand, or see what it
would do first with --dry-run.

Settings:

  recurrence.limit    how many future tasks to have ready, default 1
  recurrence.catchup  'latest' (default) only creates the most recent missed
                      task, 'all' creates every one that was missed`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: noComplete,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun := mustGetCmd[bool](cmd, "dry-run")
			rep, err := mustPoet().SpawnRecurring(dryRun)
			if err != nil {
				return err
			}
			verb := "Created"
			if dryRun {
				verb = "Would create"
			}
			fmt.Printf("%v %d tasks\n", verb, len(rep.Created))
			for _, t := range rep.Created {
				due := ""
				if t.Due != nil {
					due = t.Due.Local().Format("2006-01-02 15:04")
				}
				fmt.Printf("  %v  due %v  %v\n", t.ShortID(), due, t.Description)
			}
			for _, w := range rep.Warnings {
				log.Warn(w)
			}
			return nil
		},
	}
	cmd.Flags().Bool("dry-run", false, "Show what would be created, without creating it")
	return cmd
}
