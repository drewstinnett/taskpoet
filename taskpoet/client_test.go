package taskpoet

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestInitDB(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "taskpoet.*.db")
	log.Println(tmpfile.Name())

	dbConfig := &DBConfig{Path: tmpfile.Name()}
	err := InitDB(dbConfig)
	if err != nil {
		t.Error("Error initializing database test: ", err)
	}

	localClient, err := NewLocalClient(dbConfig)
	if err != nil {
		t.Error("Could not get db we just created: ", err)
	}
	err = localClient.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(localClient.Task.BucketName()))
		if bucket == nil {
			t.Error("Could not get the task bucket on the new db")
		}
		return nil
	})
	if err != nil {
		t.Error("Error looking up bucket: ", err)
	}
}

func TestNew(t *testing.T) {
	got, err := New(WithNamespace("foo"))
	require.NotNil(t, got)
	require.NoError(t, err)
	require.Equal(t, "foo", got.Namespace)
}
