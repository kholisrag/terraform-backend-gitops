// Create server.go test file
package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
)

func TestNewAppHealthz(t *testing.T) {
	router := NewApp(config.NewDefaultConfig())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "{\"status\":\"ok\"}", w.Body.String())
}

func TestNewAppVersion(t *testing.T) {
	router := NewApp(config.NewDefaultConfig())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/version", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.MatchRegex(t, w.Body.String(), `{"build":".*","commit":".*","version":".*"}`)
}
