package taskpoet

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

type Task struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Due         time.Time `json:"due,omitempty"`
	HideUntil   time.Time `json:"hide_until,omitempty"`
	Completed   time.Time `json:"completed,omitempty"`
	Added       time.Time `json:"added,omitempty"`
}

type TaskService interface {
	New(t, d *Task) (*Task, error)
	Log(t, d *Task) (*Task, error)
	List(prefix string) ([]Task, error)
	Complete(t *Task) error
	// Operations by ID (/$prefix/$id)
	GetByExactID(id string, prefix string) (*Task, error)
	GetByPartialID(partialID string, prefix string) (*Task, error)
	GetIDsByPrefix(prefix string) ([]string, error)
}
type TaskServiceOp struct {
	localClient *LocalClient
}

func (svc *TaskServiceOp) GetByPartialID(partialID string, prefix string) (*Task, error) {
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

func (svc *TaskServiceOp) GetByExactID(id string, prefix string) (*Task, error) {
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
	err := svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		t.Completed = now
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
func (svc *TaskServiceOp) Log(t *Task, d *Task) (*Task, error) {
	// t is the new task
	// d are the defaults
	if t.Description == "" {
		return nil, fmt.Errorf("Missing description for Task")
	}
	now := time.Now()
	if t.Added.IsZero() {
		t.Added = now
	}

	t.ID = fmt.Sprintf(uuid.New().String())
	t.Completed = now
	taskSerial, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	err = svc.localClient.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tasks"))
		key := fmt.Sprintf("/completed/%s", t.ID)
		err = bucket.Put([]byte(key), taskSerial)
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

	// Handle defaults due
	if d != nil {
		if t.Due.IsZero() && !d.Due.IsZero() {
			t.Due = d.Due
		}
	}
	t.ID = fmt.Sprintf(uuid.New().String())
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

func FilterAllTasks(t Task) bool {
	return true
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

func QueryTasks(localClient *LocalClient, queryFn func(t Task) bool, limit int) ([]Task, error) {
	var results []Task
	var err error

	err = localClient.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		if err != nil {
			return err
		}
		err = b.ForEach(func(k, v []byte) error {
			var task Task
			err = json.Unmarshal(v, &task)
			if err != nil {
				return err
			}
			//log.Println(task)
			results = append(results, task)
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

	return results, nil
}
