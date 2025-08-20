package handler

import (
	"fmt"
	"io"
	"log"
	"msa/registry/cluster"
	"msa/registry/models"
	"msa/registry/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleHeartbeat(c *gin.Context) {
	var request storage.ServiceInstance
	if err := c.ShouldBindJSON(&request); err != nil {
		response := models.ErrorResponse(400, "请求体格式错误", nil)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 验证ServiceID不能为空
	if request.ServiceID == "" {
		errorData := models.UnregisterErrorData{
			Suggestion: "请填写正确的serviceId",
		}
		response := models.ErrorResponse(400, "serviceId不能为空", errorData)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 根据ServiceID查找服务实例
	existingInstance := storage.GetInstanceByServiceID(request.ServiceID)
	if existingInstance == nil {
		errorData := models.UnregisterErrorData{
			ServiceID:  request.ServiceID,
			Suggestion: "请填写正确的serviceId",
		}
		response := models.ErrorResponse(404, "serviceId不存在", errorData)
		c.JSON(http.StatusNotFound, response)
		return
	}

	// 验证其他信息是否一致
	errorData := models.UnregisterErrorData{
		ServiceID: request.ServiceID,
	}

	// 验证 ServiceName 是否匹配
	if request.ServiceName == "" || request.ServiceName != existingInstance.ServiceName {
		errorData.FieldMismatch = "serviceName"
		errorData.Suggestion = fmt.Sprintf("serviceName不匹配，提供值: %s", request.ServiceName)
		response := models.ErrorResponse(400, "服务信息不匹配", errorData)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 验证 IPAddress 是否匹配
	if request.IPAddress == "" || request.IPAddress != existingInstance.IPAddress {
		errorData.FieldMismatch = "ipAddress"
		errorData.Suggestion = fmt.Sprintf("ipAddress不匹配，提供值: %s", request.IPAddress)
		response := models.ErrorResponse(400, "服务信息不匹配", errorData)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 验证 Port 是否匹配
	if request.Port == 0 || request.Port != existingInstance.Port {
		errorData.FieldMismatch = "port"
		errorData.Suggestion = fmt.Sprintf("port不匹配，提供值: %d", request.Port)
		response := models.ErrorResponse(400, "服务信息不匹配", errorData)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 更新心跳时间
	// 如果是从节点，尝试获取主节点的响应
	if cluster.Manager != nil && !cluster.Manager.IsMaster() {
		// 从节点：转发到主节点并直接返回主节点的响应
		resp, success := storage.UpdateHeartbeatForResponse(request.ServiceID, request)
		if !success {
			errorData := models.UnregisterErrorData{
				ServiceID:  request.ServiceID,
				Suggestion: "心跳更新失败",
			}
			response := models.ErrorResponse(500, "心跳更新失败", errorData)
			c.JSON(http.StatusInternalServerError, response)
			return
		}

		if resp != nil {
			// 直接转发主节点的响应
			defer resp.Body.Close()

			// 复制响应头
			for key, values := range resp.Header {
				for _, value := range values {
					c.Header(key, value)
				}
			}

			// 复制状态码和响应体
			c.Status(resp.StatusCode)
			io.Copy(c.Writer, resp.Body)
			return
		}
	}

	// 主节点或无集群时的处理
	success := storage.UpdateHeartbeat(request.ServiceID)
	if !success {
		errorData := models.UnregisterErrorData{
			ServiceID:  request.ServiceID,
			Suggestion: "心跳更新失败",
		}
		response := models.ErrorResponse(500, "心跳更新失败", errorData)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	log.Printf("[success] Heartbeat updated: ServiceID=%s\n", request.ServiceID)

	// 获取更新后的实例信息
	updatedInstance := storage.GetInstanceByServiceID(request.ServiceID)
	if updatedInstance == nil {
		response := models.ErrorResponse(500, "心跳更新成功但无法获取实例信息", nil)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// 返回成功响应
	successData := models.RegisterSuccessData{
		ServiceName:          updatedInstance.ServiceName,
		ServiceID:            updatedInstance.ServiceID,
		IPAddress:            updatedInstance.IPAddress,
		Port:                 updatedInstance.Port,
		RegistrationTime:     updatedInstance.RegisteredAt,
		LastHeartbeatTime:    updatedInstance.LastHeartbeat,
		RegistrationGMTTime:  updatedInstance.RegisteredGMTTime,
		LastHeartbeatGMTTime: updatedInstance.LastHeartbeatGMTTime,
	}
	response := models.SuccessResponse(200, "心跳更新成功", successData)
	c.JSON(http.StatusOK, response)
}
