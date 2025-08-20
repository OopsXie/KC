package handler

import (
	"fmt"
	"net/http"
	"time"

	"msa/time_service/config"
	"msa/time_service/models"

	"github.com/gin-gonic/gin"
)

type ResponseData struct {
	Result      string `json:"result"`
	ServiceName string `json:"serviceName"`
	ServiceId   string `json:"serviceId"`
	IpAddress   string `json:"ipAddress"`
	Port        int    `json:"port"`
}

func HandlerGetDateTime(c *gin.Context) {
	//style := c.DefaultQuery("style", "full")

	style := c.Query("style")
	msg := ""

	var result string
	switch style {
	case "full":
		result = time.Now().UTC().Format("2006-01-02 15:04:05")
		msg = "GMT-完整格式时间请求成功"
	case "date":
		result = time.Now().UTC().Format("2006-01-02")
		msg = "GMT-日期格式请求成功"
	case "time":
		result = time.Now().UTC().Format("15:04:05")
		msg = "GMT-时间格式请求成功"
	case "unix":
		result = fmt.Sprintf("%d", time.Now().UTC().UnixMilli())
		msg = "GMT-Unix时间戳请求成功"
	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "无效的 style 参数", gin.H{
			"supportedStyles": []string{"full", "date", "time", "unix"},
		}))

		return
	}

	responseData := ResponseData{
		Result:      result,
		ServiceName: config.Cfg.Service.ServiceName,
		ServiceId:   config.Cfg.Service.ServiceID,
		IpAddress:   config.Cfg.Service.IpAddress,
		Port:        config.Cfg.Service.Port,
	}

	c.JSON(http.StatusOK, models.SuccessResponse(msg, responseData))
}
