package taskpoet

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/drewstinnett/taskpoet/v2/themes"
	"golang.org/x/term"
)

var docStyle = lipgloss.NewStyle().Padding(0).Margin(0)

// TableOpts defines the data displayed in a table
type TableOpts struct {
	// Statuses to list, pending when empty
	Statuses     []Status
	FilterParams FilterParams
	Filters      []Filter
	Columns      []string
	SortBy       any
}

const descriptionColumnName = "Description"

var columnMap = map[string]func(Task) string{
	"ID":      func(t Task) string { return t.ShortID() },
	"Urgency": func(t Task) string { return fmt.Sprintf("%.2f", t.Urgency) },
	"Age": func(t Task) string {
		return shortDuration(time.Since(t.Entry))
	},
	descriptionColumnName: func(t Task) string {
		if t.IsBlocked() {
			return "⛔ " + t.DescriptionDetails()
		}
		return t.DescriptionDetails()
	},
	"Due": func(t Task) string {
		if t.Due != nil {
			return shortDuration(time.Since(*t.Due) * -1)
		}
		return ""
	},
	"Project":  func(t Task) string { return t.Project },
	"Priority": func(t Task) string { return string(t.Priority) },
	"Tags":     func(t Task) string { return strings.Join(t.Tags, ",") },
	"Completed": func(t Task) string {
		if t.End == nil {
			return ""
		}
		return t.End.Format("2006-01-02")
	},
}

func columnValue(s string, t Task) (string, error) {
	valF, ok := columnMap[s]
	if !ok {
		return "", fmt.Errorf("column not defined: %v", s)
	}
	return valF(t), nil
}

func mustColumnValue(s string, t Task) string {
	got, err := columnValue(s, t)
	if err != nil {
		panic(err)
	}
	return got
}

// Rows consists of column and row strings
type Rows [][]string

// taskTable is a printable table of tasks
type taskTable struct {
	columns []string
	styling themes.Styling
	tasks   Tasks
}

// Generate returns a real table from the struct
func (t taskTable) Generate() *table.Table {
	rows := make(Rows, len(t.tasks))
	for idx, task := range t.tasks {
		row := make([]string, len(t.columns))
		for idx, c := range t.columns {
			row[idx] = mustColumnValue(c, *task)
		}
		rows[idx] = row
	}
	return table.New().
		Border(lipgloss.HiddenBorder()).
		BorderStyle(lipgloss.NewStyle()).
		StyleFunc(t.StyleFunc).
		Headers(t.columns...).
		Rows(rows...)
}

var columnStyles = map[string]func(Tasks, int, lipgloss.Style, themes.Styling) lipgloss.Style{
	"due": func(tasks Tasks, row int, rowStyle lipgloss.Style, t themes.Styling) lipgloss.Style {
		if tasks[row-1].Due != nil {
			rdue := time.Since(*tasks[row-1].Due)
			switch {
			case rdue > 0:
				return rowStyle.Copy().Foreground(t.PastDue)
			case time.Now().Add(7 * 24 * time.Hour).After(*tasks[row-1].Due):
				return rowStyle.Copy().Foreground(t.NearingDue)
			default:
				return rowStyle
			}
		}
		return rowStyle
	},
}

// StyleFunc provides styling for a set of rows
func (t taskTable) StyleFunc(row, col int) lipgloss.Style {
	if row == 0 {
		return t.styling.RowHeader
	}

	even := row%2 == 0
	rowStyle := t.styling.Row
	if even {
		rowStyle = t.styling.RowAlt
	}

	switch t.columns[col] {
	case "Due":
		return columnStyles["due"](t.tasks, row, rowStyle, t.styling)
	default:
		return rowStyle
	}
}

func descDate(d time.Time) string {
	return fmt.Sprintf("%v (%v)", d.Format("2006-01-02 15:04"), shortDuration(time.Since(d)*-1))
}

func shortIDs(ids []string) string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id[0:min(len(id), shortIDLen)]
	}
	return strings.Join(out, ",")
}

