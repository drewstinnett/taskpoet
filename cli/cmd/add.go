package cmd

import (
	"log"
	"strings"

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

		// Make sure this is between 0 and 4
		effortImpact, _ := cmd.PersistentFlags().GetUint("effort-impact")
		if effortImpact > 4 {
			log.Fatal("EfforImpact assessment must be less than 5")
		}

		t := &taskpoet.Task{
			Description:  strings.Join(args, " "),
			EffortImpact: effortImpact,
		}
		found, err := localClient.Task.Add(t, taskDefaults)
		CheckErr(err)
		found.Describe()
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")
	addCmd.PersistentFlags().UintP("effort-impact", "e", 0, "Effort/Impact Score Assessment. See Help for more info")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
