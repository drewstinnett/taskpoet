package taskpoet

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 9, 0, 0, 0, time.UTC)
}

func TestParseRecurrence(t *testing.T) {
	tests := map[string]Recurrence{
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
		"Weekly":     {Days: 7},
		" daily ":    {Days: 1},
		"3d":         {Days: 3},
		"3days":      {Days: 3},
		"2w":         {Days: 14},
		"1week":      {Days: 7},
		"3months":    {Months: 3},
		"6mo":        {Months: 6},
		"2m":         {Months: 2},
		"1q":         {Months: 3},
		"2quarters":  {Months: 6},
		"1y":         {Months: 12},
		"3 years":    {Months: 36},
		"12h":        {Dur: 12 * time.Hour},
		"30min":      {Dur: 30 * time.Minute},
		"90s":        {Dur: 90 * time.Second},
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			got, err := ParseRecurrence(in)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}

	for _, bad := range []string{"", "0d", "d", "3", "3x", "-1d", "fortnightly", "every day", "1.5w"} {
		t.Run("invalid "+bad, func(t *testing.T) {
			_, err := ParseRecurrence(bad)
			require.Error(t, err)
		})
	}
}

func TestRecurrenceAdd(t *testing.T) {
	tests := map[string]struct {
		rule  string
		start time.Time
		n     int
		want  time.Time
	}{
		"daily":                  {"daily", day(2024, 3, 10), 3, day(2024, 3, 13)},
		"weekly":                 {"weekly", day(2024, 3, 10), 2, day(2024, 3, 24)},
		"zero periods":           {"weekly", day(2024, 3, 10), 0, day(2024, 3, 10)},
		"across a year":          {"daily", day(2023, 12, 30), 3, day(2024, 1, 2)},
		"monthly":                {"monthly", day(2024, 1, 15), 1, day(2024, 2, 15)},
		"month end clamps":       {"monthly", day(2024, 1, 31), 1, day(2024, 2, 29)},
		"month end clamps, 2023": {"monthly", day(2023, 1, 31), 1, day(2023, 2, 28)},
		"no drift after clamp":   {"monthly", day(2024, 1, 31), 2, day(2024, 3, 31)},
		"30 day month":           {"monthly", day(2024, 1, 31), 3, day(2024, 4, 30)},
		"december to january":    {"monthly", day(2023, 12, 15), 1, day(2024, 1, 15)},
		"many months":            {"monthly", day(2023, 11, 10), 15, day(2025, 2, 10)},
		"quarterly":              {"quarterly", day(2024, 1, 31), 1, day(2024, 4, 30)},
		"yearly":                 {"yearly", day(2024, 3, 10), 1, day(2025, 3, 10)},
		"leap day":               {"yearly", day(2024, 2, 29), 1, day(2025, 2, 28)},
		"leap day, four years":   {"yearly", day(2024, 2, 29), 4, day(2028, 2, 29)},
		"3months":                {"3months", day(2024, 11, 30), 1, day(2025, 2, 28)},
		"hours":                  {"12h", day(2024, 3, 10), 3, day(2024, 3, 11).Add(12 * time.Hour)},
		"friday to monday":       {"weekdays", day(2024, 3, 8), 1, day(2024, 3, 11)},
		"weekdays over weekend":  {"weekdays", day(2024, 3, 7), 3, day(2024, 3, 12)},
		"a full work week":       {"weekdays", day(2024, 3, 4), 5, day(2024, 3, 11)},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rule, err := ParseRecurrence(tt.rule)
			require.NoError(t, err)
			require.Equal(t, tt.want, rule.Add(tt.start, tt.n))
		})
	}
}

func TestRecurrenceAddKeepsLocalTimeAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no tz database")
	}
	// Clocks went forward on 2024-03-10
	start := time.Date(2024, 3, 8, 9, 0, 0, 0, loc)
	rule, err := ParseRecurrence("weekly")
	require.NoError(t, err)
	got := rule.Add(start, 1)
	require.Equal(t, 9, got.Hour(), "still 9am on the wall clock")
	require.Equal(t, 15, got.Day())
	// In UTC that's an hour earlier than before, which is the point
	require.Equal(t, 13, got.UTC().Hour())
	require.Equal(t, 14, start.UTC().Hour())
}

