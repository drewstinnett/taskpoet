package taskpoet_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/drewstinnett/taskpoet/taskpoet"
	bolt "go.etcd.io/bbolt"
)

func TestLogTask(t *testing.T) {
	// Init a db
	tmpfile, _ := ioutil.TempFile("", "taskpoet.*.db")
	dbConfig := &taskpoet.DBConfig{Path: tmpfile.Name()}
	_ = taskpoet.InitDB(dbConfig)
	localClient, _ := taskpoet.NewLocalClient(dbConfig)
	defaults := taskpoet.Task{}

	task, err := localClient.Task.Log(&taskpoet.Task{Description: "log-this-task"}, &defaults)
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
	// Init a db
	tmpfile, _ := ioutil.TempFile("", "taskpoet.*.db")
	dbConfig := &taskpoet.DBConfig{Path: tmpfile.Name()}
	_ = taskpoet.InitDB(dbConfig)
	localClient, _ := taskpoet.NewLocalClient(dbConfig)
	defaults := taskpoet.Task{}

	task, _ := localClient.Task.New(&taskpoet.Task{Description: "soon-to-complete-task"}, &defaults)

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

	log.Println(task.ID)
	//	t.Errorf("Fail")

}

func TestBlankDescription(t *testing.T) {
	// Init a db
	tmpfile, _ := ioutil.TempFile("", "taskpoet.*.db")
	dbConfig := &taskpoet.DBConfig{Path: tmpfile.Name()}
	_ = taskpoet.InitDB(dbConfig)
	localClient, _ := taskpoet.NewLocalClient(dbConfig)
	defaults := taskpoet.Task{}

	_, err := localClient.Task.New(&taskpoet.Task{}, &defaults)
	if err == nil {
		t.Error("Did not error on empty Description")
	}
}

func TestDefaults(t *testing.T) {
	tmpfile, _ := ioutil.TempFile("", "taskpoet.*.db")
	dbConfig := &taskpoet.DBConfig{Path: tmpfile.Name()}
	_ = taskpoet.InitDB(dbConfig)
	localClient, _ := taskpoet.NewLocalClient(dbConfig)
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
