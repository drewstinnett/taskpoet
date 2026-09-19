package cmd

import (
	"math/rand"
	"reflect"
	"time"

	"github.com/charmbracelet/log"

	"github.com/bxcodec/faker"
	"github.com/drewstinnett/taskpoet/v2/taskpoet"
	"github.com/spf13/cobra"
)

// newFakeitCmd represents the fakeit command
func newFakeitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "fakeit",
		Hidden: true,
		Short:  "Generate a bunch of fake tasks",
		Long:   `Generate a bunch of fake tasks. Mainly used for testing, load, boring stuff like that`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := mustPoet()
			log.Info("Generating TODO tasks")
			if err := addFakeTasks(p, 100, false); err != nil {
				return err
			}
			log.Info("Generating completed tasks")
			return addFakeTasks(p, 100, true)
		},
	}
	return cmd
}

// addFakeTasks adds n made up tasks, either pending or already completed
func addFakeTasks(p *taskpoet.Poet, n int, completed bool) error {
	for i := 0; i < n; i++ {
		desc, err := faker.GetLorem().Sentence(reflect.Value{})
		if err != nil {
			return err
		}
		opts := []taskpoet.TaskOption{taskpoet.WithEntry(randomPastDate())}
		if completed {
			opts = append(opts, taskpoet.WithCompleted(randomPastDate()))
		} else {
			opts = append(opts, taskpoet.WithDue(randomDueDate()))
		}
		t, err := taskpoet.NewTask(desc.(string), opts...)
		if err != nil {
			return err
		}
		if err := p.Store.Add(t); err != nil {
			return err
		}
	}
	return nil
}

func randomDueDate() *time.Time {
	now := time.Now()
	min := now.Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min // nolint:gosec
	r := time.Unix(sec, 0)
	return &r
}

func randomPastDate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Now().Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min // nolint:gosec
	return time.Unix(sec, 0).UTC()
}
