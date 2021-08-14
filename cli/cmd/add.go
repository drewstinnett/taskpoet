package cmd

import (
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
	Long: `Add new task`,
	Run: func(cmd *cobra.Command, args []string) {

		// Put some basics up here
		now := time.Now()
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
		}

		// Did we specify a due date?
		if float64(due.Nanoseconds()) != float64(0) {
			t.Due = now.Add(due)
		}
		if wait.Nanoseconds() != 0 {
			t.HideUntil = now.Add(wait)
		}

		found, err := localClient.Task.Add(t, taskDefaults)
		CheckErr(err)
		if parentS != "" {
			parent, err := localClient.Task.GetByPartialID(parentS)
			CheckErr(err)
			if parent != nil {
				localClient.Task.AddParent(t, parent)
			}
		}
		localClient.Task.Describe(found)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")
	addCmd.PersistentFlags().UintP("effort-impact", "e", 0, "Effort/Impact Score Assessment. See Help for more info")
	addCmd.PersistentFlags().StringP("parent", "p", "", "ID of parent task")
	addCmd.PersistentFlags().StringP("due", "d", "", "How long before this is due?")
	addCmd.PersistentFlags().StringP("wait", "w", "", "Wait until given duration to actually show up as active")

	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
