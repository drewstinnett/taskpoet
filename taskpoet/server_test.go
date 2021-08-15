package taskpoet_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/go-playground/assert/v2"
)

func TestPingRoute(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/ping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"message":"pong"}`, w.Body.String())
}

func TestActiveRoute(t *testing.T) {
	_, err := lc.Task.Add(&taskpoet.Task{
		ID:          "test-active",
		Description: "foo",
	}, nil)
	if err != nil {
		t.Error(err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/active", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var tasks []taskpoet.Task
	err = json.Unmarshal(w.Body.Bytes(), &tasks)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, tasks[0].Description, "foo")
}
