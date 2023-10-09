/*
Package themes defines how custom themes behave
*/
package themes

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme is an interface describe a theme for the UI
type Theme interface {
	Styling() Styling
}

// Styling is the thing that holds all the lipgloss.Style things
type Styling struct {
	Row       lipgloss.Style
	RowAlt    lipgloss.Style
	RowHeader lipgloss.Style
}

// New returns the default built-in theme
func New() Styling {
	return Styling{
		Row:       lipgloss.NewStyle().Padding(0, 1, 0, 1),
		RowAlt:    lipgloss.NewStyle().Padding(0, 1, 0, 1),
		RowHeader: lipgloss.NewStyle().Underline(true).Padding(0, 1, 0, 1),
	}
}
