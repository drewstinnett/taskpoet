package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// importCmd represents the import command
func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import data from TaskWarrior",
		Args:  cobra.ExactArgs(1),
		Long: `Create an export of your TaskWarrior data using the followign command:
$ task export > /tmp/tw.backup.json

Now import that in to TaskPoet with:

$ taskpoet import /tmp/tw.backup.json `,
		Run: func(cmd *cobra.Command, args []string) {
			b, err := os.ReadFile(args[0])
			checkErr(err)
			var tasks taskpoet.TaskWarriorTasks
			checkErr(json.Unmarshal(b, &tasks))

			log.Info("Importing items", "count", len(tasks))
			c := make(chan taskpoet.ProgressStatus)
			go func() {
				if _, terr := tea.NewProgram(taskpoet.NewProgressBar(taskpoet.WithStatusChannel(c))).Run(); err != nil {
					fmt.Println("Error running program:", terr)
					os.Exit(1)
				}
			}()
			imported, err := poetC.ImportTaskWarrior(tasks, c)
			checkErr(err)

			log.Info("imported tasks", "count", imported)
		},
	}
	return cmd
}
