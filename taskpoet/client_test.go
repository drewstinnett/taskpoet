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

	lc, err := New(WithDatabasePath(tmpfile.Name()))
	require.NoError(t, err)
	err = lc.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(lc.Task.BucketName()))
		require.NotNil(t, bucket)
		return nil
	})
	require.NoError(t, err)
}

func TestNew(t *testing.T) {
	got, err := New(WithNamespace("foo"))
	require.NotNil(t, got)
	require.NoError(t, err)
	require.Equal(t, "foo", got.Namespace)
}

func TestTable(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "taskpoet.*.db")
	p, err := New(WithDatabasePath(tmpfile.Name()))
	require.NoError(t, err)
	_, err = p.Task.Add(&Task{Description: "foo"}, nil)
	require.NoError(t, err)
	got := p.TaskTable("/active", nil)
	require.Contains(t, got, "foo")
}
