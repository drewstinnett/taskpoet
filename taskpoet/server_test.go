package taskpoet_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drewstinnett/taskpoet/taskpoet"
	"github.com/go-playground/assert/v2"
	log "github.com/sirupsen/logrus"
)

func TestPingRoute(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/ping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"message":"pong"}`, w.Body.String())
}

func TestActiveRoute(t *testing.T) {
	ts := []taskpoet.Task{
		{
			ID:          "test-active",
			Description: "foo",
		},
		{
			ID:          "test-active-2",
			Description: "foo",
		},
	}
	err := lc.Task.AddSet(ts, nil)
	if err != nil {
		t.Error(err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/active?limit=1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var apir taskpoet.APITaskResponse
	//var tasks []taskpoet.Task
	log.Warning(string(w.Body.Bytes()))
	err = json.Unmarshal(w.Body.Bytes(), &apir)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, apir.Data[0].Description, "foo")
	assert.Equal(t, apir.Pagination.HasMore, true)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/v1/active?limit=100", nil)
	router.ServeHTTP(w, req)

	err = json.Unmarshal(w.Body.Bytes(), &apir)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, apir.Pagination.HasMore, false)

}

func TestSwaggerFile(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/static/v1/swagger.yaml", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

}
