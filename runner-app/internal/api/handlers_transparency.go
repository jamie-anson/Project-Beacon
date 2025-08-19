package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/internal/transparency"
)

// TransparencyHandler provides endpoints for transparency log and IPFS bundles.
type TransparencyHandler struct{}

func NewTransparencyHandler() *TransparencyHandler { return &TransparencyHandler{} }

// bundleFetcher is the minimal interface needed by GetBundle.
type bundleFetcher interface {
    FetchBundle(ctx context.Context, cid string) (*ipfs.Bundle, error)
}

// ipfsStorageFactory allows tests to inject a fake implementation.
var ipfsStorageFactory = func(client *ipfs.Client) bundleFetcher { return transparency.NewIPFSStorage(client) }

// GET /api/v1/transparency/root
func (h *TransparencyHandler) GetRoot(c *gin.Context) {
	root := transparency.DefaultWriter.Root()
	c.JSON(http.StatusOK, gin.H{"root": root})
}

// GET /api/v1/transparency/proof?index=<n>
func (h *TransparencyHandler) GetProof(c *gin.Context) {
	idxStr := c.Query("index")
	if idxStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "index is required"})
		return
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "index must be a non-negative integer"})
		return
	}
	if proof, ok := transparency.DefaultWriter.GetProof(idx); ok {
		c.JSON(http.StatusOK, proof)
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "proof not found"})
}

// GET /api/v1/transparency/bundles/:cid
func (h *TransparencyHandler) GetBundle(c *gin.Context) {
	cid := c.Param("cid")
	if cid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cid is required"})
		return
	}
	storage := ipfsStorageFactory(nil) // from env (overridable in tests)
	bundle, err := storage.FetchBundle(c.Request.Context(), cid)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	// Also return a gateway URL for convenience
	gateway := ipfs.NewFromEnv().GetGatewayURL(cid)
	c.JSON(http.StatusOK, gin.H{"bundle": bundle, "cid": cid, "gateway_url": gateway})
}
