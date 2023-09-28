package cmd

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/drewstinnett/taskpoet/taskpoet"
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

			var re *regexp.Regexp
			if len(args) > 0 {
				re = regexp.MustCompile(fmt.Sprintf("(?i)%v", strings.Join(args, " ")))
				slog.Debug("Showing tasks that match", "regex", re)
			} else {
				re = regexp.MustCompile(".*")
			}
			// table := poetC.TaskTable("/active", *fp, taskpoet.FilterHidden, taskpoet.FilterRegex)
			table := poetC.TaskTable(taskpoet.TableOpts{
				Prefix:       "/active",
				FilterParams: taskpoet.FilterParams{Regex: re, Limit: limit},
				Filters: []taskpoet.Filter{
					taskpoet.FilterHidden,
					taskpoet.FilterRegex,
				},
			})
			fmt.Print(table)
		},
	}
	cmd.PersistentFlags().IntP("limit", "l", 40, "Limit to N results")
	return cmd
}

func init() {
	rootCmd.AddCommand(NewGetCmd())
}
