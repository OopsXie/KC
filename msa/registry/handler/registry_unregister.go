package handler

import (
	"fmt"
	"log"
	"net/http"

	"msa/registry/models"
	"msa/registry/storage"

	"github.com/gin-gonic/gin"
)

func HandleUnregister(c *gin.Context) {
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

	if request.ServiceName != existingInstance.ServiceName {
		errorData.FieldMismatch = "serviceName"
		errorData.Suggestion = fmt.Sprintf("serviceName不匹配，提供值: %s", request.ServiceName)
		response := models.ErrorResponse(400, "服务信息不匹配", errorData)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if request.IPAddress == "" || request.IPAddress != existingInstance.IPAddress {
		errorData.FieldMismatch = "ipAddress"
		errorData.Suggestion = fmt.Sprintf("ipAddress不匹配，提供值: %s", request.IPAddress)
		response := models.ErrorResponse(400, "服务信息不匹配", errorData)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if request.Port != existingInstance.Port {
		errorData.FieldMismatch = "port"
		errorData.Suggestion = fmt.Sprintf("port不匹配，提供值: %d", request.Port)
		response := models.ErrorResponse(400, "服务信息不匹配", errorData)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// 删除服务实例
	success := storage.RemoveInstance(request.ServiceID)
	if !success {
		errorData := models.UnregisterErrorData{
			ServiceID:  request.ServiceID,
			Suggestion: "服务实例删除失败",
		}
		response := models.ErrorResponse(500, "注销失败", errorData)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	log.Printf("[success] Unregistered: ServiceID=%s\n", request.ServiceID)

	// 返回成功响应
	successData := models.UnregisterSuccessData{
		ServiceName: existingInstance.ServiceName,
		ServiceID:   existingInstance.ServiceID,
		IPAddress:   existingInstance.IPAddress,
		Port:        existingInstance.Port,
		Message:     "服务已成功注销",
	}
	response := models.SuccessResponse(200, "注销成功", successData)
	c.JSON(http.StatusOK, response)
}
