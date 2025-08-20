package register

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"msa/client/config"
	"net/http"
	"time"
)

func Register(cfg *config.Config) error {
	// 构造注册请求体
	body := map[string]interface{}{
		"serviceName": cfg.Client.ServiceName,
		"serviceId":   cfg.Client.ServiceID,
		"ipAddress":   cfg.Client.IpAddress,
		"port":        cfg.Client.Port,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("[error] JSON 编码失败: %v", err)
	}
	log.Printf("[info] 发送注册请求体: %s", string(data))

	// 循环尝试所有注册中心地址
	var failedAddresses []string
	for _, address := range cfg.Registry.Addresses {
		url := fmt.Sprintf("%s/api/register", address)
		req, err := http.NewRequest("POST", url, bytes.NewReader(data))
		if err != nil {
			log.Printf("[error] 构造注册请求失败: %v", err)
			failedAddresses = append(failedAddresses, address)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[error] 发送注册请求失败: %v, 地址: %s", err, address)
			failedAddresses = append(failedAddresses, address)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("[success] 注册成功: %s -> %s", cfg.Client.ServiceID, address)
			return nil
		} else {
			log.Printf("[error] 注册失败，响应状态码: %d, 地址: %s", resp.StatusCode, address)
			failedAddresses = append(failedAddresses, address)
		}
	}

	return fmt.Errorf("[error] 所有注册中心地址均无法访问，失败地址: %v", failedAddresses)
}

// Unregister 注销当前服务
func Unregister(cfg *config.Config) {
	// 构建注销请求体
	body := map[string]interface{}{
		"serviceName": cfg.Client.ServiceName,
		"serviceId":   cfg.Client.ServiceID,
		"ipAddress":   cfg.Client.IpAddress,
		"port":        cfg.Client.Port,
	}
	data, _ := json.Marshal(body)

	// 循环尝试所有注册中心地址
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
			log.Printf("[error] 注销请求失败: %v, 地址: %s", err, address)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("[success] 注销成功: %s -> %s", cfg.Client.ServiceID, address)
			return
		} else {
			log.Printf("[error] 注销失败，响应状态码: %d, 地址: %s", resp.StatusCode, address)
		}
	}

	log.Printf("[error] 所有注册中心地址均无法访问，注销失败")
}
