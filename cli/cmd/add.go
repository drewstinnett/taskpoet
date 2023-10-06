package cmd

import (
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

func taskWithCmd(cmd *cobra.Command, args []string) *taskpoet.Task {
	opts := []taskpoet.TaskOption{
		taskpoet.WithEffortImpact(taskpoet.EffortImpact(mustGetCmd[uint](cmd, "effort-impact"))),
		taskpoet.WithDescription(strings.Join(args, " ")),
		taskpoet.WithTags(mustGetCmd[[]string](cmd, "tag")),
	}

	now := time.Now()
	dueIn := mustGetCmd[string](cmd, "due")
	if dueIn != "" {
		due := now.Add(taskpoet.MustParseDuration(dueIn))
		opts = append(opts, taskpoet.WithDue(&due))
	}

	hideIn := mustGetCmd[string](cmd, "wait")
	if hideIn != "" {
		hide := now.Add(taskpoet.MustParseDuration(hideIn))
		opts = append(opts, taskpoet.WithHideUntil(&hide))
	}

	return taskpoet.NewTask(opts...)
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:     "add",
	Short:   "Add a new task",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"a"},
	Example: `Add a new task by giving the description as an argument:
$ taskpoet add "Learn a new skill"

For us lazy folks, you can also leave the quotes out:
$ taskpoet add Learn a new skill

Set an Effort/Impact to a new task:
$ taskpoet add --effort-impact 2 Rebuild all the remote servers`,
	Long:              `Add new task`,
	ValidArgsFunction: noComplete,
	Run: func(cmd *cobra.Command, args []string) {
		// t := taskWithCmd(cmd, args)

		added, err := poetC.Task.Add(taskWithCmd(cmd, args))
		checkErr(err)

		parentS := mustGetCmd[string](cmd, "parent")
		if parentS != "" {
			parent, err := poetC.Task.GetWithPartialID(parentS, "", "")
			checkErr(err)
			if parent != nil {
				checkErr(poetC.Task.AddParent(added, parent))
			}
		}
		log.Info("Added task", "description", added.Description)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	err := bindAdd(addCmd)
	if err != nil {
		panic(err)
	}
}

func bindAdd(cmd *cobra.Command) error {
	cmd.PersistentFlags().UintP("effort-impact", "e", 0, "Effort/Impact Score Assessment. See Help for more info")
	err := cmd.RegisterFlagCompletionFunc("effort-impact", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"0\tUndefined (Default)",
			"1\tLow Effort, High Impact (Sweet Spot)",
			"2\t High Effort, High Impact (Homework)",
			"3\tLow Effort, Low Impact (Busywork)",
			"4\tHigh Effort, Low Impact (Charity)",
		}, cobra.ShellCompDirectiveNoFileComp
	})
	checkErr(err)

	cmd.PersistentFlags().StringP("parent", "p", "", "ID of parent task")
	if err := cmd.RegisterFlagCompletionFunc("parent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return poetC.CompleteIDsWithPrefix("/active", toComplete), cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		return err
	}

	cmd.PersistentFlags().StringP("due", "d", "", "How long before this is due?")
	cmd.PersistentFlags().StringP("wait", "w", "", "Wait until given duration to actually show up as active")
	cmd.PersistentFlags().StringSliceP("tag", "t", []string{}, "Tags to include in this task")
	return nil
}
