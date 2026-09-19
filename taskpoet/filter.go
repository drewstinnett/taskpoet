package taskpoet

import (
	"regexp"
	"strings"
	"time"
)

// Filter is a filter function applied to a single task
type Filter func(*FilterParams, Task) bool

// FilterParams are options for filtering tasks
type FilterParams struct {
	Regex   *regexp.Regexp
	Project string
	Tag     string
	Limit   int
	// Now is the time used for wait checks, time.Now() when unset
	Now time.Time
}

func (p *FilterParams) now() time.Time {
	if p == nil || p.Now.IsZero() {
		return time.Now()
	}
	return p.Now
}

// FilterHidden removes items that are still waiting
func FilterHidden(p *FilterParams, task Task) bool {
	return !task.IsWaiting(p.now())
}

// FilterRegex removes items not matching a given regex
func FilterRegex(p *FilterParams, task Task) bool {
	if p == nil || p.Regex == nil {
		return true
	}
	return p.Regex.MatchString(task.Description)
}

// FilterProject removes items outside of a project. Like Taskwarrior, the
// project 'home' also matches 'home.kitchen'.
func FilterProject(p *FilterParams, task Task) bool {
	if p == nil || p.Project == "" {
		return true
	}
	return task.Project == p.Project || strings.HasPrefix(task.Project, p.Project+".")
}

// FilterTag removes items that don't have a tag
func FilterTag(p *FilterParams, task Task) bool {
	if p == nil || p.Tag == "" {
		return true
	}
	return containsString(task.Tags, p.Tag)
}

// ApplyFilters applies a set of filters to a task list.
// Each record will be checked against each filter.
// The filters are applied in the order they are passed in.
func ApplyFilters(tasks Tasks, p *FilterParams, filters ...Filter) Tasks {
	// Make sure there are actually filters to be applied.
	if len(filters) == 0 {
		return tasks
	}

	filteredRecords := make(Tasks, 0, len(tasks))

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for _, r := range tasks {
		keep := true

		for _, f := range filters {
			if !f(p, *r) {
				keep = false
				break
			}
		}

		if keep {
			filteredRecords = append(filteredRecords, r)
		}
	}

	return filteredRecords
}

// ByDue is the by due date sorter
type ByDue Tasks

func (a ByDue) Len() int { return len(a) }
func (a ByDue) Less(i, j int) bool {
	if a[i].Due == nil {
		return false
	}
	if a[j].Due == nil {
		return true
	}
	return !a[j].Due.Before(*a[i].Due)
}
func (a ByDue) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// ByUrgency sorts using the Urgency field
type ByUrgency Tasks

func (a ByUrgency) Len() int { return len(a) }
func (a ByUrgency) Less(i, j int) bool {
	return a[i].Urgency > a[j].Urgency
}
func (a ByUrgency) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// ByCompleted is the by completed date sorter, most recent first
type ByCompleted Tasks

func (a ByCompleted) Len() int { return len(a) }
func (a ByCompleted) Less(i, j int) bool {
	if a[i].End == nil {
		return false
	}
	if a[j].End == nil {
		return true
	}
	return a[j].End.Before(*a[i].End)
}
func (a ByCompleted) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
