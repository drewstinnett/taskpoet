package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// newGetCmd is the new get command
func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "active",
		Short:   "Get Active tasks, waiting to be completed",
		Aliases: []string{"g", "a", "get"},
		Version: version,
		Long: `Get Active Tasks
`,
		Run: func(cmd *cobra.Command, args []string) {
			tableOpts := mustTableOptsWithCmd(cmd, args)
			tableOpts.Prefix = "/active"
			tableOpts.Columns = []string{"ID", "Age", "Due", "Description", "Urgency"}
			tableOpts.SortBy = taskpoet.ByUrgency{}
			tableOpts.Filters = []taskpoet.Filter{
				taskpoet.FilterHidden,
				taskpoet.FilterRegex,
			}

			var re *regexp.Regexp
			if len(args) > 0 {
				tableOpts.FilterParams.Regex = regexp.MustCompile(fmt.Sprintf("(?i)%v", strings.Join(args, " ")))
				log.Debug("Showing tasks that match", "regex", re)
			} else {
				tableOpts.FilterParams.Regex = regexp.MustCompile(".*")
			}
			// table := poetC.TaskTable("/active", *fp, taskpoet.FilterHidden, taskpoet.FilterRegex)
			table := poetC.TaskTable(*tableOpts)
			fmt.Print(table)
		},
	}
	bindTableOpts(cmd)
	return cmd
}