func TestParseCatchUp(t *testing.T) {
	for in, want := range map[string]CatchUp{"": CatchUpLatest, "latest": CatchUpLatest, "ALL": CatchUpAll} {
		got, err := ParseCatchUp(in)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := ParseCatchUp("everything")
	require.Error(t, err)
}

func TestPadMask(t *testing.T) {
	require.Equal(t, "XX", padMask("", 2, 'X'))
	require.Equal(t, "+-X", padMask("+-X", 1, 'X'))
	require.Equal(t, "+XX", padMask("+", 3, 'X'))
}

// weeklyTemplate is due on Monday 2024-03-04 at 9am, every week
func weeklyTemplate(opts ...TaskOption) *Task {
	due := day(2024, 3, 4)
	all := append([]TaskOption{
		WithRecur("weekly"),
		WithDue(&due),
		WithProject("home"),
		WithPriority(PriorityLow),
		WithTags([]string{"chores"}),
		WithEffortImpact(EffortImpactHigh),
	}, opts...)
	return MustNewTask("Water the plants", all...)
}

func spawnOpts(now time.Time) SpawnOptions {
	return SpawnOptions{Now: now, Location: time.UTC, Limit: 1, CatchUp: CatchUpLatest}
}

// dues lists when the pending instances are due, soonest first
func dues(t *testing.T, s *Store, parent string) []time.Time {
	t.Helper()
	kids, err := s.Instances(parent)
	require.NoError(t, err)
	out := []time.Time{}
	for _, k := range kids {
		out = append(out, *k.Due)
	}
	// insertion into a sorted slice, the list is tiny
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j].Before(out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func TestSpawnPeriodicLatestAndLimit(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	require.NoError(t, s.Add(tpl))

	// Wednesday 20th: the 4th, 11th and 18th are past, the 25th is next
	now := day(2024, 3, 20)
	rep, err := s.SpawnRecurring(spawnOpts(now))
	require.NoError(t, err)
	require.Empty(t, rep.Warnings)
	require.Len(t, rep.Created, 2, "only the latest missed one, plus one ahead")
	require.Equal(t, []time.Time{day(2024, 3, 18), day(2024, 3, 25)}, dues(t, s, tpl.UUID))

	kids, err := s.Instances(tpl.UUID)
	require.NoError(t, err)
	byIdx := map[int]*Task{}
	for _, k := range kids {
		byIdx[*k.IMask] = k
	}
	require.Contains(t, byIdx, 2)
	require.Contains(t, byIdx, 3)
	inst := byIdx[2]
	require.Equal(t, StatusPending, inst.Status)
	require.Equal(t, tpl.UUID, inst.Parent)
	require.Equal(t, "Water the plants", inst.Description)
	require.Equal(t, "home", inst.Project)
	require.Equal(t, PriorityLow, inst.Priority)
	require.Equal(t, []string{"chores"}, inst.Tags)
	require.Equal(t, EffortImpactHigh, inst.EffortImpact)
	require.Equal(t, "weekly", inst.Recur)
	require.Equal(t, now, inst.Entry)
	require.NotEqual(t, tpl.UUID, inst.UUID)

	// The template's mask records them, and the periods that were skipped
	got, err := s.Get(tpl.UUID)
	require.NoError(t, err)
	require.Equal(t, "XX--", got.Mask)
	require.Equal(t, StatusRecurring, got.Status)
	checkIndexes(t, s)
}

func TestSpawnIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Add(weeklyTemplate()))
	now := day(2024, 3, 20)
	_, err := s.SpawnRecurring(spawnOpts(now))
	require.NoError(t, err)

	rep, err := s.SpawnRecurring(spawnOpts(now))
	require.NoError(t, err)
	require.Empty(t, rep.Created)
	all, err := s.List(StatusPending)
	require.NoError(t, err)
	require.Len(t, all, 2)
}

func TestSpawnAdvancesWithTime(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	require.NoError(t, s.Add(tpl))
	_, err := s.SpawnRecurring(spawnOpts(day(2024, 3, 20)))
	require.NoError(t, err)

	// A week later the 25th is due, so the 1st of April is the one ahead
	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 3, 27)))
	require.NoError(t, err)
	require.Len(t, rep.Created, 1)
	require.Equal(t, day(2024, 4, 1), *rep.Created[0].Due)
	require.Equal(t, 4, *rep.Created[0].IMask)

	// Nothing on the same day again
	rep, err = s.SpawnRecurring(spawnOpts(day(2024, 3, 27)))
	require.NoError(t, err)
	require.Empty(t, rep.Created)
}

