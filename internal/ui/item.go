package ui

type taskItem struct {
	title       string
	description string
}

func (i taskItem) Title() string       { return i.title }
func (i taskItem) Description() string { return i.description }
func (i taskItem) FilterValue() string { return i.title }
