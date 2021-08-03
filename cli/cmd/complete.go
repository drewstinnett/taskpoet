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
		task, err := localClient.Task.GetByPartialID(args[0], "/active")
		CheckErr(err)
		err = localClient.Task.Complete(task)
		CheckErr(err)
		log.Printf("Completed task: '%v', nice work!", task.Description)
	},
}

func init() {
	rootCmd.AddCommand(completeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// completeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// completeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
