package taskpoet_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/drewstinnett/taskpoet/taskpoet"
	bolt "go.etcd.io/bbolt"
)

// Local Client for lookups
var lc *taskpoet.LocalClient
var emptyDefaults taskpoet.Task

func setup() {
	// Init a db
	tmpfile, _ := ioutil.TempFile("", "taskpoet.*.db")
	dbConfig := &taskpoet.DBConfig{Path: tmpfile.Name()}
	_ = taskpoet.InitDB(dbConfig)
	lc, _ = taskpoet.NewLocalClient(dbConfig)
	emptyDefaults = taskpoet.Task{}

	// Populate with some various tasks to filter on
}
func shutdown() {}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestLogTask(t *testing.T) {

	_, err := lc.Task.Log(&taskpoet.Task{ID: "log-this-task", Description: "foo"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	_, err = lc.Task.GetByIDWithPrefix("log-this-task", "/completed")
	if err != nil {
		t.Error(err)
	}

}

func TestCompleteTask(t *testing.T) {

	task, err := lc.Task.Add(&taskpoet.Task{Description: "soon-to-complete-task"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}

	err = lc.Task.Complete(task)
	if err != nil {
		t.Errorf("Error completing Task")
	}

	// Now make sure we actually did something
	err = lc.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		old := b.Get([]byte(fmt.Sprintf("/active/%s", task.ID)))
		if old != nil {
			t.Errorf("When completing a task, the /active id is not removed")
		}
		new := b.Get([]byte(fmt.Sprintf("/completed/%s", task.ID)))
		if new == nil {
			t.Errorf("When completing a task, the /completed id is not created")
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestBlankDescription(t *testing.T) {
	_, err := lc.Task.Add(&taskpoet.Task{}, &emptyDefaults)
	if err == nil {
		t.Error("Did not error on empty Description")
	}
}

func TestGetByPartialIDWithPath(t *testing.T) {
	ts := []taskpoet.Task{
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
	task, err := lc.Task.GetByPartialIDWithPath("fake", "/active")
	if err != nil {
		t.Error(err)
	} else if task.ID != "fakeid" {
		t.Errorf("Expected to retrive 'fakeid' but got %v", task.ID)
	}

	// Test for a non-unique partial
	_, err = lc.Task.GetByPartialIDWithPath("dupthing", "/active")
	if err == nil {
		t.Error("Tried to get a partial that has duplicates, but got no error")
	}

	// Test for a non existant prefix
	_, err = lc.Task.GetByPartialIDWithPath("this-will-never-exist", "/active")
	if err == nil {
		t.Error("Tried to match on a non existant partial id, but did not error")
	}

}

func TestDefaults(t *testing.T) {
	defaults := taskpoet.Task{}

	// Set some defaults
	now := time.Now()
	fakeDuration, _ := taskpoet.ParseDuration("2h")
	duration := now.Add(fakeDuration)
	defaults.Due = duration

	task, _ := lc.Task.Add(&taskpoet.Task{Description: "foo"}, &defaults)

	if task.Due != duration {
		t.Error("Default setting of Due did not work")
	}

}
func TestGetByID(t *testing.T) {
	ts := []taskpoet.Task{
		{Description: "foo", ID: "id-stay-active"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	_, err = lc.Task.GetByID("id-stay-active")
	if err != nil {
		t.Errorf("Could not GetByID for id-stay-active")
	}

	// Check completed
	_, err = lc.Task.Log(&taskpoet.Task{ID: "id-in-completed", Description: "foo"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	_, err = lc.Task.GetByID("id-in-completed")
	if err != nil {
		t.Errorf("Could not GetByID for id-in-completed")
	}

}

func TestDuplicateIDs(t *testing.T) {
	// Put something new in the completed bucket
	_, err := lc.Task.Log(&taskpoet.Task{ID: "duplicate-id", Description: "foo"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}

	// Try to create a new task with the same id
	_, err = lc.Task.Add(&taskpoet.Task{ID: "duplicate-id", Description: "foo"}, &emptyDefaults)
	if err == nil {
		t.Error("Creating a duplicate ID did not present an error")
	}

}

func TestGetByExactID(t *testing.T) {
	ts := []taskpoet.Task{
		{Description: "foo", ID: "fakeid"},
		{Description: "foo", ID: "another_fakeid"},
		{Description: "foo"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	activePrefix := "/active"
	task, err := lc.Task.GetByIDWithPrefix("another_fakeid", activePrefix)
	if err != nil {
		t.Error(err)
	} else if task.ID != "another_fakeid" {
		t.Errorf("Expected to retrive 'another_fakeid' but got %v", task.ID)
	}

	_, err = lc.Task.GetByIDWithPrefix("another", activePrefix)
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
		t.Errorf("Did not return an empty list when listing a non existant prefix")
	}
}

func TestAddParent(t *testing.T) {
	tasks := []taskpoet.Task{
		{ID: "kid", Description: "Kid task"},
		{ID: "parent", Description: "Parent task"},
	}

	err := lc.Task.AddSet(tasks, nil)
	if err != nil {
		t.Error(err)
	}
	kid, _ := lc.Task.GetByIDWithPrefix("kid", "/active")
	parent, _ := lc.Task.GetByIDWithPrefix("parent", "/active")

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
	kid, _ := lc.Task.Add(&taskpoet.Task{ID: "test-self-add-parent", Description: "foo"}, nil)

	kid.Parents = append(kid.Parents, kid.ID)
	_, err := lc.Task.Edit(kid)
	if err == nil {
		t.Error("Adding the a task as it's own parent did not return an error")
	}

}

func TestTaskSelfAddChildren(t *testing.T) {
	// Definitely don't add yourself to the children array
	kid, _ := lc.Task.Add(&taskpoet.Task{ID: "test-self-add-child", Description: "foo"}, nil)

	kid.Children = append(kid.Children, kid.ID)
	_, err := lc.Task.Edit(kid)
	if err == nil {
		t.Error("Adding the a task as it's own children did not return an error")
	}

}

func TestShortID(t *testing.T) {
	tasks := []taskpoet.Task{
		{ID: "a", Description: "Short ID"},
		{ID: "foo-bar-baz-bazinga", Description: "Long ID"},
	}
	lc.Task.AddSet(tasks, nil)
	short, _ := lc.Task.GetByIDWithPrefix("a", "/active")
	long, _ := lc.Task.GetByIDWithPrefix("foo-bar-baz-bazinga", "/active")

	if short.ShortID() != "a" {
		t.Errorf("Short ID for %v did not return 'a'", short.ID)
	}
	if long.ShortID() != "foo-b" {
		t.Errorf("Short ID for %v did not return 'foo-b'", long.ID)
	}

}

func TestEditNonExisting(t *testing.T) {
	task := &taskpoet.Task{ID: "non-existing-edit"}
	_, err := lc.Task.Edit(task)
	if err == nil {
		t.Error("No error on editing a non existant task")
	}
}

func TestEditInvalid(t *testing.T) {
	task := &taskpoet.Task{ID: "soon-to-be-invalid", Description: "foo"}
	_, err := lc.Task.Add(task, nil)
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
	task := &taskpoet.Task{ID: "test-completed-edit", Description: "foo"}
	_, err := lc.Task.Add(task, nil)
	if err != nil {
		t.Error(err)
	}

	task.Completed = time.Now()
	_, err = lc.Task.Edit(task)
	if err == nil {
		t.Error("Did not error when editing a task and changing the Completed field")
	}
}

func TestEditDescription(t *testing.T) {
	task := &taskpoet.Task{ID: "test-edit-description", Description: "original"}
	lc.Task.Add(task, nil)

	task.Description = "New"
	edited, err := lc.Task.Edit(task)
	if err != nil {
		t.Error(err)
	}
	if edited.Description != "New" {
		t.Error("Failed editing new description in Task")
	}

}

func TestEditSet(t *testing.T) {
	ts := []taskpoet.Task{
		{Description: "Foo", ID: "edit-set-1"},
		{Description: "Bar", ID: "edit-set-2"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}

	test1, _ := lc.Task.GetByIDWithPrefix("edit-set-1", "/active")
	test2, _ := lc.Task.GetByIDWithPrefix("edit-set-2", "/active")

	editSet := []taskpoet.Task{*test1, *test2}

	test2.Description = "New Description"
	err = lc.Task.EditSet(editSet)
	if err != nil {
		t.Error(err)
	}

}
func TestAddParentFunc(t *testing.T) {
	tasks := []taskpoet.Task{
		{ID: "kid-func", Description: "Kid task"},
		{ID: "parent-func", Description: "Parent task"},
	}

	err := lc.Task.AddSet(tasks, nil)
	if err != nil {
		t.Error(err)
	}

	// Get newly created items
	kid, _ := lc.Task.GetByIDWithPrefix("kid-func", "/active")
	parent, _ := lc.Task.GetByIDWithPrefix("parent-func", "/active")

	// Make sure adding a parent works
	err = lc.Task.AddParent(kid, parent)
	if err != nil {
		t.Error(err)
	}

	// Get newly Editted
	kid, _ = lc.Task.GetByIDWithPrefix("kid-func", "/active")
	parent, _ = lc.Task.GetByIDWithPrefix("parent-func", "/active")

	if !taskpoet.ContainsString(kid.Parents, parent.ID) {
		t.Error("Setting parent via functiono did not work")
	}

	if !taskpoet.ContainsString(parent.Children, kid.ID) {
		t.Error("Setting parent did not also set child on parent resource")
	}

}

func TestAddChildFunc(t *testing.T) {
	tasks := []taskpoet.Task{
		{ID: "kid-func2", Description: "Kid task"},
		{ID: "parent-func2", Description: "Parent task"},
	}

	err := lc.Task.AddSet(tasks, nil)
	if err != nil {
		t.Error(err)
	}

	// Get newly created items
	kid, _ := lc.Task.GetByIDWithPrefix("kid-func2", "/active")
	parent, _ := lc.Task.GetByIDWithPrefix("parent-func2", "/active")

	// Make sure adding a parent works
	err = lc.Task.AddChild(parent, kid)
	if err != nil {
		t.Error(err)
	}

	// Get newly Editted
	kid, _ = lc.Task.GetByIDWithPrefix("kid-func2", "/active")
	parent, _ = lc.Task.GetByIDWithPrefix("parent-func2", "/active")

	if !taskpoet.ContainsString(kid.Parents, parent.ID) {
		t.Error("Setting parent via functiono did not work")
	}

	if !taskpoet.ContainsString(parent.Children, kid.ID) {
		t.Error("Setting parent did not also set child on parent resource")
	}

}

func TestGetByPartialID(t *testing.T) {
	ts := []taskpoet.Task{
		{Description: "foo", ID: "partial-id-test"},
		{Description: "foo", ID: "partial-id-test-2"},
		{Description: "foo", ID: "unique-partial-id-test-2"},
	}
	err := lc.Task.AddSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	task, err := lc.Task.GetByPartialID("partial-id-test-2")
	if err != nil {
		t.Error(err)
	} else if task.ID != "partial-id-test-2" {
		t.Errorf("Expected to retrive 'partial-id-test-2' but got %v", task.ID)
	}

	// Test for a non-unique partial
	_, err = lc.Task.GetByPartialID("partial-id")
	if err == nil {
		t.Error("Tried to get a partial that has duplicates, but got no error")
	}

	// Test for a non existant prefix
	_, err = lc.Task.GetByPartialIDWithPath("this-will-never-exist", "/active")
	if err == nil {
		t.Error("Tried to match on a non existant partial id, but did not error")
	}

}

func TestDescribe(t *testing.T) {
	ts := []taskpoet.Task{
		{Description: "foo", ID: "describe-test"},
		{Description: "Some parent", ID: "describe-parent"},
	}
	lc.Task.AddSet(ts, &emptyDefaults)
	task, _ := lc.Task.GetByPartialIDWithPath("describe-test", "/active")
	taskP, _ := lc.Task.GetByPartialIDWithPath("describe-parent", "/active")
	lc.Task.Describe(task)

	// Describe with parent test
	lc.Task.AddParent(task, taskP)
	task, err := lc.Task.Edit(task)
	if err != nil {
		t.Error(err)
	}
	lc.Task.Describe(task)

	// Describe with parent test
	taskP, err = lc.Task.GetByPartialIDWithPath("describe-parent", "/active")
	if err != nil {
		t.Error(err)
	}

	lc.Task.Describe(taskP)

}

func TestHideAfterDue(t *testing.T) {
	now := time.Now()
	sooner := now.Add(time.Minute * 5)
	later := now.Add(time.Minute * 10)
	ts := &taskpoet.Task{
		ID:          "test-hide-after-due",
		Description: "test-hide-after-due",
		HideUntil:   later,
		Due:         sooner,
	}
	_, err := lc.Task.Add(ts, nil)

	if err == nil {
		t.Error("Adding a task with hideuntil later than due did not produce an error")
	}
}
