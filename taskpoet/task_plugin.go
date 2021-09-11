package taskpoet

type Creator func() TaskPlugin

var TaskPlugins = map[string]Creator{}

/*
All Plugins must register the following functions
ExampleConfig() - Return an example configuration
Description() - Return the description of the plugin
Sync() Do the actual sync'ing of tasks, and return an error object if it fails
*/
type TaskPlugin interface {
	ExampleConfig() string
	Description() string
	Sync() ([]Task, error)
}

func AddPlugin(name string, creator Creator) {
	TaskPlugins[name] = creator
}
