package ui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/dustin/go-humanize"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

/*
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render
*/

type model struct {
	// choices  []string
	list list.Model
	// cursor   int
	// selected map[int]struct{}

	client *taskpoet.Poet
}

func initialModel(l *taskpoet.Poet) model {
	m := model{
		client: l,
	}
	results, _ := m.client.Task.List("/active")
	sort.Slice(results, func(i, j int) bool {
		return results[i].Added.Before(results[j].Added)
	})
	var tasks []list.Item
	for _, r := range results {
		// m.list = append(m.choices, r.Description)
		ti := taskItem{
			title:       r.Description,
			description: fmt.Sprintf("Due: %v Age: %v", humanize.Time(*r.Due), humanize.Time(r.Added)),
		}
		tasks = append(tasks, ti)
		m.list = list.New(tasks, list.NewDefaultDelegate(), 0, 0)
		m.list.Title = "TODO Tasks"
		/*
			i := list.Item{
				ti,
			}
		*/
		// tasks = append(tasks, list.Item{name: r.Description, description: r.ID})
	}
	return m
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, nil
		}
	case tea.WindowSizeMsg:
		top, right, bottom, left := docStyle.GetMargin()
		m.list.SetSize(msg.Width-left-right, msg.Height-top-bottom)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

// NewUI returns a new instance of the UI
func NewUI(l *taskpoet.Poet) *tea.Program {
	p := tea.NewProgram(initialModel(l), tea.WithAltScreen())
	return p
}

/*
type listKeyMap struct {
	toggleSpinner    key.Binding
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	insertItem       key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		insertItem: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add item"),
		),
		toggleSpinner: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle spinner"),
		),
		toggleTitleBar: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle title"),
		),
		toggleStatusBar: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle status"),
		),
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
	}
}
*/