func descRows(t Task, blocks Tasks) [][]string {
	rows := [][]string{
		{"ID", fmt.Sprintf("%v (%v)", t.UUID, t.ShortID())},
		{"Status", string(t.Status)},
		{"Description", t.DescriptionDetails()},
	}
	if t.Project != "" {
		rows = append(rows, []string{"Project", t.Project})
	}
	if t.Priority != PriorityNone {
		rows = append(rows, []string{"Priority", string(t.Priority)})
	}
	rows = append(rows, []string{"Entered", descDate(t.Entry)})
	for _, d := range []struct {
		name string
		when *time.Time
	}{
		{"Due", t.Due},
		{"Scheduled", t.Scheduled},
		{"Wait", t.Wait},
		{"Until", t.Until},
		{"Start", t.Start},
		{"Reviewed", t.Reviewed},
		{"End", t.End},
	} {
		if d.when != nil {
			rows = append(rows, []string{d.name, descDate(*d.when)})
		}
	}
	if len(t.Tags) > 0 {
		rows = append(rows, []string{"Tags", strings.Join(t.Tags, ",")})
	}
	if t.EffortImpact != EffortImpactUnset {
		rows = append(rows, []string{"Effort/Impact", fmt.Sprintf("%v %v", t.EffortImpact.Emoji(), t.EffortImpact)})
	}
	if len(t.Depends) > 0 {
		rows = append(rows, []string{"Depends", shortIDs(t.Depends)})
	}
	if len(blocks) > 0 {
		ids := make([]string, len(blocks))
		for i, b := range blocks {
			ids[i] = b.UUID
		}
		rows = append(rows, []string{"Blocks", shortIDs(ids)})
	}
	if t.Recur != "" {
		rows = append(rows, []string{"Recur", t.Recur})
	}
	if t.Mask != "" {
		rows = append(rows, []string{"Mask", t.Mask})
	}
	if t.Parent != "" {
		rows = append(rows, []string{"Parent", shortIDs([]string{t.Parent})})
	}
	udas := make([]string, 0, len(t.UDA))
	for k := range t.UDA {
		udas = append(udas, k)
	}
	sort.Strings(udas)
	for _, k := range udas {
		rows = append(rows, []string{k, fmt.Sprint(t.UDA[k])})
	}
	return append(rows, []string{"Urgency", fmt.Sprintf("%.2f", t.Urgency)})
}

// DescribeTask returns a pretty table describing a given task
func (p *Poet) DescribeTask(t Task) (string, error) {
	tasks := Tasks{&t}
	if err := p.refresh(tasks); err != nil {
		return "", err
	}
	blocks, err := p.Store.Blocking(t.UUID)
	if err != nil {
		return "", err
	}
	doc := strings.Builder{}
	doc.WriteString(table.New().
		Border(lipgloss.HiddenBorder()).
		BorderStyle(lipgloss.NewStyle()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return p.styling.RowHeader
			}
			even := row%2 == 0
			rowStyle := p.styling.Row
			if even {
				rowStyle = p.styling.RowAlt
			}
			return rowStyle
		}).
		Headers("Name", "Value").
		Rows(descRows(t, blocks)...).Render())

	doc.WriteString("\n  Urgency Calculation\n")
	urg, reasons := p.curator.WeighAndDescribe(t)
	reasonRows := [][]string{}
	for _, reason := range reasons {
		reasonRows = append(reasonRows, []string{
			reason.Name,
			fmt.Sprintf("%.2f", reason.Coefficient),
			"x",
			fmt.Sprint(reason.Multiplier),
			reason.Unit,
			"=",
			fmt.Sprintf("%.2f", reason.Coefficient*float64(reason.Multiplier)),
		})
	}
	if len(reasonRows) > 0 {
		eerow := append(make([]string, len(reasonRows[len(reasonRows)-1])-1), fmt.Sprintf("%.2f", urg))
		reasonRows = append(reasonRows, eerow)
	}
	doc.WriteString(table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			even := row%2 == 1
			rowStyle := p.styling.Row
			if even {
				rowStyle = p.styling.RowAlt
			}
			return rowStyle
		}).
		Rows(reasonRows...).Render())

	w, _, _ := term.GetSize(int(os.Stdout.Fd()))
	maxW := min(w, 180)
	docStyle = docStyle.MaxWidth(maxW)
	return docStyle.Render(doc.String()), nil
}

// TaskTable returns a table of the tasks matching the options
func (p *Poet) TaskTable(opts TableOpts) (string, error) {
	for _, c := range opts.Columns {
		if _, ok := columnMap[c]; !ok {
			return "", fmt.Errorf("column not defined: %v", c)
		}
	}
	statuses := opts.Statuses
	if len(statuses) == 0 {
		statuses = []Status{StatusPending}
	}
	all, err := p.List(statuses...)
	if err != nil {
		return "", err
	}
	tasks := ApplyFilters(all, &opts.FilterParams, opts.Filters...)
	allTasksLen := len(tasks)

	tasks.SortBy(opts.SortBy)

	if opts.FilterParams.Limit > 0 {
		tasks = tasks[0:min(len(tasks), opts.FilterParams.Limit)]
	}

	doc := strings.Builder{}
	tr := taskTable{
		tasks:   tasks,
		columns: opts.Columns,
		styling: p.styling,
	}.Generate().Render()

	doc.WriteString(tr)
	width := lipgloss.Width(tr)
	addLimitWarning(&doc, width-4, opts.FilterParams.Limit, allTasksLen)

	w, _, _ := term.GetSize(int(os.Stdout.Fd()))
	maxW := min(w, width)
	docStyle = docStyle.MaxWidth(maxW)
	return docStyle.Render(doc.String()), nil
}

func addLimitWarning(doc io.StringWriter, width, limit, total int) {
	if (limit > 0) && limit < total {
		_, _ = doc.WriteString("\n")
		_, _ = doc.WriteString(
			lipgloss.NewStyle().Italic(true).Width(width - 3).Align(lipgloss.Right).Render(
				fmt.Sprintf("* %v more records to display, increase the limit to see it",
					total-limit)),
		)
	}
}
