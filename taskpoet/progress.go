package taskpoet

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	doneStyle   = lipgloss.NewStyle().Margin(1, 2)
	warningIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).SetString("⚠️")
)

func readStatus(s chan ProgressStatus) tea.Cmd {
	return tea.Tick(time.Millisecond, func(t time.Time) tea.Msg {
		st := <-s
		return statusMsg(st)
	})
}

// ProgressStatus is a helper to show a progress bar on a bunch of item things
type ProgressStatus struct {
	Current int64
	Total   int64
	Info    string
	Warning string
	Done    bool
}

// ProgressModel represents our fancy little progress bar
type ProgressModel struct {
	// packages []string
	width    int
	height   int
	spinner  spinner.Model
	progress progress.Model
	// done     bool
	statusC chan ProgressStatus
	status  ProgressStatus
	// title    string
}

// Init satisfies the model interface
func (m ProgressModel) Init() tea.Cmd {
	return tea.Batch(readStatus(m.statusC), m.spinner.Tick)
}

type statusMsg ProgressStatus

// Update satiesfies the model interface
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	case statusMsg:
		m.status = <-m.statusC
		progressCmd := m.progress.SetPercent(float64(m.status.Current) / float64(m.status.Total))
		batch := []tea.Cmd{
			progressCmd,
			readStatus(m.statusC),
		}
		if m.status.Warning != "" {
			batch = append(batch, tea.Printf("%v %v", warningIcon, m.status.Warning))
		}
		return m, tea.Batch(batch...)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		if newModel, ok := newModel.(progress.Model); ok {
			m.progress = newModel
		}
		return m, cmd
	}
	return m, nil
}

// View satisfies the model interface
func (m ProgressModel) View() string {
	// n := m.status.Total
	// w := lipgloss.Width(fmt.Sprintf("%d", n))

	if m.status.Done {
		return doneStyle.Render("Done!")
	}

	// pkgCount := fmt.Sprintf(" %*d/%*d", w, m.status.Current, w, n-1)

	spin := m.spinner.View() + " "
	prog := m.progress.View()
	cellsAvail := max(0, m.width-lipgloss.Width(spin+prog))

	// pkgName := currentPkgNameStyle.Render(m.packages[m.index])
	info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render(m.status.Info)

	// cellsRemaining := max(20, m.width-lipgloss.Width(spin+info+prog))
	// gap := strings.Repeat(" ", cellsRemaining)
	gap := "     "

	return spin + prog + gap + info
}

// WithStatusChannel sets the channel on a new object
func WithStatusChannel(s chan ProgressStatus) func(*ProgressModel) {
	return func(m *ProgressModel) {
		m.statusC = s
	}
}

// NewProgressBar returns a new progress bar
func NewProgressBar(options ...func(*ProgressModel)) ProgressModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	m := ProgressModel{
		spinner: s,
		progress: progress.New(
			// progress.WithDefaultGradient(),
			// progress.WithWidth(80),
			progress.WithDefaultScaledGradient(),
			// progress.WithoutPercentage(),
		),
	}
	for _, opt := range options {
		opt(&m)
	}
	return m
}
