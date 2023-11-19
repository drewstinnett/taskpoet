package taskpoet

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
)

// Local Client for lookups
var (
	lc            *Poet
	emptyDefaults Task
	router        *gin.Engine
	testDBPath    string
)

/*
func newTestPoet(t *testing.T) (*Poet, string) {
	dbPath := path.Join(t.TempDir(), "testtaskpoet.db")
	p, err := New(WithDatabasePath(dbPath))
	panicIfErr(err)
	return p, testDBPath
}
*/

func setup() {
	// Init a db
	tmpfile, err := os.CreateTemp("", "taskpoet.*.db")
	panicIfErr(err)
	testDBPath = tmpfile.Name()
	lc, err = New(WithDatabasePath(testDBPath))
	panicIfErr(err)
	emptyDefaults = Task{}

	// Init Router
	router = NewRouter(&RouterConfig{LocalClient: lc})

	os.Setenv("TZ", "America/New_York")
}

func shutdown() {
	err := os.Remove(testDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not remove:%v ", testDBPath)
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestIDSlash(t *testing.T) {
	/*
		//task:  Task{ID: "foo/bar", Description: "Invalid-id"},
		task:  Task{ID: "foo/bar", Description: "Invalid-id"},
		valid: false,
	*/
	_, err := NewTask("foo", WithID("foo/bar"))
	require.EqualError(t, err, "ID Cannot contain a slash (/)")
}

func TestLogTask(t *testing.T) {
	_, err := lc.Task.Log(&Task{ID: "log-this-task", Description: "foo"}, &emptyDefaults)
	require.NoError(t, err)
	_, err = lc.Task.GetWithID("log-this-task", "", "/completed")
	require.NoError(t, err)
}

func TestCompleteTask(t *testing.T) {
	task, err := lc.Task.Add(&Task{Description: "soon-to-complete-task"})
	require.NoError(t, err)
	activePath := task.DetectKeyPath()

	require.NoError(t, lc.Task.Complete(task))
	completePath := task.DetectKeyPath()

	_, err = lc.Task.GetWithExactPath(activePath)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "could not find task:"))

	_, err = lc.Task.GetWithExactPath(completePath)
	require.NoError(t, err)
}

func TestBlankDescription(t *testing.T) {
	_, err := NewTask("")
	require.Error(t, err)
	require.EqualError(t, err, "missing description for Task")
}

func TestGetByPartialIDWithPath(t *testing.T) {
	ts := Tasks{
		MustNewTask("foo", WithID("again with the fakeid again")),
		MustNewTask("foo", WithID("fakeid")),
		MustNewTask("foo", WithID("another_fakeid")),
		MustNewTask("foo", WithID("dupthing-num-1")),
		MustNewTask("foo", WithID("dupthing-num-2")),
		MustNewTask("foo"),
	}
	require.NoError(t, lc.Task.AddSet(ts))
	task, err := lc.Task.GetWithPartialID("fake", "", "/active")
	require.NoError(t, err)
	require.Equal(t, "fakeid", task.ID)

	// Test for a non-unique partial
	_, err = lc.Task.GetWithPartialID("dupthing", "", "/active")
	require.Error(t, err)

	// Test for a non existent prefix
	_, err = lc.Task.GetWithPartialID("this-will-never-exist", "", "/active")
	require.Error(t, err)
}

func TestDefaults(t *testing.T) {
	// Set some defaults
	fakeDuration, _ := ParseDuration("2h")
	duration := time.Now().Add(fakeDuration)

	p, err := New(
		WithDatabasePath(mustTempDB(t)),
	)
	require.NoError(t, err)
	p.Default = Task{Due: &duration}

	task, _ := p.Task.Add(&Task{Description: "foo"})
	require.EqualValues(t, &duration, task.Due)
}

func TestGetByExactPath(t *testing.T) {
	require.NoError(t, lc.Task.AddSet(
		Tasks{
			MustNewTask("foo", WithID("id-stay-active")),
		},
	))
	_, err := lc.Task.GetWithID("id-stay-active", "", "")
	require.NoError(t, err)

	// Check completed
	_, err = lc.Task.Log(&Task{ID: "id-in-completed", Description: "foo"}, &emptyDefaults)
	require.NoError(t, err)
	_, err = lc.Task.GetWithID("id-in-completed", "", "")
	require.NoError(t, err)
}

