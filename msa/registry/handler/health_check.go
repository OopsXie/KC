package handler

import (
	"msa/registry/cluster"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleHealthCheck 处理健康检查请求
func HandleHealthCheck(c *gin.Context) {
	role := "slave"
	if cluster.Manager.IsMaster() {
		role = "master"
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"role":   role,
	})
}
