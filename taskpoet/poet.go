/*
Package taskpoet is the main worker library
*/
package taskpoet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	"github.com/drewstinnett/taskpoet/themes"
	"github.com/mitchellh/go-homedir"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/term"
)

// Option helper for functional options with error reporting
type Option func() (func(*Poet), error)

func success(opt func(*Poet)) Option {
	return func() (func(*Poet), error) {
		return opt, nil
	}
}

func failure(err error) Option {
	return func() (func(*Poet), error) {
		return nil, err
	}
}

// MustNew returns a new poet object or panics
func MustNew(options ...Option) *Poet {
	got, err := New(options...)
	if err != nil {
		panic(err)
	}
	return got
}

// New returns a new poet object and optional error
func New(options ...Option) (*Poet, error) {
	p := &Poet{
		Namespace: "default",
		dbPath:    path.Join(mustHomeDir(), ".taskpoet.db"),
		curator:   NewCurator(),
	}
	// Default to homedir database
	for _, option := range options {
		opt, err := option()
		if err != nil {
			return nil, err
		}
		opt(p)
	}

	// We may want to make this more flexible later
	p.bucket = []byte(fmt.Sprintf("/%v/tasks", p.Namespace))

	var err error
	p.DB, err = bolt.Open(p.dbPath, 0o600, nil)
	if err != nil {
		return nil, err
	}

	p.Task = &TaskServiceOp{
		localClient: p,
	}

	// InitDB
	if err := p.initDB(); err != nil {
		return nil, err
	}

	// Open the db
	return p, nil
}

/*
// DBConfig configures the database
type DBConfig struct {
	Path      string
	Namespace string
}
*/

// WithStyling gives the poet a certain style at create
func WithStyling(s themes.Styling) Option {
	return success(func(p *Poet) {
		p.styling = s
	})
}

// WithDatabasePath gives the Poet a path to a database file
func WithDatabasePath(s string) Option {
	if s != "" {
		return success(func(p *Poet) {
			p.dbPath = s
		})
	}
	return success(func(p *Poet) {})
}

// WithNamespace passes a namespace in to the new Poet object
func WithNamespace(n string) Option {
	if n == "" {
		return failure(errors.New("namespace cannot be empty"))
	}
	return success(func(p *Poet) {
		p.Namespace = n
	})
}

// WithRecurringTasks sets the recurring tasks for a poet
func WithRecurringTasks(r RecurringTasks) Option {
	return success(func(p *Poet) {
		p.RecurringTasks = r
	})
}

// Poet is the main operator for this whole thing
type Poet struct {
	DB             *bolt.DB
	Namespace      string
	Default        Task
	Task           TaskService
	dbPath         string
	RecurringTasks RecurringTasks
	bucket         []byte
	styling        themes.Styling
	curator        *Curator
}

func (p Poet) refresh(ts Tasks) {
	for _, task := range ts {
		newW := p.curator.Weigh(*task)
		if task.Urgency != newW {
			log.Debug("Updating urgency", "from", task.Urgency, "to", newW, "task", task.Description)
			task.Urgency = newW
		}
	}
}

// initDB initializes the database
func (p *Poet) initDB() error {
	// store some data
	return p.DB.Update(func(tx *bolt.Tx) error {
		bucket := p.getBucket(tx)
		if bucket == nil {
			if _, berr := tx.CreateBucket(p.bucket); berr != nil {
				return berr
			}
		}
		return nil
	})
}

/*
func dclose(c io.Closer) {
	err := c.Close()
	if err != nil {
		panic(err)
	}
}
*/

func mustHomeDir() string {
	h, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return h
}

const (
	// XS is extra small
	XS int = 1
	// SM is small
	SM int = 3
	// MD is medium
	MD int = 5
	// LG is large
	LG int = 10
)

var docStyle = lipgloss.NewStyle().Padding(0).Margin(0)

// TableOpts defines the data displayed in a table
type TableOpts struct {
	Prefix       string
	FilterParams FilterParams
	Filters      []Filter
	Columns      []string
	SortBy       any
}

// MustList returns tasks from a prefix or panics
func (p Poet) MustList(s string) Tasks {
	tasks, err := p.Task.List(s)
	if err != nil {
		panic(err)
	}
	return tasks
}

const descriptionColumnName = "Description"

