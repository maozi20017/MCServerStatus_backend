package main

import (
	"backend/internal/api"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// 根據環境變量設置 gin 模式
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)
	log.Printf("Gin mode: %s", ginMode)

	// 創建 gin 引擎
	r := gin.Default()

	// 設置路由
	api.SetupRoutes(r)
	log.Println("Routes set up successfully")

	// 獲取端口
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // 默認端口
	}

	// 啟動服務器
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
