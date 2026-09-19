package taskpoet

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func canonicalTasks(t *testing.T) Tasks {
	t.Helper()
	return mustParseFixture(t, "canonical.json", ParseOptions{}).Tasks
}

func TestImportTasks(t *testing.T) {
	s := newTestStore(t)
	rep, err := s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)

	require.Equal(t, 11, rep.Read)
	require.Equal(t, 11, rep.Imported)
	require.Zero(t, rep.Skipped)
	require.Zero(t, rep.Overwritten)
	require.Empty(t, rep.Warnings)
	want := map[Status]int{StatusPending: 6, StatusCompleted: 2, StatusDeleted: 1, StatusRecurring: 2}
	require.Equal(t, want, rep.ByStatus)

	counts, err := s.CountByStatus()
	require.NoError(t, err)
	require.Equal(t, want, counts)
	checkIndexes(t, s)

	// Everything came across, not just the pending tasks
	got, err := s.Get(uuidTplWeek)
	require.NoError(t, err)
	require.Equal(t, "+-", got.Mask)

	kids, err := s.Instances(uuidTplWeek)
	require.NoError(t, err)
	require.Len(t, kids, 2)

	blocked, err := s.Blocking(uuidReport)
	require.NoError(t, err)
	require.Len(t, blocked, 2)

	// Timestamps are kept as they were in Taskwarrior
	got, err = s.Get(uuidReport)
	require.NoError(t, err)
	require.Equal(t, twTime(t, "20240110T101500Z"), got.Modified)
}

func TestImportSurvivesReopen(t *testing.T) {
	path := mustTempDB(t)
	s, err := OpenStore(path, "default")
	require.NoError(t, err)
	_, err = s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)
	require.NoError(t, s.Close())

	s, err = OpenStore(path, "default")
	require.NoError(t, err)
	defer func() { _ = s.Close() }()
	all, err := s.List()
	require.NoError(t, err)
	require.Len(t, all, 11)

	// And the whole thing exports back out the way it came in
	var out bytes.Buffer
	require.NoError(t, WriteTaskWarrior(&out, all))
	orig := canonicalTasks(t)
	var want bytes.Buffer
	require.NoError(t, WriteTaskWarrior(&want, orig))
	require.Equal(t, normalize(t, want.Bytes()), normalize(t, out.Bytes()))
}

func TestImportIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)

	rep, err := s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 11, rep.Skipped)
	require.Zero(t, rep.Imported)
	all, err := s.List()
	require.NoError(t, err)
	require.Len(t, all, 11)
	checkIndexes(t, s)
}

func TestImportSkipLeavesExistingAlone(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)
	_, err = s.Update(uuidMilk, func(x *Task) error {
		x.Description = "edited locally"
		return nil
	})
	require.NoError(t, err)

	_, err = s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)
	got, err := s.Get(uuidMilk)
	require.NoError(t, err)
	require.Equal(t, "edited locally", got.Description)
}

func TestImportOverwrite(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)

	// Local edits: change the description, move the dependencies, set an
	// effort/impact that Taskwarrior doesn't know about
	_, err = s.Update(uuidSend, func(x *Task) error {
		x.Description = "edited locally"
		x.Depends = nil
		x.EffortImpact = EffortImpactLow
		return nil
	})
	require.NoError(t, err)
	checkIndexes(t, s)

	rep, err := s.ImportTasks(canonicalTasks(t), ImportOptions{Overwrite: true})
	require.NoError(t, err)
	require.Equal(t, 11, rep.Overwritten)
	require.Zero(t, rep.Imported)
	require.Zero(t, rep.Skipped)

	got, err := s.Get(uuidSend)
	require.NoError(t, err)
	require.Equal(t, "Send the report to the board", got.Description)
	require.Equal(t, []string{uuidReport}, got.Depends, "index and record both restored")
	require.Equal(t, EffortImpactLow, got.EffortImpact, "taskpoet's own field survives an overwrite")
	checkIndexes(t, s)
}