func TestSpawnCatchUpAll(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	require.NoError(t, s.Add(tpl))
	opts := spawnOpts(day(2024, 3, 20))
	opts.CatchUp = CatchUpAll
	rep, err := s.SpawnRecurring(opts)
	require.NoError(t, err)
	require.Len(t, rep.Created, 4, "the three missed weeks, plus one ahead")
	require.Equal(t, []time.Time{
		day(2024, 3, 4), day(2024, 3, 11), day(2024, 3, 18), day(2024, 3, 25),
	}, dues(t, s, tpl.UUID)[:4])
	got, err := s.Get(tpl.UUID)
	require.NoError(t, err)
	require.Equal(t, "----", got.Mask[:4])
}

func TestSpawnCatchUpAllIsCapped(t *testing.T) {
	s := newTestStore(t)
	due := day(2020, 1, 1)
	require.NoError(t, s.Add(MustNewTask("daily grind", WithRecur("daily"), WithDue(&due))))
	opts := spawnOpts(day(2024, 3, 20)) // 1540 days later
	opts.CatchUp = CatchUpAll
	rep, err := s.SpawnRecurring(opts)
	require.NoError(t, err)
	require.Len(t, rep.Created, maxCatchUp+1)
	require.Contains(t, strings.Join(rep.Warnings, "\n"), "stopped after 1000 instances")
}

func TestSpawnLimit(t *testing.T) {
	for limit, want := range map[int]int{0: 1, 1: 2, 3: 4} {
		s := newTestStore(t)
		tpl := weeklyTemplate()
		require.NoError(t, s.Add(tpl))
		opts := spawnOpts(day(2024, 3, 20))
		opts.Limit = limit
		rep, err := s.SpawnRecurring(opts)
		require.NoError(t, err)
		require.Len(t, rep.Created, want, "limit %d", limit)
	}
}

func TestSpawnFutureTemplate(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	require.NoError(t, s.Add(tpl))
	// Before the first due date there is nothing overdue, just the first one
	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 2, 1)))
	require.NoError(t, err)
	require.Len(t, rep.Created, 1)
	require.Equal(t, day(2024, 3, 4), *rep.Created[0].Due)
	require.Equal(t, 0, *rep.Created[0].IMask)
}

func TestSpawnStopsAtUntil(t *testing.T) {
	s := newTestStore(t)
	until := day(2024, 3, 12)
	tpl := weeklyTemplate(WithUntil(&until))
	require.NoError(t, s.Add(tpl))
	opts := spawnOpts(day(2024, 3, 20))
	opts.CatchUp = CatchUpAll
	rep, err := s.SpawnRecurring(opts)
	require.NoError(t, err)
	require.Equal(t, []time.Time{day(2024, 3, 4), day(2024, 3, 11)}, dues(t, s, tpl.UUID))
	require.Len(t, rep.Created, 2, "nothing after the until date, past or future")

	// Latest mode leaves a series that is already over alone
	s = newTestStore(t)
	tpl = weeklyTemplate(WithUntil(&until))
	require.NoError(t, s.Add(tpl))
	rep, err = s.SpawnRecurring(spawnOpts(day(2024, 3, 20)))
	require.NoError(t, err)
	require.Empty(t, rep.Created, "not resurrecting a period from a finished series")

	// But a series that is still running gets its latest missed one
	s = newTestStore(t)
	tpl = weeklyTemplate(WithUntil(&until))
	require.NoError(t, s.Add(tpl))
	_, err = s.SpawnRecurring(spawnOpts(day(2024, 3, 12)))
	require.NoError(t, err)
	require.Equal(t, []time.Time{day(2024, 3, 11)}, dues(t, s, tpl.UUID), "the 11th, and nothing ahead because the 18th is past the until date")
}

