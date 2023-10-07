package taskpoet

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func mustTempDB(t *testing.T) string {
	return path.Join(t.TempDir(), "taskpoet.db")
}

func TestInitDB(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "taskpoet.*.db")

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

func TestActiveTable(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "taskpoet.*.db")
	p, err := New(WithDatabasePath(tmpfile.Name()))
	require.NoError(t, err)
	_, err = p.Task.Add(&Task{Description: "foo"})
	require.NoError(t, err)
	require.Contains(t, p.TaskTable(TableOpts{Prefix: "/active"}), "foo")
}

func TestCompletedTable(t *testing.T) {
	// tmpfile, _ := os.CreateTemp("", "taskpoet.*.db")
	p, err := New(WithDatabasePath(mustTempDB(t)))
	require.NoError(t, err)
	_, err = p.Task.Log(&Task{ID: "log-this-task", Description: "foo"}, &emptyDefaults)
	require.NoError(t, err)
	require.Contains(t, p.TaskTable(TableOpts{
		Prefix: "/completed", FilterParams: FilterParams{},
		Filters: []Filter{
			FilterHidden,
		},
	}), "foo")
}

func newTestPoet(t *testing.T) *Poet {
	return MustNew(
		WithDatabasePath(mustTempDB(t)),
	)
}

func TestDelete(t *testing.T) {
	p := newTestPoet(t)
	created, err := p.Task.Add(MustNewTask(WithDescription("about to delete this")))
	require.NoError(t, err)
	require.NotNil(t, created)

	// Delete it!
	results, err := p.Task.GetIDsByPrefix("/active")
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	require.NoError(t, p.Delete(created))

	// Make sure it's no longer here
	results, err = p.Task.GetIDsByPrefix("/active")
	require.NoError(t, err)
	require.Equal(t, 0, len(results))

	// Make sure it's in the deleted path
	results, err = p.Task.GetIDsByPrefix("/deleted")
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	require.NoError(t, p.Task.Purge(created))

	// Make sure it's purged out!
	results, err = p.Task.GetIDsByPrefix("/deleted")
	require.NoError(t, err)
	require.Equal(t, 0, len(results))
}
