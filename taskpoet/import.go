package taskpoet

import (
	"fmt"
	"strings"
	"time"
)

// TWTime is the format that TaskWarrior uses for timestamps
type TWTime time.Time

const twTimeLayout = "20060102T150405Z"

var errExists = fmt.Errorf("task already exists")

// UnmarshalJSON Parses the json string in the custom format
func (t *TWTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), `"`)
	nt, err := time.Parse(twTimeLayout, s)
	*t = TWTime(nt)
	return
}

// MarshalJSON writes a quoted string in the custom format
func (t TWTime) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

// String returns the time in the custom format
func (t *TWTime) String() string {
	return time.Time(*t).String()
}

// TaskWarriorTask is a task from TaskWarrior
type TaskWarriorTask struct {
	ID          int64          `json:"id,omitempty"`
	Description string         `json:"description,omitempty"`
	UUID        string         `json:"uuid,omitempty"`
	Status      string         `json:"status,omitempty"`
	Entry       *TWTime        `json:"entry,omitempty"`
	Modified    *TWTime        `json:"modified,omitempty"`
	Due         *TWTime        `json:"due,omitempty"`
	Wait        *TWTime        `json:"wait,omitempty"`
	End         *TWTime        `json:"end,omitempty"`
	Reviewed    *TWTime        `json:"reviewed,omitempty"`
	Until       *TWTime        `json:"until,omitempty"`
	Mask        string         `json:"mask,omitempty"`
	Urgency     float64        `json:"urgency,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Annotations []TWAnnotation `json:"annotations,omitempty"`
}

// TWAnnotation is a TaskWarrior Annotation
type TWAnnotation struct {
	Entry       *TWTime `json:"entry,omitempty"`
	Description string  `json:"description,omitempty"`
}

// TaskWarriorTasks is multiple TaskWarriorTasks items
type TaskWarriorTasks []TaskWarriorTask

// ImportTaskWarrior imports a set of TaskWarrior items and returns the number
// it imported, and an optional error
func (p *Poet) ImportTaskWarrior(ts TaskWarriorTasks, c chan ProgressStatus) (int, error) {
	// total := len(ts)
	var imported int
	// Erase the defaults
	p.Default = Task{}
	total := len(ts)
	for idx, twItem := range ts {
		s := ProgressStatus{
			Current: int64(idx),
			Total:   int64(total),
			Info:    fmt.Sprintf("Importing: %v", twItem.Description),
		}
		if twItem.Mask != "" {
			s.Warning = fmt.Sprintf("Skipping item with recursion mask since we know how to handle it yet: %v", twItem.Description)
			pushStatus(c, s)
			continue
		}
		t := MustNewTask(WithTaskWarriorTask(twItem))

		_, err := p.Task.Add(t)
		if p.Task.Validate(t, &TaskValidateOpts{IsExisting: true}) != nil {
			s.Warning = fmt.Sprintf("Error importing task: %v (%v)", twItem.Description, err.Error())
		} else if (err == nil) || (err != nil) && (err == errExists) {
			imported++
		} else if err != nil {
			s.Warning = fmt.Sprintf("Error importing task: %v (%v)", twItem.Description, err.Error())
		}

		/*
			if _, err := p.Task.Add(t); err != nil && err != errExists {
				s.Warning = fmt.Sprintf("Error importing task: %v (%v)", twItem.Description, err.Error())
			} else {
				imported++
			}
		*/
		pushStatus(c, s)
	}
	return imported, nil
}

func pushStatus(c chan ProgressStatus, s ProgressStatus) {
	if c != nil {
		c <- s
	}
}