func TestDuplicateIDs(t *testing.T) {
	// Put something new in the completed bucket
	_, err := lc.Task.Log(&Task{ID: "duplicate-id", Description: "foo"}, &emptyDefaults)
	require.NoError(t, err)

	// Try to create a new task with the same id
	_, err = lc.Task.Add(&Task{ID: "duplicate-id", Description: "foo"})
	require.Error(t, err)
	require.Equal(t, errExists, err)

	// Make sure IDs and PluginIDs are UniqueTogether
	_, err = lc.Task.Add(&Task{ID: "duplicate-id-plugin", PluginID: "plugin-1", Description: "foo"})
	require.NoError(t, err)

	// Try to create a new task with the same id
	_, err = lc.Task.Add(&Task{ID: "duplicate-id-plugin", PluginID: "plugin-2", Description: "foo"})
	require.NoError(t, err)
}

func TestGetByExactID(t *testing.T) {
	require.NoError(t, lc.Task.AddSet(
		Tasks{
			MustNewTask("foo", WithID("fakeid-exact")),
			MustNewTask("foo", WithID("another_fakeid-exact")),
			MustNewTask("foo"),
		}))
	activePrefix := "/active"
	task, err := lc.Task.GetWithID("another_fakeid-exact", "", activePrefix)
	require.NoError(t, err)
	require.Equal(t, "another_fakeid-exact", task.ID)

	_, err = lc.Task.GetWithID("another", "", activePrefix)
	require.Error(t, err)
}

func TestListNonExistant(t *testing.T) {
	r, err := lc.Task.List("/never-exist")
	require.NoError(t, err)
	require.Equal(t, 0, len(r))
}

func TestAddParent(t *testing.T) {
	tasks := Tasks{
		{ID: "kid", Description: "Kid task"},
		{ID: "parent", Description: "Parent task"},
	}

	require.NoError(t, lc.Task.AddSet(tasks))
	kid, _ := lc.Task.GetWithID("kid", "", "/active")
	parent, _ := lc.Task.GetWithID("parent", "", "/active")

	// Make sure adding a parent works
	kid.Parents = append(kid.Parents, parent.ID)
	_, err := lc.Task.Edit(kid)
	require.NoError(t, err)

	// Make sure you can't add the same parent multiple times
	kid.Parents = append(kid.Parents, parent.ID)
	_, err = lc.Task.Edit(kid)
	require.Error(t, err)
}

func TestTaskSelfAddParent(t *testing.T) {
	_, err := NewTask("foo",
		WithID("test-self-add-parent"),
		WithParents([]string{"test-self-add-parent"}),
	)
	require.Error(t, err)
	require.EqualError(t, err, "self id is set in the parents, we don't do that")
}

func TestTaskSelfAddChild(t *testing.T) {
	_, err := NewTask("foo",
		WithID("test-self-add-child"),
		WithChildren([]string{"test-self-add-child"}),
	)
	require.Error(t, err)
	require.EqualError(t, err, "self id is set in the children, we don't do that")
}

func TestTaskDuplicateParents(t *testing.T) {
	_, err := NewTask("some-id",
		WithDescription("foo"),
		WithParents([]string{"dup", "dup"}),
	)
	require.Error(t, err)
	require.EqualError(t, err, "found duplicate ids in the Parents field")
}

func TestShortID(t *testing.T) {
	lc.Task.AddSet(Tasks{
		MustNewTask("Short ID", WithID("a")),
		MustNewTask("Long ID", WithID("foo-bar-baz-bazinga")),
	})
	require.Equal(t, "a", lc.MustGet("a").ShortID())
	require.Equal(t, "foo-b", lc.MustGet("foo-bar-baz-bazinga").ShortID())
}

func TestEditNonExisting(t *testing.T) {
	got, err := lc.Task.Edit(&Task{ID: "non-existing-edit"})
	require.Nil(t, got)
	require.EqualError(t, err, "cannot edit a task that did not previously exist: non-existing-edit")
}

func TestEditInvalid(t *testing.T) {
	p := newTestPoet(t)
	task := MustNewTask("foo", WithID("soon-to-be-valid"))
	_, aerr := p.Task.Add(task)
	require.NoError(t, aerr)

	task.Description = ""
	_, err := p.Task.Edit(task)
	require.EqualError(t, err, "missing description for Task")
}

func TestEditCompletedInvalid(t *testing.T) {
	p := newTestPoet(t)
	task := &Task{ID: "test-completed-edit", Description: "foo"}
	_, err := p.Task.Add(task)
	require.NoError(t, err)

	task.Completed = ptr(time.Now())
	_, err = p.Task.Edit(task)
	require.EqualError(t, err, "cannot edit a task that did not previously exist: test-completed-edit")
}

func TestEditDescription(t *testing.T) {
	task := MustNewTask("original", WithID("test-edit-description"))
	_, err := lc.Task.Add(task)
	require.NoError(t, err)

	task.Description = "New"
	edited, err := lc.Task.Edit(task)
	require.NoError(t, err)
	require.Equal(t, "New", edited.Description)
}

