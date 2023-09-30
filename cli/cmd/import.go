package cmd

import (
	"encoding/json"
	"log"
	"os"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
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
		err = json.Unmarshal(b, &tasks)
		checkErr(err)

		log.Printf("Importing %v items", len(tasks))
		imported, err := poetC.ImportTaskWarrior(tasks)
		checkErr(err)

		log.Printf("imported tasks: %v", imported)
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// importCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// importCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
