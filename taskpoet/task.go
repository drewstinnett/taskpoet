package taskpoet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var prefixes []string

type Task struct {
	ID           string    `json:"id"`
	Description  string    `json:"description"`
	Due          time.Time `json:"due,omitempty"`
	HideUntil    time.Time `json:"hide_until,omitempty"`
	Completed    time.Time `json:"completed,omitempty"`
	Added        time.Time `json:"added,omitempty"`
	EffortImpact uint      `json:"effort_impact,omitempty"`
	Children     []string  `json:"children,omitempty"`
	Parents      []string  `json:"parents,omitempty"`
}

type TaskValidateOpts struct {
	IsExisting bool
}

type TaskService interface {
	// This should replace New, and Log
	Add(t, d *Task) (*Task, error)
	AddSet(t []Task, d *Task) error

	// Helper for adding Parent
	AddParent(c, p *Task) error
	AddChild(p, c *Task) error

	// Describe something
	Describe(t *Task) error

	// Edit a Task entry
	Edit(t *Task) (*Task, error)
	EditSet(t []Task) error

	// Check to ensure Task is in a valid state
	Validate(t *Task, o *TaskValidateOpts) error

	// Just a little helper function to add and immediately mark as completed
	Log(t, d *Task) (*Task, error)

	List(prefix string) ([]Task, error)
	Complete(t *Task) error
	// Operations by ID (/$prefix/$id)
	GetByID(id string) (*Task, error)
	GetByIDWithPrefix(id string, prefix string) (*Task, error)
	GetByPartialID(partialID string) (*Task, error)
	GetByPartialIDWithPath(partialID string, prefix string) (*Task, error)
	GetIDsByPrefix(prefix string) ([]string, error)
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

func (t *Task) DetectKeyPath() []byte {
	// Is this a new active task, or just logging completed?
	var keyPath string
	if t.Completed.IsZero() {
		keyPath = fmt.Sprintf("/active/%s", t.ID)
	} else {
		keyPath = fmt.Sprintf("/completed/%s", t.ID)
	}
	return []byte(keyPath)

}

func (svc *TaskServiceOp) EditSet(tasks []Task) error {
	for _, t := range tasks {
		originalTask, err := svc.GetByID(t.ID)
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
	}

	err := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		for _, t := range tasks {
			taskSerial, err := json.Marshal(t)
			if err != nil {
				return err
			}
			b := tx.Bucket([]byte("tasks"))
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

func (svc *TaskServiceOp) Edit(t *Task) (*Task, error) {
	originalTask, err := svc.GetByID(t.ID)
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
		b := tx.Bucket([]byte("tasks"))
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

	// If not specified
	if o == nil {
		o = &TaskValidateOpts{}
	}

	// IF NEW, Make sure this ID doesn't already exist
	if !o.IsExisting {
		_, err := svc.GetByID(t.ID)
		if err == nil {
			return fmt.Errorf("Task with ID %v already exists", t.ID)
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

func (t *Task) ShortID() string {
	if len(t.ID) > 5 {
		return t.ID[0:5]
	} else {
		return t.ID
	}
}

func (svc *TaskServiceOp) Describe(t *Task) error {
	//data := make([][]string, 0)
	now := time.Now()
	due := HumanizeDuration(t.Due.Sub(now))
	added := HumanizeDuration(t.Added.Sub(now))
	completed := HumanizeDuration(t.Completed.Sub(now))
	var parentsBuff bytes.Buffer
	var childrenBuff bytes.Buffer
	for _, p := range t.Parents {
		parent, err := svc.GetByID(p)
		if err != nil {
			return nil
		}
		parentsBuff.Write([]byte(parent.Description))
	}
	for _, c := range t.Children {
		child, err := svc.GetByID(c)
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
		{"Effort/Impact", EffortImpactText(int(t.EffortImpact)), fmt.Sprintf("%+v", t.EffortImpact)},
		{"Parents", parentsBuff.String(), fmt.Sprintf("%+v", t.Parents)},
		{"Children", childrenBuff.String(), fmt.Sprintf("%+v", t.Children)},
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	return nil
}

type TaskServiceOp struct {
	localClient *LocalClient
}

func (svc *TaskServiceOp) GetByPartialID(partialID string) (*Task, error) {
	prefixes := []string{"/active", "/completed"}
	matches := []string{}
	for _, prefix := range prefixes {
		ids, err := svc.GetIDsByPrefix(prefix)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if strings.HasPrefix(id, fmt.Sprintf("%v/%v", prefix, partialID)) {
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
	t, err := svc.GetByExactPath(matches[0])
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (svc *TaskServiceOp) GetByPartialIDWithPath(partialID string, prefix string) (*Task, error) {
	ids, err := svc.GetIDsByPrefix(prefix)
	if err != nil {
		return nil, err
	}
	matches := []string{}
	for _, id := range ids {
		if strings.HasPrefix(id, fmt.Sprintf("%v/%v", prefix, partialID)) {
			matches = append(matches, id)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("No matches for %v found in %v", partialID, prefix)
	} else if len(matches) > 1 {
		return nil, fmt.Errorf(
			"More than 1 match for %v found in %v, please try using more of the ID. Returned: %v", partialID, prefix, matches)
	}
	t, err := svc.GetByExactPath(matches[0])
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (svc *TaskServiceOp) GetByExactPath(path string) (*Task, error) {
	var task Task
	err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
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

func (svc *TaskServiceOp) GetByID(id string) (*Task, error) {
	prefixes := []string{"/active", "/completed"}
	var task *Task
	var err error
	for _, prefix := range prefixes {
		task, err = svc.GetByIDWithPrefix(id, prefix)
		if err != nil {
			log.Debugf("No task with id %v in %v", id, prefix)
		} else {
			log.Debugf("Found task %v %v", prefix, id)
			return task, nil
		}
	}

	return nil, fmt.Errorf("No task found in %v", prefixes)
}

func (svc *TaskServiceOp) GetByIDWithPrefix(id string, prefix string) (*Task, error) {
	var realPrefix string
	if prefix == "" {
		realPrefix = "/active"
	} else {
		realPrefix = prefix
	}

	var task Task
	err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		taskBytes := b.Get([]byte(fmt.Sprintf("%s/%s", realPrefix, id)))
		if taskBytes == nil {
			return fmt.Errorf("Could not find task: %s/%s", realPrefix, id)
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

func (svc *TaskServiceOp) Complete(t *Task) error {
	now := time.Now()
	t.Completed = now
	err := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		taskSerial, err := json.Marshal(t)
		if err != nil {
			return err
		}
		err = b.Put([]byte(fmt.Sprintf("/completed/%s", t.ID)), taskSerial)
		if err != nil {
			return err
		}
		b.Delete([]byte(fmt.Sprintf("/active/%s", t.ID)))
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
		b := tx.Bucket([]byte("tasks"))
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
	if t.Completed.IsZero() {
		t.Completed = time.Now()
	}
	ret, err := svc.Add(t, d)
	if err != nil {
		return nil, err
	}
	return ret, nil

}
func (svc *TaskServiceOp) AddSet(t []Task, d *Task) error {
	for _, task := range t {
		_, err := svc.New(&task, d)
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
	now := time.Now()
	if t.Added.IsZero() {
		t.Added = now
	}

	// Handle defaults due
	if d != nil {
		if t.Due.IsZero() && !d.Due.IsZero() {
			t.Due = d.Due
		}
	}
	// If no ID is set, just generate one
	if t.ID == "" {
		t.ID = fmt.Sprintf(uuid.New().String())
	}

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
		bucket := tx.Bucket([]byte("tasks"))
		err = bucket.Put(keyPath, taskSerial)
		if err != nil {
			return err
		}
		return nil
	})

	return t, nil

}

func (svc *TaskServiceOp) New(t *Task, d *Task) (*Task, error) {
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
		if t.Due.IsZero() && !d.Due.IsZero() {
			t.Due = d.Due
		}
	}
	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	// Is this a new active task, or just logging completed?
	var keyPath string
	if t.Completed.IsZero() {
		keyPath = fmt.Sprintf("/active/%s", t.ID)
	} else {
		keyPath = fmt.Sprintf("/completed/%s", t.ID)
	}
	err = svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tasks"))
		err = bucket.Put([]byte(keyPath), taskSerial)
		if err != nil {
			return err
		}
		return nil
	})

	return t, nil

}

func (svc *TaskServiceOp) GetIDsByPrefix(prefix string) ([]string, error) {
	allIDs := []string{}

	err := svc.localClient.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tasks"))
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
