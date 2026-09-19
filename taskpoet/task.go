package taskpoet

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status is the lifecycle state of a task. It mirrors Taskwarrior, except that
// 'waiting' is not a stored status: a pending task with a future Wait is
// waiting (see Task.IsWaiting).
type Status string

const (
	// StatusPending is a task that still needs doing
	StatusPending Status = "pending"
	// StatusCompleted is a task that has been done
	StatusCompleted Status = "completed"
	// StatusDeleted is a task that was removed, but is kept for the record
	StatusDeleted Status = "deleted"
	// StatusRecurring is a recurrence template. It is never worked on directly,
	// it spawns pending instances instead.
	StatusRecurring Status = "recurring"
)

// AllStatuses returns every valid status
func AllStatuses() []Status {
	return []Status{StatusPending, StatusCompleted, StatusDeleted, StatusRecurring}
}

// Valid returns true if this is a known status
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusCompleted, StatusDeleted, StatusRecurring:
		return true
	}
	return false
}

// Priority is the Taskwarrior style priority of a task
type Priority string

const (
	// PriorityNone is unset
	PriorityNone Priority = ""
	// PriorityLow is L
	PriorityLow Priority = "L"
	// PriorityMedium is M
	PriorityMedium Priority = "M"
	// PriorityHigh is H
	PriorityHigh Priority = "H"
)

// Valid returns true if this is a known priority
func (p Priority) Valid() bool {
	switch p {
	case PriorityNone, PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}

// ParsePriority understands 'H', 'high', 'm', etc. An empty string is no priority
func ParsePriority(s string) (Priority, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return PriorityNone, nil
	case "l", "low":
		return PriorityLow, nil
	case "m", "med", "medium":
		return PriorityMedium, nil
	case "h", "high":
		return PriorityHigh, nil
	}
	return PriorityNone, fmt.Errorf("unknown priority %q, expected one of L, M, H", s)
}

// Annotation is a timestamped note on a task. This is the Taskwarrior
// annotation.
type Annotation struct {
	Entry       time.Time `json:"entry"`
	Description string    `json:"description"`
}

