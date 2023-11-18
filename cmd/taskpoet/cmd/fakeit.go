package cmd

import (
	"math/rand"
	"reflect"
	"time"

	"github.com/charmbracelet/log"

	"github.com/bxcodec/faker"
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/spf13/cobra"
)

// fakeitCmd represents the fakeit command
func newFakeitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "fakeit",
		Hidden: true,
		Short:  "Generate a bunch of fake tasks",
		Long:   `Generate a bunch of fake tasks. Mainly used for testing, load, boring stuff like that`,
		Run: func(cmd *cobra.Command, args []string) {
			var ts taskpoet.Tasks
			log.Info("Generating TODO tasks")
			for i := 0; i < 100; i++ {
				desc, err := faker.GetLorem().Sentence(reflect.Value{})
				checkErr(err)

				t := taskpoet.Task{
					// Description: fmt.Sprintf("Task number %v", i),
					Description: desc.(string),
					Due:         randomDueDate(),
					Added:       randomAddedDate(),
				}
				ts = append(ts, &t)
			}
			checkErr(poetC.Task.AddSet(ts))

			log.Info("Generating completed tasks")
			var tsl taskpoet.Tasks
			for i := 0; i < 100; i++ {
				desc, serr := faker.GetLorem().Sentence(reflect.Value{})
				checkErr(serr)

				rad := randomAddedDate()
				tsl = append(tsl, taskpoet.MustNewTask(desc.(string),
					taskpoet.WithAdded(&rad),
					taskpoet.WithCompleted(randomCompletedDate()),
				))
			}
			checkErr(poetC.Task.AddSet(tsl))
		},
	}
	return cmd
}

func randomDueDate() *time.Time {
	// min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	now := time.Now()
	min := now.Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min // nolint:gosec
	r := time.Unix(sec, 0)
	return &r
}

func randomAddedDate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Now().Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min // nolint:gosec
	return time.Unix(sec, 0)
}

func randomCompletedDate() *time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Now().Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min // nolint:gosec
	r := time.Unix(sec, 0)
	return &r
}