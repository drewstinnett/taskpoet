package cmd

import (
	"fmt"
	"sort"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// getPendingCmd represents the getPending command
var getCompleteCmd = &cobra.Command{
	Use:               "completed",
	Short:             "Get Completed tasks",
	Aliases:           []string{"c", "complete"},
	ValidArgsFunction: noComplete,
	Long: `Get Completed Tasks
`,
	Run: func(cmd *cobra.Command, args []string) {
		limit, err := cmd.PersistentFlags().GetInt("limit")
		checkErr(err)
		var results []taskpoet.Task
		results, err = poetC.Task.List("/completed")
		checkErr(err)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Added.Before(results[j].Added)
		})

		table := poetC.TaskTable(taskpoet.TableOpts{
			Prefix:       "/completed",
			Columns:      []string{"ID", "Description", "Completed", "Tags"},
			FilterParams: taskpoet.FilterParams{Limit: limit},
			Filters: []taskpoet.Filter{
				taskpoet.FilterHidden,
			},
		})
		fmt.Print(table)
	},
}

func init() {
	rootCmd.AddCommand(getCompleteCmd)
	getCompleteCmd.PersistentFlags().IntP("limit", "l", 100, "Limit to N results")
}