func TestSpawnMonthEndDoesNotDrift(t *testing.T) {
	s := newTestStore(t)
	due := day(2024, 1, 31)
	tpl := MustNewTask("pay rent", WithRecur("monthly"), WithDue(&due))
	require.NoError(t, s.Add(tpl))
	opts := spawnOpts(day(2024, 4, 15))
	opts.CatchUp = CatchUpAll
	_, err := s.SpawnRecurring(opts)
	require.NoError(t, err)
	require.Equal(t, []time.Time{
		day(2024, 1, 31), day(2024, 2, 29), day(2024, 3, 31), day(2024, 4, 30), day(2024, 5, 31),
	}[:4], dues(t, s, tpl.UUID)[:4])
}

func TestSpawnKeepsWaitAndScheduledOffsets(t *testing.T) {
	s := newTestStore(t)
	wait := day(2024, 3, 3)      // a day before it's due
	scheduled := day(2024, 3, 2) // two days before
	tpl := weeklyTemplate()
	tpl.Wait, tpl.Scheduled = &wait, &scheduled
	require.NoError(t, s.Add(tpl))

	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 3, 5)))
	require.NoError(t, err)
	var next *Task
	for _, c := range rep.Created {
		if c.Due.Equal(day(2024, 3, 11)) {
			next = c
		}
	}
	require.NotNil(t, next)
	require.Equal(t, day(2024, 3, 10), *next.Wait)
	require.Equal(t, day(2024, 3, 9), *next.Scheduled)
}

func TestSpawnCopiesUDAs(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	tpl.UDA = map[string]any{"estimate": "2h"}
	require.NoError(t, s.Add(tpl))
	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 3, 5)))
	require.NoError(t, err)
	require.NotEmpty(t, rep.Created)
	for _, c := range rep.Created {
		require.Equal(t, map[string]any{"estimate": "2h"}, c.UDA)
	}
	// Changing one later doesn't leak into the template
	rep.Created[0].UDA["estimate"] = "changed"
	got, err := s.Get(tpl.UUID)
	require.NoError(t, err)
	require.Equal(t, "2h", got.UDA["estimate"])
}

func TestSpawnDryRun(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Add(weeklyTemplate()))
	opts := spawnOpts(day(2024, 3, 20))
	opts.DryRun = true
	rep, err := s.SpawnRecurring(opts)
	require.NoError(t, err)
	require.Len(t, rep.Created, 2)
	all, err := s.List(StatusPending)
	require.NoError(t, err)
	require.Empty(t, all, "nothing was written")
}

func TestSpawnWarnsAboutBrokenTemplates(t *testing.T) {
	s := newTestStore(t)
	due := day(2024, 3, 4)
	noDue := MustNewTask("no due", WithRecur("weekly"))
	noRule := MustNewTask("no rule", WithStatus(StatusRecurring), WithDue(&due))
	badRule := MustNewTask("bad rule", WithRecur("whenever"), WithDue(&due))
	for _, x := range (Tasks{noDue, noRule, badRule}) {
		require.NoError(t, s.Add(x))
	}
	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 3, 20)))
	require.NoError(t, err)
	require.Empty(t, rep.Created)
	joined := strings.Join(rep.Warnings, "\n")
	require.Contains(t, joined, "no due")
	require.Contains(t, joined, "has no due date")
	require.Contains(t, joined, "has no recur rule")
	require.Contains(t, joined, `unknown recurrence "whenever"`)
}

func TestSpawnIgnoresDeletedAndCompletedTemplates(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	require.NoError(t, s.Add(tpl))
	_, err := s.Delete(tpl.UUID)
	require.NoError(t, err)
	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 3, 20)))
	require.NoError(t, err)
	require.Empty(t, rep.Created)
}

func TestSpawnContinuesAnImportedTemplate(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)

	// The imported weekly template has instance 0 (done) and 1 (pending, due
	// the 15th), and a mask of '+-'. Twelve days later it carries on from there.
	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 1, 20)))
	require.NoError(t, err)
	var got []*Task
	for _, c := range rep.Created {
		if c.Parent == uuidTplWeek {
			got = append(got, c)
		}
	}
	require.Len(t, got, 1, "the 22nd is next, the ones that exist aren't repeated")
	require.Equal(t, 2, *got[0].IMask)
	require.Equal(t, day(2024, 1, 22), *got[0].Due)

	tpl, err := s.Get(uuidTplWeek)
	require.NoError(t, err)
	require.Equal(t, "+--", tpl.Mask)

	// The chained one already has a pending instance, so it's left alone
	for _, c := range rep.Created {
		require.NotEqual(t, uuidTplChain, c.Parent)
	}
	checkIndexes(t, s)
}

