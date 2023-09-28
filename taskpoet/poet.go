/*
Package taskpoet is the main worker library
*/
package taskpoet

import (
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

// Poet isi the main operator for this whole thing
type Poet struct {
	DB        *bolt.DB
	Namespace string
	Task      TaskService
	dbPath    string
}

// initDB initializes the database
func (p *Poet) initDB() error {
	// store some data
	return p.DB.Update(func(tx *bolt.Tx) error {
		// localClient.
		bucket := tx.Bucket([]byte(p.Task.BucketName()))
		if bucket == nil {
			_, berr := tx.CreateBucket([]byte(p.Task.BucketName()))
			if berr != nil {
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
	docStyle  = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	subtle    = lipgloss.AdaptiveColor{Light: "#f3f4f0", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	header = lipgloss.NewStyle().
		Foreground(highlight).
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
}

// MustList returns tasks from a prefix or panics
func (p Poet) MustList(s string) Tasks {
	tasks, err := p.Task.List(s)
	if err != nil {
		panic(err)
	}
	return tasks
}

var columnMap map[string]func(Task) string = map[string]func(Task) string{
	"ID":          func(t Task) string { return t.ShortID() },
	"Age":         func(t Task) string { return t.Added.Format("2006-01-02") },
	"Description": func(t Task) string { return t.Description },
	"Due":         func(t Task) string { return t.DueString() },
	"Tags":        func(t Task) string { return strings.Join(t.Tags, ",") },
	"Completed": func(t Task) string {
		if t.Completed == nil {
			return ""
		}
		return t.Completed.Format("2006-01-02")
	},
}

var columnSizeMap map[string]int = map[string]int{
	"ID":          8,
	"Age":         15,
	"Description": 50,
	"Due":         15,
	"Tags":        15,
	"Completed":   15,
}

func columnSize(s string) (int, error) {
	val, ok := columnSizeMap[s]
	if !ok {
		return 0, fmt.Errorf("column size not defined: %v", s)
	}
	return val, nil
}

func mustColumnSize(s string) int {
	got, err := columnSize(s)
	if err != nil {
		panic(err)
	}
	return got
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

func iterateColumnHeaders(c []string) []string {
	ret := make([]string, len(c))
	for idx, item := range c {
		ret[idx] = header.Width(mustColumnSize(item)).Render(item)
	}
	return ret
}

func iterateColumnValues(c []string, t Task, s lipgloss.Style) []string {
	ret := make([]string, len(c))
	for idx, item := range c {
		ret[idx] = s.Width(mustColumnSize(item)).Render(mustColumnValue(item, t))
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
	tasks := ApplyFilters(p.MustList(opts.Prefix), &opts.FilterParams, opts.Filters...)
	sort.Sort(tasks)
	allTasksLen := len(tasks)
	if opts.FilterParams.Limit > 0 {
		tasks = tasks[0:min(len(tasks), opts.FilterParams.Limit)]
	}

	columns := columnsOrDefault(opts.Columns)

	dLen := min(longestDescription(tasks)+5, 50)
	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		iterateColumnHeaders(columns)...,
	)
	doc := strings.Builder{}
	doc.WriteString(row + "\n")

	for idx, task := range tasks {
		rs := altRowStyle(idx, entry, entryAlt)
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			iterateColumnValues(columns, task, rs)...,
		)
		doc.WriteString(row + "\n")
	}
	addLimitWarning(&doc, 53+dLen, opts.FilterParams.Limit, allTasksLen)

	w, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if w > 0 {
		docStyle = docStyle.MaxWidth(w)
	}
	return docStyle.Render(doc.String())
}

func addLimitWarning(doc io.StringWriter, width, limit, total int) {
	if limit < total {
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
