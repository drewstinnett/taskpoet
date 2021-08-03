package taskpoet_test

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/drewstinnett/taskpoet/taskpoet"
	bolt "go.etcd.io/bbolt"
)

func TestInitDB(t *testing.T) {

	tmpfile, _ := ioutil.TempFile("", "taskpoet.*.db")
	log.Println(tmpfile.Name())

	dbConfig := &taskpoet.DBConfig{Path: tmpfile.Name()}
	err := taskpoet.InitDB(dbConfig)
	if err != nil {
		t.Error("Error initializing database test: ", err)
	}

	localClient, err := taskpoet.NewLocalClient(dbConfig)
	if err != nil {
		t.Error("Could not get db we just created: ", err)
	}
	err = localClient.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tasks"))
		if bucket == nil {
			t.Error("Could not get the task bucket on the new db")
		}
		return nil
	})
	if err != nil {
		t.Error("Error looking up bucket: ", err)
	}

}
