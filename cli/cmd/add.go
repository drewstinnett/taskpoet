package cmd

import (
	"log"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:     "add",
	Short:   "Add a new task",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"a"},
	Example: `$ taskpoet add "Learn a new skill"`,
	Long:    `Add new task`,
	Run: func(cmd *cobra.Command, args []string) {
		t := &taskpoet.Task{
			Description: args[0],
		}
		found, err := localClient.Task.New(t, taskDefaults)
		CheckErr(err)
		log.Println("Got ", found)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
