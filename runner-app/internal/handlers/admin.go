package handlers

import (
 "net/http"
 "sync"

 "github.com/gin-gonic/gin"
 "github.com/jamie-anson/project-beacon-runner/internal/config"
 "github.com/jamie-anson/project-beacon-runner/internal/middleware"
)

var (
mu  sync.RWMutex
cfg = config.DefaultAdminConfig()
)

func Health(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"ok": true})
}

func WhoAmI(c *gin.Context) {
 role := middleware.GetRole(c)
 c.JSON(http.StatusOK, gin.H{"role": role})
}

func GetAdminConfig(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != middleware.RoleAdmin && role != middleware.RoleOperator {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": role})
		return
	}
	mu.RLock()
	defer mu.RUnlock()
	c.JSON(http.StatusOK, cfg)
}

func PutAdminConfig(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != middleware.RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": role})
		return
	}
	var upd config.AdminConfigUpdate
	if err := c.ShouldBindJSON(&upd); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	mu.Lock()
	cfg = config.SanitizeAndMerge(cfg, upd)
	mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"ok": true, "config": cfg})
}
