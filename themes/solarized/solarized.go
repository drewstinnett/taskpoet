/*
Package solarized is easy on the eyes 👀
*/
package solarized

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/drewstinnett/taskpoet/themes"
)

// Colors is the thing that holds all the styles
type colors struct {
	Base03   lipgloss.TerminalColor
	Base02   lipgloss.TerminalColor
	Base01   lipgloss.TerminalColor
	Base00   lipgloss.TerminalColor
	Base0    lipgloss.TerminalColor
	Base1    lipgloss.TerminalColor
	Base2    lipgloss.TerminalColor
	Base3    lipgloss.TerminalColor
	Accent00 lipgloss.TerminalColor
	Accent01 lipgloss.TerminalColor
	Accent02 lipgloss.TerminalColor
	Accent03 lipgloss.TerminalColor
	Accent0  lipgloss.TerminalColor
	Accent1  lipgloss.TerminalColor
	Accent2  lipgloss.TerminalColor
	Accent3  lipgloss.TerminalColor
}

var lightColors colors = colors{
	Base03:   lipgloss.Color("#022B35"), // Background
	Base02:   lipgloss.Color("#0A3641"), // Background
	Base01:   lipgloss.Color("#596E75"), // Content
	Base00:   lipgloss.Color("#667B82"), // Content
	Base0:    lipgloss.Color("#819294"), // Content
	Base1:    lipgloss.Color("#93A1A1"), // Content
	Base2:    lipgloss.Color("#EEE8D6"), // Background
	Base3:    lipgloss.Color("#FDF6E4"), // Background
	Accent00: lipgloss.Color("#B0851C"), // Yellow
	Accent01: lipgloss.Color("#C94C22"), // Orange
	Accent02: lipgloss.Color("#DA3435"), // Red
	Accent03: lipgloss.Color("#D13A82"), // Magenta
	Accent0:  lipgloss.Color("#6D73C2"), // Violet
	Accent1:  lipgloss.Color("#2E8CCF"), // Blue
	Accent2:  lipgloss.Color("#32A198"), // Cyan
	Accent3:  lipgloss.Color("#85981C"), // Green
}

// NewLight returns a new light theme solarized styler
func NewLight() themes.Styling {
	return themes.Styling{
		RowHeader:  lipgloss.NewStyle().Bold(false).Foreground(lightColors.Accent0).Underline(true).Padding(0, 1, 0, 1),
		Row:        lipgloss.NewStyle().Bold(false).Foreground(lightColors.Base0).Background(lightColors.Base3).Padding(0, 1, 0, 1),
		RowAlt:     lipgloss.NewStyle().Bold(false).Foreground(lightColors.Base0).Background(lightColors.Base2).Padding(0, 1, 0, 1),
		NearingDue: lightColors.Accent00,
		PastDue:    lightColors.Accent02,
	}
}

// NewDark returns a new dark theme solarized styler
func NewDark() themes.Styling {
	return themes.Styling{
		Row:        lipgloss.NewStyle().Foreground(lightColors.Base00).Background(lightColors.Base03).Padding(0, 1, 0, 1),
		RowAlt:     lipgloss.NewStyle().Foreground(lightColors.Base00).Background(lightColors.Base02).Padding(0, 1, 0, 1),
		RowHeader:  lipgloss.NewStyle().Foreground(lightColors.Accent01).Underline(true).Padding(0, 1, 0, 1),
		NearingDue: lightColors.Accent00,
		PastDue:    lightColors.Accent02,
	}
}
