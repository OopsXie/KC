package handler

import (
	"log"
	"msa/registry/cluster"
	"msa/registry/models"
	"msa/registry/storage"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func HandleRegister(c *gin.Context) {
	var instance storage.ServiceInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		response := models.ErrorResponse(400, "请求体格式错误", nil)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 验证所有必要字段都不能为空
	if instance.ServiceName == "" {
		response := models.ErrorResponse(400, "serviceName为必填字段，不能为空", nil)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if instance.ServiceID == "" {
		response := models.ErrorResponse(400, "serviceId为必填字段，不能为空", nil)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if instance.IPAddress == "" {
		response := models.ErrorResponse(400, "ipAddress为必填字段，不能为空", nil)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if instance.Port == 0 {
		response := models.ErrorResponse(400, "port为必填字段，不能为空或0", nil)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 验证端口号范围
	if instance.Port < 1 || instance.Port > 65535 {
		response := models.ErrorResponse(400, "端口号必须在1-65535之间", nil)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 检查ServiceID是否已存在
	if storage.ServiceIDExists(instance.ServiceID) {
		conflictData := models.RegisterConflictData{
			ConflictServiceID: instance.ServiceID,
			Suggestion:        "请使用不同的serviceId或检查是否重复注册",
		}
		response := models.ErrorResponse(409, "serviceId已经存在", conflictData)
		c.JSON(http.StatusConflict, response)
		return
	}

	// 检查 ipAddress 和 port 是否已被占用
	if storage.IPPortExists(instance.IPAddress, instance.Port) {
		response := models.ErrorResponse(409, "该 ipAddress 和 port 已被其他服务使用", nil)
		c.JSON(http.StatusConflict, response)
		return
	}

	// 保存服务实例
	success := storage.SaveInstance(instance)
	if !success {
		conflictData := models.RegisterConflictData{
			ConflictServiceID: instance.ServiceID,
			Suggestion:        "该 serviceId 已被其他服务使用",
		}
		response := models.ErrorResponse(409, "serviceId已经存在", conflictData)
		c.JSON(http.StatusConflict, response)
		return
	}

	log.Printf("[success] Registered: %+v\n", instance)

	// 如果是从节点，直接返回成功响应（因为数据是转发到主节点处理的）
	if cluster.Manager != nil && !cluster.Manager.IsMaster() {
		// 从节点：构建响应数据（使用当前时间作为注册时间）
		now := time.Now().UTC()
		successData := models.RegisterSuccessData{
			ServiceName:          instance.ServiceName,
			ServiceID:            instance.ServiceID,
			IPAddress:            instance.IPAddress,
			Port:                 instance.Port,
			RegistrationTime:     now.Unix(),
			LastHeartbeatTime:    now.Unix(),
			RegistrationGMTTime:  now.Format("2006-01-02 15:04:05"),
			LastHeartbeatGMTTime: now.Format("2006-01-02 15:04:05"),
		}
		response := models.SuccessResponse(200, "注册成功", successData)
		c.JSON(http.StatusOK, response)
		return
	}

	// 主节点：获取已保存的服务实例（包含时间戳信息）
	savedInstance := storage.GetInstanceByServiceID(instance.ServiceID)
	if savedInstance == nil {
		response := models.ErrorResponse(500, "服务注册失败", nil)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// 返回成功响应
	successData := models.RegisterSuccessData{
		ServiceName:          savedInstance.ServiceName,
		ServiceID:            savedInstance.ServiceID,
		IPAddress:            savedInstance.IPAddress,
		Port:                 savedInstance.Port,
		RegistrationTime:     savedInstance.RegisteredAt,
		LastHeartbeatTime:    savedInstance.LastHeartbeat,
		RegistrationGMTTime:  savedInstance.RegisteredGMTTime,
		LastHeartbeatGMTTime: savedInstance.LastHeartbeatGMTTime,
	}
	response := models.SuccessResponse(200, "注册成功", successData)
	c.JSON(http.StatusOK, response)
}
