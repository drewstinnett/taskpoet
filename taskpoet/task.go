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
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var prefixes []string

type Task struct {
	ID           string     `json:"id"`
	PluginID     string     `json:"plugin_id"`
	Description  string     `json:"description"`
	Due          *time.Time `json:"due,omitempty"`
	HideUntil    *time.Time `json:"hide_until,omitempty"`
	Completed    *time.Time `json:"completed,omitempty"`
	Added        time.Time  `json:"added,omitempty"`
	EffortImpact uint       `json:"effort_impact"`
	Children     []string   `json:"children,omitempty"`
	Parents      []string   `json:"parents,omitempty"`
}

func (t *Task) DetectKeyPath() []byte {
	// Is this a new active task, or just logging completed?
	var keyPath string
	var pluginID string
	if t.PluginID == "" {
		pluginID = "builtin"
	} else {
		pluginID = t.PluginID
	}
	if t.Completed == nil {
		// keyPath = fmt.Sprintf("/active/%s", t.ID)
		keyPath = filepath.Join("/active", pluginID, t.ID)
	} else {
		keyPath = filepath.Join("/completed", pluginID, t.ID)
		//keyPath = fmt.Sprintf("/completed/%s", t.ID)
	}
	return []byte(keyPath)

}

func (t *Task) SetDefaults(d *Task) {
	now := time.Now()
	if t.Added.IsZero() {
		t.Added = now
	}

	// If no ID is set, just generate one
	if t.ID == "" {
		t.ID = fmt.Sprintf(uuid.New().String())
	}

	// Set the plugin id to default if not set
	if t.PluginID == "" {
		t.PluginID = "builtin"
	}
	// Handle defaults due
	if d != nil {
		if t.Due == nil && d.Due != nil {
			t.Due = d.Due
		}
	}

}

func (t *Task) ShortID() string {
	if len(t.ID) > 5 {
		return t.ID[0:5]
	} else {
		return t.ID
	}
}

type TaskValidateOpts struct {
	IsExisting bool
}

type TaskService interface {
	// This should replace New, and Log
	Add(t, d *Task) (*Task, error)
	AddSet(t []Task, d *Task) error

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
	List(prefix string) ([]Task, error)
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

func (svc *TaskServiceOp) BucketName() string {
	return fmt.Sprintf("/%v/tasks", svc.localClient.Namespace)
}
func (svc *TaskServiceOp) GetStates() []string {
	return []string{"active", "completed"}
}

func (svc *TaskServiceOp) GetStatePaths() []string {
	var r []string
	for _, item := range svc.GetStates() {
		r = append(r, filepath.Join("/", item))

	}
	return r
}

func (svc *TaskServiceOp) SyncPlugin(tp TaskPlugin) error {
	ts, err := tp.Sync()
	if err != nil {
		return err
	}
	err = svc.localClient.Task.AddOrEditSet(ts)
	if err != nil {
		return err
	}
	log.Println(ts)
	return nil
}

func (svc *TaskServiceOp) GetPlugins() (map[string]Creator, error) {
	return TaskPlugins, nil

}

func (svc *TaskServiceOp) AddParent(c, p *Task) error {

	c.Parents = append(c.Parents, p.ID)
	p.Children = append(p.Children, c.ID)

	err := svc.EditSet([]Task{*c, *p})
	if err != nil {
		return err
	}

	return nil
}

func (svc *TaskServiceOp) AddChild(p, c *Task) error {

	p.Children = append(p.Children, c.ID)
	c.Parents = append(c.Parents, p.ID)

	err := svc.EditSet([]Task{*c, *p})
	if err != nil {
		return err
	}

	return nil
}

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

	err := svc.localClient.Task.AddSet(addSet, nil)
	if err != nil {
		return err
	}
	err = svc.localClient.Task.EditSet(editSet)
	if err != nil {
		return err
	}

	return nil
}

func (svc *TaskServiceOp) EditSet(tasks []Task) error {
	// We need to merge the new value with the old
	var mergedTasks []Task

	for _, t := range tasks {
		originalTask, err := svc.GetWithExactPath(t.DetectKeyPath())
		if err != nil {
			return errors.New("Cannot edit a task that did not previously exist: " + t.ID)
		}

		// Right now we wanna use the Complete function to do this, not edit...at least yet
		if originalTask.Completed != t.Completed {
			return errors.New("Editing the Completed field is not yet supported as it changes the path")
		}

		err = svc.Validate(&t, &TaskValidateOpts{IsExisting: true})
		if err != nil {
			return err
		}

		// Now do the merge
		if err != nil {
			return err
		}
		//var mergedTask Task
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
			log.Warning(t)
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

func (svc *TaskServiceOp) Delete(t *Task) error {
	originalTask, err := svc.GetWithExactPath(t.DetectKeyPath())
	if err != nil {
		return errors.New("Cannot delete a task that did not previously exist: " + t.ID)
	}

	err = svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		err = b.Delete(originalTask.DetectKeyPath())
		if err != nil {
			return err
		}
		return nil
	})

	return nil
}

func (svc *TaskServiceOp) Edit(t *Task) (*Task, error) {
	//originalTask, err := svc.GetByID(t.ID)
	originalTask, err := svc.GetWithExactPath(t.DetectKeyPath())
	if err != nil {
		return nil, errors.New("Cannot edit a task that did not previously exist: " + t.ID)
	}

	// Right now we wanna use the Complete function to do this, not edit...at least yet
	if originalTask.Completed != t.Completed {
		return nil, errors.New("Editing the Completed field is not yet supported as it changes the path")
	}

	err = svc.Validate(t, &TaskValidateOpts{IsExisting: true})
	if err != nil {
		return nil, err
	}

	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	err = svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		err = b.Put(t.DetectKeyPath(), taskSerial)
		if err != nil {
			return err
		}
		return nil
	})

	return t, nil
}

