package example_plugin

import (
	"github.com/drewstinnett/taskpoet/taskpoet"
	log "github.com/sirupsen/logrus"
)

type Example struct {
}

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
	log.Debug("Syncing tasks: ", ts)

	return ts, nil

}

func (p *Example) ExampleConfig() string {
	return "# No configuration yet"
}

func (p *Example) Description() string {
	return "This is meant to be a little structure to help you create your own Task Plugin"
}

func init() {
	taskpoet.AddPlugin("example", func() taskpoet.TaskPlugin {
		return &Example{}
	})
}
