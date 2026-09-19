package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/charmbracelet/log"
	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/spf13/cobra"
)

// newImportCmd is the parent of the importers
func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import tasks from another tool",
	}
	cmd.AddCommand(newImportTaskwarriorCmd())
	return cmd
}

func newImportTaskwarriorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "taskwarrior [FILE]",
		Short: "Import everything from Taskwarrior",
		Args:  cobra.MaximumNArgs(1),
		Long: `Import all of your Taskwarrior tasks: pending, waiting, completed, deleted and
recurring, along with projects, priorities, tags, dependencies, annotations,
recurrence details and user defined attributes.

The easiest way is to let taskpoet run Taskwarrior for you:

$ taskpoet import taskwarrior --from-task

Or make an export yourself, and import that (use - to read from stdin):

$ task export > tw.json
$ taskpoet import taskwarrior tw.json

Tasks are matched by their UUID, so it is safe to import again. Tasks that are
already here are left alone unless --overwrite is given. Nothing is written if
any part of the import fails. Use --dry-run to see what would happen first.`,
		Example: `Check what would happen, without changing anything:
$ taskpoet import taskwarrior --from-task --dry-run`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fromTask := mustGetCmd[bool](cmd, "from-task")
			if fromTask == (len(args) > 0) {
				return errors.New("give either a FILE (or - for stdin), or --from-task")
			}
			raw, err := readTaskWarrior(cmd, args, fromTask)
			if err != nil {
				return err
			}

			parsed, err := taskpoet.ParseTaskWarrior(bytes.NewReader(raw), taskpoet.ParseOptions{
				SkipInvalid: mustGetCmd[bool](cmd, "skip-invalid"),
			})
			if err != nil {
				return err
			}

			rep, err := mustPoet().Store.ImportTasks(parsed.Tasks, taskpoet.ImportOptions{
				Overwrite: mustGetCmd[bool](cmd, "overwrite"),
				DryRun:    mustGetCmd[bool](cmd, "dry-run"),
			})
			if err != nil {
				return err
			}
			printImportReport(rep, parsed)
			return nil
		},
	}
	cmd.Flags().Bool("from-task", false, "Run Taskwarrior to get the tasks, instead of reading a file")
	cmd.Flags().String("task-bin", "task", "Taskwarrior binary to run for --from-task")
	cmd.Flags().Bool("dry-run", false, "Show what would be imported, without writing anything")
	cmd.Flags().Bool("overwrite", false, "Replace tasks that were already imported, instead of skipping them")
	cmd.Flags().Bool("skip-invalid", false, "Skip records that can't be converted, instead of stopping")
	return cmd
}

func readTaskWarrior(cmd *cobra.Command, args []string, fromTask bool) ([]byte, error) {
	switch {
	case fromTask:
		bin := mustGetCmd[string](cmd, "task-bin")
		log.Info("Running Taskwarrior to export your tasks", "bin", bin)
		return taskpoet.RunTaskWarriorExport(context.Background(), bin)
	case args[0] == "-":
		return io.ReadAll(os.Stdin)
	default:
		return os.ReadFile(args[0])
	}
}

func printImportReport(rep *taskpoet.ImportReport, parsed *taskpoet.ParseResult) {
	statuses := make([]string, 0, len(rep.ByStatus))
	for st, n := range rep.ByStatus {
		statuses = append(statuses, fmt.Sprintf("%v %d", st, n))
	}
	sort.Strings(statuses)

	verb := "Imported"
	if rep.DryRun {
		fmt.Println("Dry run, no tasks were written.")
		verb = "Would import"
	}
	fmt.Printf("Read %d tasks from Taskwarrior %v\n", rep.Read, statuses)
	fmt.Printf("%v %d new tasks, %d overwritten, %d already present and skipped", verb, rep.Imported, rep.Overwritten, rep.Skipped)
	if parsed.Skipped > 0 {
		fmt.Printf(", %d invalid records skipped", parsed.Skipped)
	}
	fmt.Println()
	for _, w := range append(append([]string{}, parsed.Warnings...), rep.Warnings...) {
		log.Warn(w)
	}
}
