package handlers

import (
	mcstatus "backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetServerStatus(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "伺服器地址不能為空"})
		return
	}

	status, err := mcstatus.GetServerStatus(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}
