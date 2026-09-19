package taskpoet

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

/*
Recurrence works like Taskwarrior's. A task with status 'recurring' is a
template: it has a Recur rule and a Due date. It is never worked on itself, it
spawns pending instances that point back at it with Parent and IMask (the
index of the period the instance is for).

Periodic templates (the default) have their due dates fixed by the template:
instance n is due at template.Due plus n periods. Chained templates only ever
have one instance, and the next one is due a period after the last was
completed.
*/

// RTypeChained is the rtype of a chained template. Anything else is periodic.
const RTypeChained = "chained"

// Recurrence is a parsed recur rule, like 'weekly' or '3months'
type Recurrence struct {
	Months   int
	Days     int
	Dur      time.Duration // hours, minutes and seconds
	Weekdays bool          // every weekday, skipping the weekend
}

var recurUnit = map[string]func(n int) Recurrence{}

func init() {
	unit := func(f func(n int) Recurrence, names ...string) {
		for _, n := range names {
			recurUnit[n] = f
		}
	}
	unit(func(n int) Recurrence { return Recurrence{Dur: time.Duration(n) * time.Second} }, "s", "sec", "secs", "second", "seconds")
	unit(func(n int) Recurrence { return Recurrence{Dur: time.Duration(n) * time.Minute} }, "min", "mins", "minute", "minutes")
	unit(func(n int) Recurrence { return Recurrence{Dur: time.Duration(n) * time.Hour} }, "h", "hr", "hrs", "hour", "hours")
	unit(func(n int) Recurrence { return Recurrence{Days: n} }, "d", "day", "days")
	unit(func(n int) Recurrence { return Recurrence{Days: 7 * n} }, "w", "wk", "wks", "week", "weeks")
	// Taskwarrior has no bare 'm' for months, but nobody recurs every N minutes with it
	unit(func(n int) Recurrence { return Recurrence{Months: n} }, "m", "mo", "mos", "mth", "mths", "month", "months")
	unit(func(n int) Recurrence { return Recurrence{Months: 3 * n} }, "q", "qtr", "qtrs", "quarter", "quarters")
	unit(func(n int) Recurrence { return Recurrence{Months: 12 * n} }, "y", "yr", "yrs", "year", "years")
}

var recurNamed = map[string]Recurrence{
	"daily":      {Days: 1},
	"weekdays":   {Weekdays: true},
	"weekly":     {Days: 7},
	"biweekly":   {Days: 14},
	"fortnight":  {Days: 14},
	"monthly":    {Months: 1},
	"bimonthly":  {Months: 2},
	"quarterly":  {Months: 3},
	"semiannual": {Months: 6},
	"annual":     {Months: 12},
	"yearly":     {Months: 12},
	"biannual":   {Months: 24},
	"biyearly":   {Months: 24},
}

var recurCount = regexp.MustCompile(`^(\d+)\s*([a-z]+)$`)

// ParseRecurrence understands the Taskwarrior recur values: names like
// 'weekly' or 'quarterly', and counts like '3d', '2w', '6months' or '1y'
func ParseRecurrence(s string) (Recurrence, error) {
	in := strings.ToLower(strings.TrimSpace(s))
	if r, ok := recurNamed[in]; ok {
		return r, nil
	}
	if m := recurCount.FindStringSubmatch(in); m != nil {
		n, err := strconv.Atoi(m[1])
		if err == nil && n > 0 {
			if f, ok := recurUnit[m[2]]; ok {
				return f(n), nil
			}
		}
	}
	return Recurrence{}, fmt.Errorf("unknown recurrence %q, use something like daily, weekdays, weekly, monthly, quarterly, yearly, 3d, 2w or 6months", s)
}

