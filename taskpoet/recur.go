package taskpoet

import (
	"fmt"
	"os"
	"time"
)

// RecurringTask is a task that recurs
type RecurringTask struct {
	Description string        `yaml:"description"`
	Frequency   time.Duration `yaml:"frequency"`
}

// RecurringTasks represents multiple RecurringTask items
type RecurringTasks []RecurringTask

func (p *Poet) checkRecurring() error {
	now := time.Now()
	for _, recur := range p.RecurringTasks {
		needsCreate := true
		for _, task := range p.MustList("/completed") {
			earliest := now.Add(recur.Frequency)
			fmt.Fprintf(os.Stderr, "NOW THEN: %v %v\n", now, earliest)
			if (task.Description == recur.Description) && task.Completed.After(now.Add(-recur.Frequency)) {
				needsCreate = false
				break
			}
		}

		if needsCreate {
			if _, err := p.Task.Add(MustNewTask(WithDescription(recur.Description))); err != nil {
				return err
			}
		}
	}
	return nil
}