func TestImportDryRun(t *testing.T) {
	s := newTestStore(t)
	rep, err := s.ImportTasks(canonicalTasks(t), ImportOptions{DryRun: true})
	require.NoError(t, err)
	require.True(t, rep.DryRun)
	require.Equal(t, 11, rep.Imported)
	require.Equal(t, 6, rep.ByStatus[StatusPending])

	all, err := s.List()
	require.NoError(t, err)
	require.Empty(t, all, "a dry run writes nothing")

	// Against a populated store it reports what would be skipped
	_, err = s.ImportTasks(canonicalTasks(t), ImportOptions{})
	require.NoError(t, err)
	rep, err = s.ImportTasks(canonicalTasks(t), ImportOptions{DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 11, rep.Skipped)
	rep, err = s.ImportTasks(canonicalTasks(t), ImportOptions{DryRun: true, Overwrite: true})
	require.NoError(t, err)
	require.Equal(t, 11, rep.Overwritten)
}

func TestImportIsAtomic(t *testing.T) {
	s := newTestStore(t)
	good := canonicalTasks(t)
	bad := &Task{UUID: "bad", Status: StatusPending} // no description
	ts := append(Tasks{}, good...)
	ts = append(ts, bad)

	_, err := s.ImportTasks(ts, ImportOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad")

	all, err := s.List()
	require.NoError(t, err)
	require.Empty(t, all, "nothing is written when any task is bad")
	checkIndexes(t, s)

	// Dry runs catch the same problem
	_, err = s.ImportTasks(ts, ImportOptions{DryRun: true})
	require.Error(t, err)
}

func TestImportIntegrityWarnings(t *testing.T) {
	s := newTestStore(t)
	res := mustParseFixture(t, "legacy.json", ParseOptions{})
	rep, err := s.ImportTasks(res.Tasks, ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 11, rep.Imported, "warnings never stop an import")
	joined := strings.Join(rep.Warnings, "\n")
	require.Contains(t, joined, "1 tasks depend on tasks that are not in the database")
	require.Contains(t, joined, "1 recurring instances point at a parent that is not in the database")
	require.Contains(t, joined, "1 recurring templates have no 'recur' rule")
	require.Contains(t, joined, "1 dependency cycles")
	checkIndexes(t, s)

	// The dangling references really are kept
	got, err := s.Get("aaaaaaaa-0000-4000-8000-000000000007")
	require.NoError(t, err)
	require.Equal(t, []string{"ffffffff-ffff-4fff-8fff-ffffffffffff"}, got.Depends)
}

func TestImportWarningsSeeExistingTasks(t *testing.T) {
	s := newTestStore(t)
	// The parent of this instance arrives in an earlier import, so it isn't an orphan
	first := Tasks{MustNewTask("template", WithUUID("tpl"), WithRecur("daily"))}
	_, err := s.ImportTasks(first, ImportOptions{})
	require.NoError(t, err)
	inst := MustNewTask("instance", WithUUID("inst"))
	inst.Parent = "tpl"
	inst.IMask = ptr(0)
	rep, err := s.ImportTasks(Tasks{inst}, ImportOptions{})
	require.NoError(t, err)
	require.Empty(t, rep.Warnings)
}

func TestImportCycleReport(t *testing.T) {
	s := newTestStore(t)
	a := MustNewTask("a", WithUUID("aaaaaaaa1"), WithDepends([]string{"bbbbbbbb2"}))
	b := MustNewTask("b", WithUUID("bbbbbbbb2"), WithDepends([]string{"cccccccc3"}))
	c := MustNewTask("c", WithUUID("cccccccc3"), WithDepends([]string{"aaaaaaaa1"}))
	rep, err := s.ImportTasks(Tasks{a, b, c}, ImportOptions{})
	require.NoError(t, err)
	require.Len(t, rep.Warnings, 1)
	require.Contains(t, rep.Warnings[0], "1 dependency cycles")
	require.Contains(t, rep.Warnings[0], "aaaaaaaa -> bbbbbbbb -> cccccccc -> aaaaaaaa")
}