// Task is the actual task item. The core fields match Taskwarrior so that a
// 'task export' can be imported without losing anything.
type Task struct {
	UUID        string       `json:"uuid"`
	Status      Status       `json:"status"`
	Description string       `json:"description"`
	Project     string       `json:"project,omitempty"`
	Priority    Priority     `json:"priority,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Depends     []string     `json:"depends,omitempty"` // UUIDs of tasks that must be done first
	Annotations []Annotation `json:"annotations,omitempty"`

	Entry     time.Time  `json:"entry"`
	Modified  time.Time  `json:"modified"`
	Start     *time.Time `json:"start,omitempty"`
	End       *time.Time `json:"end,omitempty"` // when it was completed or deleted
	Due       *time.Time `json:"due,omitempty"`
	Wait      *time.Time `json:"wait,omitempty"`  // hidden until this time
	Until     *time.Time `json:"until,omitempty"` // expires after this time
	Scheduled *time.Time `json:"scheduled,omitempty"`
	Reviewed  *time.Time `json:"reviewed,omitempty"`

	// Recurrence. A template has Status recurring and Recur set. Its instances
	// point back at it with Parent and IMask.
	Recur  string `json:"recur,omitempty"`
	RType  string `json:"rtype,omitempty"` // periodic (default) or chained
	Mask   string `json:"mask,omitempty"`  // template: one char per spawned instance
	Parent string `json:"parent,omitempty"`
	IMask  *int   `json:"imask,omitempty"`

	// EffortImpact is the taskpoet Limoncelli extension
	EffortImpact EffortImpact `json:"effort_impact,omitempty"`

	// UDA holds every key we don't know about, so nothing is lost on import
	UDA map[string]any `json:"uda,omitempty"`

	// Urgency and the blocking info are computed when listing, never stored
	Urgency  float64 `json:"-"`
	blocking int     // number of pending tasks waiting on this one
	blocked  bool    // has an unfinished dependency
}

// Names of task fields that are used as keys in more than one place: the
// Taskwarrior export, and the urgency weights.
const (
	fieldDescription = "description"
	fieldPriority    = "priority"
	fieldDue         = "due"
	fieldEnd         = "end"
	fieldEntry       = "entry"
	fieldModified    = "modified"
	fieldStart       = "start"
	fieldWait        = "wait"
	fieldUntil       = "until"
	fieldScheduled   = "scheduled"
	fieldReviewed    = "reviewed"
)

// shortIDLen is how many characters of the UUID we show. This is the same as
// Taskwarrior, and is enough to stay unique in a large history.
const shortIDLen = 8

// ShortID is just the first few characters of the UUID
func (t *Task) ShortID() string {
	return t.UUID[0:min(len(t.UUID), shortIDLen)]
}

// IsWaiting is true for a pending task that is hidden until later
func (t Task) IsWaiting(now time.Time) bool {
	return t.Status == StatusPending && t.Wait != nil && t.Wait.After(now)
}

// IsBlocked is true when the task depends on something that isn't done yet.
// This is only known for tasks that came from Poet.List
func (t Task) IsBlocked() bool {
	return t.blocked
}

// DescriptionDetails is the details along with any annotations or extra info we like to include
func (t Task) DescriptionDetails() string {
	ret := strings.Builder{}
	ret.WriteString(t.Description + "\n")
	for _, a := range t.Annotations {
		fmt.Fprintf(&ret, " %v - %v\n", a.Entry.Format("2006-01-02"), a.Description)
	}
	return strings.TrimSpace(ret.String())
}

// AddAnnotation adds an annotation to the task
func (t *Task) AddAnnotation(s string, now time.Time) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("annotation must not be empty")
	}
	t.Annotations = append(t.Annotations, Annotation{Entry: now, Description: s})
	return nil
}

// Validate makes sure the task isn't malformed. This is deliberately loose
// about anything Taskwarrior allows, like a wait that is after the due date,
// so imports stay lossless.
func (t Task) Validate() error {
	switch {
	case t.UUID == "":
		return errors.New("missing UUID for Task")
	case !t.Status.Valid():
		return fmt.Errorf("invalid status %q", t.Status)
	case t.Description == "":
		return errors.New("missing description for Task")
	case !t.Priority.Valid():
		return fmt.Errorf("invalid priority %q", t.Priority)
	case containsString(t.Depends, t.UUID):
		return errors.New("a task cannot depend on itself")
	case !CheckUniqueStringSlice(t.Depends):
		return errors.New("found duplicate ids in the Depends field")
	case t.Parent != "" && t.Parent == t.UUID:
		return errors.New("a task cannot be its own parent")
	default:
		return nil
	}
}

// Tasks represents multiple Task items
type Tasks []*Task

// SortBy specifies how to sort the tasks
func (t *Tasks) SortBy(s any) {
	switch s.(type) {
	case ByDue:
		sort.Sort(ByDue(*t))
	case ByCompleted:
		sort.Sort(ByCompleted(*t))
	case ByUrgency:
		sort.Sort(ByUrgency(*t))
	default:
		sort.Sort(*t)
	}
}

// Len helps to satisfy the sort interface
func (t Tasks) Len() int {
	return len(t)
}

// Swap helps to satisfy the sort interface
func (t Tasks) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// Less helps to satisfy the sort interface
func (t Tasks) Less(i, j int) bool {
	return t[i].Entry.Before(t[j].Entry)
}

// TaskOption is a functional option for a new Task
type TaskOption func(*Task)

// WithEffortImpact sets the impact statement on create
func WithEffortImpact(e EffortImpact) TaskOption {
	return func(t *Task) {
		t.EffortImpact = e
	}
}

// WithUUID sets the UUID on create
func WithUUID(i string) TaskOption {
	return func(t *Task) {
		t.UUID = i
	}
}

// WithStatus sets the status on create
func WithStatus(s Status) TaskOption {
	return func(t *Task) {
		t.Status = s
	}
}

// WithDepends sets the tasks this one depends on
func WithDepends(d []string) TaskOption {
	return func(t *Task) {
		t.Depends = d
	}
}

// WithTags sets the tags on create
func WithTags(s []string) TaskOption {
	return func(t *Task) {
		t.Tags = s
	}
}

// WithProject sets the project on create
func WithProject(p string) TaskOption {
	return func(t *Task) {
		t.Project = p
	}
}

// WithPriority sets the priority on create
func WithPriority(p Priority) TaskOption {
	return func(t *Task) {
		t.Priority = p
	}
}

// WithDue sets the due date on create
func WithDue(d *time.Time) TaskOption {
	return func(t *Task) {
		t.Due = d
	}
}

// WithWait sets the wait date on create
func WithWait(d *time.Time) TaskOption {
	return func(t *Task) {
		t.Wait = d
	}
}

// WithScheduled sets the scheduled date on create
func WithScheduled(d *time.Time) TaskOption {
	return func(t *Task) {
		t.Scheduled = d
	}
}

// WithUntil sets the until date on create
func WithUntil(d *time.Time) TaskOption {
	return func(t *Task) {
		t.Until = d
	}
}

// WithRecur turns the task into a recurrence template
func WithRecur(recur string) TaskOption {
	return func(t *Task) {
		t.Recur = recur
		t.Status = StatusRecurring
	}
}

// WithEntry sets the entry (added) time on create
func WithEntry(d time.Time) TaskOption {
	return func(t *Task) {
		t.Entry = d
		t.Modified = d
	}
}

// WithCompleted marks a task as already completed at the given time
func WithCompleted(d time.Time) TaskOption {
	return func(t *Task) {
		t.Status = StatusCompleted
		t.End = &d
	}
}

// MustNewTask returns a task or panics
func MustNewTask(description string, options ...TaskOption) *Task {
	got, err := NewTask(description, options...)
	if err != nil {
		panic(err)
	}
	return got
}

// NewTask returns a new pending task given functional options
func NewTask(desc string, options ...TaskOption) (*Task, error) {
	now := time.Now().UTC().Truncate(time.Second)
	task := &Task{
		UUID:        uuid.New().String(),
		Status:      StatusPending,
		Description: desc,
		Entry:       now,
		Modified:    now,
	}
	for _, opt := range options {
		opt(task)
	}
	// Sort the tags alphabetically
	sort.Strings(task.Tags)
	return task, task.Validate()
}
