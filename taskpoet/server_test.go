package taskpoet

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestPingRoute(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/ping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"message":"pong"}`, w.Body.String())
}

func TestSwaggerFile(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/static/v1/swagger.yaml", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
