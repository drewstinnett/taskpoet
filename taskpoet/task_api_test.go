package taskpoet

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
)

func TestActiveRoute(t *testing.T) {
	ts := Tasks{
		{
			ID:          "test-active",
			Description: "foo",
		},
		{
			ID:          "test-active-2",
			Description: "foo",
		},
	}
	require.NoError(t, lc.Task.AddSet(ts))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/tasks?limit=1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var apir APITaskResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apir))
	assert.Equal(t, apir.Data[0].Description, "foo")
	assert.Equal(t, apir.Pagination.HasMore, true)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tasks?limit=100", nil)
	router.ServeHTTP(w, req)

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apir))
	assert.Equal(t, apir.Pagination.HasMore, false)
}

func TestCompleted(t *testing.T) {
	now := time.Now()
	ts := Tasks{
		{
			ID:          "test-completed",
			Description: "foo-completed",
			Completed:   &now,
		},
		{
			ID:          "test-completed-2",
			Description: "foo-completed-2",
			Completed:   &now,
		},
	}
	require.NoError(t, lc.Task.AddSet(ts))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/tasks?limit=1&include_completed=true&include_active=false", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var apir APITaskResponse
	// var tasks []Task
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &apir))
	assert.Equal(t, apir.Data[0].Description, "foo-completed")
	assert.Equal(t, apir.Pagination.HasMore, true)
}

func TestIncludeIssues(t *testing.T) {
	// Make sure include_completed not set to bool errors
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/tasks?include_completed=foo", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)

	// Make sure include_active not set to bool errors
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tasks?include_active=foo", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)

	// Make sure either include_active or include_completed is set to true
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tasks?include_active=false&include_completed=false", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
}

func TestAddTask(t *testing.T) {
	// Make sure include_completed not set to bool errors
	task := Task{
		ID:          "foo",
		Description: "foo",
	}
	taskB, _ := json.Marshal(task)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/tasks", bytes.NewBuffer(taskB))
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestGetTask(t *testing.T) {
	ts := Tasks{
		{
			ID:          "test",
			Description: "foo",
		},
	}
	require.NoError(t, lc.Task.AddSet(ts))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/tasks/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tasks/test404", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestEditTask(t *testing.T) {
	ts := Tasks{
		{
			ID:          "test-edit",
			Description: "orig",
		},
	}
	require.NoError(t, lc.Task.AddSet(ts))

	edit := Task{
		ID:          "test-edit",
		Description: "new-desc",
	}
	editB, _ := json.Marshal(edit)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/v1/tasks/test-edit", bytes.NewBuffer(editB))
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	var gotTask Task
	json.Unmarshal(w.Body.Bytes(), &gotTask)
	assert.Equal(t, gotTask.Description, "new-desc")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/v1/tasks/test404", bytes.NewBuffer(editB))
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestDeleteTest(t *testing.T) {
	ts := Tasks{
		{
			ID:          "test-delete",
			Description: "orig",
		},
	}
	require.NoError(t, lc.Task.AddSet(ts))

	taskB, err := json.Marshal(ts)
	require.NoError(t, err)

	// Make sure delete succeeds
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/v1/tasks/test-delete", bytes.NewBuffer(taskB))
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Make sure it's gone
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/tasks/test-delete", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}
