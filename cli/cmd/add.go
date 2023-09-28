package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:     "add",
	Short:   "Add a new task",
	Args:    cobra.MinimumNArgs(1),
	Aliases: []string{"a"},
	Example: `$ taskpoet add "Learn a new skill"
$ taskpoet add --effort-impact 2 Rebuild all the remote servers`,
	Long:              `Add new task`,
	ValidArgsFunction: noComplete,
	Run: func(cmd *cobra.Command, args []string) {
		// Put some basics up here
		var err error

		// Make sure this is between 0 and 4
		effortImpact, _ := cmd.PersistentFlags().GetUint("effort-impact")
		if effortImpact > 4 {
			log.Fatal("EfforImpact assessment must be less than 5")
		}

		// Due Date parsing
		dueS, _ := cmd.PersistentFlags().GetString("due")
		var due time.Duration
		if dueS != "" {
			due, err = taskpoet.ParseDuration(dueS)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Get Tags
		tags, _ := cmd.PersistentFlags().GetStringSlice("tag")

		// HideUntil parsing
		waitS, _ := cmd.PersistentFlags().GetString("wait")
		var wait time.Duration
		if waitS != "" {
			wait, err = taskpoet.ParseDuration(waitS)
			if err != nil {
				log.Fatal(err)
			}
		}

		parentS, _ := cmd.PersistentFlags().GetString("parent")

		t := &taskpoet.Task{
			Description:  strings.Join(args, " "),
			EffortImpact: effortImpact,
			Tags:         tags,
		}

		// Did we specify a due date?
		now := time.Now()
		if float64(due.Nanoseconds()) != float64(0) {
			d := now.Add(due)
			t.Due = &d
		}
		if wait.Nanoseconds() != 0 {
			h := now.Add(wait)
			t.HideUntil = &h
		}

		found, err := poetC.Task.Add(t, taskDefaults)
		checkErr(err)
		if parentS != "" {
			parent, err := poetC.Task.GetWithPartialID(parentS, "", "")
			checkErr(err)
			if parent != nil {
				checkErr(poetC.Task.AddParent(t, parent))
			}
		}
		fmt.Printf("Added task '%v'\n", found.Description)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.PersistentFlags().UintP("effort-impact", "e", 0, "Effort/Impact Score Assessment. See Help for more info")
	err := addCmd.RegisterFlagCompletionFunc("effort-impact", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"0\tUndefined (Default)",
			"1\tLow Effort, High Impact (Sweet Spot)",
			"2\t High Effort, High Impact (Homework)",
			"3\tLow Effort, Low Impact (Busywork)",
			"4\tHigh Effort, Low Impact (Charity)",
		}, cobra.ShellCompDirectiveNoFileComp
	})
	checkErr(err)

	addCmd.PersistentFlags().StringP("parent", "p", "", "ID of parent task")
	err = addCmd.RegisterFlagCompletionFunc("parent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return poetC.CompleteIDsWithPrefix("/active", toComplete), cobra.ShellCompDirectiveNoFileComp
	})
	checkErr(err)

	addCmd.PersistentFlags().StringP("due", "d", "", "How long before this is due?")
	addCmd.PersistentFlags().StringP("wait", "w", "", "Wait until given duration to actually show up as active")
	addCmd.PersistentFlags().StringSliceP("tag", "t", []string{}, "Tags to include in this task")

	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
