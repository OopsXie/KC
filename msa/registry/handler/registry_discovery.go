package handler

import (
	"io"
	"log"
	"msa/registry/cluster"
	"msa/registry/models"
	"msa/registry/storage"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HandleDiscovery 处理 /api/discovery 接口
func HandleDiscovery(c *gin.Context) {
	serviceName := c.Query("name")

	// 记录当前节点状态
	isMaster := cluster.Manager.IsMaster()
	currentMaster := cluster.Manager.GetMaster()
	log.Printf("[discovery] 处理服务发现请求: serviceName=%s, 当前节点是否主节点=%v, 当前主节点=%s",
		serviceName, isMaster, currentMaster)

	// 如果当前节点不是主节点，转发请求到主节点
	if !isMaster {
		log.Printf("[discovery] 当前节点非主节点，转发请求到主节点: %s", currentMaster)
		forwardDiscoveryToMaster(c)
		return
	}

	if serviceName == "" {
		// 不带 name 参数：返回全部服务实例列表
		allInstances := storage.GetAllInstances()

		if len(allInstances) == 0 {
			// 没有任何服务
			successData := models.DiscoverySuccessData{
				TotalCount: 0,
				Instances:  []models.DiscoveryInstanceData{},
			}
			response := models.SuccessResponse(200, "没有服务", successData)
			c.JSON(http.StatusOK, response)
			return
		}

		// 转换为响应格式
		var instances []models.DiscoveryInstanceData
		for _, inst := range allInstances {
			instances = append(instances, models.DiscoveryInstanceData{
				ServiceName:          inst.ServiceName,
				ServiceID:            inst.ServiceID,
				IPAddress:            inst.IPAddress,
				Port:                 inst.Port,
				RegistrationTime:     inst.RegisteredAt,
				LastHeartbeatTime:    inst.LastHeartbeat,
				RegistrationGMTTime:  inst.RegisteredGMTTime,
				LastHeartbeatGMTTime: inst.LastHeartbeatGMTTime,
			})
		}

		successData := models.DiscoverySuccessData{
			TotalCount: len(instances),
			Instances:  instances,
		}
		response := models.SuccessResponse(200, "获取所有服务实例成功", successData)
		c.JSON(http.StatusOK, response)
		return
	}

	// 带 name 参数：返回一个实例（轮询）
	instance := storage.SelectOneInstance(serviceName)
	if instance == nil {
		errorData := models.DiscoveryErrorData{
			ServiceName: serviceName,
			Suggestion:  "请检查服务名称是否正确，或确认该服务已注册",
		}
		response := models.ErrorResponse(404, "service not found", errorData)
		c.JSON(http.StatusNotFound, response)
		return
	}

	// 成功找到服务实例
	instances := []models.DiscoveryInstanceData{
		{
			ServiceName:          instance.ServiceName,
			ServiceID:            instance.ServiceID,
			IPAddress:            instance.IPAddress,
			Port:                 instance.Port,
			RegistrationTime:     instance.RegisteredAt,
			LastHeartbeatTime:    instance.LastHeartbeat,
			RegistrationGMTTime:  instance.RegisteredGMTTime,
			LastHeartbeatGMTTime: instance.LastHeartbeatGMTTime,
		},
	}

	successData := models.DiscoverySuccessData{
		ServiceName: serviceName,
		TotalCount:  1,
		Instances:   instances,
	}
	response := models.SuccessResponse(200, "服务发现成功", successData)
	c.JSON(http.StatusOK, response)
}

// forwardDiscoveryToMaster 将discovery请求转发到主节点
func forwardDiscoveryToMaster(c *gin.Context) {
	masterAddr := cluster.Manager.GetMaster()
	if masterAddr == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "主节点不可用",
		})
		return
	}

	// 构建转发URL
	forwardURL := masterAddr + "/api/discovery"
	if serviceName := c.Query("name"); serviceName != "" {
		forwardURL += "?name=" + serviceName
	}

	// 设置较短的超时时间，用于快速检测故障
	client := &http.Client{
		Timeout: 3 * time.Second, // 3秒超时
	}

	req, err := http.NewRequest("GET", forwardURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建转发请求失败",
		})
		return
	}

	// 复制原始请求的headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		// 转发失败，可能是主节点故障，触发故障检测
		log.Printf("[discovery] 转发到主节点 %s 失败: %v，触发故障检测", masterAddr, err)

		// 异步触发主节点健康检查
		go cluster.Manager.TriggerMasterHealthCheck()

		// 等待短暂时间，看是否能切换主节点
		time.Sleep(100 * time.Millisecond)

		// 重新检查当前节点是否已成为主节点
		if cluster.Manager.IsMaster() {
			log.Printf("[discovery] 当前节点已成为主节点，本地处理请求")
			// 递归调用自己，但这次会走主节点逻辑
			HandleDiscovery(c)
			return
		}

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "转发到主节点失败，主节点可能不可用",
		})
		return
	}
	defer resp.Body.Close()

	// 直接转发响应
	c.Status(resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 复制响应体
	io.Copy(c.Writer, resp.Body)
}
