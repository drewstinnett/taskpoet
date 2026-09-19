package taskpoet

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

var testEpoch = time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)

func mustTempDB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(mustTempDB(t), "default", WithClock(func() time.Time { return testEpoch }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// checkIndexes rebuilds every index from the tasks bucket and compares it to
// what is really stored
func checkIndexes(t *testing.T, s *Store) {
	t.Helper()
	wantStatus, wantParent, wantBlocks := map[string]bool{}, map[string]bool{}, map[string]bool{}
	require.NoError(t, s.db.View(func(tx *bolt.Tx) error {
		if err := s.bucket(tx, bucketTasks).ForEach(func(_, v []byte) error {
			task, err := decodeTask(v)
			require.NoError(t, err)
			wantStatus[string(idxKey(string(task.Status), task.UUID))] = true
			if task.Parent != "" {
				wantParent[string(idxKey(task.Parent, task.UUID))] = true
			}
			for _, d := range task.Depends {
				wantBlocks[string(idxKey(d, task.UUID))] = true
			}
			return nil
		}); err != nil {
			return err
		}
		for name, want := range map[string]map[string]bool{
			string(bucketIdxStatus): wantStatus,
			string(bucketIdxParent): wantParent,
			string(bucketIdxBlocks): wantBlocks,
		} {
			got := map[string]bool{}
			if err := s.bucket(tx, []byte(name)).ForEach(func(k, _ []byte) error {
				got[string(k)] = true
				return nil
			}); err != nil {
				return err
			}
			require.Equal(t, want, got, "index %v is out of step with tasks", name)
		}
		return nil
	}))
}

func TestOpenStoreCreatesSchema(t *testing.T) {
	path := mustTempDB(t)
	s, err := OpenStore(path, "default")
	require.NoError(t, err)
	require.NoError(t, s.Add(MustNewTask("persist me")))
	require.NoError(t, s.Close())

	// Opening again keeps the data
	s, err = OpenStore(path, "default")
	require.NoError(t, err)
	defer func() { _ = s.Close() }()
	got, err := s.List()
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "persist me", got[0].Description)
}

func TestOpenStoreEmptyNamespace(t *testing.T) {
	_, err := OpenStore(mustTempDB(t), "")
	require.Error(t, err)
}

func TestOpenStoreCreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "tasks.db")
	s, err := OpenStore(path, "default")
	require.NoError(t, err)
	require.NoError(t, s.Close())
}

func TestOpenStoreRefusesLegacyDatabase(t *testing.T) {
	path := mustTempDB(t)
	db, err := bolt.Open(path, 0o600, nil)
	require.NoError(t, err)
	require.NoError(t, db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte("/default/tasks"))
		require.NoError(t, err)
		return b.Put([]byte("/active/builtin/abc"), []byte(`{"id":"abc"}`))
	}))
	require.NoError(t, db.Close())

	_, err = OpenStore(path, "default")
	require.ErrorIs(t, err, ErrLegacyDatabase)
	// Other namespaces in a legacy file are refused too
	_, err = OpenStore(path, "other")
	require.ErrorIs(t, err, ErrLegacyDatabase)
}

func TestOpenStoreRefusesNewerSchema(t *testing.T) {
	path := mustTempDB(t)
	s, err := OpenStore(path, "default")
	require.NoError(t, err)
	require.NoError(t, s.db.Update(func(tx *bolt.Tx) error {
		return s.bucket(tx, bucketMeta).Put(keySchema, []byte("99"))
	}))
	require.NoError(t, s.Close())

	_, err = OpenStore(path, "default")
	require.ErrorIs(t, err, ErrNewerSchema)
}

func TestOpenStoreLocked(t *testing.T) {
	path := mustTempDB(t)
	s, err := OpenStore(path, "default")
	require.NoError(t, err)
	defer func() { _ = s.Close() }()

	_, err = OpenStore(path, "default")
	require.ErrorIs(t, err, ErrLocked)
}

