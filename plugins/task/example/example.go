/*
Package exampleplugin is an example plugin
*/
package exampleplugin

import (
	"log"

	"github.com/drewstinnett/taskpoet/taskpoet"
)

// Example thing
type Example struct{}

// Sync thing
func (p *Example) Sync() ([]taskpoet.Task, error) {
	ts := []taskpoet.Task{
		{
			Description: "Fist Synced Task",
			PluginID:    "EXAMPLE-1",
		},
		{
			Description: "Second Synced Task",
			PluginID:    "EXAMPLE-2",
		},
	}
	log.Print("Syncing tasks", "tasks", ts)

	return ts, nil
}

// ExampleConfig thing thingy
func (p *Example) ExampleConfig() string {
	return "# No configuration yet"
}

// Description thing
func (p *Example) Description() string {
	return "This is meant to be a little structure to help you create your own Task Plugin"
}

func init() {
	taskpoet.AddPlugin("example", func() taskpoet.TaskPlugin {
		return &Example{}
	})
}
