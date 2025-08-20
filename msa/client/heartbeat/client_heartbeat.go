package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"msa/client/config"
	"net/http"
	"time"
)

func StartHeartbeat(cfg *config.Config) {
	// 启动心跳服务
	interval := time.Duration(cfg.Client.HeartbeatInterval) * time.Second
	log.Printf("启动心跳服务，每 %d 秒发送一次", cfg.Client.HeartbeatInterval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		<-ticker.C

		// 构造心跳请求体
		body := map[string]interface{}{
			"serviceName": cfg.Client.ServiceName,
			"serviceId":   cfg.Client.ServiceID,
			"ipAddress":   cfg.Client.IpAddress,
			"port":        cfg.Client.Port,
		}

		data, err := json.Marshal(body)
		if err != nil {
			log.Printf("[error] JSON 编码失败: %v", err)
			continue
		}

		// 循环尝试所有注册中心地址
		success := false
		for _, address := range cfg.Registry.Addresses {
			url := fmt.Sprintf("%s/api/heartbeat", address)
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
			if err != nil {
				log.Printf("[error] 心跳发送失败: %v, 地址: %s", err, address)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				log.Printf("[success] 心跳成功: [%s:%d] -> %s", cfg.Client.IpAddress, cfg.Client.Port, url)
				success = true
				break
			} else {
				log.Printf("[error] 心跳失败，响应状态码: %d, 地址: %s", resp.StatusCode, address)
			}
		}

		if !success {
			log.Printf("[error] 所有注册中心地址均无法访问，心跳发送失败")
		}
	}

}
