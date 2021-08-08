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
var getActiveCmd = &cobra.Command{
	Use:     "active",
	Short:   "Get Active tasks, waiting to be completed",
	Aliases: []string{"a"},
	Long: `Get Active Tasks
`,
	Run: func(cmd *cobra.Command, args []string) {
		var results []taskpoet.Task
		var err error
		results, err = localClient.Task.List("/active")
		CheckErr(err)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Added.Before(results[j].Added)
		})

		data := make([][]string, 0)
		data = append(data, []string{"ID", "Age", "Description", "Due"})
		for _, task := range results {
			//duration := task.Due.Sub(now)
			age := humanize.Time(task.Added)
			//hrDuration := durafmt.Parse(duration).LimitFirstN(1) // // limit first two parts.

			row := []string{fmt.Sprintf("%v", task.ShortID()), age, task.Description, humanize.Time(task.Due)}
			data = append(data, row)
		}
		pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	},
}

func init() {
	getCmd.AddCommand(getActiveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getPendingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getPendingCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