func (svc *TaskServiceOp) Validate(t *Task, o *TaskValidateOpts) error {
	if t.Description == "" {
		return fmt.Errorf("Missing description for Task")
	}

	if strings.Contains(t.ID, "/") {
		return fmt.Errorf("ID Cannot contain a slash (/)")
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
		return fmt.Errorf("Self id is set in the parents, we don't do that")
	}
	if ContainsString(t.Children, t.ID) {
		return fmt.Errorf("Self id is set in the children, we don't do that")
	}

	// Make sure Parents contains no duplicates
	if !CheckUniqueStringSlice(t.Parents) {
		return fmt.Errorf("Found duplicate ids in the Parents field")
	}
	return nil
}

func (svc *TaskServiceOp) Describe(t *Task) error {
	//data := make([][]string, 0)
	now := time.Now()
	var due string
	var wait string
	var completed string
	if t.Due == nil {
		due = "-"
	} else {
		due = HumanizeDuration(t.Due.Sub(now))
	}
	if t.HideUntil == nil {
		wait = "-"
	} else {
		wait = HumanizeDuration(t.HideUntil.Sub(now))
	}
	added := HumanizeDuration(t.Added.Sub(now))
	if t.Completed == nil {
		completed = "-"
	} else {
		completed = HumanizeDuration(t.Completed.Sub(now))
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
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	return nil
}

type TaskServiceOp struct {
	localClient *LocalClient
	plugins     []TaskPlugin
}

func (svc *TaskServiceOp) GetWithPartialID(partialID, pluginID, state string) (*Task, error) {
	var possibleStates []string
	if state == "" {
		possibleStates = svc.GetStatePaths()
	} else {
		possibleStates = append(possibleStates, state)
	}
	matches := []string{}
	if pluginID == "" {
		pluginID = "builtin"
	}
	qualifiedPartialID := filepath.Join(pluginID, partialID)
	log.Warning(qualifiedPartialID)
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
		return nil, fmt.Errorf("No matches for %v found in %v", partialID, prefixes)
	} else if len(matches) > 1 {
		return nil, fmt.Errorf(
			"More than 1 match for %v found in %v, please try using more of the ID. Returned: %v", partialID, prefixes, matches)
	}
	t, err := svc.GetWithExactPath([]byte(matches[0]))
	if err != nil {
		return nil, err
	}
	return t, nil

}

func (svc *TaskServiceOp) GetWithExactPath(path []byte) (*Task, error) {
	var task Task
	err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		taskBytes := b.Get([]byte(fmt.Sprintf("%s", path)))
		if taskBytes == nil {
			return fmt.Errorf("Could not find task: %s", path)
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

func (svc *TaskServiceOp) GetWithID(id, pluginID, state string) (*Task, error) {
	var possibleKeypaths []string
	if pluginID == "" {
		pluginID = "builtin"
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
	return nil, fmt.Errorf("Could not find that task at any of %v", possibleKeypaths)

}

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
		b.Delete(activePath)
		return nil
	})

	if err != nil {
		return err
	}
	return nil

}

func (svc *TaskServiceOp) List(prefix string) ([]Task, error) {
	var tasks []Task
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

// Shortcut utility to add a new task, but a completed time of 'now'
func (svc *TaskServiceOp) Log(t *Task, d *Task) (*Task, error) {
	if t.Completed == nil {
		n := time.Now()
		t.Completed = &n
	}
	ret, err := svc.Add(t, d)
	if err != nil {
		return nil, err
	}
	return ret, nil

}
func (svc *TaskServiceOp) AddSet(t []Task, d *Task) error {
	for _, task := range t {
		_, err := svc.Add(&task, d)
		if err != nil {
			log.Debug("Error adding task in set", t)
			return err
		}
	}
	//return errors.New("Thing")
	return nil
}

func (svc *TaskServiceOp) Add(t, d *Task) (*Task, error) {
	// t is the new task
	// d are the defaults
	// Detect the path based on if t.Completed has been used

	t.SetDefaults(d)

	// Validate that Task is actually good
	err := svc.Validate(t, &TaskValidateOpts{IsExisting: false})
	if err != nil {
		return nil, err
	}
	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	keyPath := t.DetectKeyPath()
	err = svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		err = bucket.Put(keyPath, taskSerial)
		if err != nil {
			return err
		}
		return nil
	})

	return t, nil

}

/*func (svc *TaskServiceOp) New(t *Task, d *Task) (*Task, error) {
	// t is the new task
	// d are the defaults
	if t.Description == "" {
		return nil, fmt.Errorf("Missing description for Task")
	}
	now := time.Now()
	if t.Added.IsZero() {
		t.Added = now
	}

	// If no ID is set, just generate one
	if t.ID == "" {
		t.ID = fmt.Sprintf(uuid.New().String())
	}

	// Handle defaults due
	if d != nil {
		if t.Due == nil && d.Due != nil {
			t.Due = d.Due
		}
	}
	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	// Is this a new active task, or just logging completed?
	var keyPath string
	if t.Completed == nil {
		keyPath = fmt.Sprintf("/active/%s", t.ID)
	} else {
		keyPath = fmt.Sprintf("/completed/%s", t.ID)
	}
	err = svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(svc.localClient.Task.BucketName()))
		err = bucket.Put([]byte(keyPath), taskSerial)
		if err != nil {
			return err
		}
		return nil
	})

	return t, nil

}
*/

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
