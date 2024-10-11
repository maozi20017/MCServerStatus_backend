package api

import (
	"backend/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/api/server-status", handlers.GetServerStatus)
}