func TestSpawnPeriodicDoesNotBackfillBeforeExisting(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	require.NoError(t, s.Add(tpl))
	// Someone (say, Taskwarrior) already made instance 3, the 25th
	inst := newInstance(tpl, 3, day(2024, 3, 25), day(2024, 3, 20))
	require.NoError(t, s.Add(inst))

	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 3, 27)))
	require.NoError(t, err)
	require.Len(t, rep.Created, 1)
	require.Equal(t, day(2024, 4, 1), *rep.Created[0].Due, "the missed 11th and 18th stay missed")
}

func chainedTemplate(t *testing.T, s *Store) *Task {
	t.Helper()
	due := day(2024, 3, 1)
	tpl := MustNewTask("Change the filter", WithRecur("1w"), WithDue(&due))
	tpl.RType = RTypeChained
	require.NoError(t, s.Add(tpl))
	return tpl
}

func TestChainedSpawnsOneAtATime(t *testing.T) {
	s := newTestStore(t) // the clock is stuck at testEpoch, 2024-03-10 12:00
	tpl := chainedTemplate(t, s)

	rep, err := s.SpawnRecurring(spawnOpts(testEpoch))
	require.NoError(t, err)
	require.Len(t, rep.Created, 1)
	first := rep.Created[0]
	require.Equal(t, day(2024, 3, 1), *first.Due, "the first one is due when the template says")
	require.Equal(t, 0, *first.IMask)
	require.Equal(t, RTypeChained, first.RType)

	// While it's pending there is nothing more to do, however late it is
	rep, err = s.SpawnRecurring(spawnOpts(testEpoch.Add(90 * 24 * time.Hour)))
	require.NoError(t, err)
	require.Empty(t, rep.Created)

	// Completing it starts the next, a period after it was completed
	_, err = s.Complete(first.UUID)
	require.NoError(t, err)
	kids, err := s.Instances(tpl.UUID)
	require.NoError(t, err)
	require.Len(t, kids, 2)
	var next *Task
	for _, k := range kids {
		if k.Status == StatusPending {
			next = k
		}
	}
	require.NotNil(t, next)
	require.Equal(t, testEpoch.Add(7*24*time.Hour), *next.Due)
	require.Equal(t, 1, *next.IMask)

	got, err := s.Get(tpl.UUID)
	require.NoError(t, err)
	require.Equal(t, "+-", got.Mask)
	checkIndexes(t, s)

	// And it keeps going
	_, err = s.Complete(next.UUID)
	require.NoError(t, err)
	kids, err = s.Instances(tpl.UUID)
	require.NoError(t, err)
	require.Len(t, kids, 3)
}

func TestChainedPicksUpWhereTaskwarriorStopped(t *testing.T) {
	s := newTestStore(t)
	tpl := chainedTemplate(t, s)
	// The last instance was completed, but the next was never made
	inst := newInstance(tpl, 0, day(2024, 3, 1), day(2024, 3, 1))
	inst.Status = StatusCompleted
	end := day(2024, 3, 3)
	inst.End = &end
	require.NoError(t, s.Add(inst))

	rep, err := s.SpawnRecurring(spawnOpts(testEpoch))
	require.NoError(t, err)
	require.Len(t, rep.Created, 1)
	require.Equal(t, 1, *rep.Created[0].IMask)
	require.Equal(t, day(2024, 3, 10), *rep.Created[0].Due)
}

func TestChainedEndsWhenAnInstanceIsDeleted(t *testing.T) {
	s := newTestStore(t)
	tpl := chainedTemplate(t, s)
	rep, err := s.SpawnRecurring(spawnOpts(testEpoch))
	require.NoError(t, err)
	_, err = s.Delete(rep.Created[0].UUID)
	require.NoError(t, err)

	rep, err = s.SpawnRecurring(spawnOpts(testEpoch))
	require.NoError(t, err)
	require.Empty(t, rep.Created)
	kids, err := s.Instances(tpl.UUID)
	require.NoError(t, err)
	require.Len(t, kids, 1)
}

