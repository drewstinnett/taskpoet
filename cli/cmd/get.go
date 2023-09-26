package cmd

import (
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/dustin/go-humanize"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// NewGetCmd is the new get command
func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "active",
		Short:   "Get Active tasks, waiting to be completed",
		Aliases: []string{"g", "a", "get"},
		Long: `Get Active Tasks
`,
		Run: func(cmd *cobra.Command, args []string) {
			limit, err := cmd.PersistentFlags().GetInt("limit")
			checkErr(err)

			results, err := localClient.Task.List("/active")
			checkErr(err)
			sort.Sort(results)

			var re *regexp.Regexp
			if len(args) > 0 {
				re = regexp.MustCompile(fmt.Sprintf("(?i)%v", strings.Join(args, " ")))
				slog.Debug("Showing tasks that match", "regex", re)
			} else {
				re = regexp.MustCompile(".*")
			}

			data := make([][]string, 1, len(results)+1)
			// data = append(data, []string{"ID", "Age", "Description", "Due", "Tags"})
			data[0] = []string{"ID", "Age", "Description", "Due", "Tags"}
			for _, task := range results {
				// Ignore things still hidden
				if task.HideUntil != nil && task.HideUntil.After(time.Now()) || (!re.Match([]byte(task.Description))) {
					continue
				}
				var dueHR string
				if task.Due != nil {
					dueHR = humanize.Time(*task.Due)
				}
				var desc string
				if task.PluginID != taskpoet.DefaultPluginID {
					desc = fmt.Sprintf("%v (%v)", task.Description, task.PluginID)
				} else {
					desc = task.Description
				}
				data = append(data, []string{
					fmt.Sprintf("%v", task.ShortID()),
					humanize.Time(task.Added),
					desc,
					dueHR,
					fmt.Sprintf("%+v", task.Tags),
				})
			}
			checkErr(pterm.DefaultTable.WithHasHeader().WithData(
				data[0:min(len(data), limit+1)],
			).Render())

			if limit < len(data) {
				slog.Warn("more records to display, increase the limit to see it", "n-more", len(data)-limit)
			}
		},
	}
	cmd.PersistentFlags().IntP("limit", "l", 40, "Limit to N results")
	return cmd
}

func init() {
	rootCmd.AddCommand(NewGetCmd())

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getPendingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getPendingCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