func TestEditSet(t *testing.T) {
	require.NoError(t, lc.Task.AddSet(Tasks{
		MustNewTask("Foo", WithID("edit-set-1")),
		MustNewTask("Bar", WithID("edit-set-2")),
	}))

	test2 := lc.MustGet("edit-set-2")
	editSet := []Task{
		*lc.MustGet("edit-set-1"),
		*test2,
	}
	test2.Description = "New Description"
	require.NoError(t, lc.Task.EditSet(editSet))
}

func TestAddParentFunc(t *testing.T) {
	require.NoError(t, lc.Task.AddSet(Tasks{
		MustNewTask("Kid task", WithID("kid-func")),
		MustNewTask("Parent task", WithID("parent-func")),
	}))

	// Make sure adding a parent works
	require.NoError(t, lc.Task.AddParent(
		lc.MustGet("kid-func"),
		lc.MustGet("parent-func"),
	))

	// Get newly Editted
	kid := lc.MustGet("kid-func")
	parent := lc.MustGet("parent-func")

	require.Contains(t, kid.Parents, parent.ID)
	require.Contains(t, parent.Children, kid.ID)
}

func TestAddChildFunc(t *testing.T) {
	p := newTestPoet(t)

	require.NoError(t, p.Task.AddSet(
		Tasks{
			MustNewTask("Kid task", WithID("kid-func2")),
			MustNewTask("Parent task", WithID("parent-func2")),
		}))

	// Make sure adding a parent works
	require.NoError(t, p.Task.AddChild(
		p.MustGet("parent-func2"),
		p.MustGet("kid-func2"),
	))

	// Get newly Editted
	kid := p.MustGet("kid-func2")
	parent := p.MustGet("parent-func2")
	require.Contains(t, kid.Parents, parent.ID)
	require.Contains(t, parent.Children, kid.ID)
}

func TestGetByPartialID(t *testing.T) {
	p := newTestPoet(t)
	ts := Tasks{
		{Description: "foo", ID: "partial-id-test"},
		{Description: "foo", ID: "partial-id-test-2"},
		{Description: "foo", ID: "unique-partial-id-test-2"},
	}
	require.NoError(t, p.Task.AddSet(ts))
	task, err := p.Task.GetWithPartialID("partial-id-test-2", "", "")
	require.NoError(t, err)
	require.Equal(t, "partial-id-test-2", task.ID)

	// Test for a non-unique partial
	_, err = p.Task.GetWithPartialID("partial-id", "", "")
	require.EqualError(t, err, "more than 1 match for partial-id found in [], please try using more of the ID. Returned: [/active/builtin/partial-id-test /active/builtin/partial-id-test-2]")

	// Test for a non existent prefix
	_, err = p.Task.GetWithPartialID("this-will-never-exist", "", "/active")
	require.EqualError(t, err, "no matches for this-will-never-exist found in []")
}

func TestDescribe(t *testing.T) {
	// pterm.SetDefaultOutput(os.NewFile(0, os.DevNull))
	ts := Tasks{
		{Description: "foo", ID: "describe-test"},
		{Description: "Some parent", ID: "describe-parent"},
	}
	lc.Task.AddSet(ts)
	task, _ := lc.Task.GetWithID("describe-test", "builtin", "/active")
	taskP, _ := lc.Task.GetWithID("describe-parent", "builtin", "/active")
	lc.DescribeTask(*task)

	// Describe with parent test
	lc.Task.AddParent(task, taskP)
	task, err := lc.Task.Edit(task)
	require.NoError(t, err)
	lc.DescribeTask(*task)

	// Describe with parent test
	taskP, err = lc.Task.GetWithID("describe-parent", "builtin", "/active")
	require.NoError(t, err)

	lc.DescribeTask(*taskP)

	// Describe a task with more things set
	n := time.Now()
	wait := n.Add(time.Hour * 1)
	due := n.Add(time.Hour * 24)
	completed := n.Add(time.Hour * 12)
	lc.DescribeTask(Task{
		ID:          "describe-descriptive",
		Description: "foo",
		Due:         &due,
		HideUntil:   &wait,
		Completed:   &completed,
	})
}

func TestHideAfterDue(t *testing.T) {
	now := time.Now()
	sooner := now.Add(time.Minute * 5)
	later := now.Add(time.Minute * 10)
	_, err := NewTask("test-hide-after-due",
		WithID("test-hide-after-due"),
		WithHideUntil(&later),
		WithDue(&sooner),
	)
	require.Error(t, err)
}

func TestDefaultBucketName(t *testing.T) {
	assert.Equal(t, string(lc.bucket), "/default/tasks")
}

func TestPurgeTask(t *testing.T) {
	added, err := lc.Task.Add(&Task{
		ID:          "delete-me",
		Description: "foo",
	})
	require.NoError(t, err)

	// Delete it now
	require.NoError(t, lc.Task.Purge(added))

	_, err = lc.Task.GetWithID("delete-me", "", "")
	if err == nil {
		t.Error("Got task we should have deleted")
	}
}

