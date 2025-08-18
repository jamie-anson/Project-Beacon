package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Missing API handler methods - stub implementations

func (s *APIServer) getJob(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "getJob not implemented yet"})
}

func (s *APIServer) listJobs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "listJobs not implemented yet"})
}

func (s *APIServer) deleteJob(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "deleteJob not implemented yet"})
}

func (s *APIServer) listDiffs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "listDiffs not implemented yet"})
}

func (s *APIServer) getDiff(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "getDiff not implemented yet"})
}

func (s *APIServer) executeJob(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "executeJob not implemented yet"})
}

func (s *APIServer) getExecution(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "getExecution not implemented yet"})
}

func (s *APIServer) listExecutions(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "listExecutions not implemented yet"})
}

func (s *APIServer) cancelExecution(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "cancelExecution not implemented yet"})
}

func (s *APIServer) discoverProviders(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "discoverProviders not implemented yet"})
}

func (s *APIServer) getProvider(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "getProvider not implemented yet"})
}

func (s *APIServer) latestResult(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "latestResult not implemented yet"})
}

func (s *APIServer) getExecutionDiffs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "getExecutionDiffs not implemented yet"})
}

func (s *APIServer) analyzeDiffs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "analyzeDiffs not implemented yet"})
}
