package cmd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	. "github.com/ahmetb/go-linq/v3"
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/dustin/go-humanize"
	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// getPendingCmd represents the getPending command
var getCmd = &cobra.Command{
	Use:     "active",
	Short:   "Get Active tasks, waiting to be completed",
	Aliases: []string{"g", "a", "get"},
	Long: `Get Active Tasks
`,
	Run: func(cmd *cobra.Command, args []string) {
		var results []taskpoet.Task
		var err error
		var filter string

		// Figure out the limit
		limit, _ := cmd.PersistentFlags().GetInt("limit")

		now := time.Now()
		results, err = localClient.Task.List("/active")
		CheckErr(err)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Added.Before(results[j].Added)
		})

		var re *regexp.Regexp
		if len(args) > 0 {
			filter = fmt.Sprintf("(?i)%v", strings.Join(args, " "))
			re = regexp.MustCompile(filter)
			log.Debugf("Showing tasks that match '%v' regex", re)
		}

		data := make([][]string, 0)
		data = append(data, []string{"ID", "Age", "Description", "Due"})
		for _, task := range results {
			// Ignore things still hidden
			if task.HideUntil != nil && task.HideUntil.After(now) {
				continue
			}
			if filter != "" && !re.Match([]byte(task.Description)) {
				continue
			}
			age := humanize.Time(task.Added)
			row := []string{fmt.Sprintf("%v", task.ShortID()), age, task.Description, humanize.Time(*task.Due)}
			data = append(data, row)
		}
		page := make([][]string, 0)

		From(data).Skip(0).Take(limit).ToSlice(&page)

		//pterm.DefaultTable.WithHasHeader().WithData(data).Render()
		pterm.DefaultTable.WithHasHeader().WithData(page).Render()

		if limit < len(data) {
			log.Warningf("%v more records to display, increase the limit to see it", len(data)-limit)
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getPendingCmd.PersistentFlags().String("foo", "", "A help for foo")
	getCmd.PersistentFlags().IntP("limit", "l", 40, "Limit to N results")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getPendingCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
