package cmd

import (
	"fmt"
	"sort"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/dustin/go-humanize"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// getPendingCmd represents the getPending command
var getCompleteCmd = &cobra.Command{
	Use:     "complete",
	Short:   "Get Complete tasks",
	Aliases: []string{"c"},
	Long: `Get Complete Tasks
`,
	Run: func(cmd *cobra.Command, args []string) {
		var results []taskpoet.Task
		var err error
		results, err = localClient.Task.List("/completed")
		CheckErr(err)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Added.Before(results[j].Added)
		})

		data := make([][]string, 0)
		data = append(data, []string{"ID", "Description", "Completed"})
		for _, task := range results {
			row := []string{fmt.Sprintf("%v", task.ID[0:5]), task.Description, humanize.Time(task.Completed)}
			data = append(data, row)
		}
		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	},
}

func init() {
	getCmd.AddCommand(getCompleteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getPendingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getPendingCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
