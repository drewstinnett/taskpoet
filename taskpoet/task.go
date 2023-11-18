package taskpoet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	bolt "go.etcd.io/bbolt"
)

var prefixes []string

// Task is the actual task item
type Task struct {
	ID           string       `json:"id"`
	PluginID     string       `json:"plugin_id"`
	Description  string       `json:"description"`
	Due          *time.Time   `json:"due,omitempty"`
	HideUntil    *time.Time   `json:"hide_until,omitempty"`   // HideUntil is similar to 'wait' in taskwarrior
	CancelAfter  *time.Time   `json:"cancel_after,omitempty"` // CancelAfter is similar to 'until' in taskwarrior
	Completed    *time.Time   `json:"completed,omitempty"`
	Reviewed     *time.Time   `json:"reviewed,omitempty"`
	Deleted      *time.Time   `json:"deleted,omitempty"`
	Added        time.Time    `json:"added,omitempty"`
	EffortImpact EffortImpact `json:"effort_impact"`
	Children     []string     `json:"children,omitempty"`
	Parents      []string     `json:"parents,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Comments     []Comment    `json:"comments,omitempty"`
	Project      string       `json:"project,omitempty"`
	Urgency      float64      `json:"urgency,omitempty"`
}

// DescriptionDetails is the details along with any comments or extra info we like to include
func (t Task) DescriptionDetails() string {
	ret := strings.Builder{}
	ret.WriteString(t.Description + "\n")
	if len(t.Comments) > 0 {
		for _, c := range t.Comments {
			ret.WriteString(fmt.Sprintf(" %v - %v\n", c.Added.Format("2006-01-02"), c.Text))
		}
	}
	return strings.TrimSpace(ret.String())
}

// Comment is just a little comment/note on a task
type Comment struct {
	Added time.Time `json:"added,omitempty"`
	Text  string    `json:"text,omitempty"`
}

// NewComment returns a new comment item using functional arguments
func NewComment(s string) (*Comment, error) {
	if s == "" {
		return nil, errors.New("text must not be empty")
	}
	return &Comment{
		Added: time.Now(),
		Text:  s,
	}, nil
}

// AddComment adds a comment to the task
func (t *Task) AddComment(s string) error {
	c, err := NewComment(s)
	if err != nil {
		return err
	}
	t.Comments = append(t.Comments, *c)
	return nil
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
	return t[i].Added.Before(t[j].Added)
}

// DetectKeyPath finds the key path for a given task
func (t *Task) DetectKeyPath() []byte {
	// Is this a new active task, or just logging completed?
	pluginID := DefaultPluginID
	if t.PluginID != "" {
		pluginID = t.PluginID
	}
	switch {
	case t.Deleted != nil:
		return []byte(filepath.Join("/deleted", pluginID, t.ID))
	case t.Completed != nil:
		return []byte(filepath.Join("/completed", pluginID, t.ID))
	default:
		return []byte(filepath.Join("/active", pluginID, t.ID))
	}
}

func (t *Task) setDefaults(d *Task) {
	// Handle defaults due
	if d != nil {
		if t.Due == nil && d.Due != nil {
			t.Due = d.Due
		}
	}
}

// ShortID is just the first 5 characters of the ID
func (t *Task) ShortID() string {
	return t.ID[0:min(len(t.ID), 5)]
}

// TaskService is the interface to tasks...we'll delete this junk
type TaskService interface {
	// This should replace New, and Log
	Add(t *Task) (*Task, error)
	AddSet(t Tasks) error

	// BucketName() string

	// Helper for adding Parent
	AddParent(c, p *Task) error
	AddChild(p, c *Task) error

	// Describe something
	Describe(t *Task) error

	// Edit a Task entry
	Edit(t *Task) (*Task, error)
	EditSet(t []Task) error

	// Should we use this instead of above funcs?
	AddOrEditSet(tasks []Task) error

	// Purge completely removes a task out of the DB
	Purge(t *Task) error

	// Complete a task
	Complete(t *Task) error

	// Sweet lord, this gettin' confusin' Drew
	GetPlugins() (map[string]Creator, error)
	SyncPlugin(TaskPlugin) error

	// States of a Task
	GetStates() []string
	GetStatePaths() []string

	// Just a little helper function to add and immediately mark as completed
	Log(t, d *Task) (*Task, error)

	// This on prolly needs work
	List(prefix string) (Tasks, error)
	/*
	   Path Conventions
	   /${state}/${plugin-id}/${id}
	   $state -> Active or Completed
	   $plugin-id -> Could be 'default'...or ''...or 'plugin-1'
	   $id -> Unique identifier

	   $plugin-id/$id should be unique together. Think things like:
	   gitlab.com/ISSUE-2
	   gitlab.local.io/ISSUE-2
	*/
	// New way to get stuff
	GetWithID(id, pluginID, state string) (*Task, error)
	GetWithPartialID(partialID, pluginID, state string) (*Task, error)
	GetWithExactPath(path []byte) (*Task, error)

	// Old way to get stuff, delete soon
	GetIDsByPrefix(prefix string) ([]string, error)
}

// GetStates returns types of states
func (svc *TaskServiceOp) GetStates() []string {
	return []string{"active", "completed", "deleted"}
}

// GetStatePaths is each of the states with a / prefix i guess
func (svc *TaskServiceOp) GetStatePaths() []string {
	s := svc.GetStates()
	r := make([]string, len(s))
	for idx, item := range s {
		r[idx] = filepath.Join("/", item)
	}
	return r
}

// SyncPlugin syncs a plugin or whatever
func (svc *TaskServiceOp) SyncPlugin(tp TaskPlugin) error {
	ts, err := tp.Sync()
	if err != nil {
		return err
	}
	err = svc.localClient.Task.AddOrEditSet(ts)
	if err != nil {
		return err
	}
	return nil
}

// GetPlugins returns all plugins
func (svc *TaskServiceOp) GetPlugins() (map[string]Creator, error) {
	return TaskPlugins, nil
}

// AddParent adds a parent to a child command
func (svc *TaskServiceOp) AddParent(c, p *Task) error {
	c.Parents = append(c.Parents, p.ID)
	p.Children = append(p.Children, c.ID)

	if err := svc.EditSet([]Task{*c, *p}); err != nil {
		return err
	}

	return nil
}

// AddChild adds a child to a parent
func (svc *TaskServiceOp) AddChild(p, c *Task) error {
	p.Children = append(p.Children, c.ID)
	c.Parents = append(c.Parents, p.ID)

	if err := svc.EditSet([]Task{*c, *p}); err != nil {
		return err
	}

	return nil
}

// AddOrEditSet adds or edits a set of tasks
func (svc *TaskServiceOp) AddOrEditSet(tasks []Task) error {
	var addSet Tasks
	var editSet []Task
	for _, t := range tasks {
		t := t
		_, err := svc.GetWithExactPath(t.DetectKeyPath())
		if err != nil {
			addSet = append(addSet, &t)
		} else {
			editSet = append(editSet, t)
		}
	}

	if err := svc.localClient.Task.AddSet(addSet); err != nil {
		return err
	}
	if err := svc.localClient.Task.EditSet(editSet); err != nil {
		return err
	}

	return nil
}

// EditSet edits a single set of tasks
func (svc *TaskServiceOp) EditSet(tasks []Task) error { //nolint:funlen,gocognit
	// We need to merge the new value with the old
	var mergedTasks []Task

	for _, t := range tasks {
		t := t
		originalTask, err := svc.GetWithExactPath(t.DetectKeyPath())
		if err != nil {
			return errors.New("cannot edit a task that did not previously exist: " + t.ID)
		}

		// Right now we wanna use the Complete function to do this, not edit...at least yet
		if originalTask.Completed != t.Completed {
			return errors.New("editing the Completed field is not yet supported as it changes the path")
		}

		// var mergedTask Task
		// Decide how to handle missing data and such
		// Added should never change
		t.Added = originalTask.Added

		// Re-add the description if needed
		if t.Description == "" {
			t.Description = originalTask.Description
		}

		if t.Due == nil {
			t.Due = originalTask.Due
		}

		if t.Completed == nil {
			t.Completed = originalTask.Completed
		}
		if t.HideUntil == nil {
			t.HideUntil = originalTask.HideUntil
		}

		if t.EffortImpact == 0 {
			t.EffortImpact = originalTask.EffortImpact
		}

		mergedTasks = append(mergedTasks, t)
	}

	err := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		for _, t := range mergedTasks {
			taskSerial, err := json.Marshal(t)
			if err != nil {
				return err
			}
			if perr := svc.localClient.getBucket(tx).Put(t.DetectKeyPath(), taskSerial); perr != nil {
				return perr
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Purge removes a task from the database
func (svc *TaskServiceOp) Purge(t *Task) error {
	originalTask, err := svc.GetWithExactPath(t.DetectKeyPath())
	if err != nil {
		return errors.New("Cannot delete a task that did not previously exist: " + t.ID)
	}

	if svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		if derr := svc.localClient.getBucket(tx).Delete(originalTask.DetectKeyPath()); derr != nil {
			return err
		}
		return nil
	}) != nil {
		return err
	}

	return nil
}

// Edit edits an existing task
func (svc *TaskServiceOp) Edit(t *Task) (*Task, error) {
	originalTask, err := svc.GetWithExactPath(t.DetectKeyPath())
	if err != nil {
		return nil, errors.New("cannot edit a task that did not previously exist: " + t.ID)
	}

	if verr := t.Validate(); verr != nil {
		return nil, verr
	}

	// Right now we wanna use the Complete function to do this, not edit...at least yet
	if originalTask.Completed != t.Completed {
		return nil, errors.New("editing the Completed field is not yet supported as it changes the path")
	}

	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	if uerr := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		if err := svc.localClient.getBucket(tx).Put(t.DetectKeyPath(), taskSerial); err != nil {
			return err
		}
		return nil
	}); uerr != nil {
		return nil, uerr
	}

	return t, nil
}

// Describe describes a task
func (svc *TaskServiceOp) Describe(t *Task) error {
	now := time.Now()
	var due string
	var wait string
	var completed string
	if t.Due == nil {
		due = "-"
	} else {
		due = humanizeDuration(t.Due.Sub(now))
	}
	if t.HideUntil == nil {
		wait = "-"
	} else {
		wait = humanizeDuration(t.HideUntil.Sub(now))
	}
	added := humanizeDuration(t.Added.Sub(now))
	if t.Completed == nil {
		completed = "-"
	} else {
		completed = humanizeDuration(t.Completed.Sub(now))
	}
	var parentsBuff bytes.Buffer
	var childrenBuff bytes.Buffer
	for _, p := range t.Parents {
		parent, err := svc.GetWithID(p, "", "")
		if err != nil {
			return nil
		}
		parentsBuff.Write([]byte(parent.Description))
	}
	for _, c := range t.Children {
		child, err := svc.GetWithID(c, "", "")
		if err != nil {
			return nil
		}
		childrenBuff.Write([]byte(child.Description))
	}
	data := [][]string{
		{"Field", "Value", "Read-Value"},
		{"ID", t.ShortID(), t.ID},
		{"Description", t.DescriptionDetails(), ""},
		{"Added", added, fmt.Sprintf("%+v", t.Added)},
		{"Completed", completed, fmt.Sprintf("%+v", t.Completed)},
		{"Due", due, fmt.Sprintf("%+v", t.Due)},
		{"Wait", wait, fmt.Sprintf("%+v", t.HideUntil)},
		{"Effort/Impact", t.EffortImpact.String(), fmt.Sprintf("%+v", t.EffortImpact)},
		{"Parents", parentsBuff.String(), fmt.Sprintf("%+v", t.Parents)},
		{"Children", childrenBuff.String(), fmt.Sprintf("%+v", t.Children)},
		{"Tags", fmt.Sprintf("%+v", t.Tags), ""},
	}

	return pterm.DefaultTable.WithHasHeader().WithData(data).Render()
}

// TaskServiceOp is the TaskService Operator
type TaskServiceOp struct {
	localClient *Poet
}

// DefaultPluginID is just the string used as the built in plugin default
const DefaultPluginID string = "builtin"

// GetWithPartialID returns using a partial id of the task
func (svc *TaskServiceOp) GetWithPartialID(partialID, pluginID, state string) (*Task, error) {
	var possibleStates []string
	if state == "" {
		possibleStates = svc.GetStatePaths()
	} else {
		possibleStates = append(possibleStates, state)
	}
	matches := []string{}
	if pluginID == "" {
		pluginID = DefaultPluginID
	}
	qualifiedPartialID := filepath.Join(pluginID, partialID)
	for _, prefix := range possibleStates {
		ids, err := svc.GetIDsByPrefix(fmt.Sprintf("%v/%v", prefix, qualifiedPartialID))
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if strings.HasPrefix(id, fmt.Sprintf("%v/%v", prefix, qualifiedPartialID)) {
				matches = append(matches, id)
			}
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches for %v found in %v", partialID, prefixes)
	} else if len(matches) > 1 {
		return nil, fmt.Errorf(
			"more than 1 match for %v found in %v, please try using more of the ID. Returned: %v", partialID, prefixes, matches)
	}
	return svc.GetWithExactPath([]byte(matches[0]))
}

// GetWithExactPath returns a task from an exact path
func (svc *TaskServiceOp) GetWithExactPath(path []byte) (*Task, error) {
	var task Task
	if err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		taskBytes := svc.localClient.getBucket(tx).Get(path)
		if taskBytes == nil {
			return fmt.Errorf("could not find task: %s", path)
		}
		return json.Unmarshal(taskBytes, &task)
	}); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetWithID returns a task from an ID
func (svc *TaskServiceOp) GetWithID(id, pluginID, state string) (*Task, error) {
	var possibleKeypaths []string
	if pluginID == "" {
		pluginID = DefaultPluginID
	}

	if state != "" {
		possibleKeypaths = append(possibleKeypaths, filepath.Join(state, pluginID, id))
	} else {
		for _, state := range svc.GetStatePaths() {
			possibleKeypaths = append(possibleKeypaths, filepath.Join(state, pluginID, id))
		}
	}
	for _, possible := range possibleKeypaths {
		task, err := svc.GetWithExactPath([]byte(possible))
		if err == nil {
			return task, nil
		}
	}
	return nil, fmt.Errorf("could not find that task at any of %v", possibleKeypaths)
}

// Complete marks a task as completed
func (svc *TaskServiceOp) Complete(t *Task) error {
	activePath := t.DetectKeyPath()
	t.Completed = nowPTR()
	completePath := t.DetectKeyPath()
	if err := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		taskSerial, err := json.Marshal(t)
		if err != nil {
			return err
		}
		b := svc.localClient.getBucket(tx)
		if perr := b.Put(completePath, taskSerial); perr != nil {
			return perr
		}
		return b.Delete(activePath)
	}); err != nil {
		return err
	}
	return nil
}

// List lists items under a given prefix
func (svc *TaskServiceOp) List(prefix string) (Tasks, error) {
	var tasks Tasks
	if err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		if err := svc.localClient.getBucket(tx).ForEach(func(k, v []byte) error {
			var task Task
			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}
			if strings.HasPrefix(string(k), prefix) {
				tasks = append(tasks, &task)
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return tasks, nil
}

// Log is a Shortcut utility to add a new task, but a completed time of 'now'
func (svc *TaskServiceOp) Log(t *Task, d *Task) (*Task, error) {
	if t.Completed == nil {
		t.Completed = nowPTR()
	}
	return svc.Add(t)
}

// AddSet adds a task set
func (svc *TaskServiceOp) AddSet(t Tasks) error {
	for _, task := range t {
		task := task
		if _, err := svc.Add(task); err != nil {
			return err
		}
	}
	return nil
}

// Add adds a new task
func (svc *TaskServiceOp) Add(t *Task) (*Task, error) {
	// t is the new task
	t.setDefaults(&svc.localClient.Default)

	// Assign a weight/urgency
	t.Urgency = svc.localClient.curator.Weigh(*t)

	// Does this already exist??
	if svc.localClient.exists(t) {
		return nil, errExists
	}

	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	if uerr := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		if perr := svc.localClient.getBucket(tx).Put(t.DetectKeyPath(), taskSerial); perr != nil {
			return perr
		}
		return nil
	}); uerr != nil {
		return nil, uerr
	}

	return t, nil
}

// GetIDsByPrefix returns a list of ids matching the given prefix
func (svc *TaskServiceOp) GetIDsByPrefix(prefix string) ([]string, error) {
	allIDs := []string{}

	if err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		if err := svc.localClient.getBucket(tx).ForEach(func(k, v []byte) error {
			if strings.HasPrefix(string(k), prefix) {
				allIDs = append(allIDs, string(k))
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return allIDs, nil
}

// CompleteIDsWithPrefix returns a list of ids matching the given prefix and
// autocomplete pattern
func (p Poet) CompleteIDsWithPrefix(prefix, toComplete string) []string {
	allIDs := []string{}

	if err := p.DB.View(func(tx *bolt.Tx) error {
		if err := p.getBucket(tx).ForEach(func(k, v []byte) error {
			if strings.HasPrefix(string(k), prefix) {
				idPieces := strings.Split(string(k), "/")
				id := idPieces[len(idPieces)-1]
				var task Task
				panicIfErr(json.Unmarshal(v, &task))
				if strings.HasPrefix(id, toComplete) || strings.Contains(task.Description, toComplete) {
					allIDs = append(allIDs, fmt.Sprintf("%v\t%v", id[0:5], task.Description))
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil
	}

	return allIDs
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

// TaskOption is a functional option for a new Task
type TaskOption func(*Task)

// WithEffortImpact sets the impact statement on create
func WithEffortImpact(e EffortImpact) TaskOption {
	return func(t *Task) {
		t.EffortImpact = e
	}
}

// WithDescription sets the description on create
func WithDescription(d string) TaskOption {
	return func(t *Task) {
		t.Description = d
	}
}

// WithID sets the UUID on create
func WithID(i string) TaskOption {
	return func(t *Task) {
		t.ID = i
	}
}

// WithChildren set the children of a task
func WithChildren(c []string) TaskOption {
	return func(t *Task) {
		t.Children = c
	}
}

// WithParents set the parents of a task
func WithParents(p []string) TaskOption {
	return func(t *Task) {
		t.Parents = p
	}
}

// WithTags sets the tags on create
func WithTags(s []string) TaskOption {
	return func(t *Task) {
		t.Tags = s
	}
}

// WithDue sets the due date on create
func WithDue(d *time.Time) TaskOption {
	return func(t *Task) {
		t.Due = d
	}
}

// WithCompleted sets the completed date on create
func WithCompleted(d *time.Time) TaskOption {
	return func(t *Task) {
		t.Completed = d
	}
}

// WithAdded sets the added date on create
func WithAdded(d *time.Time) TaskOption {
	return func(t *Task) {
		t.Added = *d
	}
}

// WithHideUntil sets the hide until on create
func WithHideUntil(d *time.Time) TaskOption {
	return func(t *Task) {
		t.HideUntil = d
	}
}

// WithTaskWarriorTask imports a task from a task warrior task
func WithTaskWarriorTask(twItem TaskWarriorTask) TaskOption {
	return func(t *Task) {
		t.Description = twItem.Description
		if twItem.UUID != "" {
			t.ID = twItem.UUID
		} else {
			t.ID = uuid.New().String()
		}
		t.Tags = twItem.Tags
		t.Due = (*time.Time)(twItem.Due)
		t.Completed = (*time.Time)(twItem.End)
		t.Reviewed = (*time.Time)(twItem.Reviewed)
		t.CancelAfter = (*time.Time)(twItem.Until)
		if twItem.Status == "deleted" {
			t.Deleted = (*time.Time)(twItem.End)
		}
		if twItem.Entry == nil {
			t.Added = time.Now()
		} else {
			t.Added = time.Time(*twItem.Entry)
		}
		if twItem.Annotations != nil {
			t.Comments = make([]Comment, len(twItem.Annotations))
			for idx, a := range twItem.Annotations {
				t.Comments[idx].Text = a.Description
				t.Comments[idx].Added = time.Time(*a.Entry)
			}
		}

		if (twItem.Wait != nil) && (twItem.Due != nil) && (*time.Time)(twItem.Wait).After((time.Time)(*twItem.Due)) {
			nh := t.Due.Add(-1 * time.Minute)
			t.HideUntil = &nh
		} else {
			t.HideUntil = (*time.Time)(twItem.Wait)
		}
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

// Validate makes sure the task isn't malformed
func (t Task) Validate() error {
	switch {
	case t.Description == "":
		return errors.New("missing description for Task")
	case strings.Contains(t.ID, "/"):
		return errors.New("ID Cannot contain a slash (/)")
		// If both HideUntil and Due are set, make sure HideUntil isn't after Due
	case (t.HideUntil != nil && t.Due != nil) && t.HideUntil.After(*t.Due):
		return fmt.Errorf("HideUntil cannot be later than Due")
	// Make sure we didn't add ourself
	case containsString(t.Parents, t.ID):
		return fmt.Errorf("self id is set in the parents, we don't do that")
	case containsString(t.Children, t.ID):
		return fmt.Errorf("self id is set in the children, we don't do that")
	// Make sure Parents contains no duplicates
	case !CheckUniqueStringSlice(t.Parents):
		return fmt.Errorf("found duplicate ids in the Parents field")
	default:
		return nil
	}
}

// NewTask returns a new task given functional options
func NewTask(desc string, options ...TaskOption) (*Task, error) {
	task := &Task{
		ID:          uuid.New().String(),
		Description: desc,
		Added:       time.Now(),
		PluginID:    DefaultPluginID,
	}
	for _, opt := range options {
		opt(task)
	}
	// Sort the tags alphabetically
	sort.Strings(task.Tags)
	return task, task.Validate()
}
