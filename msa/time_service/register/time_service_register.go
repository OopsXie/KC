package register

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"msa/time_service/config"
	"net/http"
	"time"
)

// Register 向注册中心注册当前服务
func Register(cfg *config.Config) error {
	var failedAddresses []string

	for _, address := range cfg.Registry.Addresses {
		body := map[string]interface{}{
			"serviceName": cfg.Service.ServiceName,
			"serviceId":   cfg.Service.ServiceID,
			"ipAddress":   cfg.Service.IpAddress,
			"port":        cfg.Service.Port,
		}
		data, err := json.Marshal(body)
		if err != nil {
			log.Printf("[error] JSON 编码失败: %v", err)
			continue
		}

		url := fmt.Sprintf("%s/api/register", address)
		req, err := http.NewRequest("POST", url, bytes.NewReader(data))
		if err != nil {
			log.Printf("[error] 构造注册请求失败: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[error] 注册请求失败: %v", err)
			failedAddresses = append(failedAddresses, address)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("[success] 注册成功: %s -> %s", cfg.Service.ServiceID, address)
			return nil
		} else {
			log.Printf("[error] 注册失败，状态码: %d, 地址: %s", resp.StatusCode, address)
			failedAddresses = append(failedAddresses, address)
		}
	}

	return fmt.Errorf("[error] 所有注册中心地址均无法访问，失败地址: %v", failedAddresses)
}

// Unregister 注销当前服务
func Unregister(cfg *config.Config) {
	body := map[string]interface{}{
		"serviceName": cfg.Service.ServiceName,
		"serviceId":   cfg.Service.ServiceID,
		"ipAddress":   cfg.Service.IpAddress,
		"port":        cfg.Service.Port,
	}
	data, _ := json.Marshal(body)

	for _, address := range cfg.Registry.Addresses {
		url := fmt.Sprintf("%s/api/unregister", address)
		req, err := http.NewRequest("POST", url, bytes.NewReader(data))
		if err != nil {
			log.Printf("[error] 构造注销请求失败: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[error] 注销请求失败: %v", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("[success] 注销成功: %s -> %s", cfg.Service.ServiceID, address)
			return
		} else {
			log.Printf("[error] 注销失败，状态码: %d, 地址: %s", resp.StatusCode, address)
		}
	}

	log.Printf("[error] 所有注册中心地址均无法访问，注销失败")
}