func TestDetectKeyPath(t *testing.T) {
	tests := []struct {
		task   Task
		wanted string
	}{
		{
			task:   Task{ID: "foo", Description: "bar"},
			wanted: "/active/builtin/foo",
		},
		{
			task:   Task{ID: "foo", Description: "bar", PluginID: "plugin-1"},
			wanted: "/active/plugin-1/foo",
		},
		{
			task:   Task{ID: "foo", Description: "bar", Deleted: ptr(time.Now())},
			wanted: "/deleted/builtin/foo",
		},
		{
			task:   Task{ID: "foo", Description: "bar", Completed: ptr(time.Now())},
			wanted: "/completed/builtin/foo",
		},
	}

	for _, test := range tests {
		got := string(test.task.DetectKeyPath())
		require.Equal(t, test.wanted, got)
	}
}

func TestAddOrEditSet(t *testing.T) {
	require.NoError(t, lc.Task.AddSet(Tasks{
		{Description: "Foo", ID: "add-or-edit-do-edit-1"},
	}))

	require.NoError(t, lc.Task.AddOrEditSet([]Task{
		{Description: "Edited-desc", ID: "add-or-edit-do-edit-1"},
		{Description: "Added-desc", ID: "add-or-edit-do-add-1"},
	}))

	edited, err := lc.Task.GetWithID("add-or-edit-do-edit-1", "", "")
	require.NoError(t, err)
	added, err := lc.Task.GetWithID("add-or-edit-do-add-1", "", "")
	require.NoError(t, err)

	require.Equal(t, edited.Description, "Edited-desc")
	require.Equal(t, added.Description, "Added-desc")
}

func TestEditExistingValues(t *testing.T) {
	ts := Tasks{
		MustNewTask("Foo", WithID("edit-existing-1")),
	}
	require.NoError(t, lc.Task.AddSet(ts))

	aets := []Task{
		*MustNewTask("Update", WithID("edit-existing-1")),
	}
	require.NoError(t, lc.Task.AddOrEditSet(aets))

	edited, _ := lc.Task.GetWithID("edit-existing-1", "", "")
	assert.Equal(t, false, edited.Added.IsZero())
}

func TestCompleteIDs(t *testing.T) {
	p := newTestPoet(t)
	p.Task.Add(MustNewTask("This is foo"))
	p.Task.Add(MustNewTask("This is bar"))
	got := p.CompleteIDsWithPrefix("/active", "bar")
	require.True(t, strings.HasSuffix(got[0], "\tThis is bar"))
	require.Equal(t, 1, len(got))
}

func TestTaskTable(t *testing.T) {
	p := newTestPoet(t)
	_, err := p.Task.Add(MustNewTask("draw a table and test it"))
	require.NoError(t, err)

	table := p.TaskTable(TableOpts{
		Prefix:  "/active",
		Columns: []string{"ID", "Description", "Due"},
	})
	require.Contains(
		t,
		table,
		"draw a table and test it",
	)
}

func TestUrgency(t *testing.T) {
	p := newTestPoet(t)
	got, err := p.Task.Add(MustNewTask("something in the past",
		WithDue(datePTR(time.Now().Add(-24*time.Hour)))),
	)
	require.NoError(t, err)
	require.Greater(t, got.Urgency, float64(0))
	// Again from cache
	require.Greater(t, got.Urgency, float64(0))
}

func TestMiscNewWith(t *testing.T) {
	a := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	require.Equal(
		t,
		&Task{
			ID:           "some-id",
			Description:  "some description",
			Urgency:      float64(0),
			EffortImpact: 2,
			Tags:         []string{"bar", "foo"},
			Added:        a,
			PluginID:     "builtin",
		},
		MustNewTask("some description",
			WithID("some-id"),
			WithEffortImpact(EffortImpactMedium),
			WithTags([]string{"foo", "bar"}),
			WithAdded(&a),
		),
	)
}

func TestNewComment(t *testing.T) {
	got, err := NewComment("")
	require.Nil(t, got)
	require.EqualError(t, err, "text must not be empty")

	got, err = NewComment("foo")
	require.NotNil(t, got)
	require.NoError(t, err)
	require.Equal(t, "foo", got.Text)

	task := MustNewTask("this is a test")
	require.NoError(t, task.AddComment("test comment"))
	require.Equal(t, 1, len(task.Comments))

	require.Error(t, task.AddComment(""))
}

func TestDescriptionDetails(t *testing.T) {
	task := MustNewTask("task with comments")
	require.NoError(t, task.AddComment("this is a comment"))
	require.True(t, strings.HasSuffix(task.DescriptionDetails(), "this is a comment"))
}
