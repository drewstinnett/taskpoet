package taskpoet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestPoet(t *testing.T, tasks ...*Task) *Poet {
	t.Helper()
	p, err := New(
		WithDatabasePath(mustTempDB(t)),
		WithNow(func() time.Time { return testEpoch }),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })
	for _, task := range tasks {
		require.NoError(t, p.Add(task))
	}
	return p
}

func TestNewPoetDefaults(t *testing.T) {
	p := newTestPoet(t)
	require.Equal(t, "default", p.Namespace)
	_, err := New(WithNamespace(""))
	require.Error(t, err)
}

func TestDefaultDBPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/xdg")
	got, err := DefaultDBPath()
	require.NoError(t, err)
	require.Equal(t, "/xdg/taskpoet/taskpoet.db", got)

	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/someone")
	got, err = DefaultDBPath()
	require.NoError(t, err)
	require.Equal(t, "/home/someone/.local/share/taskpoet/taskpoet.db", got)
}

func TestAddAppliesDefaultDue(t *testing.T) {
	p, err := New(
		WithDatabasePath(mustTempDB(t)),
		WithNow(func() time.Time { return testEpoch }),
		WithDefaultDue(48*time.Hour),
	)
	require.NoError(t, err)
	defer func() { _ = p.Close() }()

	plain := MustNewTask("plain")
	require.NoError(t, p.Add(plain))
	require.Equal(t, testEpoch.Add(48*time.Hour), *plain.Due)

	due := testEpoch.Add(time.Hour)
	explicit := MustNewTask("explicit", WithDue(&due))
	require.NoError(t, p.Add(explicit))
	require.Equal(t, due, *explicit.Due)

	logged := MustNewTask("already done", WithCompleted(testEpoch))
	require.NoError(t, p.Add(logged))
	require.Nil(t, logged.Due, "completed tasks don't get a due date")
}

func TestListFlagsBlockedTasks(t *testing.T) {
	p := newTestPoet(t)
	_, err := p.Store.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)

	pending, err := p.List(StatusPending)
	require.NoError(t, err)
	all := byUUID(pending)
	require.Len(t, pending, 6)

	// The report is pending and holds up two others
	require.False(t, all[uuidReport].IsBlocked())
	require.Equal(t, 2, all[uuidReport].blocking)
	// These are waiting on it
	require.True(t, all[uuidSend].IsBlocked())
	require.True(t, all[uuidEffort].IsBlocked())
	require.False(t, all[uuidUnicode].IsBlocked())

	// Finishing the report unblocks the board mail, and the weights follow
	_, err = p.Store.Complete(uuidReport)
	require.NoError(t, err)
	pending, err = p.List(StatusPending)
	require.NoError(t, err)
	all = byUUID(pending)
	require.False(t, all[uuidSend].IsBlocked())
	require.True(t, all[uuidEffort].IsBlocked(), "still waiting on the board mail")
	require.Equal(t, 1, all[uuidSend].blocking)
}

func TestUrgencyReflectsBlockedAndPriority(t *testing.T) {
	p := newTestPoet(t)
	blocker := MustNewTask("blocker", WithPriority(PriorityLow))
	blocked := MustNewTask("blocked", WithPriority(PriorityLow), WithDepends([]string{blocker.UUID}))
	high := MustNewTask("high", WithPriority(PriorityHigh))
	for _, task := range (Tasks{blocker, blocked, high}) {
		require.NoError(t, p.Add(task))
	}
	pending, err := p.List(StatusPending)
	require.NoError(t, err)
	all := byUUID(pending)
	require.Greater(t, all[high.UUID].Urgency, all[blocker.UUID].Urgency)
	require.Greater(t, all[blocker.UUID].Urgency, all[blocked.UUID].Urgency)
}

func TestTaskTable(t *testing.T) {
	due := testEpoch.Add(24 * time.Hour)
	wait := time.Now().Add(24 * time.Hour)
	p := newTestPoet(t,
		MustNewTask("visible task", WithProject("home"), WithDue(&due)),
		MustNewTask("hidden until later", WithWait(&wait)),
		MustNewTask("finished task", WithCompleted(testEpoch)),
	)

	got, err := p.TaskTable(TableOpts{
		Columns:      []string{"ID", "Project", "Description", "Urgency"},
		Filters:      []Filter{FilterHidden},
		FilterParams: FilterParams{Now: time.Now()},
		SortBy:       ByUrgency{},
	})
	require.NoError(t, err)
	require.Contains(t, got, "visible task")
	require.Contains(t, got, "home")
	require.NotContains(t, got, "hidden until later")
	require.NotContains(t, got, "finished task")

	got, err = p.TaskTable(TableOpts{
		Statuses: []Status{StatusCompleted},
		Columns:  []string{"ID", "Description", "Completed"},
		SortBy:   ByCompleted{},
	})
	require.NoError(t, err)
	require.Contains(t, got, "finished task")
	require.Contains(t, got, "2024-03-10")

	_, err = p.TaskTable(TableOpts{Columns: []string{"NoSuchColumn"}})
	require.ErrorContains(t, err, "column not defined: NoSuchColumn")
}

func TestFilters(t *testing.T) {
	kitchen := MustNewTask("fix tap", WithProject("home.kitchen"), WithTags([]string{"diy"}))
	home := MustNewTask("mow lawn", WithProject("home"))
	work := MustNewTask("file taxes", WithProject("homework"))
	all := Tasks{kitchen, home, work}

	got := ApplyFilters(all, &FilterParams{Project: "home"}, FilterProject)
	require.Equal(t, Tasks{kitchen, home}, got, "subprojects match, lookalikes don't")
	got = ApplyFilters(all, &FilterParams{Tag: "diy"}, FilterTag)
	require.Equal(t, Tasks{kitchen}, got)
	got = ApplyFilters(all, &FilterParams{}, FilterProject, FilterTag, FilterRegex)
	require.Len(t, got, 3, "empty params filter nothing, and nil regex is fine")
}

func TestDescribeTask(t *testing.T) {
	p := newTestPoet(t)
	_, err := p.Store.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)
	task, err := p.Store.GetByPrefix("11111111")
	require.NoError(t, err)

	got, err := p.DescribeTask(*task)
	require.NoError(t, err)
	for _, want := range []string{
		uuidReport, "work.reports", "Priority", "Blocks", "22222222", "storypoints", "Urgency Calculation",
	} {
		require.Contains(t, got, want)
	}
}

func TestCompleteIDs(t *testing.T) {
	a := MustNewTask("buy bread", WithUUID("abcdef12-0000-0000-0000-000000000000"))
	b := MustNewTask("bake bread", WithUUID("fedcba98-0000-0000-0000-000000000000"))
	p := newTestPoet(t, a, b)
	require.ElementsMatch(t, []string{"abcdef12\tbuy bread"}, p.CompleteIDs("abc", StatusPending))
	require.Len(t, p.CompleteIDs("BREAD", StatusPending), 2, "matches descriptions, ignoring case")
	require.Empty(t, p.CompleteIDs("abc", StatusCompleted))
}
