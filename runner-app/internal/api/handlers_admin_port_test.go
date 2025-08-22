package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
)

func TestAdmin_GetPortInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		HTTPPort:       ":8090",
		PortStrategy:   "fallback",
		PortRangeStart: 8090,
		PortRangeEnd:   8099,
		AddrFile:       ".runner-http.addr",
		ResolvedAddr:   "[::]:8091",
	}
	h := NewAdminHandler(cfg)
	r := gin.New()
	r.GET("/admin/port", func(c *gin.Context) { h.GetPortInfo(c) })

	req := httptest.NewRequest(http.MethodGet, "/admin/port", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "\"strategy\":\"fallback\"")
	assert.Contains(t, body, "\"http_port\":\":8090\"")
	assert.Contains(t, body, "\"resolved_addr\":\"[::]:8091\"")
	assert.Contains(t, body, "\"range_start\":8090")
	assert.Contains(t, body, "\"range_end\":8099")
}