func TestChainedStopsAtUntil(t *testing.T) {
	s := newTestStore(t)
	due := day(2024, 3, 1)
	until := testEpoch.Add(3 * 24 * time.Hour) // the next one would land after this
	tpl := MustNewTask("Change the filter", WithRecur("1w"), WithDue(&due), WithUntil(&until))
	tpl.RType = RTypeChained
	require.NoError(t, s.Add(tpl))
	rep, err := s.SpawnRecurring(spawnOpts(testEpoch))
	require.NoError(t, err)
	_, err = s.Complete(rep.Created[0].UUID)
	require.NoError(t, err)
	kids, err := s.Instances(tpl.UUID)
	require.NoError(t, err)
	require.Len(t, kids, 1)
}

func TestDeletingTemplateDeletesPendingInstances(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	require.NoError(t, s.Add(tpl))
	opts := spawnOpts(day(2024, 3, 20))
	opts.CatchUp = CatchUpAll
	rep, err := s.SpawnRecurring(opts)
	require.NoError(t, err)
	require.Len(t, rep.Created, 4)
	// One of them was finished already
	_, err = s.Complete(rep.Created[0].UUID)
	require.NoError(t, err)

	_, err = s.Delete(tpl.UUID)
	require.NoError(t, err)

	pending, err := s.List(StatusPending)
	require.NoError(t, err)
	require.Empty(t, pending)
	done, err := s.List(StatusCompleted)
	require.NoError(t, err)
	require.Len(t, done, 1, "history is kept")
	deleted, err := s.List(StatusDeleted)
	require.NoError(t, err)
	require.Len(t, deleted, 4, "the template and its three pending instances")
	checkIndexes(t, s)

	// And it stays quiet afterwards
	rep, err = s.SpawnRecurring(opts)
	require.NoError(t, err)
	require.Empty(t, rep.Created)
}

func TestCompletingPeriodicInstanceDoesNotSpawnImmediately(t *testing.T) {
	s := newTestStore(t)
	tpl := weeklyTemplate()
	require.NoError(t, s.Add(tpl))
	rep, err := s.SpawnRecurring(spawnOpts(day(2024, 3, 20)))
	require.NoError(t, err)
	_, err = s.Complete(rep.Created[0].UUID)
	require.NoError(t, err)
	kids, err := s.Instances(tpl.UUID)
	require.NoError(t, err)
	require.Len(t, kids, 2, "the next comes from a spawn run, not from completing")
}

func TestPoetAddRecurring(t *testing.T) {
	p := newTestPoet(t)
	// No due date
	err := p.Add(MustNewTask("no due", WithRecur("weekly")))
	require.ErrorContains(t, err, "due date")
	// Bad rule
	due := day(2024, 3, 4)
	err = p.Add(MustNewTask("bad", WithRecur("sometimes"), WithDue(&due)))
	require.ErrorContains(t, err, "unknown recurrence")
	// A good one
	require.NoError(t, p.Add(MustNewTask("good", WithRecur("weekly"), WithDue(&due))))
	tpls, err := p.Store.List(StatusRecurring)
	require.NoError(t, err)
	require.Len(t, tpls, 1)
}

func TestPoetSpawnRecurring(t *testing.T) {
	p, err := New(
		WithDatabasePath(mustTempDB(t)),
		WithNow(func() time.Time { return day(2024, 3, 20) }),
		WithRecurrence(2, CatchUpAll),
	)
	require.NoError(t, err)
	defer func() { _ = p.Close() }()
	due := day(2024, 3, 4)
	require.NoError(t, p.Add(MustNewTask("weekly thing", WithRecur("weekly"), WithDue(&due))))

	rep, err := p.SpawnRecurring(true)
	require.NoError(t, err)
	require.Len(t, rep.Created, 5, "3 due so far and 2 ahead")
	pending, err := p.Store.List(StatusPending)
	require.NoError(t, err)
	require.Empty(t, pending, "dry run")

	rep, err = p.SpawnRecurring(false)
	require.NoError(t, err)
	require.Len(t, rep.Created, 5)
	pending, err = p.Store.List(StatusPending)
	require.NoError(t, err)
	require.Len(t, pending, 5)
}
