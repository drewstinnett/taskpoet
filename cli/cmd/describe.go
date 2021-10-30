package cmd

import (
	"github.com/spf13/cobra"
)

// describeCmd represents the describe command
var describeCmd = &cobra.Command{
	Use:     "describe",
	Short:   "Describe a task",
	Long:    `Describe a task...showing details, all that cool stuff`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"desc", "d"},
	Run: func(cmd *cobra.Command, args []string) {
		task, err := localClient.Task.GetWithPartialID(args[0], "", "")
		CheckErr(err)
		err = localClient.Task.Describe(task)
		CheckErr(err)
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// describeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// describeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
