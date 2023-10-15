package taskpoet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSimpleRecursion(t *testing.T) {
	p, _ := New(
		WithDatabasePath(mustTempDB(t)),
		WithRecurringTasks(RecurringTasks{{
			Description: "do something frequently",
			Frequency:   time.Second * 1,
		}}),
	)
	time.Sleep(2 * time.Second)
	p.checkRecurring()
	items := p.MustList("/active")
	require.Equal(t, 1, len(items))
	require.Equal(t, "do something frequently", items[0].Description)
}

func TestSimpleRecursionHit(t *testing.T) {
	p, err := New(
		WithDatabasePath(mustTempDB(t)),
		WithRecurringTasks(RecurringTasks{{
			Description: "do something frequently",
			Frequency:   time.Minute * 1,
		}}),
	)
	now := time.Now()
	require.NoError(t, err)
	_, err = p.Task.Add(MustNewTask(
		WithDescription("do something frequently"),
		WithCompleted(&now),
	))
	require.NoError(t, err)
	p.checkRecurring()
	items := p.MustList("/active")
	require.Equal(t, 0, len(items))
	items = p.MustList("/completed")
	require.Equal(t, 1, len(items))
	require.Equal(t, "do something frequently", items[0].Description)
}
