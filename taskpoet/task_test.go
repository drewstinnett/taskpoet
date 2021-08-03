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

var localClient *taskpoet.LocalClient
var emptyDefaults taskpoet.Task

func setup() {
	// Init a db
	tmpfile, _ := ioutil.TempFile("", "taskpoet.*.db")
	dbConfig := &taskpoet.DBConfig{Path: tmpfile.Name()}
	_ = taskpoet.InitDB(dbConfig)
	localClient, _ = taskpoet.NewLocalClient(dbConfig)
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

	task, err := localClient.Task.Log(&taskpoet.Task{Description: "log-this-task"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	err = localClient.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		b.Get([]byte(fmt.Sprintf("completed/%s", task.ID)))
		return nil
	})
	if err != nil {
		t.Error(err)
	}

}

func TestCompleteTask(t *testing.T) {

	task, _ := localClient.Task.New(&taskpoet.Task{Description: "soon-to-complete-task"}, &emptyDefaults)

	err := localClient.Task.Complete(task)
	if err != nil {
		t.Errorf("Error completing Task")
	}

	// Now make sure we actually did something
	err = localClient.DB.View(func(tx *bolt.Tx) error {
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
	_, err := localClient.Task.New(&taskpoet.Task{}, &emptyDefaults)
	if err == nil {
		t.Error("Did not error on empty Description")
	}
}

func TestGetByPartialID(t *testing.T) {
	ts := []taskpoet.Task{
		{Description: "foo", ID: "again with the fakeid again"},
		{Description: "foo", ID: "fakeid"},
		{Description: "foo", ID: "another_fakeid"},
		{Description: "foo"},
	}
	err := localClient.Task.NewSet(ts, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	task, err := localClient.Task.GetByPartialID("fake", "/active")
	if err != nil {
		t.Error(err)
	} else if task.ID != "fakeid" {
		t.Errorf("Expected to retrive 'fakeid' but got %v", task.ID)
	}

}

func TestDefaults(t *testing.T) {
	defaults := taskpoet.Task{}

	// Set some defaults
	now := time.Now()
	fakeDuration, _ := taskpoet.ParseDuration("2h")
	duration := now.Add(fakeDuration)
	defaults.Due = duration

	task, _ := localClient.Task.New(&taskpoet.Task{Description: "foo"}, &defaults)

	if task.Due != duration {
		t.Error("Default setting of Due did not work")
	}

}

func TestGetByExactD(t *testing.T) {
	_, err := localClient.Task.New(&taskpoet.Task{Description: "foo", ID: "fakeid"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	_, err = localClient.Task.New(&taskpoet.Task{Description: "foo", ID: "another_fakeid"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	_, err = localClient.Task.New(&taskpoet.Task{Description: "foo"}, &emptyDefaults)
	if err != nil {
		t.Error(err)
	}
	task, err := localClient.Task.GetByExactID("another_fakeid", "/active")
	if err != nil {
		t.Error(err)
	} else if task.ID != "another_fakeid" {
		t.Errorf("Expected to retrive 'another_fakeid' but got %v", task.ID)
	}

}

func TestListNonExistant(t *testing.T) {
	r, err := localClient.Task.List("/never-exist")
	if err != nil {
		t.Error(err)
	}
	if len(r) != 0 {
		t.Errorf("Did not return an empty list when listing a non existant prefix")
	}
}
