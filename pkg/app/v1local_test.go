package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kholisrag/terraform-backend-gitops/pkg/config"
)

func TestRouterGroupV1Local(t *testing.T) {
	r := gin.Default()
	group := r.Group("/")

	config := &config.Config{} // Initialize your config here

	routerGroupV1Local(config, group)

	httpRecorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/local/", nil)
	r.ServeHTTP(httpRecorder, req)

	if status := httpRecorder.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"apiVersion":"v1","backend":"local"}`
	if httpRecorder.Body.String() != expected {
		t.Errorf("Handler returned unexpected body: got %v want %v",
			httpRecorder.Body.String(), expected)
	}
}