func TestNamespacesAreSeparate(t *testing.T) {
	path := mustTempDB(t)
	a, err := OpenStore(path, "a")
	require.NoError(t, err)
	require.NoError(t, a.Add(MustNewTask("in a")))
	require.NoError(t, a.Close())

	b, err := OpenStore(path, "b")
	require.NoError(t, err)
	defer func() { _ = b.Close() }()
	got, err := b.List()
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestAddGet(t *testing.T) {
	s := newTestStore(t)
	task := MustNewTask("do a thing", WithProject("home.kitchen"), WithPriority(PriorityHigh), WithTags([]string{"b", "a"}))
	require.NoError(t, s.Add(task))

	got, err := s.Get(task.UUID)
	require.NoError(t, err)
	require.Equal(t, task, got)
	require.Equal(t, []string{"a", "b"}, got.Tags)
	checkIndexes(t, s)

	_, err = s.Get("nope")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestAddFillsDefaults(t *testing.T) {
	s := newTestStore(t)
	task := &Task{Description: "bare"}
	require.NoError(t, s.Add(task))
	require.NotEmpty(t, task.UUID)
	require.Equal(t, StatusPending, task.Status)
	require.Equal(t, testEpoch, task.Entry)
	require.Equal(t, testEpoch, task.Modified)
}

func TestAddDuplicate(t *testing.T) {
	s := newTestStore(t)
	task := MustNewTask("once")
	require.NoError(t, s.Add(task))
	require.ErrorIs(t, s.Add(task), ErrExists)
}

func TestAddInvalid(t *testing.T) {
	s := newTestStore(t)
	require.Error(t, s.Add(&Task{}))
	got, err := s.List()
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestValidate(t *testing.T) {
	self := MustNewTask("self")
	tests := map[string]Task{
		"no uuid":         {Status: StatusPending, Description: "x"},
		"no description":  {UUID: "a", Status: StatusPending},
		"bad status":      {UUID: "a", Status: "waiting", Description: "x"},
		"bad priority":    {UUID: "a", Status: StatusPending, Description: "x", Priority: "Z"},
		"self depends":    {UUID: self.UUID, Status: StatusPending, Description: "x", Depends: []string{self.UUID}},
		"dupe depends":    {UUID: "a", Status: StatusPending, Description: "x", Depends: []string{"b", "b"}},
		"self parent":     {UUID: "a", Status: StatusPending, Description: "x", Parent: "a"},
		"missing statues": {UUID: "a", Description: "x"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require.Error(t, tt.Validate())
		})
	}
	// TW allows a wait past the due date, and so do we
	due := testEpoch
	wait := testEpoch.Add(time.Hour)
	require.NoError(t, MustNewTask("late wait", WithDue(&due), WithWait(&wait)).Validate())
}

func TestGetByPrefix(t *testing.T) {
	s := newTestStore(t)
	a := MustNewTask("a", WithUUID("aaaa1111-0000-0000-0000-000000000000"))
	b := MustNewTask("b", WithUUID("aaaa2222-0000-0000-0000-000000000000"))
	c := MustNewTask("c", WithUUID("cccc3333-0000-0000-0000-000000000000"), WithCompleted(testEpoch))
	for _, task := range (Tasks{a, b, c}) {
		require.NoError(t, s.Add(task))
	}

	got, err := s.GetByPrefix("aaaa1")
	require.NoError(t, err)
	require.Equal(t, "a", got.Description)

	// Case does not matter
	got, err = s.GetByPrefix("AAAA2")
	require.NoError(t, err)
	require.Equal(t, "b", got.Description)

	// Full uuid works
	got, err = s.GetByPrefix(c.UUID)
	require.NoError(t, err)
	require.Equal(t, "c", got.Description)

	_, err = s.GetByPrefix("aaaa")
	require.ErrorIs(t, err, ErrAmbiguous)

	_, err = s.GetByPrefix("zzz")
	require.ErrorIs(t, err, ErrNotFound)

	_, err = s.GetByPrefix("")
	require.Error(t, err)

	// Status filters narrow the search
	_, err = s.GetByPrefix("cccc", StatusPending)
	require.ErrorIs(t, err, ErrNotFound)
	got, err = s.GetByPrefix("cccc", StatusPending, StatusCompleted)
	require.NoError(t, err)
	require.Equal(t, "c", got.Description)
}

func TestListByStatus(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Add(MustNewTask("p1")))
	require.NoError(t, s.Add(MustNewTask("p2")))
	require.NoError(t, s.Add(MustNewTask("done", WithCompleted(testEpoch))))
	require.NoError(t, s.Add(MustNewTask("template", WithRecur("weekly"))))

	for status, want := range map[Status]int{
		StatusPending:   2,
		StatusCompleted: 1,
		StatusDeleted:   0,
		StatusRecurring: 1,
	} {
		got, err := s.List(status)
		require.NoError(t, err)
		require.Len(t, got, want, status)
	}
	got, err := s.List(StatusPending, StatusCompleted)
	require.NoError(t, err)
	require.Len(t, got, 3)
	got, err = s.List()
	require.NoError(t, err)
	require.Len(t, got, 4)

	counts, err := s.CountByStatus()
	require.NoError(t, err)
	require.Equal(t, map[Status]int{StatusPending: 2, StatusCompleted: 1, StatusRecurring: 1}, counts)
	checkIndexes(t, s)
}

func TestCompleteAndDelete(t *testing.T) {
	s := newTestStore(t)
	a := MustNewTask("finish me")
	b := MustNewTask("bin me")
	require.NoError(t, s.Add(a))
	require.NoError(t, s.Add(b))

	done, err := s.Complete(a.UUID)
	require.NoError(t, err)
	require.Equal(t, StatusCompleted, done.Status)
	require.Equal(t, testEpoch, *done.End)
	checkIndexes(t, s)

	gone, err := s.Delete(b.UUID)
	require.NoError(t, err)
	require.Equal(t, StatusDeleted, gone.Status)
	checkIndexes(t, s)

	pending, err := s.List(StatusPending)
	require.NoError(t, err)
	require.Empty(t, pending)
	completed, err := s.List(StatusCompleted)
	require.NoError(t, err)
	require.Len(t, completed, 1)

	_, err = s.Complete(a.UUID)
	require.Error(t, err, "already completed")
	_, err = s.Complete("nope")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCompleteTemplateRefused(t *testing.T) {
	s := newTestStore(t)
	tpl := MustNewTask("template", WithRecur("daily"))
	require.NoError(t, s.Add(tpl))
	_, err := s.Complete(tpl.UUID)
	require.Error(t, err)
	// but it can be deleted
	_, err = s.Delete(tpl.UUID)
	require.NoError(t, err)
}

func TestUpdateKeepsIndexesInStep(t *testing.T) {
	s := newTestStore(t)
	dep1 := MustNewTask("dep1")
	dep2 := MustNewTask("dep2")
	tpl := MustNewTask("tpl", WithRecur("daily"))
	task := MustNewTask("worker", WithDepends([]string{dep1.UUID}))
	for _, x := range (Tasks{dep1, dep2, tpl, task}) {
		require.NoError(t, s.Add(x))
	}
	checkIndexes(t, s)

	blocks, err := s.Blocking(dep1.UUID)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	require.Equal(t, task.UUID, blocks[0].UUID)

	// Move the dependency, add a parent
	updated, err := s.Update(task.UUID, func(x *Task) error {
		x.Depends = []string{dep2.UUID}
		x.Parent = tpl.UUID
		x.IMask = ptr(0)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, testEpoch, updated.Modified)
	checkIndexes(t, s)

	blocks, err = s.Blocking(dep1.UUID)
	require.NoError(t, err)
	require.Empty(t, blocks)
	blocks, err = s.Blocking(dep2.UUID)
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	kids, err := s.Instances(tpl.UUID)
	require.NoError(t, err)
	require.Len(t, kids, 1)

	// Drop them again
	_, err = s.Update(task.UUID, func(x *Task) error {
		x.Depends = nil
		x.Parent = ""
		x.IMask = nil
		return nil
	})
	require.NoError(t, err)
	checkIndexes(t, s)
	kids, err = s.Instances(tpl.UUID)
	require.NoError(t, err)
	require.Empty(t, kids)
}

func TestUpdateErrorLeavesTaskAlone(t *testing.T) {
	s := newTestStore(t)
	task := MustNewTask("stay")
	require.NoError(t, s.Add(task))
	_, err := s.Update(task.UUID, func(x *Task) error {
		x.Description = "changed"
		x.Depends = []string{x.UUID} // invalid
		return nil
	})
	require.Error(t, err)
	got, err := s.Get(task.UUID)
	require.NoError(t, err)
	require.Equal(t, "stay", got.Description)
	checkIndexes(t, s)
}

func TestMaskUpkeep(t *testing.T) {
	s := newTestStore(t)
	tpl := MustNewTask("tpl", WithRecur("daily"))
	require.NoError(t, s.Add(tpl))
	var kids Tasks
	for i := 0; i < 3; i++ {
		kid := MustNewTask("kid")
		kid.Parent = tpl.UUID
		kid.IMask = ptr(i)
		require.NoError(t, s.Add(kid))
		kids = append(kids, kid)
	}

	_, err := s.Complete(kids[0].UUID)
	require.NoError(t, err)
	_, err = s.Delete(kids[2].UUID)
	require.NoError(t, err)

	got, err := s.Get(tpl.UUID)
	require.NoError(t, err)
	require.Equal(t, "+-X", got.Mask)
	checkIndexes(t, s)
}

func TestSetMaskChar(t *testing.T) {
	require.Equal(t, "+", setMaskChar("", 0, '+'))
	require.Equal(t, "--X", setMaskChar("", 2, 'X'))
	require.Equal(t, "-+-", setMaskChar("---", 1, '+'))
	require.Equal(t, "+-X", setMaskChar("+-X", 1, '-'))
}
