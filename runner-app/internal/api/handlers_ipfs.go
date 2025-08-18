package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// IPFS bundle handlers

// createIPFSBundle creates and stores an IPFS bundle for a completed job
func (s *APIServer) createIPFSBundle(c *gin.Context) {
	jobID := c.Param("id")

	if s.ipfsBundler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "IPFS service not available",
		})
		return
	}

	cid, err := s.ipfsBundler.StoreBundle(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create IPFS bundle",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "IPFS bundle created successfully",
		"job_id": jobID,
		"cid": cid,
		"gateway_url": s.ipfsClient.GetGatewayURL(cid),
	})
}

// getIPFSBundle retrieves bundle metadata by CID
func (s *APIServer) getIPFSBundle(c *gin.Context) {
	cid := c.Param("cid")

	if s.ipfsRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "IPFS service not available",
		})
		return
	}

	bundle, err := s.ipfsRepo.GetBundleByCID(cid)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Bundle not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve bundle",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bundle": bundle,
		"gateway_url": s.ipfsClient.GetGatewayURL(cid),
	})
}

// listIPFSBundles lists IPFS bundles with optional filtering
func (s *APIServer) listIPFSBundles(c *gin.Context) {
	if s.ipfsRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "IPFS service not available",
		})
		return
	}

	jobID := c.Query("job_id")

	var bundles []store.IPFSBundle
	var err error

	if jobID != "" {
		bundles, err = s.ipfsRepo.GetBundlesByJobID(jobID)
	} else {
		bundles, err = s.ipfsRepo.ListBundles(0, 100) // Default pagination
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list bundles",
			"details": err.Error(),
		})
		return
	}

	bundleResponses := make([]map[string]interface{}, len(bundles))
	for i, bundle := range bundles {
		bundleResponses[i] = map[string]interface{}{
			"id":              bundle.ID,
			"job_id":          bundle.JobID,
			"cid":             bundle.CID,
			"bundle_size":     bundle.BundleSize,
			"execution_count": bundle.ExecutionCount,
			"regions":         bundle.Regions,
			"created_at":      bundle.CreatedAt,
			"pinned_at":       bundle.PinnedAt,
			"gateway_url":     s.ipfsClient.GetGatewayURL(bundle.CID),
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"bundles": bundleResponses,
		"count": len(bundleResponses),
	})
}
