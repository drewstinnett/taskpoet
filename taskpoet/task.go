package taskpoet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	bolt "go.etcd.io/bbolt"
)

var prefixes []string

// Task is the actual task item
type Task struct {
	ID           string     `json:"id"`
	PluginID     string     `json:"plugin_id"`
	Description  string     `json:"description"`
	Due          *time.Time `json:"due,omitempty"`
	HideUntil    *time.Time `json:"hide_until,omitempty"`   // HideUntil is similar to 'wait' in taskwarrior
	CancelAfter  *time.Time `json:"cancel_after,omitempty"` // CancelAfter is similar to 'until' in taskwarrior
	Completed    *time.Time `json:"completed,omitempty"`
	Reviewed     *time.Time `json:"reviewed,omitempty"`
	Added        time.Time  `json:"added,omitempty"`
	EffortImpact uint       `json:"effort_impact"`
	Children     []string   `json:"children,omitempty"`
	Parents      []string   `json:"parents,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
	Comments     []Comment  `json:"comments,omitempty"`
	Project      string     `json:"project,omitempty"`
}

// Comment is just a little comment/note on a task
type Comment struct {
	Added   time.Time `json:"added,omitempty"`
	Comment string    `json:"comment,omitempty"`
}

// DueString returns a string representation of Due
func (t Task) DueString() string {
	if t.Due != nil {
		return t.Due.Format("2006-01-02")
	}
	return ""
}

// Tasks represents multiple Task items
type Tasks []Task

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
	var keyPath string
	var pluginID string
	if t.PluginID == "" {
		pluginID = DefaultPluginID
	} else {
		pluginID = t.PluginID
	}
	if t.Completed == nil {
		keyPath = filepath.Join("/active", pluginID, t.ID)
	} else {
		keyPath = filepath.Join("/completed", pluginID, t.ID)
	}
	return []byte(keyPath)
}

func (t *Task) setDefaults(d *Task) {
	if t.Added.IsZero() {
		t.Added = time.Now()
	}

	// If no ID is set, just generate one
	if t.ID == "" {
		t.ID = uuid.New().String()
	}

	// Set the plugin id to default if not set
	if t.PluginID == "" {
		t.PluginID = DefaultPluginID
	}
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

// TaskValidateOpts is a dumb struct...why is this here?
type TaskValidateOpts struct {
	IsExisting bool
}

// TaskService is the interface to tasks...we'll delete this junk
type TaskService interface {
	// This should replace New, and Log
	Add(t *Task) (*Task, error)
	AddSet(t []Task) error

	BucketName() string

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

	// Delete
	Delete(t *Task) error

	// Complete a task
	Complete(t *Task) error

	// Check to ensure Task is in a valid state
	Validate(t *Task, o *TaskValidateOpts) error

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

// BucketName returns the name of the task bucket
func (svc *TaskServiceOp) BucketName() string {
	return fmt.Sprintf("/%v/tasks", svc.localClient.Namespace)
}

// GetStates returns types of states
func (svc *TaskServiceOp) GetStates() []string {
	return []string{"active", "completed"}
}

// GetStatePaths is each of the states with a / prefix i guess
func (svc *TaskServiceOp) GetStatePaths() []string {
	var r []string
	for _, item := range svc.GetStates() {
		r = append(r, filepath.Join("/", item))
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
	var addSet []Task
	var editSet []Task
	for _, t := range tasks {
		_, err := svc.GetWithExactPath(t.DetectKeyPath())
		if err != nil {
			addSet = append(addSet, t)
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

		err = svc.Validate(&t, &TaskValidateOpts{IsExisting: true})
		if err != nil {
			return err
		}

		// Now do the merge
		if err != nil {
			return err
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
			b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
			err = b.Put(t.DetectKeyPath(), taskSerial)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Delete deletes a task
func (svc *TaskServiceOp) Delete(t *Task) error {
	originalTask, err := svc.GetWithExactPath(t.DetectKeyPath())
	if err != nil {
		return errors.New("Cannot delete a task that did not previously exist: " + t.ID)
	}

	if svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		if derr := b.Delete(originalTask.DetectKeyPath()); derr != nil {
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

	// Right now we wanna use the Complete function to do this, not edit...at least yet
	if originalTask.Completed != t.Completed {
		return nil, errors.New("editing the Completed field is not yet supported as it changes the path")
	}

	if verr := svc.Validate(t, &TaskValidateOpts{IsExisting: true}); verr != nil {
		return nil, verr
	}

	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	uerr := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		err = b.Put(t.DetectKeyPath(), taskSerial)
		if err != nil {
			return err
		}
		return nil
	})
	if uerr != nil {
		return nil, uerr
	}

	return t, nil
}

// Validate validates a task i guess...
func (svc *TaskServiceOp) Validate(t *Task, o *TaskValidateOpts) error {
	if t.Description == "" {
		return errors.New("missing description for Task")
	}

	if strings.Contains(t.ID, "/") {
		return errors.New("ID Cannot contain a slash (/)")
	}

	// If not specified
	if o == nil {
		o = &TaskValidateOpts{}
	}

	// IF NEW, Make sure this ID doesn't already exist
	if !o.IsExisting {
		_, err := svc.GetWithID(t.ID, t.PluginID, "")
		if err == nil {
			return fmt.Errorf("Task with ID %v already exists", t.ID)
		}
	}

	// If both HideUntil and Due are set, make sure HideUntil isn't after Due
	if t.HideUntil != nil && t.Due != nil {
		if t.HideUntil.After(*t.Due) {
			return fmt.Errorf("HideUntil cannot be later than Due")
		}
	}

	// Make sure we didn't add ourself
	if ContainsString(t.Parents, t.ID) {
		return fmt.Errorf("self id is set in the parents, we don't do that")
	}
	if ContainsString(t.Children, t.ID) {
		return fmt.Errorf("self id is set in the children, we don't do that")
	}

	// Make sure Parents contains no duplicates
	if !CheckUniqueStringSlice(t.Parents) {
		return fmt.Errorf("found duplicate ids in the Parents field")
	}
	return nil
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
		{"Description", t.Description, ""},
		{"Added", added, fmt.Sprintf("%+v", t.Added)},
		{"Completed", completed, fmt.Sprintf("%+v", t.Completed)},
		{"Due", due, fmt.Sprintf("%+v", t.Due)},
		{"Wait", wait, fmt.Sprintf("%+v", t.HideUntil)},
		{"Effort/Impact", EffortImpactText(int(t.EffortImpact)), fmt.Sprintf("%+v", t.EffortImpact)},
		{"Parents", parentsBuff.String(), fmt.Sprintf("%+v", t.Parents)},
		{"Children", childrenBuff.String(), fmt.Sprintf("%+v", t.Children)},
		{"Tags", fmt.Sprintf("%+v", t.Tags), ""},
	}

	return pterm.DefaultTable.WithHasHeader().WithData(data).Render()
}

// TaskServiceOp is the TaskService Operator
type TaskServiceOp struct {
	localClient *Poet
	// plugins     []TaskPlugin
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
	t, err := svc.GetWithExactPath([]byte(matches[0]))
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetWithExactPath returns a task from an exact path
func (svc *TaskServiceOp) GetWithExactPath(path []byte) (*Task, error) {
	var task Task
	err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		taskBytes := b.Get(path)
		if taskBytes == nil {
			return fmt.Errorf("could not find task: %s", path)
		}
		err := json.Unmarshal(taskBytes, &task)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
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
	now := time.Now()
	activePath := t.DetectKeyPath()
	t.Completed = &now
	completePath := t.DetectKeyPath()
	err := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		taskSerial, err := json.Marshal(t)
		if err != nil {
			return err
		}
		err = b.Put(completePath, taskSerial)
		if err != nil {
			return err
		}
		return b.Delete(activePath)
	})
	if err != nil {
		return err
	}
	return nil
}

// List lists items under a given prefix
func (svc *TaskServiceOp) List(prefix string) (Tasks, error) {
	var tasks Tasks
	err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		err := b.ForEach(func(k, v []byte) error {
			var task Task
			err := json.Unmarshal(v, &task)
			if err != nil {
				return err
			}
			if strings.HasPrefix(string(k), prefix) {
				tasks = append(tasks, task)
			}
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// Log is a Shortcut utility to add a new task, but a completed time of 'now'
func (svc *TaskServiceOp) Log(t *Task, d *Task) (*Task, error) {
	if t.Completed == nil {
		n := time.Now()
		t.Completed = &n
	}
	ret, err := svc.Add(t)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// AddSet adds a task set
func (svc *TaskServiceOp) AddSet(t []Task) error {
	for _, task := range t {
		task := task
		_, err := svc.Add(&task)
		if err != nil {
			return err
		}
	}
	return nil
}

// Add adds a new task
func (svc *TaskServiceOp) Add(t *Task) (*Task, error) {
	// t is the new task
	t.setDefaults(&svc.localClient.Default)

	// Validate that Task is actually good
	err := svc.Validate(t, &TaskValidateOpts{IsExisting: false})
	if err != nil {
		return nil, err
	}
	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	err = svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		if perr := bucket.Put(t.DetectKeyPath(), taskSerial); perr != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return t, nil
}

// GetIDsByPrefix returns a list of ids matching the given prefix
func (svc *TaskServiceOp) GetIDsByPrefix(prefix string) ([]string, error) {
	allIDs := []string{}

	err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		err := bucket.ForEach(func(k, v []byte) error {
			if strings.HasPrefix(string(k), prefix) {
				allIDs = append(allIDs, string(k))
			}
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return allIDs, nil
}

// CompleteIDsWithPrefix returns a list of ids matching the given prefix and
// autocomplete pattern
func (p Poet) CompleteIDsWithPrefix(prefix, toComplete string) []string {
	allIDs := []string{}

	err := p.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(p.Task.BucketName()))
		err := bucket.ForEach(func(k, v []byte) error {
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
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil
	}

	return allIDs
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
