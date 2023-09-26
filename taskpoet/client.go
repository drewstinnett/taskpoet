/*
Package taskpoet is the main worker library
*/
package taskpoet

import (
	"errors"
	"fmt"
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

// TaskTable returns a table of the given tasks
func (p *Poet) TaskTable(prefix string, fp *FilterParams, filters ...Filter) string {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		fmt.Fprint(os.Stderr, "unable to calculate height and width of terminal")
	}
	tasks, err := p.Task.List("/active")
	if err != nil {
		panic(err)
	}
	sort.Sort(tasks)
	tasks = ApplyFilters(tasks, fp, filters...)

	allTasksLen := len(tasks)
	if fp.Limit > 0 {
		tasks = tasks[0:min(len(tasks), fp.Limit)]
	}

	doc := strings.Builder{}
	dLen := min(longestDescription(tasks)+5, 50)
	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		header.Width(8).Render("ID"),
		header.Width(15).Render("Age"),
		header.Width(dLen).Render("Description"),
		header.Width(15).Render("Due"),
		header.Width(15).Render("Tags"),
	)
	doc.WriteString(row + "\n")

	for idx, task := range tasks {
		var rs lipgloss.Style
		if idx%2 == 0 {
			rs = entry
		} else {
			rs = entryAlt
		}
		var due string
		if task.Due != nil {
			due = task.Due.Format("2006-01-02")
		}
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			rs.Width(8).Render(task.ShortID()),
			rs.Width(15).Render(task.Added.Format("2006-01-02")),
			rs.Width(dLen).Render(task.Description),
			rs.Width(15).Render(due),
			rs.Width(15).Render(strings.Join(task.Tags, ",")),
		)
		doc.WriteString(row + "\n")
	}

	if fp.Limit < allTasksLen {
		doc.WriteString(
			lipgloss.NewStyle().Width(53 + dLen).Align(lipgloss.Right).Render(fmt.Sprintf("\n* %v more records to display, increase the limit to see it\n", allTasksLen-fp.Limit)),
		)
	}

	if w > 0 {
		docStyle = docStyle.MaxWidth(w)
	}
	return docStyle.Render(doc.String())
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
