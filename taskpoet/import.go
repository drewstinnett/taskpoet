package taskpoet

import (
	"strings"
	"time"

	"github.com/charmbracelet/log"
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
func (p *Poet) ImportTaskWarrior(ts TaskWarriorTasks) (int, error) {
	// total := len(ts)
	var imported int
	// Erase the defaults
	p.Default = Task{}
	for _, twItem := range ts {
		if twItem.Status == "deleted" {
			log.Warn("skipping deleted task since we don't really do deleted in the poet thing", "description", twItem.Description)
			continue
		}
		if twItem.Mask != "" {
			log.Warn("skipping item with a recursion mask since we don't really do that yet", "description", twItem.Description)
			continue
		}
		t := &Task{
			Description: twItem.Description,
			ID:          twItem.UUID,
			Tags:        twItem.Tags,
			Due:         (*time.Time)(twItem.Due),
			Completed:   (*time.Time)(twItem.End),
			HideUntil:   (*time.Time)(twItem.Wait),
			Reviewed:    (*time.Time)(twItem.Reviewed),
			CancelAfter: (*time.Time)(twItem.Until),
		}
		if twItem.Entry == nil {
			t.Added = time.Now()
		} else {
			t.Added = time.Time(*twItem.Entry)
		}
		if twItem.Annotations != nil {
			t.Comments = make([]Comment, len(twItem.Annotations))
			for idx, a := range twItem.Annotations {
				t.Comments[idx].Comment = a.Description
				t.Comments[idx].Added = time.Time(*a.Entry)
			}
		}
		if (t.HideUntil != nil) && (t.Due != nil) && t.HideUntil.After(*t.Due) {
			log.Warn("importing task: Due was after HideUntil, so we tweaked that")
			nh := t.Due.Add(-1 * time.Minute)
			t.HideUntil = &nh
			// twItem.Due = twItem.Wait + (1 * time.Minute)
		}

		_, err := p.Task.Add(t)
		if err != nil {
			log.With("err", err).Warn("error importing task", "task", twItem.Description)
		} else {
			imported++
		}
	}
	return imported, nil
}
