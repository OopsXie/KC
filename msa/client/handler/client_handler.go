package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"msa/client/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 注册中心返回结构
type DiscoveryResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ServiceName string `json:"serviceName"`
		TotalCount  int    `json:"totalCount"`
		Instances   []struct {
			ServiceName          string `json:"serviceName"`
			ServiceID            string `json:"serviceId"`
			IPAddress            string `json:"ipAddress"`
			Port                 int    `json:"port"`
			RegistrationTime     int64  `json:"registrationTime"`
			LastHeartbeatTime    int64  `json:"lastHeartbeatTime"`
			RegistrationGMTTime  string `json:"registrationGMTTime"`
			LastHeartbeatGMTTime string `json:"lastHeartbeatGMTTime"`
		} `json:"instances"`
	} `json:"data"`
}

// time-service 响应结构
type TimeServiceWrapper struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Result      string `json:"result"`
		ServiceName string `json:"serviceName"`
		ServiceID   string `json:"serviceId"`
		IPAddress   string `json:"ipAddress"`
		Port        int    `json:"port"`
	} `json:"data"`
}

// 统一响应结构
type APIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// 定义响应结构体，确保字段顺序
type OrderedResponse struct {
	Result            string `json:"result"`
	ClientServiceName string `json:"clientServiceName"`
	ClientServiceID   string `json:"clientServiceId"`
	ClientIPAddress   string `json:"clientIpAddress"`
	ClientPort        int    `json:"clientPort"`
	TimeServiceName   string `json:"timeServiceName"`
	TimeServiceID     string `json:"timeServiceId"`
	TimeIPAddress     string `json:"timeIpAddress"`
	TimePort          int    `json:"timePort"`
}

// 处理 /api/getInfo 请求
func HandleGetInfo(c *gin.Context) {
	// 1. 调用注册中心进行服务发现
	var discovery DiscoveryResponse
	var discoverySuccess bool

	for _, registryAddr := range config.Cfg.Registry.Addresses {
		discoveryURL := fmt.Sprintf("%s/api/discovery?name=time-service", registryAddr)

		resp, err := http.Get(discoveryURL)
		if err != nil {
			log.Printf("[error] 无法连接注册中心: %v, 地址: %s", err, registryAddr)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &discovery); err != nil {
			log.Printf("[error] 无法解析注册中心响应, 地址: %s", registryAddr)
			continue
		}

		if discovery.Data.TotalCount > 0 && len(discovery.Data.Instances) > 0 {
			discoverySuccess = true
			break
		}
	}

	if !discoverySuccess {
		c.JSON(http.StatusServiceUnavailable, APIResponse{
			Code: 503,
			Msg:  "未发现可用的 time-service 实例",
			Data: nil,
		})
		return
	}

	// 2. 调用 time-service 获取时间
	var ts TimeServiceWrapper
	var timeServiceSuccess bool
	var selectedInstance struct {
		ServiceName string
		ServiceID   string
		IPAddress   string
		Port        int
	}

	for _, instance := range discovery.Data.Instances {
		timeURL := fmt.Sprintf("http://%s:%d/api/getDateTime?style=full", instance.IPAddress, instance.Port)
		// 2. 调用 time-service 接口
		// port := discovery.Data.Instances[0].Port
		// timeURL := fmt.Sprintf("http://127.0.0.1:%d/api/getDateTime?style=full", instance.Port)

		timeResp, err := http.Get(timeURL)
		if err != nil {
			log.Printf("[error] 请求 time-service 失败: %v, 地址: %s:%d", err, instance.IPAddress, instance.Port)
			continue
		}
		defer timeResp.Body.Close()

		timeBody, _ := io.ReadAll(timeResp.Body)
		if err := json.Unmarshal(timeBody, &ts); err != nil || ts.Data.Result == "" {
			log.Printf("[error] time-service 响应格式错误, 地址: %s:%d", instance.IPAddress, instance.Port)
			continue
		}

		// 保存成功调用的时间服务实例信息
		selectedInstance.ServiceName = instance.ServiceName
		selectedInstance.ServiceID = instance.ServiceID
		selectedInstance.IPAddress = instance.IPAddress
		selectedInstance.Port = instance.Port

		timeServiceSuccess = true
		break
	}

	if !timeServiceSuccess {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Code: 500,
			Msg:  "无法连接 time-service",
			Data: nil,
		})
		return
	}

	// 3. 解析时间为北京时间（GMT+8）
	parsedTime, err := time.Parse("2006-01-02 15:04:05", ts.Data.Result)
	if err != nil {
		log.Println("[error] 无法解析时间格式:", err)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Code: 500,
			Msg:  "时间格式无效",
			Data: nil,
		})
		return
	}

	beijingTime := parsedTime.In(time.FixedZone("CST", 8*3600))
	formatted := beijingTime.Format("2006-01-02 15:04:05")

	// 4. 构造响应
	response := OrderedResponse{
		Result:            fmt.Sprintf("Hello Kingsoft Cloud Star Camp - %s - %s", config.Cfg.Client.ServiceID, formatted),
		ClientServiceName: config.Cfg.Client.ServiceName,
		ClientServiceID:   config.Cfg.Client.ServiceID,
		ClientIPAddress:   config.Cfg.Client.IpAddress,
		ClientPort:        config.Cfg.Client.Port,
		TimeServiceName:   selectedInstance.ServiceName,
		TimeServiceID:     selectedInstance.ServiceID,
		TimeIPAddress:     selectedInstance.IPAddress,
		TimePort:          selectedInstance.Port,
	}

	c.JSON(http.StatusOK, APIResponse{
		Code: 200,
		Msg:  "获取客户端时间成功",
		Data: response,
	})
}
