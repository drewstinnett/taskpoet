package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
)

// debugCmd represents the debug command
var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Mostly just barf out the DB",
	Long:  `Barf the DB to stdout for debugging purposes 🤮`,
	Run: func(cmd *cobra.Command, args []string) {
		err := localClient.DB.View(func(tx *bolt.Tx) error {
			tx.ForEach(func(bucketName []byte, bucket *bolt.Bucket) error {
				fmt.Println("Bucket: ", string(bucketName))
				err := bucket.ForEach(func(k, v []byte) error {
					fmt.Println(string(k), " -> ", string(v))
					return nil
				})
				CheckErr(err)
				return nil
			})

			return nil
		})
		CheckErr(err)
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// debugCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// debugCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
