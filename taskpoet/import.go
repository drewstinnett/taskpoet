package taskpoet

import (
	"log"
	"strings"
	"time"
)

// TWTime is the format that TaskWarrior uses for timestamps
type TWTime time.Time

const twTimeLayout = "20060102T150405Z"

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
	ID          int64    `json:"id,omitempty"`
	Description string   `json:"description,omitempty"`
	UUID        string   `json:"uuid,omitempty"`
	Status      string   `json:"status,omitempty"`
	Entry       *TWTime  `json:"entry,omitempty"`
	Modified    *TWTime  `json:"modified,omitempty"`
	Due         *TWTime  `json:"due,omitempty"`
	Wait        *TWTime  `json:"wait,omitempty"`
	End         *TWTime  `json:"end,omitempty"`
	Urgency     float64  `json:"urgency,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// TaskWarriorTasks is multiple TaskWarriorTasks items
type TaskWarriorTasks []TaskWarriorTask

// ImportTaskWarrior imports a set of TaskWarrior items and returns the number
// it imported, and an optional error
func (p *Poet) ImportTaskWarrior(ts TaskWarriorTasks) (int, error) {
	// total := len(ts)
	var imported int
	for _, twItem := range ts {
		t := &Task{
			Description: twItem.Description,
			ID:          twItem.UUID,
			Tags:        twItem.Tags,
		}
		if twItem.Due != nil {
			t.Due = (*time.Time)(twItem.Due)
		}
		if twItem.End != nil {
			t.Completed = (*time.Time)(twItem.End)
		}
		if twItem.Wait != nil {
			t.HideUntil = (*time.Time)(twItem.Wait)
		}
		_, err := p.Task.Add(t)
		if err != nil {
			log.Printf("error importing task: %v", err)
		} else {
			imported++
		}
	}
	return imported, nil
}
