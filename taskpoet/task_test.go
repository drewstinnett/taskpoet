package taskpoet

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"
)

// Local Client for lookups
var (
	lc            *Poet
	emptyDefaults Task
	router        *gin.Engine
	testDBPath    string
	// dbConfig      *taskpoet.DBConfig
)

func tempDB() string {
	tmpfile, err := os.CreateTemp("", "taskpoet.*.db")
	panicIfErr(err)
	return tmpfile.Name()
}

func newTestPoet() (*Poet, string) {
	testDBPath = tempDB()
	p, err := New(WithDatabasePath(testDBPath))
	panicIfErr(err)
	return p, testDBPath
}

func setup() {
	// Init a db
	tmpfile, err := os.CreateTemp("", "taskpoet.*.db")
	panicIfErr(err)
	testDBPath = tmpfile.Name()
	lc, err = New(WithDatabasePath(testDBPath))
	panicIfErr(err)
	emptyDefaults = Task{}

	// Init Router
	rc := &RouterConfig{
		LocalClient: lc,
	}
	router = NewRouter(rc)
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

/*
func TestAddLocalTask(t *testing.T) {
	tests := []struct {
		task Task
		path string
	}{
		{
			Task{ID: "add-1", Description: "foo-added-1"},
			"/active/add-1",
		},
	}

	for _, test := range tests {
		lc.Task.Add(&test.task, nil)
		gotTask := lc.Task.GetBy

	}

}
*/

func TestValidate(t *testing.T) {
	tests := []struct {
		task  Task
		valid bool
	}{
		{
			task:  Task{ID: "foo/bar", Description: "Invalid-id"},
			valid: false,
		},
	}

	for _, test := range tests {
		err := lc.Task.Validate(&test.task, nil)
		var valid bool
		if err != nil {
			valid = false
		} else {
			valid = true
		}
		if valid != test.valid {
			t.Errorf("Invalid result when testing validation. Wanted %v and got %v for %v", test.valid, valid, test.task)
		}
	}
}

func TestLogTask(t *testing.T) {
	_, err := lc.Task.Log(&Task{ID: "log-this-task", Description: "foo"}, &emptyDefaults)
	require.NoError(t, err)
	_, err = lc.Task.GetWithID("log-this-task", "", "/completed")
	require.NoError(t, err)
}

func TestCompleteTask(t *testing.T) {
	task, err := lc.Task.Add(&Task{Description: "soon-to-complete-task"})
	if err != nil {
		t.Error(err)
	}
	activePath := task.DetectKeyPath()

	err = lc.Task.Complete(task)
	if err != nil {
		t.Errorf("Error completing Task")
	}
	completePath := task.DetectKeyPath()

	_, err = lc.Task.GetWithExactPath(activePath)
	if err == nil {
		t.Errorf("When completing a task, the /active id is not removed")
	}

	_, err = lc.Task.GetWithExactPath(completePath)
	if err != nil {
		t.Errorf("When completing a task, the /completed id is not created")
	}
}

func TestBlankDescription(t *testing.T) {
	_, err := lc.Task.Add(&Task{})
	if err == nil {
		t.Error("Did not error on empty Description")
	}
}

func TestGetByPartialIDWithPath(t *testing.T) {
	ts := []Task{
		{Description: "foo", ID: "again with the fakeid again"},
		{Description: "foo", ID: "fakeid"},
		{Description: "foo", ID: "another_fakeid"},
		{Description: "foo", ID: "dupthing-num-1"},
		{Description: "foo", ID: "dupthing-num-2"},
		{Description: "foo"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	task, err := lc.Task.GetWithPartialID("fake", "", "/active")
	if err != nil {
		t.Error(err)
	} else if task.ID != "fakeid" {
		t.Errorf("Expected to retrieve 'fakeid' but got %v", task.ID)
	}

	// Test for a non-unique partial
	_, err = lc.Task.GetWithPartialID("dupthing", "", "/active")
	if err == nil {
		t.Error("Tried to get a partial that has duplicates, but got no error")
	}

	// Test for a non existent prefix
	_, err = lc.Task.GetWithPartialID("this-will-never-exist", "", "/active")
	if err == nil {
		t.Error("Tried to match on a non existent partial id, but did not error")
	}
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

// func TestGetByID(t *testing.T) {
func TestGetByExactPath(t *testing.T) {
	ts := []Task{
		{Description: "foo", ID: "id-stay-active"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	_, err = lc.Task.GetWithID("id-stay-active", "", "")
	if err != nil {
		t.Errorf("Could not GetWithID for id-stay-active")
	}

	// Check completed
	_, err = lc.Task.Log(&Task{ID: "id-in-completed", Description: "foo"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	_, err = lc.Task.GetWithID("id-in-completed", "", "")
	if err != nil {
		t.Errorf("Could not GetByID for id-in-completed")
	}
}

func TestDuplicateIDs(t *testing.T) {
	// Put something new in the completed bucket
	_, err := lc.Task.Log(&Task{ID: "duplicate-id", Description: "foo"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}

	// Try to create a new task with the same id
	_, err = lc.Task.Add(&Task{ID: "duplicate-id", Description: "foo"})
	if err == nil {
		t.Error("Creating a duplicate ID did not present an error")
	}

	// Make sure IDs and PluginIDs are UniqueTogether
	_, err = lc.Task.Add(&Task{ID: "duplicate-id-plugin", PluginID: "plugin-1", Description: "foo"})
	if err != nil {
		t.Error(err)
	}

	// Try to create a new task with the same id
	_, err = lc.Task.Add(&Task{ID: "duplicate-id-plugin", PluginID: "plugin-2", Description: "foo"})
	if err != nil {
		t.Error("Creating a duplicate ID with Plugin presented an error")
	}
}

func TestGetByExactID(t *testing.T) {
	ts := []Task{
		{Description: "foo", ID: "fakeid-exact"},
		{Description: "foo", ID: "another_fakeid-exact"},
		{Description: "foo"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	activePrefix := "/active"
	task, err := lc.Task.GetWithID("another_fakeid-exact", "", activePrefix)
	if err != nil {
		t.Error(err)
	} else if task.ID != "another_fakeid-exact" {
		t.Errorf("Expected to retrieve 'another_fakeid-exact' but got %v", task.ID)
	}

	_, err = lc.Task.GetWithID("another", "", activePrefix)
	if err == nil {
		t.Error("Did not error when checking for an exact id that does not exist")
	}
}

func TestListNonExistant(t *testing.T) {
	r, err := lc.Task.List("/never-exist")
	if err != nil {
		t.Error(err)
	}
	if len(r) != 0 {
		t.Errorf("Did not return an empty list when listing a non existent prefix")
	}
}

func TestAddParent(t *testing.T) {
	tasks := []Task{
		{ID: "kid", Description: "Kid task"},
		{ID: "parent", Description: "Parent task"},
	}

	err := lc.Task.AddSet(tasks, nil)
	if err != nil {
		t.Error(err)
	}
	kid, _ := lc.Task.GetWithID("kid", "", "/active")
	parent, _ := lc.Task.GetWithID("parent", "", "/active")

	// Make sure adding a parent works
	kid.Parents = append(kid.Parents, parent.ID)
	_, err = lc.Task.Edit(kid)
	if err != nil {
		t.Error(err)
	}

	// Make sure you can't add the same parent multiple times
	kid.Parents = append(kid.Parents, parent.ID)
	_, err = lc.Task.Edit(kid)
	if err == nil {
		t.Error("Adding the same parent twice did not generate an error")
	}
}

func TestTaskSelfAddParent(t *testing.T) {
	// Definitely don't add yourself to the parents array
	kid, _ := lc.Task.Add(&Task{ID: "test-self-add-parent", Description: "foo"})

	kid.Parents = append(kid.Parents, kid.ID)
	_, err := lc.Task.Edit(kid)
	if err == nil {
		t.Error("Adding the a task as it's own parent did not return an error")
	}
}

func TestTaskSelfAddChildren(t *testing.T) {
	// Definitely don't add yourself to the children array
	kid, _ := lc.Task.Add(&Task{ID: "test-self-add-child", Description: "foo"})

	kid.Children = append(kid.Children, kid.ID)
	_, err := lc.Task.Edit(kid)
	if err == nil {
		t.Error("Adding the a task as it's own children did not return an error")
	}
}

func TestShortID(t *testing.T) {
	tasks := []Task{
		{ID: "a", Description: "Short ID"},
		{ID: "foo-bar-baz-bazinga", Description: "Long ID"},
	}
	lc.Task.AddSet(tasks, nil)
	short, _ := lc.Task.GetWithID("a", "", "/active")
	long, _ := lc.Task.GetWithID("foo-bar-baz-bazinga", "", "/active")

	if short.ShortID() != "a" {
		t.Errorf("Short ID for %v did not return 'a'", short.ID)
	}
	if long.ShortID() != "foo-b" {
		t.Errorf("Short ID for %v did not return 'foo-b'", long.ID)
	}
}

func TestEditNonExisting(t *testing.T) {
	task := &Task{ID: "non-existing-edit"}
	_, err := lc.Task.Edit(task)
	if err == nil {
		t.Error("No error on editing a non existent task")
	}
}

func TestEditInvalid(t *testing.T) {
	task := &Task{ID: "soon-to-be-invalid", Description: "foo"}
	_, err := lc.Task.Add(task)
	if err != nil {
		t.Error(err)
	}

	task.Description = ""
	_, err = lc.Task.Edit(task)
	if err == nil {
		t.Error("Did not error when editing a task in to an invalid state")
	}
}

func TestEditCompletedInvalid(t *testing.T) {
	task := &Task{ID: "test-completed-edit", Description: "foo"}
	_, err := lc.Task.Add(task)
	if err != nil {
		t.Error(err)
	}

	n := time.Now()
	task.Completed = &n
	_, err = lc.Task.Edit(task)
	if err == nil {
		t.Error("Did not error when editing a task and changing the Completed field")
	}
}

func TestEditDescription(t *testing.T) {
	task := &Task{ID: "test-edit-description", Description: "original"}
	_, err := lc.Task.Add(task)
	if err != nil {
		t.Error(err)
	}

	task.Description = "New"
	edited, err := lc.Task.Edit(task)
	if err != nil {
		t.Error(err)
		return
	}
	if edited.Description != "New" {
		t.Error("Failed editing new description in Task")
	}
}

func TestEditSet(t *testing.T) {
	ts := []Task{
		{Description: "Foo", ID: "edit-set-1"},
		{Description: "Bar", ID: "edit-set-2"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}

	test1, _ := lc.Task.GetWithID("edit-set-1", "", "/active")
	test2, _ := lc.Task.GetWithID("edit-set-2", "", "/active")

	editSet := []Task{*test1, *test2}

	test2.Description = "New Description"
	err = lc.Task.EditSet(editSet)
	if err != nil {
		t.Error(err)
	}
}

func TestAddParentFunc(t *testing.T) {
	tasks := []Task{
		{ID: "kid-func", Description: "Kid task"},
		{ID: "parent-func", Description: "Parent task"},
	}

	err := lc.Task.AddSet(tasks, nil)
	if err != nil {
		t.Error(err)
	}

	// Get newly created items
	kid, _ := lc.Task.GetWithID("kid-func", "", "/active")
	parent, _ := lc.Task.GetWithID("parent-func", "", "/active")

	// Make sure adding a parent works
	err = lc.Task.AddParent(kid, parent)
	if err != nil {
		t.Error(err)
	}

	// Get newly Editted
	kid, _ = lc.Task.GetWithID("kid-func", "", "/active")
	parent, _ = lc.Task.GetWithID("parent-func", "", "/active")

	if !ContainsString(kid.Parents, parent.ID) {
		t.Error("Setting parent via functiono did not work")
	}

	if !ContainsString(parent.Children, kid.ID) {
		t.Error("Setting parent did not also set child on parent resource")
	}
}

func TestAddChildFunc(t *testing.T) {
	tasks := []Task{
		{ID: "kid-func2", Description: "Kid task"},
		{ID: "parent-func2", Description: "Parent task"},
	}

	err := lc.Task.AddSet(tasks, nil)
	if err != nil {
		t.Error(err)
	}

	// Get newly created items
	kid, _ := lc.Task.GetWithID("kid-func2", "", "/active")
	parent, _ := lc.Task.GetWithID("parent-func2", "", "/active")

	// Make sure adding a parent works
	err = lc.Task.AddChild(parent, kid)
	if err != nil {
		t.Error(err)
	}

	// Get newly Editted
	kid, _ = lc.Task.GetWithID("kid-func2", "", "/active")
	parent, _ = lc.Task.GetWithID("parent-func2", "", "/active")

	if !ContainsString(kid.Parents, parent.ID) {
		t.Error("Setting parent via functiono did not work")
	}

	if !ContainsString(parent.Children, kid.ID) {
		t.Error("Setting parent did not also set child on parent resource")
	}
}

func TestGetByPartialID(t *testing.T) {
	ts := []Task{
		{Description: "foo", ID: "partial-id-test"},
		{Description: "foo", ID: "partial-id-test-2"},
		{Description: "foo", ID: "unique-partial-id-test-2"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	task, err := lc.Task.GetWithPartialID("partial-id-test-2", "", "")
	if err != nil {
		t.Error(err)
	} else if task.ID != "partial-id-test-2" {
		t.Errorf("Expected to retrieve 'partial-id-test-2' but got %v", task.ID)
	}

	// Test for a non-unique partial
	_, err = lc.Task.GetWithPartialID("partial-id", "", "")
	if err == nil {
		t.Error("Tried to get a partial that has duplicates, but got no error")
	}

	// Test for a non existent prefix
	_, err = lc.Task.GetWithPartialID("this-will-never-exist", "", "/active")
	if err == nil {
		t.Error("Tried to match on a non existent partial id, but did not error")
	}
}

func TestDescribe(t *testing.T) {
	pterm.SetDefaultOutput(os.NewFile(0, os.DevNull))
	ts := []Task{
		{Description: "foo", ID: "describe-test"},
		{Description: "Some parent", ID: "describe-parent"},
	}
	lc.Task.AddSet(ts, &emptyDefaults)
	task, _ := lc.Task.GetWithID("describe-test", "builtin", "/active")
	taskP, _ := lc.Task.GetWithID("describe-parent", "builtin", "/active")
	lc.Task.Describe(task)

	// Describe with parent test
	lc.Task.AddParent(task, taskP)
	task, err := lc.Task.Edit(task)
	if err != nil {
		t.Error(err)
	}
	lc.Task.Describe(task)

	// Describe with parent test
	taskP, err = lc.Task.GetWithID("describe-parent", "builtin", "/active")
	if err != nil {
		t.Error(err)
	}

	lc.Task.Describe(taskP)

	// Describe a task with more things set
	n := time.Now()
	wait := n.Add(time.Hour * 1)
	due := n.Add(time.Hour * 24)
	completed := n.Add(time.Hour * 12)
	lc.Task.Describe(&Task{
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
	ts := &Task{
		ID:          "test-hide-after-due",
		Description: "test-hide-after-due",
		HideUntil:   &later,
		Due:         &sooner,
	}
	_, err := lc.Task.Add(ts)

	if err == nil {
		t.Error("Adding a task with hideuntil later than due did not produce an error", sooner, later)
	}
}

func TestDefaultBucketName(t *testing.T) {
	n := lc.Task.BucketName()
	assert.Equal(t, n, "/default/tasks")
}

func TestDeleteTask(t *testing.T) {
	ts := &Task{
		ID:          "delete-me",
		Description: "foo",
	}
	added, err := lc.Task.Add(ts)
	if err != nil {
		t.Error(err)
	}

	// Delete it now
	err = lc.Task.Delete(added)
	if err != nil {
		t.Error(err)
	}

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
			Task{ID: "foo", Description: "bar"},
			"/active/builtin/foo",
		},
		{
			Task{ID: "foo", Description: "bar", PluginID: "plugin-1"},
			"/active/plugin-1/foo",
		},
	}

	for _, test := range tests {
		got := string(test.task.DetectKeyPath())
		if got != test.wanted {
			t.Errorf("Failed DetectKeyPath, wanted %v but got %v", test.wanted, got)
		}
	}
}

func TestAddOrEditSet(t *testing.T) {
	ts := []Task{
		{Description: "Foo", ID: "add-or-edit-do-edit-1"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}

	aets := []Task{
		{Description: "Edited-desc", ID: "add-or-edit-do-edit-1"},
		{Description: "Added-desc", ID: "add-or-edit-do-add-1"},
	}

	err = lc.Task.AddOrEditSet(aets)
	if err != nil {
		t.Error(err)
	}

	edited, _ := lc.Task.GetWithID("add-or-edit-do-edit-1", "", "")
	added, _ := lc.Task.GetWithID("add-or-edit-do-add-1", "", "")

	assert.Equal(t, edited.Description, "Edited-desc")
	assert.Equal(t, added.Description, "Added-desc")
}

func TestEditExistingValues(t *testing.T) {
	ts := []Task{
		{Description: "Foo", ID: "edit-existing-1"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}

	aets := []Task{
		{Description: "Update", ID: "edit-existing-1"},
	}

	err = lc.Task.AddOrEditSet(aets)
	if err != nil {
		t.Error(err)
	}

	edited, _ := lc.Task.GetWithID("edit-existing-1", "", "")

	assert.Equal(t, false, edited.Added.IsZero())
}

func TestCompleteIDs(t *testing.T) {
	p, db := newTestPoet()
	defer os.RemoveAll(db)
	p.Task.Add(&Task{Description: "This is foo"})
	p.Task.Add(&Task{Description: "This is bar"})
	got := p.CompleteIDsWithPrefix("/active", "bar")
	require.True(t, strings.HasSuffix(got[0], "\tThis is bar"))
	require.Equal(t, 1, len(got))
}
