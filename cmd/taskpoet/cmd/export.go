package cmd

import (
	"fmt"
	"os"

	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/spf13/cobra"
)

// newExportCmd writes tasks out in the Taskwarrior JSON format
func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export tasks as Taskwarrior style JSON",
		Long: `Write every task to stdout as JSON, in the same format 'task export' uses.
Handy for backups. Taskpoet's own effort/impact is included as 'effort_impact'.`,
		Example: `Back everything up:
$ taskpoet export > tasks.json

Just the pending ones:
$ taskpoet export --status pending`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: noComplete,
		RunE: func(cmd *cobra.Command, args []string) error {
			var statuses []taskpoet.Status
			for _, s := range mustGetCmd[[]string](cmd, "status") {
				st := taskpoet.Status(s)
				if !st.Valid() {
					return fmt.Errorf("unknown status %q, expected one of %v", s, taskpoet.AllStatuses())
				}
				statuses = append(statuses, st)
			}
			ts, err := mustPoet().Store.List(statuses...)
			if err != nil {
				return err
			}
			return taskpoet.WriteTaskWarrior(os.Stdout, ts)
		},
	}
	cmd.Flags().StringSlice("status", nil, "Only export tasks in these statuses (pending, completed, deleted, recurring)")
	checkErr(cmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"pending", "completed", "deleted", "recurring"}, cobra.ShellCompDirectiveNoFileComp
	}))
	return cmd
}
