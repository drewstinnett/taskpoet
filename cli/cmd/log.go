package cmd

import (
	"log"
	"strings"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:     "log COMPLETED_TASK_DESCRIPTION",
	Short:   "Log a completed task",
	Aliases: []string{"l"},
	Long: `Log a completed task. Useful for when you do something that was not in your
actual TODO list`,
	Run: func(cmd *cobra.Command, args []string) {
		/*
			now := time.Now()
			t := &taskpoet.Task{
				Description: strings.Join(args, " "),
				Completed:   &now,
			}
			found, err := poetC.Task.Add(t, taskDefaults)

			checkErr(err)
		*/
		got, err := poetC.Task.Log(&taskpoet.Task{
			Description: strings.Join(args, " "),
		}, &taskpoet.Task{})
		checkErr(err)
		log.Printf("Logged Task: '%v'", got.Description)
	},
}

func init() {
	rootCmd.AddCommand(logCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
