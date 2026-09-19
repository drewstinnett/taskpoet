/*
Package taskpoet is the main worker library
*/
package taskpoet

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drewstinnett/taskpoet/v2/themes"
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

// DefaultDBPath is where the database lives when nothing else is asked for.
// This is deliberately not the v0.x location (~/.taskpoet.db).
func DefaultDBPath() (string, error) {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "taskpoet", "taskpoet.db"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".local", "share", "taskpoet", "taskpoet.db"), nil
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
		curator:   NewCurator(),
		now:       defaultNow,

		recurLimit:   1,
		recurCatchUp: CatchUpLatest,
	}
	for _, option := range options {
		opt, err := option()
		if err != nil {
			return nil, err
		}
		opt(p)
	}

	if p.dbPath == "" {
		var err error
		if p.dbPath, err = DefaultDBPath(); err != nil {
			return nil, err
		}
	}

	var err error
	p.Store, err = OpenStore(p.dbPath, p.Namespace, WithClock(p.now))
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Close closes the underlying database
func (p *Poet) Close() error {
	return p.Store.Close()
}

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

// WithNow overrides the clock, mostly useful for tests
func WithNow(now func() time.Time) Option {
	return success(func(p *Poet) {
		p.now = now
	})
}

// WithDefaultDue gives every new task without a due date this offset from now
func WithDefaultDue(d time.Duration) Option {
	return success(func(p *Poet) {
		p.defaultDue = d
	})
}

// WithRecurrence sets how many future instances of a recurring task to keep
// ready, and what to do about the periods that were missed
func WithRecurrence(limit int, catchUp CatchUp) Option {
	if limit < 0 {
		return failure(errors.New("recurrence limit cannot be negative"))
	}
	return success(func(p *Poet) {
		p.recurLimit = limit
		p.recurCatchUp = catchUp
	})
}

// Poet is the main operator for this whole thing
type Poet struct {
	Store      *Store
	Namespace  string
	dbPath     string
	styling    themes.Styling
	curator    *Curator
	now        func() time.Time
	defaultDue time.Duration

	recurLimit   int
	recurCatchUp CatchUp
}

// refresh works out the urgency of each task, and which ones are blocking or
// blocked. This always looks at every pending task, so it doesn't matter which
// subset of tasks is passed in.
func (p *Poet) refresh(ts Tasks) error {
	pending, err := p.Store.List(StatusPending)
	if err != nil {
		return err
	}
	isPending := make(map[string]bool, len(pending))
	for _, t := range pending {
		isPending[t.UUID] = true
	}
	blocking := map[string]int{}
	for _, t := range pending {
		for _, d := range t.Depends {
			if isPending[d] {
				blocking[d]++
			}
		}
	}
	for _, t := range ts {
		t.blocking = blocking[t.UUID]
		t.blocked = false
		for _, d := range t.Depends {
			if isPending[d] {
				t.blocked = true
				break
			}
		}
		t.Urgency = p.curator.Weigh(*t)
	}
	return nil
}

// List returns the tasks in the given statuses (all of them when none are
// given), with urgency filled in
func (p *Poet) List(statuses ...Status) (Tasks, error) {
	ts, err := p.Store.List(statuses...)
	if err != nil {
		return nil, err
	}
	if err := p.refresh(ts); err != nil {
		return nil, err
	}
	return ts, nil
}

// Add stores a new task, applying the configured defaults
func (p *Poet) Add(t *Task) error {
	if t.Due == nil && p.defaultDue != 0 && t.Status != StatusCompleted {
		t.Due = ptr(p.now().Add(p.defaultDue))
	}
	if t.Status == StatusRecurring {
		if _, err := ParseRecurrence(t.Recur); err != nil {
			return err
		}
		if t.Due == nil {
			return errors.New("a recurring task needs a due date to count from")
		}
	}
	if err := p.Store.Add(t); err != nil {
		return err
	}
	return p.refresh(Tasks{t})
}

// SpawnRecurring creates the instances that recurring tasks are due, using
// the configured limit and catch up mode
func (p *Poet) SpawnRecurring(dryRun bool) (*SpawnReport, error) {
	return p.Store.SpawnRecurring(SpawnOptions{
		Now:      p.now(),
		Location: time.Local,
		Limit:    p.recurLimit,
		CatchUp:  p.recurCatchUp,
		DryRun:   dryRun,
	})
}

// CompleteIDs returns 'shortid<tab>description' lines for shell completion of
// task ids
func (p *Poet) CompleteIDs(toComplete string, statuses ...Status) []string {
	ts, err := p.Store.List(statuses...)
	if err != nil {
		return nil
	}
	toComplete = strings.ToLower(toComplete)
	ids := []string{}
	for _, t := range ts {
		if strings.HasPrefix(t.UUID, toComplete) || strings.Contains(strings.ToLower(t.Description), toComplete) {
			ids = append(ids, fmt.Sprintf("%v\t%v", t.ShortID(), t.Description))
		}
	}
	return ids
}
