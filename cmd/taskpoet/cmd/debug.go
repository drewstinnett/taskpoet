package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
)

// debugCmd represents the debug command
func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "debug",
		Short:  "Mostly just barf out the DB",
		Long:   `Barf the DB to stdout for debugging purposes 🤮`,
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			err := poetC.DB.View(func(tx *bolt.Tx) error {
				return tx.ForEach(func(bucketName []byte, bucket *bolt.Bucket) error {
					fmt.Println("Bucket: ", string(bucketName))
					err := bucket.ForEach(func(k, v []byte) error {
						fmt.Println(string(k), " -> ", string(v))
						return nil
					})
					checkErr(err)
					return nil
				})
			})
			checkErr(err)
		},
	}
	return cmd
}
