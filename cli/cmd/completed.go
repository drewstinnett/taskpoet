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
		// limit, err := cmd.PersistentFlags().GetInt("limit")
		// checkErr(err)
		// tableOpts := mustTableOptsWithCmd(cmd, args)
		tableOpts := &taskpoet.TableOpts{
			Prefix:  "/completed",
			Columns: []string{"ID", "Description", "Completed", "Tags"},
			SortBy:  taskpoet.ByCompleted{},
			Filters: []taskpoet.Filter{
				taskpoet.FilterRegex,
				taskpoet.FilterHidden,
			},
		}
		checkErr(applyCobra(cmd, args, tableOpts))
		var results []*taskpoet.Task

		results, err := poetC.Task.List("/completed")
		checkErr(err)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Added.Before(results[j].Added)
		})

		table := poetC.TaskTable(*tableOpts)
		fmt.Print(table)
	},
}

func init() {
	bindTableOpts(getCompleteCmd)
	rootCmd.AddCommand(getCompleteCmd)
}
