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
			for i := 0; i < 100; i++ {
				desc, err := faker.GetLorem().Sentence(reflect.Value{})
				if err != nil {
					return err
				}
				t, err := taskpoet.NewTask(desc.(string),
					taskpoet.WithDue(randomDueDate()),
					taskpoet.WithEntry(randomPastDate()),
				)
				if err != nil {
					return err
				}
				if err := p.Store.Add(t); err != nil {
					return err
				}
			}

			log.Info("Generating completed tasks")
			for i := 0; i < 100; i++ {
				desc, err := faker.GetLorem().Sentence(reflect.Value{})
				if err != nil {
					return err
				}
				t, err := taskpoet.NewTask(desc.(string),
					taskpoet.WithEntry(randomPastDate()),
					taskpoet.WithCompleted(randomPastDate()),
				)
				if err != nil {
					return err
				}
				if err := p.Store.Add(t); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
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
