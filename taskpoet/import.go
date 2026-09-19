package taskpoet

import (
	"fmt"
	"strings"

	bolt "go.etcd.io/bbolt"
)

// ImportOptions controls what happens to tasks that are already stored
type ImportOptions struct {
	// Overwrite replaces tasks that already exist (matched by UUID) instead of
	// leaving them alone
	Overwrite bool
	// DryRun works out the report without writing anything
	DryRun bool
}

// ImportReport says what an import did, or would do for a dry run
type ImportReport struct {
	Read        int
	Imported    int
	Overwritten int
	Skipped     int
	// ByStatus counts the tasks that were read, by status
	ByStatus map[Status]int
	Warnings []string
	DryRun   bool
}

// ImportTasks stores a set of tasks in a single transaction: either all of
// them are written, or none. It is safe to run again on the same input,
// tasks are matched by UUID.
func (s *Store) ImportTasks(ts Tasks, opts ImportOptions) (*ImportReport, error) {
	rep := &ImportReport{
		Read:     len(ts),
		ByStatus: map[Status]int{},
		DryRun:   opts.DryRun,
	}

	run := func(tx *bolt.Tx, write bool) error {
		known := map[string]*Task{}
		if err := s.bucket(tx, bucketTasks).ForEach(func(_, v []byte) error {
			t, err := decodeTask(v)
			if err != nil {
				return err
			}
			known[t.UUID] = t
			return nil
		}); err != nil {
			return err
		}

		applied := make(Tasks, 0, len(ts))
		for _, t := range ts {
			rep.ByStatus[t.Status]++
			if err := t.Validate(); err != nil {
				return fmt.Errorf("task %v: %w", t.UUID, err)
			}
			old, exists := known[t.UUID]
			switch {
			case exists && !opts.Overwrite:
				rep.Skipped++
				continue
			case exists:
				rep.Overwritten++
				// This isn't in Taskwarrior, so don't lose it
				if t.EffortImpact == EffortImpactUnset {
					t.EffortImpact = old.EffortImpact
				}
			default:
				rep.Imported++
			}
			if write {
				if err := s.putTask(tx, t); err != nil {
					return fmt.Errorf("task %v: %w", t.UUID, err)
				}
			}
			known[t.UUID] = t
			applied = append(applied, t)
		}
		rep.Warnings = append(rep.Warnings, integrityWarnings(known, applied)...)
		return nil
	}

	var err error
	if opts.DryRun {
		err = s.db.View(func(tx *bolt.Tx) error { return run(tx, false) })
	} else {
		err = s.db.Update(func(tx *bolt.Tx) error { return run(tx, true) })
	}
	if err != nil {
		return nil, err
	}
	return rep, nil
}

const maxExamples = 5

func examples(l []string) string {
	if len(l) > maxExamples {
		return strings.Join(l[:maxExamples], ", ") + fmt.Sprintf(", and %d more", len(l)-maxExamples)
	}
	return strings.Join(l, ", ")
}

func shortID(id string) string {
	return id[0:min(len(id), shortIDLen)]
}

// integrityWarnings looks at the freshly applied tasks in the context of everything
// known, and reports references that don't hold up. None of these stop an
// import, Taskwarrior itself allows them.
func integrityWarnings(known map[string]*Task, applied Tasks) []string {
	var warnings []string

	var dangling, orphans, ruleless []string
	for _, t := range applied {
		for _, d := range t.Depends {
			if _, ok := known[d]; !ok {
				dangling = append(dangling, fmt.Sprintf("%v -> %v", shortID(t.UUID), shortID(d)))
				break
			}
		}
		if t.Parent != "" {
			if _, ok := known[t.Parent]; !ok {
				orphans = append(orphans, shortID(t.UUID))
			}
		}
		if t.Status == StatusRecurring && t.Recur == "" {
			ruleless = append(ruleless, shortID(t.UUID))
		}
	}
	if len(dangling) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d tasks depend on tasks that are not in the database (kept as they are): %v", len(dangling), examples(dangling)))
	}
	if len(orphans) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d recurring instances point at a parent that is not in the database (kept as they are): %v", len(orphans), examples(orphans)))
	}
	if len(ruleless) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d recurring templates have no 'recur' rule and will never spawn anything: %v", len(ruleless), examples(ruleless)))
	}

	// Depth first walk to find dependency cycles
	const (
		unvisited = iota
		visiting
		done
	)
	state := map[string]int{}
	var cycles []string
	var visit func(id string, path []string)
	visit = func(id string, path []string) {
		state[id] = visiting
		path = append(path, id)
		for _, d := range known[id].Depends {
			if _, ok := known[d]; !ok {
				continue
			}
			switch state[d] {
			case unvisited:
				visit(d, path)
			case visiting:
				var loop []string
				for i, p := range path {
					if p == d {
						for _, q := range path[i:] {
							loop = append(loop, shortID(q))
						}
						break
					}
				}
				loop = append(loop, shortID(d))
				cycles = append(cycles, strings.Join(loop, " -> "))
			}
		}
		state[id] = done
	}
	for _, t := range applied {
		if state[t.UUID] == unvisited {
			visit(t.UUID, nil)
		}
	}
	if len(cycles) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d dependency cycles, none of these tasks can ever be unblocked: %v", len(cycles), examples(cycles)))
	}
	return warnings
}
