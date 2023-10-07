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
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
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

var (
	docStyle  = lipgloss.NewStyle().Padding(0, 2, 0, 2)
	subtle    = lipgloss.AdaptiveColor{Light: "#f3f4f0", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	header = lipgloss.NewStyle().
		Foreground(highlight).
		Underline(true).
		Padding(0, 1, 0, 1)
	entry    = lipgloss.NewStyle().Foreground(special).Padding(0, 1, 0, 1)
	entryAlt = lipgloss.NewStyle().Foreground(special).Background(subtle).Padding(0, 1, 0, 1)
)

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
	"ID":                  func(t Task) string { return t.ShortID() },
	"Age":                 func(t Task) string { return t.Added.Format("2006-01-02") },
	descriptionColumnName: func(t Task) string { return t.Description },
	"Due":                 func(t Task) string { return t.DueString() },
	"Tags":                func(t Task) string { return strings.Join(t.Tags, ",") },
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

func getTaskColumn(name string, d []Task) (taskColumn, error) {
	switch name {
	case "ID":
		return &shortIDCol{}, nil
	case "Age":
		return &ageCol{}, nil
	case "Due":
		return &dueCol{}, nil
	case descriptionColumnName:
		return &descriptionCol{tasks: d}, nil
	case "Completed":
		return &completedCol{}, nil
	case "Tags":
		return &tagsCol{tasks: d}, nil
	default:
		return nil, fmt.Errorf("unknown columnn: %v", name)
	}
}

func iterateColumnHeaders(c []string, d []Task) []string {
	ret := make([]string, len(c))
	for idx, item := range c {
		cl, err := getTaskColumn(item, d)
		panicIfErr(err)
		ret[idx] = header.Width(cl.Width()).Render(item)
	}
	return ret
}

func iterateColumnValues(c []string, t Task, d []Task, s lipgloss.Style) []string {
	ret := make([]string, len(c))
	for idx, item := range c {
		// ret[idx] = s.Width(mustColumnSize(item)).Render(mustColumnValue(item, t))
		cl, err := getTaskColumn(item, d)
		panicIfErr(err)
		ret[idx] = s.Width(cl.Width()).Render(mustColumnValue(item, t))
	}
	return ret
}

func columnsOrDefault(c []string) []string {
	defaultColumns := []string{"ID", "Age", "Description", "Due", "Tags"}
	if len(c) == 0 {
		return defaultColumns
	}
	return c
}

// TaskTable returns a table of the given tasks
// func (p *Poet) TaskTable(prefix string, fp FilterParams, filters ...Filter) string {
func (p *Poet) TaskTable(opts TableOpts) string {
	if rerr := p.checkRecurring(); rerr != nil {
		log.Warn("problem looking up recurring tasks", "err", rerr)
	}
	tasks := ApplyFilters(p.MustList(opts.Prefix), &opts.FilterParams, opts.Filters...)
	allTasksLen := len(tasks)
	switch opts.SortBy.(type) {
	case ByDue:
		sort.Sort(ByDue(tasks))
	case ByCompleted:
		sort.Sort(ByCompleted(tasks))
	default:
		sort.Sort(tasks)
	}
	if opts.FilterParams.Limit > 0 {
		tasks = tasks[0:min(len(tasks), opts.FilterParams.Limit)]
	}

	columns := columnsOrDefault(opts.Columns)

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		iterateColumnHeaders(columns, tasks)...,
	)
	headerLen := lipgloss.Width(row)
	doc := strings.Builder{}
	doc.WriteString(row + "\n")

	for idx, task := range tasks {
		rs := altRowStyle(idx, entry, entryAlt)
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			iterateColumnValues(columns, task, tasks, rs)...,
		)
		doc.WriteString(row + "\n")
	}
	addLimitWarning(&doc, headerLen-4, opts.FilterParams.Limit, allTasksLen)

	w, _, _ := term.GetSize(int(os.Stdout.Fd()))
	maxW := max(w, headerLen)
	log.Debug("setting max window size", "size", maxW)
	docStyle = docStyle.MaxWidth(maxW)
	return docStyle.Render(doc.String())
}

func addLimitWarning(doc io.StringWriter, width, limit, total int) {
	if (limit > 0) && limit < total {
		_, _ = doc.WriteString(
			lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Italic(true).Render(
				fmt.Sprintf("* %v more records to display, increase the limit to see it",
					total-limit)),
		)
	}
}

func longestDescription(tasks Tasks) int {
	// Description is 11 chars long itself, add 2 for padding
	r := 13
	for _, task := range tasks {
		l := len(task.Description)
		if l > r {
			r = l
		}
	}
	return r
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
			if !f(p, r) {
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

func altRowStyle(idx int, even, odd lipgloss.Style) lipgloss.Style {
	if idx%2 == 0 {
		return even
	}
	return odd
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

type taskColumn interface {
	String() string
	Width() int
}

type shortIDCol struct{}

func (c shortIDCol) String() string { return "ID" }
func (c shortIDCol) Width() int     { return 8 }

type ageCol struct {
	name string
}

func (a ageCol) String() string { return a.name }
func (a ageCol) Width() int     { return 15 }

type dueCol struct{}

func (d dueCol) String() string { return "Due" }
func (d dueCol) Width() int     { return 15 }

type descriptionCol struct {
	tasks Tasks
}

func (d descriptionCol) String() string { return descriptionColumnName }
func (d descriptionCol) Width() int     { return min(55, longestDescription(d.tasks)+3) }

type tagsCol struct {
	tasks Tasks
}

func (t tagsCol) String() string { return "Tags" }
func (t tagsCol) Width() int     { return 15 }

type completedCol struct{}

func (d completedCol) String() string { return "Completed" }
func (d completedCol) Width() int     { return 13 }

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
