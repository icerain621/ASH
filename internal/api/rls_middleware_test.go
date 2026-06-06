package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRLSBypassForRequestRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/api/v1/orgs", true},
		{http.MethodGet, "/api/v1/spaces", true},
		{http.MethodPost, "/api/v1/orgs", false},
		{http.MethodGet, "/api/v1/runs", false},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(tc.method, tc.path, nil)
		got := h.rlsBypassRouteMatch(c)
		if got != tc.want {
			t.Fatalf("%s %s: got %v want %v", tc.method, tc.path, got, tc.want)
		}
	}
}
