package handlers

import (
 "net/http"
 "os"
 "sync"

 "github.com/gin-gonic/gin"
 "github.com/jamie-anson/project-beacon-runner/internal/config"
 "github.com/jamie-anson/project-beacon-runner/internal/db"
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

func TriggerMigration(c *gin.Context) {
	role := middleware.GetRole(c)
	if role != middleware.RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": role})
		return
	}

	// Force migration execution with golang-migrate
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DATABASE_URL not configured"})
		return
	}

	// Initialize database with forced migration
	database, err := db.Initialize(dbURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Migration failed", "details": err.Error()})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Migration triggered successfully"})
}
