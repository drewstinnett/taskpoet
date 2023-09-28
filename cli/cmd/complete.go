package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// completeCmd represents the complete command
var completeCmd = &cobra.Command{
	Use:     "complete TASK",
	Short:   "Mark a task as complete",
	Long:    `Mark a task as complete`,
	Aliases: []string{"c"},
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		task, err := poetC.Task.GetWithPartialID(args[0], "", "/active")
		checkErr(err)
		err = poetC.Task.Complete(task)
		checkErr(err)
		log.Printf("Completed task: '%v', nice work!", task.Description)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return poetC.CompleteIDsWithPrefix("/active", toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	rootCmd.AddCommand(completeCmd)
	completeCmd.PersistentFlags().IntP("limit", "l", 100, "Limit to N results")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// completeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// completeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
