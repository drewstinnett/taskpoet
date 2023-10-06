package taskpoet

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTWMarshal(t *testing.T) {
	var got TaskWarriorTasks
	err := json.Unmarshal([]byte(`[{"id":0,"description":"Did something","end":"20230928T211203Z","entry":"20230928T211203Z","modified":"20230928T211203Z","status":"completed","uuid":"e0e59a1f-70dd-46a3-a6d8-a8f153d1c9bd","urgency":0},{"id":3,"description":"Do something later","entry":"20230928T211627Z","modified":"20230928T211627Z","status":"pending","uuid":"5fbfe931-7393-40d9-b282-9ea6f4aaaf51","wait":"20231005T211627Z","urgency":-3}]`), &got)
	require.NoError(t, err)
	require.Equal(t, "Did something", got[0].Description)
	require.EqualValues(t, "2023-09-28 21:12:03 +0000 UTC", got[0].Entry.String())
}

func TestTWImport(t *testing.T) {
	p, err := New(WithDatabasePath(mustTempDB(t)))
	require.NoError(t, err)
	futureT := TWTime(time.Now().Add(24 * time.Hour))
	pastT := TWTime(time.Now().Add(-24 * time.Hour))
	ts := TaskWarriorTasks{
		{Description: "foo-1"},
		{Description: "foo-2"},
		{Description: "foo is due", Due: &futureT},
		{Description: "bar is done", End: &pastT},
		{Description: "baz is waiting", Wait: &futureT},
		{Description: "quux has a tag or two", Tags: []string{"canary", "yearly-review"}},
		{
			Description: "something comment worthy",
			Annotations: []TWAnnotation{{Entry: &pastT, Description: "This is an annotation"}},
		},
	}
	got, err := p.ImportTaskWarrior(ts, nil)
	require.NoError(t, err)
	require.Equal(t, len(ts), got)
}
