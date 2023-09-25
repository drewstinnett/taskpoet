package taskpoet

// Creator creats plugins
type Creator func() TaskPlugin

// TaskPlugins isi the creator map
var TaskPlugins = map[string]Creator{}

/*
All Plugins must register the following functions
ExampleConfig() - Return an example configuration
Description() - Return the description of the plugin
Sync() Do the actual sync'ing of tasks, and return an error object if it fails
*/

// TaskPlugin describes what a plugin is
type TaskPlugin interface {
	ExampleConfig() string
	Description() string
	Sync() ([]Task, error)
}

// AddPlugin adds a new plugin
func AddPlugin(name string, creator Creator) {
	TaskPlugins[name] = creator
}
