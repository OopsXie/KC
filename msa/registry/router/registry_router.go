package router

import (
	"msa/registry/cluster"
	"msa/registry/handler"
	"msa/registry/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRegistryRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/api/register", handler.HandleRegister)
	r.POST("/api/unregister", handler.HandleUnregister)
	r.POST("/api/heartbeat", handler.HandleHeartbeat)
	r.GET("/api/discovery", handler.HandleDiscovery)
	r.GET("/health", handler.HandleHealthCheck) // 添加健康检查端点

	// 内部同步接口
	internal := r.Group("/api/internal")
	{
		internal.POST("/sync", handleSync)
		internal.GET("/loadbalance", handleGetLoadBalanceState) // 获取轮询状态
	}

	return r
}

// handleSync 处理内部同步请求
func handleSync(c *gin.Context) {
	var syncReq cluster.SyncRequest
	if err := c.ShouldBindJSON(&syncReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	success := cluster.Manager.HandleSlaveSync(syncReq.Action, syncReq.Instance)
	if success {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "同步失败"})
	}
}

// handleGetLoadBalanceState 获取轮询状态（仅主节点响应）
func handleGetLoadBalanceState(c *gin.Context) {
	// 只有主节点才能提供轮询状态
	if !cluster.Manager.IsMaster() {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有主节点能提供轮询状态"})
		return
	}

	loadBalanceState := storage.GetLoadBalanceState()
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取轮询状态成功",
		"data": loadBalanceState,
	})
}