// Add returns t moved forward by n periods. The result is always worked out
// from t itself, never from an earlier result, so month ends don't drift: the
// 31st plus 1 month is the 28th or 29th of February, but plus 2 months is
// the 31st of March again.
func (r Recurrence) Add(t time.Time, n int) time.Time {
	if r.Weekdays {
		for i := 0; i < n; i++ {
			t = t.AddDate(0, 0, 1)
			for t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
				t = t.AddDate(0, 0, 1)
			}
		}
		return t
	}
	if r.Months != 0 {
		t = addMonthsClamped(t, r.Months*n)
	}
	if r.Days != 0 {
		t = t.AddDate(0, 0, r.Days*n)
	}
	return t.Add(time.Duration(n) * r.Dur)
}

// addMonthsClamped adds calendar months, and when the day doesn't exist in the
// target month uses its last day instead
func addMonthsClamped(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	total := int(m) - 1 + months
	ny, nm := y+total/12, total%12
	if nm < 0 {
		nm += 12
		ny--
	}
	last := time.Date(ny, time.Month(nm+1)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(ny, time.Month(nm+1), min(d, last), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// CatchUp says what to do about periods that went by while nothing was
// looking, like a daily task on a laptop that was shut for a month
type CatchUp string

const (
	// CatchUpLatest only creates the most recent missed instance
	CatchUpLatest CatchUp = "latest"
	// CatchUpAll creates every missed instance, like Taskwarrior does
	CatchUpAll CatchUp = "all"
)

// ParseCatchUp reads a catch up setting, empty means the default
func ParseCatchUp(s string) (CatchUp, error) {
	switch c := CatchUp(strings.ToLower(strings.TrimSpace(s))); c {
	case "", CatchUpLatest:
		return CatchUpLatest, nil
	case CatchUpAll:
		return c, nil
	}
	return "", fmt.Errorf("unknown catch up mode %q, expected latest or all", s)
}

const (
	// maxCatchUp stops a template with a long history from creating thousands
	// of tasks in one go
	maxCatchUp = 1000
	// maxPeriods stops a runaway walk along a template's periods
	maxPeriods = 100000
)

// SpawnOptions controls SpawnRecurring
type SpawnOptions struct {
	// Now defaults to the store's clock
	Now time.Time
	// Location is where calendar arithmetic happens, so 'weekly at 9am' stays
	// at 9am across daylight saving. Defaults to time.Local.
	Location *time.Location
	// Limit is how many future instances to have ready, beyond the ones that
	// are due already
	Limit   int
	CatchUp CatchUp
	// DryRun reports what would be created, without creating it
	DryRun bool
}

// SpawnReport says what was, or would be, created
type SpawnReport struct {
	Created  Tasks
	Warnings []string
}

// SpawnRecurring creates the instances that recurrence templates are due to
// produce. It is safe to run as often as you like, when nothing is due it
// changes nothing.
func (s *Store) SpawnRecurring(opts SpawnOptions) (*SpawnReport, error) {
	if opts.Now.IsZero() {
		opts.Now = s.now()
	}
	if opts.Location == nil {
		opts.Location = time.Local
	}
	if opts.CatchUp == "" {
		opts.CatchUp = CatchUpLatest
	}

	// Look first with a read only transaction, so the common case of nothing
	// to do doesn't cost a write
	rep, err := s.spawnRun(opts, false)
	if err != nil || opts.DryRun || len(rep.Created) == 0 {
		return rep, err
	}
	return s.spawnRun(opts, true)
}

func (s *Store) spawnRun(opts SpawnOptions, write bool) (*SpawnReport, error) {
	rep := &SpawnReport{}
	run := func(tx *bolt.Tx) error {
		templates, err := s.indexTasks(tx, bucketIdxStatus, string(StatusRecurring))
		if err != nil {
			return err
		}
		sort.Slice(templates, func(i, j int) bool { return templates[i].UUID < templates[j].UUID })
		for _, tpl := range templates {
			created, err := s.spawnTemplate(tx, tpl, opts, rep, write)
			if err != nil {
				return err
			}
			rep.Created = append(rep.Created, created...)
		}
		return nil
	}
	var err error
	if write {
		err = s.db.Update(run)
	} else {
		err = s.db.View(run)
	}
	return rep, err
}

func (s *Store) spawnTemplate(tx *bolt.Tx, tpl *Task, opts SpawnOptions, rep *SpawnReport, write bool) ([]*Task, error) {
	warn := func(format string, a ...any) {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("recurring task %v (%v): ", tpl.ShortID(), tpl.Description)+fmt.Sprintf(format, a...))
	}
	if tpl.Recur == "" {
		warn("has no recur rule, nothing to spawn")
		return nil, nil
	}
	rule, err := ParseRecurrence(tpl.Recur)
	if err != nil {
		warn("%v", err)
		return nil, nil
	}
	if tpl.Due == nil {
		warn("has no due date, nothing to spawn")
		return nil, nil
	}
	insts, err := s.indexTasks(tx, bucketIdxParent, tpl.UUID)
	if err != nil {
		return nil, err
	}

	var indexes []int
	if strings.EqualFold(tpl.RType, RTypeChained) {
		indexes, err = chainedNext(tpl, rule, insts, opts)
		if err != nil {
			warn("%v", err)
		}
	} else {
		var capped bool
		if indexes, capped = periodicNext(tpl, rule, insts, opts); capped {
			warn("stopped after %d instances, the rest can wait for the next run", maxCatchUp)
		}
	}
	if len(indexes) == 0 {
		return nil, nil
	}

	created := make([]*Task, 0, len(indexes))
	for _, n := range indexes {
		due := s.instanceDue(tpl, rule, insts, n, opts)
		created = append(created, newInstance(tpl, n, due, opts.Now))
	}
	if !write {
		return created, nil
	}
	for _, inst := range created {
		if err := s.putTask(tx, inst); err != nil {
			return nil, err
		}
		tpl.Mask = padMask(tpl.Mask, *inst.IMask, 'X')
		tpl.Mask = setMaskChar(tpl.Mask, *inst.IMask, '-')
	}
	tpl.Modified = opts.Now
	return created, s.putTask(tx, tpl)
}

// instanceDue works out when instance n is due. Periodic instances are fixed
// by the template, chained ones by when the last instance was finished.
func (s *Store) instanceDue(tpl *Task, rule Recurrence, insts Tasks, n int, opts SpawnOptions) time.Time {
	if strings.EqualFold(tpl.RType, RTypeChained) {
		if last := lastInstance(insts); last != nil && last.End != nil {
			return rule.Add(last.End.In(opts.Location), 1).UTC()
		}
		return tpl.Due.UTC()
	}
	return rule.Add(tpl.Due.In(opts.Location), n).UTC()
}

// lastInstance is the instance with the highest index
func lastInstance(insts Tasks) *Task {
	var last *Task
	for _, in := range insts {
		if in.IMask != nil && (last == nil || *in.IMask > *last.IMask) {
			last = in
		}
	}
	return last
}

// chainedNext says which instance index to create next, if any. There is only
// ever one, and the next only starts once the last one was completed.
func chainedNext(tpl *Task, rule Recurrence, insts Tasks, opts SpawnOptions) ([]int, error) {
	if len(insts) == 0 {
		return []int{0}, nil
	}
	for _, in := range insts {
		if in.Status == StatusPending {
			return nil, nil
		}
	}
	last := lastInstance(insts)
	if last == nil {
		return nil, errors.New("its instances have no imask, can't tell which one is last")
	}
	if last.Status != StatusCompleted || last.End == nil {
		// Deleted, so the chain is over
		return nil, nil
	}
	if next := rule.Add(last.End.In(opts.Location), 1); tpl.Until != nil && next.After(*tpl.Until) {
		return nil, nil
	}
	return []int{*last.IMask + 1}, nil
}

// periodicNext says which instance indexes to create
func periodicNext(tpl *Task, rule Recurrence, insts Tasks, opts SpawnOptions) (indexes []int, capped bool) {
	have := map[int]bool{}
	maxHave := -1
	for _, in := range insts {
		if in.IMask != nil {
			have[*in.IMask] = true
			maxHave = max(maxHave, *in.IMask)
		}
	}
	origin := tpl.Due.In(opts.Location)

	var overdue, future []int
	futureHave := 0
	for n := 0; n < maxPeriods; n++ {
		due := rule.Add(origin, n)
		if tpl.Until != nil && due.After(*tpl.Until) {
			break
		}
		if !due.After(opts.Now) {
			if !have[n] {
				overdue = append(overdue, n)
			}
			continue
		}
		if futureHave+len(future) >= opts.Limit {
			break
		}
		if have[n] {
			futureHave++
		} else {
			future = append(future, n)
		}
	}

	switch {
	case opts.CatchUp == CatchUpAll:
		if len(overdue) > maxCatchUp {
			overdue, capped = overdue[:maxCatchUp], true
		}
	case tpl.Until != nil && tpl.Until.Before(opts.Now):
		// The series is over. Don't dig up a period from long ago.
		overdue = nil
	case len(overdue) > 0 && overdue[len(overdue)-1] > maxHave:
		overdue = overdue[len(overdue)-1:]
	default:
		// Anything missed is older than what we already have
		overdue = nil
	}
	return append(overdue, future...), capped
}

// newInstance makes the pending task for period n of a template
func newInstance(tpl *Task, n int, due, now time.Time) *Task {
	inst := &Task{
		UUID:         uuid.New().String(),
		Status:       StatusPending,
		Description:  tpl.Description,
		Project:      tpl.Project,
		Priority:     tpl.Priority,
		Tags:         append([]string(nil), tpl.Tags...),
		Entry:        now,
		Modified:     now,
		Due:          &due,
		Recur:        tpl.Recur,
		RType:        tpl.RType,
		Parent:       tpl.UUID,
		IMask:        &n,
		EffortImpact: tpl.EffortImpact,
	}
	if len(tpl.UDA) > 0 {
		inst.UDA = make(map[string]any, len(tpl.UDA))
		for k, v := range tpl.UDA {
			inst.UDA[k] = v
		}
	}
	// Keep wait and scheduled the same distance from due as they are on the template
	if tpl.Wait != nil {
		w := due.Add(-tpl.Due.Sub(*tpl.Wait))
		inst.Wait = &w
	}
	if tpl.Scheduled != nil {
		sc := due.Add(-tpl.Due.Sub(*tpl.Scheduled))
		inst.Scheduled = &sc
	}
	return inst
}

// padMask grows a mask to hold index i, filling the new places with c. Periods
// that were never spawned are filled with 'X', the same as if they were deleted.
func padMask(mask string, i int, c byte) string {
	b := []byte(mask)
	for len(b) < i {
		b = append(b, c)
	}
	return string(b)
}

// spawnChainedAfter creates the next instance of a chained template, when the
// instance that was just completed leaves it without a pending one
func (s *Store) spawnChainedAfter(tx *bolt.Tx, templateID string) error {
	tpl, err := s.getTask(tx, templateID)
	if err != nil || tpl.Status != StatusRecurring || !strings.EqualFold(tpl.RType, RTypeChained) {
		return nil //nolint:nilerr // no template, nothing to chain
	}
	_, err = s.spawnTemplate(tx, tpl, SpawnOptions{Now: s.now(), Location: time.Local}, &SpawnReport{}, true)
	return err
}

// deleteInstances deletes the pending instances of a template that was
// deleted, so they don't linger. Finished ones are history and stay.
func (s *Store) deleteInstances(tx *bolt.Tx, tpl *Task) error {
	insts, err := s.indexTasks(tx, bucketIdxParent, tpl.UUID)
	if err != nil {
		return err
	}
	now := s.now()
	for _, in := range insts {
		if in.Status != StatusPending {
			continue
		}
		in.Status = StatusDeleted
		in.End = &now
		in.Modified = now
		if err := s.putTask(tx, in); err != nil {
			return err
		}
	}
	return nil
}
