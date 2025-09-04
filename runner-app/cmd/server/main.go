package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/handlers"
	"github.com/jamie-anson/project-beacon-runner/internal/middleware"
)

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,PUT,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), corsMiddleware(), middleware.AuthMiddleware())

	r.GET("/health", handlers.Health)
	r.GET("/auth/whoami", handlers.WhoAmI)
	r.GET("/admin/config", handlers.GetAdminConfig)
	r.PUT("/admin/config", handlers.PutAdminConfig)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}
	addr := ":" + port
	if err := r.Run(addr); err != nil {
		log.Printf("server error: %v", err)
	}
}