var columnMap map[string]func(Task) string = map[string]func(Task) string{
	"ID":      func(t Task) string { return t.ShortID() },
	"Urgency": func(t Task) string { return fmt.Sprintf("%.2f", t.Urgency) },
	"Age": func(t Task) string {
		return shortDuration(time.Since(t.Added))
	},
	descriptionColumnName: func(t Task) string { return t.DescriptionDetails() },
	"Due": func(t Task) string {
		// return t.DueString()
		if t.Due != nil {
			return shortDuration(time.Since(*t.Due) * -1)
		}
		return ""
	},
	"Tags": func(t Task) string { return strings.Join(t.Tags, ",") },
	"Completed": func(t Task) string {
		if t.Completed == nil {
			return ""
		}
		return t.Completed.Format("2006-01-02")
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
	// rows    [][]string
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
	// c := NewCalendar()
	return fmt.Sprintf("%v (%v)", d.Format("2006-01-02 15:4"), shortDuration(time.Since(d)*-1))
}

func descRows(t Task) [][]string {
	rows := [][]string{
		{"ID", fmt.Sprintf("%v (%v)", t.ID, t.ShortID())},
		{"Description", t.DescriptionDetails()},
		{"Added", descDate(t.Added)},
	}
	if t.Due != nil {
		rows = append(rows, []string{"Due", descDate(*t.Due)})
	}
	if len(t.Tags) > 0 {
		rows = append(rows, []string{"Tags", strings.Join(t.Tags, ",")})
	}
	rows = append(rows, []string{
		"Urgency", fmt.Sprint(t.Urgency),
	})

	if len(t.Parents) > 0 {
		rows = append(rows, []string{"Parents", strings.Join(t.Parents, ",")})
	}
	if len(t.Children) > 0 {
		rows = append(rows, []string{"Children", strings.Join(t.Children, ",")})
	}
	return rows
}

// DescribeTask returns a pretty table describing a given task
func (p *Poet) DescribeTask(t Task) string {
	p.refresh(Tasks{&t})
	doc := strings.Builder{}
	rows := descRows(t)
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
		Rows(rows...).Render())

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
	return docStyle.Render(doc.String())
}

// TaskTable returns a table of the given tasks
func (p *Poet) TaskTable(opts TableOpts) string {
	p.checkRecurring()
	tasks := ApplyFilters(p.MustList(opts.Prefix), &opts.FilterParams, opts.Filters...)

	p.refresh(tasks)

	allTasksLen := len(tasks)

	tasks.SortBy(opts.SortBy)

	if opts.FilterParams.Limit > 0 {
		tasks = tasks[0:min(len(tasks), opts.FilterParams.Limit)]
	}

	doc := strings.Builder{}

	rows := make(Rows, len(tasks))
	for iidx, task := range tasks {
		row := make([]string, len(opts.Columns))
		for idx, c := range opts.Columns {
			row[idx] = mustColumnValue(c, *task)
		}
		rows[iidx] = row
	}

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
	// log.Debug("setting max window size", "size", maxW)
	docStyle = docStyle.MaxWidth(maxW)
	return docStyle.Render(doc.String())
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

// Filter is a filter function applied to a single task
type Filter func(*FilterParams, Task) bool

// FilterHidden removes items that are still hidden
func FilterHidden(p *FilterParams, task Task) bool {
	if (task.HideUntil != nil) && task.HideUntil.After(time.Now()) {
		return false
	}
	return true
}

// FilterRegex removes items not matching a given regex
func FilterRegex(p *FilterParams, task Task) bool {
	return p.Regex.Match([]byte(task.Description))
}

// FilterParams are options for filtering tasks
type FilterParams struct {
	Regex *regexp.Regexp
	Limit int
}

// ApplyFilters applies a set of filters to a task list.
// Each record will be checked against each filter.
// The filters are applied in the order they are passed in.
func ApplyFilters(tasks Tasks, p *FilterParams, filters ...Filter) Tasks {
	// Make sure there are actually filters to be applied.
	if len(filters) == 0 {
		return tasks
	}

	filteredRecords := make(Tasks, 0, len(tasks))

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for _, r := range tasks {
		keep := true

		for _, f := range filters {
			if !f(p, *r) {
				keep = false
				break
			}
		}

		if keep {
			filteredRecords = append(filteredRecords, r)
		}
	}

	return filteredRecords
}

// ByDue is the by due date sorter
type ByDue Tasks

func (a ByDue) Len() int { return len(a) }
func (a ByDue) Less(i, j int) bool {
	if a[i].Due == nil {
		return false
	}
	if a[j].Due == nil {
		return true
	}
	return !a[j].Due.Before(*a[i].Due)
}
func (a ByDue) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// ByUrgency sorts using the Urgency field
type ByUrgency Tasks

func (a ByUrgency) Len() int { return len(a) }
func (a ByUrgency) Less(i, j int) bool {
	return a[i].Urgency > a[j].Urgency
}
func (a ByUrgency) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// ByCompleted is the by completed date sorter
type ByCompleted Tasks

func (a ByCompleted) Len() int { return len(a) }
func (a ByCompleted) Less(i, j int) bool {
	if a[i].Completed == nil {
		return false
	}
	if a[j].Completed == nil {
		return true
	}
	return a[j].Completed.Before(*a[i].Completed)
}
func (a ByCompleted) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (p *Poet) exists(t *Task) bool {
	_, err := p.Task.GetWithID(t.ID, t.PluginID, "")
	return err == nil
}

// Delete marks a task as deleted
func (p *Poet) Delete(t *Task) error {
	curPath := t.DetectKeyPath()
	t.Deleted = nowPTR()
	newPath := t.DetectKeyPath()
	if err := p.DB.Update(func(tx *bolt.Tx) error {
		taskSerial, err := json.Marshal(t)
		if err != nil {
			return err
		}
		b := tx.Bucket(p.bucket)
		if perr := b.Put(newPath, taskSerial); perr != nil {
			return perr
		}
		return b.Delete(curPath)
	}); err != nil {
		return err
	}
	return nil
}

func (p Poet) getBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(p.bucket)
}
