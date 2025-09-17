package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/flags"
)

// FlagsHandler handles feature flag operations
type FlagsHandler struct{}

// NewFlagsHandler creates a new FlagsHandler
func NewFlagsHandler() *FlagsHandler {
	return &FlagsHandler{}
}

// GetFlags returns current feature flags
func (h *FlagsHandler) GetFlags(c *gin.Context) {
	c.JSON(http.StatusOK, flags.Get())
}

// UpdateFlags updates feature flags from JSON body
func (h *FlagsHandler) UpdateFlags(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unable to read body"})
		return
	}
	if err := flags.UpdateFromJSON(body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flags.Get())
}
